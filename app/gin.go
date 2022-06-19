package app

import (
	"context"
	"fmt"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/stacktasec/circle/ioc"
	"net/http"
	"reflect"
	"time"
)

const (
	suffixService = "Service"
	ctxKeyID      = "id"
)

type app struct {
	container     *ioc.Container
	options       options
	versionGroups map[int]*versionGroup
	engine        *gin.Engine
}

type UserPayload interface {
	UserID() string
	UserRole() string
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
	if err := a.container.LoadConstructors(constructors...); err != nil {
		panic(err)
	}
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

func (a *app) Run() {
	a.build()

	httpServer := http.Server{
		Addr:           a.options.addr,
		Handler:        a.engine,
		ReadTimeout:    time.Second * 10,
		WriteTimeout:   time.Second * 10,
		MaxHeaderBytes: 1 << 16,
	}

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
		c.String(http.StatusOK, "You guys found out")
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

	pointerValue, err := a.container.ResolveConstructor(constructor)
	if err != nil {
		panic(err)
	}

	actions := makeReflect(*pointerValue)

	for _, action := range actions {

		var path string
		if action.Omitted {
			path = action.MethodName
		} else {
			path = fmt.Sprintf("/%s/%s", action.ServiceName, action.MethodName)
		}

		g.POST(path, func(c *gin.Context) {

			if !action.Anonymous {

				if a.options.authenticator == nil || a.options.authorizer == nil {
					c.AbortWithStatus(http.StatusServiceUnavailable)
					return
				}

				payload, err := a.options.authenticator(c.Request.Header)
				if err != nil {
					c.AbortWithStatus(http.StatusUnauthorized)
					return
				}

				if err := a.options.authorizer(payload, c.Request.URL.Path); err != nil {
					c.AbortWithStatus(http.StatusForbidden)
					return
				}

				c.Set(ctxKeyID, payload)
			}

			req := action.BindData
			if err := c.ShouldBindJSON(&req); err != nil {
				c.AbortWithStatus(http.StatusBadRequest)
				return
			}

			i := req.(Request)
			if err := i.Validate(); err != nil {
				c.AbortWithStatus(http.StatusBadRequest)
				return
			}

			ctx := context.Background()

			ctxValue := reflect.ValueOf(ctx)
			reqValue := reflect.ValueOf(req).Elem()

			rtnList := action.MethodValue.Call([]reflect.Value{ctxValue, reqValue})

			errValue := rtnList[1].Interface()
			if errValue != nil {
				if errValue == context.DeadlineExceeded {
					c.AbortWithStatus(http.StatusGatewayTimeout)
					return
				}

				if err, ok := errValue.(internalError); ok {
					c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": err})
					return
				} else {
					c.AbortWithStatus(http.StatusInternalServerError)
					return
				}
			}

			result := rtnList[0].Interface()

			if result == nil {
				c.Status(http.StatusNotFound)
				return
			}
			c.JSON(http.StatusOK, gin.H{"result": result})
		})
	}
}
