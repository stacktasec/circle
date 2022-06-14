package app

import (
	"circle/core/ioc"
	"circle/core/klog"
	"context"
	"fmt"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/iancoleman/strcase"
	"io/fs"
	"net/http"
	"reflect"
	"strings"
	"time"
)

const suffixService = "Service"
const keyRequestID = "X-Request-ID"

type app struct {
	container     *ioc.Container
	options       options
	versionGroups map[int]*versionGroup
	engine        *gin.Engine
}

func NewApp(opts ...Option) App {

	o := &options{}

	for _, opt := range opts {
		opt.apply(o)
	}

	o.ensure()

	return &app{container: ioc.NewContainer(), options: *o, versionGroups: make(map[int]*versionGroup)}
}

func (a *app) Load(constructors ...any) {
	a.container.LoadConstructors(constructors...)
}

func (a *app) Map(groups ...*versionGroup) {
	loadGroups(a.versionGroups, groups...)
}

func (a *app) Run() {
	a.build()

	httpServer := http.Server{
		Addr:           a.options.addr,
		Handler:        a.engine,
		ReadTimeout:    time.Second * 10,
		WriteTimeout:   time.Second * 10,
		MaxHeaderBytes: 1 << 16,
	}

	klog.Info("http server is listening on %s", a.options.addr)
	if err := httpServer.ListenAndServe(); err != nil {
		panic(err)
	}
}

func (a *app) build() {

	r := gin.Default()

	r.NoRoute(func(c *gin.Context) {
		c.AbortWithStatus(http.StatusNotImplemented)
	})

	r.Use(cors.Default())

	a.discovery(r)

	for _, g := range a.versionGroups {
		a.fillGroups(r.Group(a.options.baseURL), g)
	}

	r.Use(gzip.Gzip(gzip.DefaultCompression))

	a.engine = r
}

func (a *app) discovery(r *gin.Engine) {
	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "Welcome")
	})
}

func (a *app) fillGroups(routerGroup *gin.RouterGroup, vg *versionGroup) {

	for _, constructor := range vg.stableConstructors {
		g := routerGroup.Group(fmt.Sprintf("/v%d", vg.mainVersion))
		a.fillActions(g, constructor)
	}

	for _, constructor := range vg.betaConstructors {
		g := routerGroup.Group(fmt.Sprintf("/v%dbeta", vg.mainVersion))
		a.fillActions(g, constructor)
	}

	for _, constructor := range vg.alphaConstructors {
		g := routerGroup.Group(fmt.Sprintf("/v%dalpha", vg.mainVersion))
		a.fillActions(g, constructor)
	}
}

func (a *app) fillActions(g *gin.RouterGroup, constructor any) {

	pointerValue := a.container.ResolveConstructor(constructor)

	actions := makeReflect(pointerValue)

	for _, action := range actions {

		var route string
		if action.Omitted {
			route = action.MethodName
		} else {
			route = fmt.Sprintf("/%s/%s", action.ServiceName, action.MethodName)
		}

		g.POST(route, func(c *gin.Context) {

			if !action.Anonymous {
				h := c.Request.Header
				if a.options.idInterceptor != nil {
					if err := a.options.idInterceptor(h); err != nil {
						c.AbortWithStatus(http.StatusUnauthorized)
						return
					}

					if a.options.permInterceptor != nil {
						if err := a.options.permInterceptor(h); err != nil {
							c.AbortWithStatus(http.StatusForbidden)
							return
						}
					}
				}
			}

			req := action.BindData
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
			ctx = context.WithValue(ctx, keyRequestID, reqID)

			c.Writer.Header().Set(keyRequestID, reqID)

			ctxValue := reflect.ValueOf(ctx)
			reqValue := reflect.ValueOf(req).Elem()
			rtnList := action.MethodValue.Call([]reflect.Value{ctxValue, reqValue})

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
			if action.RespType == respTypeFile {
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

func loadGroups(versionGroups map[int]*versionGroup, groups ...*versionGroup) {
	for _, g := range groups {
		_, ok := versionGroups[g.mainVersion]
		if ok {
			panic("duplicated main version")
		}
		versionGroups[g.mainVersion] = g
	}
}

type reflectAction struct {
	ServiceName string
	MethodName  string
	Omitted     bool
	Anonymous   bool
	BindData    any
	MethodValue reflect.Value
	RespType    string
}

func makeReflect(pointerValue reflect.Value) []reflectAction {

	pointerType := pointerValue.Type()

	rawSvcName := pointerType.Elem().Name()
	if !strings.HasSuffix(rawSvcName, suffixService) {
		panic("must ends with Service")
	}
	svcName := strcase.ToSnake(strings.ReplaceAll(rawSvcName, suffixService, ""))

	var omitted bool
	omittedAttribute := reflect.TypeOf((*OmittedAttribute)(nil)).Elem()
	if pointerType.Implements(omittedAttribute) {
		impl := pointerValue.Interface().(OmittedAttribute)
		omitted = impl.Omitted()
	}

	var anonymous bool
	anonymousAttribute := reflect.TypeOf((*AnonymousAttribute)(nil)).Elem()
	if pointerType.Implements(anonymousAttribute) {
		impl := pointerValue.Interface().(AnonymousAttribute)
		anonymous = impl.Anonymous()
	}

	var actions []reflectAction
	for i := 0; i < pointerType.NumMethod(); i++ {
		method := pointerType.Method(i)

		if !method.IsExported() {
			continue
		}

		methodType := method.Type
		inParams := methodType.NumIn()
		outParams := methodType.NumOut()
		if inParams != 3 || outParams != 2 {
			continue
		}

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

		methodName := strcase.ToSnake(method.Name)
		action := reflectAction{
			ServiceName: svcName,
			MethodName:  methodName,
			Anonymous:   anonymous,
			Omitted:     omitted,
			BindData:    reflect.New(in2).Interface(),
			MethodValue: pointerValue.Method(i),
			RespType:    respType,
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
	pt := reflect.New(t).Type()
	reqType := reflect.TypeOf((*Request)(nil)).Elem()
	return pt.Implements(reqType)
}

func mustResponse(t reflect.Type) string {
	if t.Kind() != reflect.Pointer || t.Elem().Kind() != reflect.Struct {
		panic("this position type must be a pointer of struct")
	}

	streamType := reflect.TypeOf((*fs.File)(nil)).Elem()
	if t.Implements(streamType) {
		return respTypeFile
	}

	return respTypeJson
}

func mustError(t reflect.Type) {
	errType := reflect.TypeOf((*error)(nil)).Elem()
	if !t.Implements(errType) {
		panic("this position type must be error")
	}
}
