# Circle

### An RPC-style dependency injection-based Http Server development kit

**Warning: The current version has not been rigorously tested. The purpose of open source is only to verify ideas and learn. It is strictly forbidden to use in production environment, otherwise the consequences will be at your own risk.**

Mainly packaged based on the following open source libraries

`github.com/gin-gonic/gin`

`go.uber.org/dig`

`go.uber.org/zap`

Features:

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

TODO:

- Automatically generate OpenAPI documentation

Get Started

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
