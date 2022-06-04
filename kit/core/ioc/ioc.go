package ioc

import (
	"go.uber.org/dig"
	"reflect"
	"runtime"
	"strings"
)

type Container struct {
	container *dig.Container
}

func NewContainer() *Container {
	return &Container{container: dig.New()}
}

func (c *Container) MustConstructor(constructor any) {
	// 只接受 函数
	funcType := reflect.TypeOf(constructor)
	if funcType.Kind() != reflect.Func {
		panic("constructor must be func")
	}

	var funcName string
	name := runtime.FuncForPC(reflect.ValueOf(constructor).Pointer()).Name()
	arr := strings.Split(name, ".")
	if len(arr) == 1 {
		funcName = arr[0]
	} else {
		funcName = arr[len(arr)-1]
	}

	// 必须 New开头
	if !strings.HasPrefix(funcName, "New") {
		panic("constructor must start with New")
	}

	// 不能是可变函数
	if funcType.IsVariadic() {
		panic("do not accept variadic func")
	}

	// return值暂时只支持1个
	if funcType.NumOut() != 1 {
		panic("only support one return value")
	}

	// return值暂时支持1个
	if funcType.Out(0).Kind() != reflect.Pointer && funcType.Out(0).Kind() != reflect.Interface {
		panic("rtn value type must be pointer or interface")
	}
}

func (c *Container) LoadConstructors(constructors ...any) {
	for _, constructor := range constructors {
		c.MustConstructor(constructor)
	}

	for _, item := range constructors {
		_ = c.container.Provide(item)
	}
}

func (c *Container) ResolveConstructor(constructor any) reflect.Value {
	c.MustConstructor(constructor)

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

	_ = c.container.Invoke(invokerValue.Interface())

	return reflect.ValueOf(rtn)
}
