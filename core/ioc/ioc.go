package ioc

import (
	"go.uber.org/dig"
	"reflect"
)

type Container struct {
	container *dig.Container
}

func NewContainer() *Container {
	return &Container{container: dig.New()}
}

func (c *Container) MustConstructor(constructor any) {
	funcType := reflect.TypeOf(constructor)
	if funcType.Kind() != reflect.Func {
		panic("constructor must be func")
	}

	if funcType.IsVariadic() {
		panic("do not accept variadic func")
	}

	if funcType.NumOut() != 1 {
		panic("only support one return value")
	}

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
