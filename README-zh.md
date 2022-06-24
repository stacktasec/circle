警告：当前版本未经过严密测试，开源目的只在验证想法和学习，严禁用于生产环境，否则后果自负



主要基于以下开源库包装而成<br />

特点：

- 全局的依赖注入
- 按约定自动将业务逻辑代码映射为Http接口
- 匿名路由/路径省略的支持
- 多版本支持
- 自定义验证/授权处理器
- 集成请求参数验证
- 常见的业务Http状态码约定
- Cors/Gzip支持
- 自动健康检查端点
- 自动匹配业务错误、内部错误、未找到错误
- 易用的zap日志包装

TODO:

- 自动生成OpenAPI文档

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

Http状态码对照

```go
200 OK: 业务成功
400 BadRequest：模型绑定以及校验未通过
401 Unauthorized：身份验证未通过
403 Forbidden：功能验证未允许
404 NotFound: 业务中的对象不存在
409 Conflict: 业务逻辑错误
500 InternalServerError：未知的内部错误
501 NotImplemented: 没有该函数
503 ServiceUnavailable：没有实现验证
504 GatewayTimeout：某些业务超时
```
