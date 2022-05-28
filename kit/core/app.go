package core

import (
	"context"
	"fmt"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/iancoleman/strcase"
	"github.com/juju/ratelimit"
	"github.com/lucas-clemente/quic-go/http3"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/stacktasec/circle/zlog"
	"io/fs"
	"net/http"
	"reflect"
	"strings"
	"sync/atomic"
	"time"
)

const ServiceSuffix = "service"
const RequestID = "RequestID"

const (
	respTypeJson   = "json"
	respTypeStream = "stream"
)

type Request interface {
	Validate() error
}

type knownError struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func (k knownError) Error() string {
	return fmt.Sprintf("[Status] %s [Message] %s", k.Status, k.Message)
}

func (k knownError) Is(err error) bool {
	nErr, ok := err.(knownError)
	if !ok {
		return false
	}

	return k.Status == nErr.Status && k.Message == nErr.Message
}

func MakeKnownError(status, message string) error {
	return knownError{
		Status:  status,
		Message: message,
	}
}

type versionGroup struct {
	mainVersion    int
	stableServices []any
	betaServices   []any
	alphaServices  []any
}

func NewVersionGroup(mainVersion int) *versionGroup {
	if mainVersion < 1 {
		panic("main version must larger than one")
	}

	return &versionGroup{
		mainVersion: mainVersion,
	}
}

func (v *versionGroup) SetStable(services ...any) {
	if len(services) == 0 {
		panic("services must more than one")
	}

	v.stableServices = append(v.stableServices, services...)
}

func (v *versionGroup) SetBeta(services ...any) {
	if len(services) == 0 {
		panic("services must more than one")
	}

	v.betaServices = append(v.betaServices, services...)
}

func (v *versionGroup) SetAlpha(services ...any) {
	if len(services) == 0 {
		panic("services must more than one")
	}

	v.alphaServices = append(v.alphaServices, services...)
}

type app struct {
	versionGroups map[int]*versionGroup
	engine        *gin.Engine
	baseURL       string
	baseGroup     *gin.RouterGroup

	addr string

	enableTLS  bool
	enableQUIC bool
	cert       string
	key        string

	ctxTimeout time.Duration // default 30s

	enableRateLimit bool
	limitNum        int64
	limitBucket     *ratelimit.Bucket

	enableMonitor bool
	loadNum       float64
	loadValue     atomic.Value
}

func NewApp() *app {
	return &app{versionGroups: make(map[int]*versionGroup), ctxTimeout: time.Second * 30, addr: ":8080"}
}

func (a *app) SetAddr(addr string) {
	a.addr = addr
}

func (a *app) SetTLS(cert, key string) {
	a.enableTLS = true
	a.cert = cert
	a.key = key
}

func (a *app) SetQUIC(cert, key string) {
	a.enableQUIC = true
	a.cert = cert
	a.key = key
}

func (a *app) SetBaseURL(url string) {
	a.baseURL = url
}

func (a *app) SetGroup(g *versionGroup) {
	if g == nil {
		panic("version group must be non-nil")
	}

	_, ok := a.versionGroups[g.mainVersion]
	if ok {
		panic("duplicated main version")
	}
	a.versionGroups[g.mainVersion] = g
}

func (a *app) SetCtxTimeout(d time.Duration) {
	a.ctxTimeout = d
}

func (a *app) SetMonitor(num float64) {
	a.enableMonitor = true
	a.loadNum = num
}

func (a *app) EnableRateLimit(num int) {
	a.enableRateLimit = true
	a.limitNum = int64(num)
}

