package core

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/iancoleman/strcase"
	"github.com/lucas-clemente/quic-go/http3"
	"io/fs"
	"net/http"
	"reflect"
	"strings"
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

const (
	respTypeJson   = "json"
	respTypeStream = "stream"
)

type app struct {
	engine *gin.Engine
	groups map[int]*versionGroup

	addr string
	cert string
	key  string

	enableQUIC bool
}

func NewApp() *app {
	return &app{groups: make(map[int]*versionGroup)}
}

func (a *app) SetAddr(addr string) {
	a.addr = addr
}

func (a *app) SetCertAndKey(cert, key string) {
	a.cert = cert
	a.key = key
}

func (a *app) SetGroup(g *versionGroup) {
	if g == nil {
		panic("version group must be non-nil")
	}

	_, ok := a.groups[g.mainVersion]
	if ok {
		panic("duplicated main version")
	}
	a.groups[g.mainVersion] = g
}

func (a *app) SetQUIC() {
	a.enableQUIC = true
}

func (a *app) Build() {
	if len(a.groups) == 0 {
		panic("must call set group")
	}

	r := gin.Default()

	for _, g := range a.groups {
		fillGroups(r, *g)
	}

	a.engine = r
}

func (a *app) Run() {
	if a.engine == nil {
		panic("must call build")
	}

	if a.cert == "" || a.key == "" {
		if err := http.ListenAndServe(a.addr, a.engine); err != nil {
			panic(err)
		}
	}

	if a.enableQUIC {
		if err := http3.ListenAndServeQUIC(a.addr, a.cert, a.key, a.engine); err != nil {
			panic(err)
		}
	} else {
		if err := http.ListenAndServeTLS(a.addr, a.cert, a.key, a.engine); err != nil {
			panic(err)
		}
	}
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

func fillGroups(r *gin.Engine, vg versionGroup) {

	for _, item := range vg.stableServices {
		g := r.Group(fmt.Sprintf("/v%d", vg.mainVersion))
		fillActions(g, item)
	}

	for _, item := range vg.betaServices {
		g := r.Group(fmt.Sprintf("/v%dbeta", vg.mainVersion))
		fillActions(g, item)
	}

	for _, item := range vg.alphaServices {
		g := r.Group(fmt.Sprintf("/v%dalpha", vg.mainVersion))
		fillActions(g, item)
	}
}

func fillActions(g *gin.RouterGroup, service any) {

	actions := makeActions(service)

	for _, action := range actions {

		g.POST(fmt.Sprintf("/%s/%s", action.serviceName, action.methodName), func(c *gin.Context) {
			req := action.bindData
			if err := c.ShouldBind(&req); err != nil {
				c.Status(http.StatusBadRequest)
				return
			}

			i := req.(Request)
			if err := i.Validate(); err != nil {
				c.Status(http.StatusBadRequest)
				return
			}

			ctx := context.Background()

			reqID := uuid.NewString()
			ctx = context.WithValue(ctx, "RequestID", reqID)
			c.Writer.Header().Set("X-Request-ID", reqID)

			ctxValue := reflect.ValueOf(ctx)

			reqValue := reflect.ValueOf(req).Elem()
			rtnList := action.methodData.Call([]reflect.Value{ctxValue, reqValue})

			// 判断第二个值 是自定义错误
			// 还是原生error
			errValue := rtnList[1].Interface()
			if errValue != nil {

				err, ok := errValue.(knownError)
				if ok {
					c.JSON(http.StatusConflict, gin.H{"error": err})
					return
				} else {
					c.Status(http.StatusInternalServerError)
					return
				}
			}
			result := rtnList[0]
			if action.respType == respTypeStream {
				file := result.Interface().(fs.File)
				stat, err := file.Stat()
				if err != nil {
					c.Status(http.StatusInternalServerError)
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
			c.JSON(http.StatusOK, gin.H{"data": value})
		})
	}
}

// 获取该结构体里的所有receiver method
func makeActions(service any) []reflectAction {

	rawType := reflect.TypeOf(service)
	if rawType.Kind() != reflect.Struct {
		panic("service should be struct")
	}

	const mustSuffix = "service"
	rawTypeName := strings.ToLower(rawType.Name())
	if !strings.HasSuffix(rawTypeName, mustSuffix) {
		panic("struct must have suffix [Service]")
	}
	serviceName := strings.ReplaceAll(rawTypeName, mustSuffix, "")

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
		methodValue := serviceValue.Method(i)

		// 检查参数是否符合规定格式
		inParams := methodType.Type.NumIn()
		outParams := methodType.Type.NumOut()
		if inParams != 3 || outParams != 2 {
			continue
		}

		var respType string

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

		if ok, r := satisfyResponse(out0); !ok {
			continue
		} else {
			respType = r
		}

		if ok := satisfyError(out1); !ok {
			continue
		}

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

func satisfyResponse(t reflect.Type) (bool, string) {
	// 指针类型 直接用
	streamType := reflect.TypeOf((*fs.File)(nil)).Elem()
	if t.Implements(streamType) {
		return true, respTypeStream
	}
	return true, respTypeJson
}

func satisfyError(t reflect.Type) bool {
	errType := reflect.TypeOf((*error)(nil)).Elem()
	return t.Implements(errType)
}
