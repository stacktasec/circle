package app

import (
	"context"
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

		mustResponse(out0)

		mustError(out1)

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

func satisfyContext(t reflect.Type) bool {
	ctxType := reflect.TypeOf((*context.Context)(nil)).Elem()
	return t.AssignableTo(ctxType)
}

func satisfyRequest(t reflect.Type) bool {
	pt := reflect.New(t).Type()
	reqType := reflect.TypeOf((*Request)(nil)).Elem()
	return pt.Implements(reqType)
}

func mustResponse(t reflect.Type) {
	if t.Kind() != reflect.Pointer || t.Elem().Kind() != reflect.Struct {
		panic("this position type must be a pointer of struct")
	}
}

func mustError(t reflect.Type) {
	errType := reflect.TypeOf((*error)(nil)).Elem()
	if !t.Implements(errType) {
		panic("this position type must be error")
	}
}