func (a *app) Build() {
	if len(a.versionGroups) == 0 {
		panic("must call set group")
	}

	r := gin.Default()

	if a.enableMonitor {
		r.Use(func(c *gin.Context) {
			value := a.loadValue.Load()
			if value == true {
				c.AbortWithStatus(http.StatusServiceUnavailable)
				return
			}
			c.Next()
		})
	}

	if a.enableRateLimit {
		a.limitBucket = ratelimit.NewBucketWithQuantum(time.Millisecond, a.limitNum, 1)
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

	baseGroup := r.Group(a.baseURL)
	a.baseGroup = baseGroup

	for _, g := range a.versionGroups {
		a.fillGroups(*g)
	}

	r.Use(gzip.Gzip(gzip.DefaultCompression))

	a.engine = r
}

func (a *app) Run() {
	if a.engine == nil {
		panic("must call build")
	}

	if a.enableMonitor {
		a.monitor()
	}

	httpServer := http.Server{
		Addr:           a.addr,
		Handler:        a.engine,
		ReadTimeout:    time.Second * 10,
		WriteTimeout:   time.Second * 10,
		MaxHeaderBytes: 1 << 16,
	}
	http3Server := http3.Server{
		Server: &httpServer,
	}

	if a.enableQUIC {
		zlog.Infof("run http3 server listen on %s", a.addr)
		if err := http3Server.ListenAndServeTLS(a.addr, a.cert); err != nil {
			panic(err)
		}
	}

	if a.enableTLS {
		zlog.Infof("run https server listen on %s", a.addr)
		if err := httpServer.ListenAndServeTLS(a.cert, a.key); err != nil {
			panic(err)
		}
	}

	zlog.Infof("run http server listen on %s", a.addr)
	if err := httpServer.ListenAndServe(); err != nil {
		panic(err)
	}
}

func (a *app) monitor() {

	go func() {
		defer func() {
			if r := recover(); r != nil {
				zlog.Panicf(r)
			}
		}()

		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()

		for t := range ticker.C {
			cpuPercents, err := cpu.Percent(time.Second, true)
			if err != nil || len(cpuPercents) == 0 {
				zlog.Errorf("monitor cpu percent error %s,%s", t, err)
				a.loadValue.Store(false)
				continue
			}

			var sum float64
			for _, u := range cpuPercents {
				sum += u
			}
			if sum/float64(len(cpuPercents)) > 80 {
				a.loadValue.Store(true)
				continue
			}

			stat, err := mem.VirtualMemory()
			if err != nil {
				zlog.Errorf("monitor mem usage error %s,%s", t, err)
				a.loadValue.Store(false)
				continue
			}
			if stat.UsedPercent > 80 {
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

func (a *app) fillActions(g *gin.RouterGroup, service any) {

	actions := makeActions(service)

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

			ctx := context.Background()

			reqID := uuid.NewString()
			ctx = context.WithValue(ctx, RequestID, reqID)
			ctx, _ = context.WithTimeout(ctx, a.ctxTimeout)

			c.Writer.Header().Set("X-Request-ID", reqID)

			ctxValue := reflect.ValueOf(ctx)
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
			result := rtnList[0]
			if action.respType == respTypeStream {
				file := result.Interface().(fs.File)
				stat, err := file.Stat()
				if err != nil {
					c.AbortWithStatus(http.StatusInternalServerError)
					return
				}

				c.DataFromReader(http.StatusOK, stat.Size(), "application/octet-stream", file, nil)
				return
			}

			value := result.Interface()
			if value == nil {
				c.Status(http.StatusNotFound)
				return
			}
			c.JSON(http.StatusOK, gin.H{"result": value})
		})
	}
}

// 获取该结构体里的所有receiver method
func makeActions(service any) []reflectAction {

	rawType := reflect.TypeOf(service)
	if rawType.Kind() != reflect.Struct {
		panic("service should be struct")
	}

	rawTypeName := strings.ToLower(rawType.Name())
	if !strings.HasSuffix(rawTypeName, ServiceSuffix) {
		panic("struct must have suffix [Service]")
	}
	serviceName := strings.ReplaceAll(rawTypeName, ServiceSuffix, "")

	serviceValue := reflect.New(reflect.TypeOf(service))
	serviceType := serviceValue.Type()

	numMethods := serviceType.NumMethod()
	if numMethods == 0 {
		return nil
	}

	var actions []reflectAction
	for i := 0; i < numMethods; i++ {
		// 获得方法
		methodType := serviceType.Method(i)

		// 必须满足 导出 有 2个入参 2个出参
		// 入参是context.Context Request 则认定为待映射方法
		// 此时 出参 必须是 结构体指针 和 error
		if !methodType.IsExported() {
			continue
		}

		// 检查参数是否符合规定格式
		inParams := methodType.Type.NumIn()
		outParams := methodType.Type.NumOut()
		if inParams != 3 || outParams != 2 {
			continue
		}

		// 必须满足 如下 四元组
		in1 := methodType.Type.In(1)
		in2 := methodType.Type.In(2)
		out0 := methodType.Type.Out(0)
		out1 := methodType.Type.Out(1)

		if !satisfyContext(in1) {
			continue
		}

		if ok := satisfyRequest(in2); !ok {
			continue
		}

		respType := mustResponse(out0)

		mustError(out1)

		methodValue := serviceValue.Method(i)
		action := reflectAction{
			serviceName: strcase.ToSnake(serviceName),
			methodName:  strcase.ToSnake(methodType.Name),
			bindData:    reflect.New(in2).Interface(),
			methodData:  methodValue,
			respType:    respType,
		}

		actions = append(actions, action)
	}

	return actions
}

func satisfyContext(t reflect.Type) bool {
	ctxType := reflect.TypeOf((*context.Context)(nil)).Elem()
	return t.AssignableTo(ctxType)
}

func satisfyRequest(t reflect.Type) bool {
	// 值类型 需要先变成指针
	pt := reflect.New(t)
	pti := reflect.TypeOf(pt.Interface())
	reqType := reflect.TypeOf((*Request)(nil)).Elem()
	return pti.Implements(reqType)
}

func mustResponse(t reflect.Type) string {
	if t.Kind() != reflect.Pointer || t.Elem().Kind() != reflect.Struct {
		panic("this position type must be a pointer of struct")
	}

	// 指针类型 直接用
	streamType := reflect.TypeOf((*fs.File)(nil)).Elem()
	if t.Implements(streamType) {
		return respTypeStream
	}

	return respTypeJson
}

func mustError(t reflect.Type) {
	errType := reflect.TypeOf((*error)(nil)).Elem()
	if !t.Implements(errType) {
		panic("this position type must be error")
	}
}
