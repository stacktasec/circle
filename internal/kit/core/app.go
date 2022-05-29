package core

import (
	"context"
	"fmt"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/juju/ratelimit"
	"github.com/lucas-clemente/quic-go/http3"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/stacktasec/circle/internal/kit/zlog"
	"io/fs"
	"net/http"
	"reflect"
	"sync/atomic"
	"time"
)

const keyRequestID = "X-Request-ID"

const (
	respTypeJson   = "json"
	respTypeStream = "stream"
)

type options struct {
	appName string
	ctxFunc func() context.Context

	addr string

	enableTLS  bool
	enableQUIC bool
	cert       string
	key        string

	baseURL    string
	ctxTimeout time.Duration

	enableRateLimit bool
	fillInterval    time.Duration
	capacity        int64
	quantum         int64

	enableOverloadClose bool
	maxCpuPercent       float64
	maxMemPercent       float64
}

func (o *options) ensure() {
	if o.addr == "" {
		o.addr = ":8080"
	}

	if o.ctxTimeout == 0 {
		o.ctxTimeout = time.Second * 30
	}
}

type AppOption interface {
	apply(*options)
}

// jsOptFn configures an option for the JetStreamContext.
type appOptionFunc func(opts *options)

func (opt appOptionFunc) apply(opts *options) {
	opt(opts)
}

func WithAppName(name string) AppOption {
	return appOptionFunc(func(opts *options) {
		opts.appName = name
	})
}

func WithCtxFunc(f func() context.Context) AppOption {
	return appOptionFunc(func(opts *options) {
		opts.ctxFunc = f
	})
}

func WithAddr(addr string) AppOption {
	return appOptionFunc(func(opts *options) {
		opts.addr = addr
	})
}

func WithTLS(cert, key string) AppOption {
	return appOptionFunc(func(opts *options) {
		opts.enableTLS = true
		opts.cert = cert
		opts.key = key
	})
}

func WithQUIC(cert, key string) AppOption {
	return appOptionFunc(func(opts *options) {
		opts.enableQUIC = true
		opts.cert = cert
		opts.key = key
	})
}

func WithBaseURL(url string) AppOption {
	return appOptionFunc(func(opts *options) {
		opts.baseURL = url
	})
}

func WithCtxTimeout(d time.Duration) AppOption {
	return appOptionFunc(func(opts *options) {
		opts.ctxTimeout = d
	})
}

func WithRateLimit(fillInterval time.Duration, capacity, quantum int) AppOption {
	return appOptionFunc(func(opts *options) {
		opts.enableRateLimit = true
		opts.fillInterval = fillInterval
		opts.capacity = int64(capacity)
		opts.quantum = int64(quantum)
	})
}

func WithOverloadClose(maxCpu, maxMem float64) AppOption {
	return appOptionFunc(func(opts *options) {
		opts.enableOverloadClose = true
		opts.maxCpuPercent = maxCpu
		opts.maxMemPercent = maxMem
	})
}

type app struct {
	options       options
	versionGroups map[int]*versionGroup
	engine        *gin.Engine
	baseGroup     *gin.RouterGroup
	limitBucket   *ratelimit.Bucket
	loadValue     atomic.Value
}

func NewApp(opts ...AppOption) *app {
	o := &options{}

	for _, opt := range opts {
		opt.apply(o)
	}

	o.ensure()

	return &app{options: *o, versionGroups: make(map[int]*versionGroup)}
}

func (a *app) Map(groups ...*versionGroup) {
	for _, g := range groups {
		_, ok := a.versionGroups[g.mainVersion]
		if ok {
			panic("duplicated main version")
		}
		a.versionGroups[g.mainVersion] = g
	}
}

