package ioc

import (
	"errors"
	"go.uber.org/dig"
	"reflect"
)

type Container struct {
	container *dig.Container
}

func NewContainer() *Container {
	return &Container{container: dig.New()}
}

func (c *Container) LoadConstructors(constructors ...any) error {

	for _, constructor := range constructors {
		if err := c.validateConstructor(constructor); err != nil {
			return err
		}
	}

	for _, item := range constructors {
		if err := c.container.Provide(item); err != nil {
			return err
		}
	}

	return nil
}

func (c *Container) ResolveConstructor(constructor any) (*reflect.Value, error) {

	if err := c.validateConstructor(constructor); err != nil {
		return nil, err
	}

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

	if err := c.container.Invoke(invokerValue.Interface()); err != nil {
		return nil, err
	}

	value := reflect.ValueOf(rtn)
	return &value, nil
}

func (c *Container) validateConstructor(constructor any) error {
	funcType := reflect.TypeOf(constructor)
	if funcType.Kind() != reflect.Func {
		return errors.New("constructor must be func")
	}

	if funcType.IsVariadic() {
		return errors.New("do not accept variadic func")
	}

	if funcType.NumOut() != 1 {
		return errors.New("only support one return value")
	}

	outKind := funcType.Out(0).Kind()
	if outKind != reflect.Pointer && outKind != reflect.Interface {
		return errors.New("rtn value type must be pointer or interface")
	}

	return nil
}
