package gin

import (
	"context"
	"fmt"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/juju/ratelimit"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/stacktasec/circle/kit/app/internal"
	"github.com/stacktasec/circle/kit/app/klog"
	"go.uber.org/dig"
	"io/fs"
	"net/http"
	"reflect"
	"sync/atomic"
	"time"
)

const keyRequestID = "X-Request-ID"

type App struct {
	container     *dig.Container
	options       internal.Options
	versionGroups map[int]*internal.VersionGroup
	engine        *gin.Engine
	limitBucket   *ratelimit.Bucket
	overload      atomic.Value
}

func NewApp(opts ...internal.AppOption) *App {

	o := &internal.Options{}

	for _, opt := range opts {
		opt.Apply(o)
	}

	o.Ensure()

	return &App{container: dig.New(), options: *o, versionGroups: make(map[int]*internal.VersionGroup)}
}

func (a *App) Provide(constructors ...any) {
	internal.LoadConstructors(a.container, constructors...)
}

func (a *App) Map(groups ...*internal.VersionGroup) {
	internal.LoadGroups(a.versionGroups, groups...)
}

func (a *App) Run() {
	a.build()

	if a.options.EnableLoadLimit {
		a.watch()
	}

	httpServer := http.Server{
		Addr:           a.options.Addr,
		Handler:        a.engine,
		ReadTimeout:    time.Second * 10,
		WriteTimeout:   time.Second * 10,
		MaxHeaderBytes: 1 << 16,
	}

	if a.options.EnableTLS {
		klog.Info("https server is listening on %s", a.options.Addr)
		if err := httpServer.ListenAndServeTLS(a.options.Cert, a.options.Key); err != nil {
			panic(err)
		}
	}

	klog.Info("http server is listening on %s", a.options.Addr)
	if err := httpServer.ListenAndServe(); err != nil {
		panic(err)
	}
}

func (a *App) build() {

	r := gin.Default()

	if a.options.EnableLoadLimit {
		r.Use(func(c *gin.Context) {
			value := a.overload.Load()
			if value == true {
				c.AbortWithStatus(http.StatusServiceUnavailable)
				return
			}
			c.Next()
		})
	}

	if a.options.EnableRateLimit {
		a.limitBucket = ratelimit.NewBucket(a.options.FillInterval, a.options.Capacity)
		r.Use(func(c *gin.Context) {
			count := a.limitBucket.TakeAvailable(1)
			if count == 0 {
				c.AbortWithStatus(http.StatusTooManyRequests)
				return
			}
			c.Next()
		})
	}

	r.NoRoute(func(c *gin.Context) {
		c.AbortWithStatus(http.StatusNotImplemented)
	})

	r.Use(cors.Default())

	a.discovery(r)

	for _, g := range a.versionGroups {
		a.fillGroups(r.Group(a.options.BaseURL), g)
	}

	r.Use(gzip.Gzip(gzip.DefaultCompression))

	a.engine = r
}

func (a *App) discovery(r *gin.Engine) {
	r.GET("/", func(c *gin.Context) {
		welcomeMsg := "Welcome"
		c.String(http.StatusOK, welcomeMsg)
	})
}

func (a *App) watch() {

	go func() {
		defer func() {
			if r := recover(); r != nil {
				klog.Error(r)
			}
		}()

		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()

		for t := range ticker.C {
			cpuPercents, err := cpu.Percent(time.Second*5, true)
			if err != nil || len(cpuPercents) == 0 {
				klog.Error("watch cpu percent error %s,%s", t, err)
				a.overload.Store(false)
				continue
			}

			var sum float64
			for _, u := range cpuPercents {
				sum += u
			}
			if sum/float64(len(cpuPercents)) > a.options.MaxCpuPercent {
				a.overload.Store(true)
				continue
			}

			stat, err := mem.VirtualMemory()
			if err != nil {
				klog.Error("watch mem usage error %s,%s", t, err)
				a.overload.Store(false)
				continue
			}
			if stat.UsedPercent > a.options.MaxMemPercent {
				a.overload.Store(true)
				continue
			}
		}
	}()
}

func (a *App) fillGroups(routerGroup *gin.RouterGroup, vg *internal.VersionGroup) {

	for _, constructor := range vg.StableConstructors {
		g := routerGroup.Group(fmt.Sprintf("/v%d", vg.MainVersion))
		a.fillActions(g, constructor)
	}

	for _, constructor := range vg.BetaConstructors {
		g := routerGroup.Group(fmt.Sprintf("/v%dbeta", vg.MainVersion))
		a.fillActions(g, constructor)
	}

	for _, constructor := range vg.AlphaConstructors {
		g := routerGroup.Group(fmt.Sprintf("/v%dalpha", vg.MainVersion))
		a.fillActions(g, constructor)
	}
}

func (a *App) fillActions(g *gin.RouterGroup, constructor any) {

	actions := internal.MakeReflect(a.container, constructor, a.options.Suffixes)

	for _, action := range actions {

		g.POST(fmt.Sprintf("/%s/%s", action.ServiceName, action.MethodName), func(c *gin.Context) {
			if ok := a.handleInterceptors(c); !ok {
				return
			}

			req := action.BindData
			if err := c.ShouldBind(&req); err != nil {
				c.AbortWithStatus(http.StatusBadRequest)
				return
			}

			i := req.(internal.Request)
			if err := i.Validate(); err != nil {
				c.AbortWithStatus(http.StatusBadRequest)
				return
			}

			ctx := context.Background()

			reqID := uuid.NewString()
			ctx = context.WithValue(ctx, keyRequestID, reqID)
			timeoutCtx, cancel := context.WithTimeout(ctx, a.options.CtxTimeout)
			defer cancel()

			c.Writer.Header().Set(keyRequestID, reqID)

			ctxValue := reflect.ValueOf(timeoutCtx)
			reqValue := reflect.ValueOf(req).Elem()
			rtnList := action.MethodValue.Call([]reflect.Value{ctxValue, reqValue})

			// 判断第二个值 是自定义错误
			// 还是原生error
			errValue := rtnList[1].Interface()
			if errValue != nil {
				if errValue == context.DeadlineExceeded {
					c.AbortWithStatus(http.StatusGatewayTimeout)
					return
				}

				err, ok := errValue.(internal.KnownError)
				if ok {
					c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": err})
					return
				} else {
					c.AbortWithStatus(http.StatusInternalServerError)
					return
				}
			}

			result := rtnList[0].Interface()
			if action.RespType == internal.RespTypeStream {
				file := result.(fs.File)
				stat, err := file.Stat()
				if err != nil {
					c.AbortWithStatus(http.StatusInternalServerError)
					return
				}

				c.DataFromReader(http.StatusOK, stat.Size(), "application/octet-stream", file, nil)
				return
			}

			if result == nil {
				c.Status(http.StatusNotFound)
				return
			}
			c.JSON(http.StatusOK, gin.H{"result": result})
		})
	}
}

func (a *App) handleInterceptors(c *gin.Context) bool {
	h := c.Request.Header

	if a.options.IDInterceptor != nil {
		if err := a.options.IDInterceptor(h); err != nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return false
		}

		// 隐含：必须有身份 才有权限
		if a.options.PermInterceptor != nil {
			if err := a.options.PermInterceptor(h); err != nil {
				c.AbortWithStatus(http.StatusForbidden)
				return false
			}
		}
	}

	return true
}