func (a *app) build() {

	r := gin.Default()

	if a.options.enableOverloadClose {
		r.Use(func(c *gin.Context) {
			value := a.loadValue.Load()
			if value == true {
				c.AbortWithStatus(http.StatusServiceUnavailable)
				return
			}
			c.Next()
		})
	}

	if a.options.enableRateLimit {
		a.limitBucket = ratelimit.NewBucketWithQuantum(a.options.fillInterval, a.options.capacity, a.options.quantum)
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

	defaultRoute(r, a.options.appName)

	baseGroup := r.Group(a.options.baseURL)
	a.baseGroup = baseGroup

	for _, g := range a.versionGroups {
		a.fillGroups(*g)
	}

	r.Use(gzip.Gzip(gzip.DefaultCompression))

	a.engine = r
}

func defaultRoute(r *gin.Engine, appName string) {
	r.GET("/", func(c *gin.Context) {
		welcomeMsg := "Welcome"
		if appName != "" {
			welcomeMsg = fmt.Sprintf("%s to %s", welcomeMsg, appName)
		}

		c.String(http.StatusOK, welcomeMsg)
	})
}

func (a *app) Run() {
	a.build()

	if a.options.enableOverloadClose {
		a.watch()
	}

	httpServer := http.Server{
		Addr:           a.options.addr,
		Handler:        a.engine,
		ReadTimeout:    time.Second * 10,
		WriteTimeout:   time.Second * 10,
		MaxHeaderBytes: 1 << 16,
	}
	http3Server := http3.Server{
		Server: &httpServer,
	}

	if a.options.enableQUIC {
		zlog.Infof("http3 server is listening on %s", a.options.addr)
		if err := http3Server.ListenAndServeTLS(a.options.addr, a.options.cert); err != nil {
			panic(err)
		}
	}

	if a.options.enableTLS {
		zlog.Infof("https server is listening on %s", a.options.addr)
		if err := httpServer.ListenAndServeTLS(a.options.cert, a.options.key); err != nil {
			panic(err)
		}
	}

	zlog.Infof("http server is listening on %s", a.options.addr)
	if err := httpServer.ListenAndServe(); err != nil {
		panic(err)
	}
}

func (a *app) watch() {

	go func() {
		defer func() {
			if r := recover(); r != nil {
				zlog.Panicf(r)
			}
		}()

		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()

		for t := range ticker.C {
			cpuPercents, err := cpu.Percent(time.Second*5, true)
			if err != nil || len(cpuPercents) == 0 {
				zlog.Errorf("watch cpu percent error %s,%s", t, err)
				a.loadValue.Store(false)
				continue
			}

			var sum float64
			for _, u := range cpuPercents {
				sum += u
			}
			if sum/float64(len(cpuPercents)) > a.options.maxCpuPercent {
				a.loadValue.Store(true)
				continue
			}

			stat, err := mem.VirtualMemory()
			if err != nil {
				zlog.Errorf("watch mem usage error %s,%s", t, err)
				a.loadValue.Store(false)
				continue
			}
			if stat.UsedPercent > a.options.maxMemPercent {
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
	methodData reflect.Value
	// 请求 返回类型
	respType string
}

func (a *app) fillGroups(vg versionGroup) {

	for _, item := range vg.stableServices {
		g := a.baseGroup.Group(fmt.Sprintf("/v%d", vg.mainVersion))
		a.fillActions(g, item)
	}

	for _, item := range vg.betaServices {
		g := a.baseGroup.Group(fmt.Sprintf("/v%dbeta", vg.mainVersion))
		a.fillActions(g, item)
	}

	for _, item := range vg.alphaServices {
		g := a.baseGroup.Group(fmt.Sprintf("/v%dalpha", vg.mainVersion))
		a.fillActions(g, item)
	}
}

func (a *app) fillActions(g *gin.RouterGroup, impl any) {

	actions := makeActions(impl)

	for _, action := range actions {

		g.POST(fmt.Sprintf("/%s/%s", action.serviceName, action.methodName), func(c *gin.Context) {
			req := action.bindData
			if err := c.ShouldBind(&req); err != nil {
				c.AbortWithStatus(http.StatusBadRequest)
				return
			}

			i := req.(Request)
			if err := i.Validate(); err != nil {
				c.AbortWithStatus(http.StatusBadRequest)
				return
			}

			var ctx context.Context
			if a.options.ctxFunc != nil {
				ctx = a.options.ctxFunc()
			} else {
				ctx = context.Background()
			}

			reqID := uuid.NewString()
			ctx = context.WithValue(ctx, keyRequestID, reqID)
			timeoutCtx, cancel := context.WithTimeout(ctx, a.options.ctxTimeout)
			defer cancel()

			c.Writer.Header().Set(keyRequestID, reqID)

			ctxValue := reflect.ValueOf(timeoutCtx)
			reqValue := reflect.ValueOf(req).Elem()
			rtnList := action.methodData.Call([]reflect.Value{ctxValue, reqValue})

			// 判断第二个值 是自定义错误
			// 还是原生error
			errValue := rtnList[1].Interface()
			if errValue != nil {
				if errValue == context.DeadlineExceeded {
					c.AbortWithStatus(http.StatusGatewayTimeout)
					return
				}

				err, ok := errValue.(knownError)
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
