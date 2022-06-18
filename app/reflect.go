package app

import (
	"context"
	"errors"
	"github.com/iancoleman/strcase"
	"reflect"
	"strings"
)

type reflectAction struct {
	ServiceName string
	MethodName  string
	Omitted     bool
	Anonymous   bool
	BindData    any
	MethodValue reflect.Value
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
			panic("exported method must be action")
		}

		in1 := methodType.In(1)
		in2 := methodType.In(2)
		out0 := methodType.Out(0)
		out1 := methodType.Out(1)

		if err := validateAction(in1, in2, out0, out1); err != nil {
			panic(err)
		}

		methodName := strcase.ToSnake(method.Name)
		action := reflectAction{
			ServiceName: svcName,
			MethodName:  methodName,
			Anonymous:   anonymous,
			Omitted:     omitted,
			BindData:    reflect.New(in2).Interface(),
			MethodValue: pointerValue.Method(i),
		}

		actions = append(actions, action)
	}

	return actions
}

func validateAction(t1, t2, t3, t4 reflect.Type) error {
	if err := validateContext(t1); err != nil {
		return err
	}

	if err := validateRequest(t2); err != nil {
		return err
	}
	if err := validateResponse(t3); err != nil {
		return err
	}
	if err := validateError(t4); err != nil {
		return err
	}
	return nil
}

func validateContext(t reflect.Type) error {
	ctxType := reflect.TypeOf((*context.Context)(nil)).Elem()
	if !t.AssignableTo(ctxType) {
		return errors.New("this position type must be context.Context")
	}
	return nil
}

func validateRequest(t reflect.Type) error {
	pt := reflect.New(t).Type()
	reqType := reflect.TypeOf((*Request)(nil)).Elem()
	if !pt.Implements(reqType) {
		return errors.New("this position type must impl Request")
	}
	return nil
}

func validateResponse(t reflect.Type) error {
	if t.Kind() != reflect.Pointer || t.Elem().Kind() != reflect.Struct {
		return errors.New("this position type must be a pointer of struct")
	}
	return nil
}

func validateError(t reflect.Type) error {
	errType := reflect.TypeOf((*error)(nil)).Elem()
	if !t.Implements(errType) {
		return errors.New("this position type must be error")
	}
	return nil
}
