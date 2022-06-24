# Circle

 [Chinese](https://github.com/stacktasec/circle/blob/main/README-zh.md).



### An RPC-style dependency injection-based Http Server development kit



According to [Clean Coder Blog](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html) ，

We can follow the following principles to write a clean layered, maintainable, and testable Go server application

Rule of Clean Architecture by Uncle Bob

- Independent of Frameworks. The architecture does not depend on the existence of some library of feature laden software. This allows you to use such frameworks as tools, rather than having to cram your system into their limited constraints.
- Testable. The business rules can be tested without the UI, Database, Web Server, or any other external element.
- Independent of UI. The UI can change easily, without changing the rest of the system. A Web UI could be replaced with a console UI, for example, without changing the business rules.
- Independent of Database. You can swap out Oracle or SQL Server, for Mongo, BigTable, CouchDB, or something else. Your business rules are not bound to the database.
- Independent of any external agency. In fact your business rules simply don’t know anything at all about the outside world.



**As long as your infrastructure layer uses interfaces for abstraction and exports constructors with interfaces as return values, and your business service functions are written in the following format**

```
func(context.Context,Request)(*Resp,error)
```

**So the `Circle`that is such a kit that help you convert some of these functions into HTTP APIs with agreed-upon behavior**



**Warning: The current version has not been rigorously tested. The purpose of open source is only to verify ideas and learn. It is strictly forbidden to use in production environment, otherwise the consequences will be at your own risk.**



### Prerequisites

Go 1.18 installed



### Installing

```
go get github.com/stacktasec/circle
```



### Mainly packaged based on the following open source libraries

`github.com/gin-gonic/gin`

`go.uber.org/dig`

`go.uber.org/zap`



### Features:

- Global dependency injection
- Automatically map business logic code to Http interface by convention
- Anonymous routing/path elision support
- Multi-version support
- custom authentication/authorization handler
- Integrated request parameter validation
- Common business Http status code conventions
- Cors/Gzip support
- Automatic health check endpoints
- Automatic matching of business errors, internal errors, not found errors
- Easy to use zap log wrapper
  
  

### TODO:

- Automatically generate OpenAPI documentation



### Get Started

```go
package main

import (
    "context"
    "errors"
    "github.com/stacktasec/circle/app"
)

// Calculator Abstract Infra
type Calculator interface {
    Add(x, y int) int
}

// SimpleCalculator Infra impl
type SimpleCalculator struct {
}

func (c *SimpleCalculator) Add(x, y int) int {
    return x + y
}

// NewSimpleCalculator Notice!!! must return interface type
// only accept one returned interface value
func NewSimpleCalculator() Calculator {
    return &SimpleCalculator{}
}

type DemoService struct {
    calculator Calculator
}

// NewDemoService Notice!!! must return pointer type
// only accept one returned pointer value
func NewDemoService(calculator Calculator) *DemoService {
    return &DemoService{calculator: calculator}
}

type MathReq struct {
    X int `json:"x" binding:"required"`
    Y int `json:"y" binding:"required"`
}

type SumResp struct {
    Sum int `json:"sum"`
}

// Validate Must provide Validate() error function Impl
func (m *MathReq) Validate() error {
    if m.X < 0 || m.Y < 0 {
        return errors.New("must be positive")
    }
    return nil
}

// Sum Biz Logic Implementation, will be mapping to http api endpoint
func (d *DemoService) Sum(ctx context.Context, req MathReq) (*SumResp, error) {
    resp := SumResp{Sum: d.calculator.Add(req.X, req.Y)}
    return &resp, nil
}

func main() {
    a := app.NewApp(app.WithBaseURL("api"))
    a.Load(NewSimpleCalculator, NewDemoService)

    g := app.NewGroup(1)
    g.SetStable(NewDemoService)

    a.Map(g)

    a.Run()
}
```



then you can

```
curl --location --request POST 'http://localhost:8080/api/v1/demo/sum' \
--header 'Content-Type: application/json' \
--data-raw '{
    "x":1000,
    "y":500
}'
```





Http status code comparison

```go
200 OK: Business Success
400 BadRequest: Model binding and validation failed
401 Unauthorized: Authentication failed
403 Forbidden: Functional verification not allowed
404 NotFound: The object in the business does not exist
409 Conflict: Business logic error
500 InternalServerError: Unknown internal error
501 NotImplemented: No such function
503 ServiceUnavailable: Authentication not implemented
504 GatewayTimeout: Some services timed out
```
