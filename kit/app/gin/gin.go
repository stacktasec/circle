package gin

import (
	"context"
	"fmt"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/iancoleman/strcase"
	"github.com/juju/ratelimit"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/stacktasec/circle/kit/app/internal"
	"github.com/stacktasec/circle/kit/klog"
	"go.uber.org/dig"
	"io/fs"
	"net/http"
	"reflect"
	"strings"
	"sync/atomic"
	"time"
)

const keyRequestID = "X-Request-ID"

const (
	respTypeJson   = "json"
	respTypeStream = "stream"
)

type App struct {
	container     *dig.Container
	options       internal.Options
	versionGroups map[int]*internal.VersionGroup
	engine        *gin.Engine
	limitBucket   *ratelimit.Bucket
	loadValue     atomic.Value
}

func NewApp(opts ...internal.AppOption) *App {

	o := &internal.Options{}

	for _, opt := range opts {
		opt.Apply(o)
	}

	o.Ensure()

	return &App{container: dig.New(), options: *o, versionGroups: make(map[int]*internal.VersionGroup)}
}

func (a *App) Map(groups ...*internal.VersionGroup) {
	for _, g := range groups {
		_, ok := a.versionGroups[g.MainVersion]
		if ok {
			panic("duplicated main version")
		}
		a.versionGroups[g.MainVersion] = g
	}
}

func (a *App) Provide(constructors ...any) {
	for _, c := range constructors {
		verifyConstructor(c)
	}

	for _, item := range constructors {
		if err := a.container.Provide(item); err != nil {
			panic(err)
		}
	}
}

func (a *App) Run() {
	a.build()

	if a.options.EnableOverloadBreak {
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

	if a.options.EnableOverloadBreak {
		r.Use(func(c *gin.Context) {
			value := a.loadValue.Load()
			if value == true {
				c.AbortWithStatus(http.StatusServiceUnavailable)
				return
			}
			c.Next()
		})
	}

	if a.options.EnableRateLimit {
		a.limitBucket = ratelimit.NewBucketWithQuantum(a.options.FillInterval, a.options.Capacity, a.options.Quantum)
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
				a.loadValue.Store(false)
				continue
			}

			var sum float64
			for _, u := range cpuPercents {
				sum += u
			}
			if sum/float64(len(cpuPercents)) > a.options.MaxCpuPercent {
				a.loadValue.Store(true)
				continue
			}

			stat, err := mem.VirtualMemory()
			if err != nil {
				klog.Error("watch mem usage error %s,%s", t, err)
				a.loadValue.Store(false)
				continue
			}
			if stat.UsedPercent > a.options.MaxMemPercent {
				a.loadValue.Store(true)
				continue
			}
		}
	}()
}

type reflectAction struct {
	// Service 资源名称
	serviceName string
	// 方法名
	methodName string
	// 用来绑定的数据
	bindData any
	// 用来调用的
	methodValue reflect.Value
	// 请求 返回类型
	respType string
}

func (a *App) makeActions(constructor any) []reflectAction {

	verifyConstructor(constructor)

	funcType := reflect.TypeOf(constructor)
	funcValue := reflect.ValueOf(constructor)

	numIn := funcType.NumIn()
	var params []reflect.Type
	for i := 0; i < numIn; i++ {
		t := funcType.In(i)
		params = append(params, reflect.New(t).Elem().Type())
	}

	var rtn any

	invokerType := reflect.FuncOf(params, nil, false)
	invokerValue := reflect.MakeFunc(invokerType, func(args []reflect.Value) (results []reflect.Value) {
		rtnList := funcValue.Call(args)
		rtn = rtnList[0].Interface()
		return nil
	})

	if err := a.container.Invoke(invokerValue.Interface()); err != nil {
		panic(err)
	}

	pointerValue := reflect.ValueOf(rtn)
	pointerType := pointerValue.Type()

	var actions []reflectAction
	for i := 0; i < pointerType.NumMethod(); i++ {
		// 获得方法
		method := pointerType.Method(i)

		// 必须满足 导出 有 2个入参 2个出参
		// 入参是context.Context Request 则认定为待映射方法
		// 此时 出参 必须是 结构体指针 和 error
		if !method.IsExported() {
			continue
		}

		methodType := method.Type
		// 检查参数是否符合规定格式
		inParams := methodType.NumIn()
		outParams := methodType.NumOut()
		if inParams != 3 || outParams != 2 {
			continue
		}

		// 必须满足 如下 四元组
		in1 := methodType.In(1)
		in2 := methodType.In(2)
		out0 := methodType.Out(0)
		out1 := methodType.Out(1)

		if !satisfyContext(in1) {
			continue
		}

		if !satisfyRequest(in2) {
			continue
		}

		respType := mustResponse(out0)

		mustError(out1)

		svcName, methodName := a.makeName(pointerType.Elem().Name(), method.Name)
		action := reflectAction{
			serviceName: svcName,
			methodName:  methodName,
			bindData:    reflect.New(in2).Interface(),
			methodValue: pointerValue.Method(i),
			respType:    respType,
		}

		actions = append(actions, action)
	}

	return actions
}

func (a *App) makeName(resource, action string) (string, string) {
	lr := strings.ToLower(resource)

	for _, s := range a.options.Suffixes {
		if strings.HasSuffix(lr, s) {
			lr = strings.ReplaceAll(lr, s, "")
			break
		}
	}

	return strcase.ToSnake(lr), strcase.ToSnake(action)
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

	actions := a.makeActions(constructor)

	for _, action := range actions {

		g.POST(fmt.Sprintf("/%s/%s", action.serviceName, action.methodName), func(c *gin.Context) {
			if ok := a.handleInterceptors(c); !ok {
				return
			}

			req := action.bindData
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
			rtnList := action.methodValue.Call([]reflect.Value{ctxValue, reqValue})

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
			if action.respType == respTypeStream {
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
