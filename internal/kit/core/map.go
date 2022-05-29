package core

import (
	"context"
	"github.com/iancoleman/strcase"
	"io/fs"
	"reflect"
	"strings"
)

const (
	methodInit    = "Init"
	suffixService = "Service"
)

// 获取该结构体里的所有receiver method
func (a *app) makeActions(service any) []reflectAction {

	structType := reflect.TypeOf(service)
	if structType.Kind() != reflect.Struct {
		panic("service must be struct")
	}

	pointerValue := reflect.New(structType)
	pointerType := pointerValue.Type()

	typeName := structType.Name()
	if !strings.HasSuffix(typeName, suffixService) {
		panic("service must have suffix [Service]")
	}

	initValue := pointerValue.MethodByName(methodInit)
	if !initValue.IsValid() {
		panic("service must have method Init")
	}
	initValueData := initValue.Interface()
	if err := a.container.Invoke(initValueData); err != nil {
		panic(err)
	}

	var actions []reflectAction
	for i := 0; i < pointerType.NumMethod(); i++ {
		// 获得方法
		method := pointerType.Method(i)
		methodType := method.Type

		// 必须满足 导出 有 2个入参 2个出参
		// 入参是context.Context Request 则认定为待映射方法
		// 此时 出参 必须是 结构体指针 和 error
		if !method.IsExported() {
			continue
		}

		// 检查参数是否符合规定格式
		inParams := methodType.NumIn()
		outParams := methodType.NumOut()
		if inParams != 3 || outParams != 2 {
			continue
		}

		// 必须满足 如下 四元组
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

		svcName, methodName := a.makeName(typeName, method.Name)
		action := reflectAction{
			serviceName: svcName,
			methodName:  methodName,
			bindData:    reflect.New(in2).Interface(),
			methodData:  pointerValue.Method(i),
			respType:    respType,
		}

		actions = append(actions, action)
	}

	return actions
}

func (a *app) makeName(resource, action string) (string, string) {
	lr := strings.ToLower(resource)

	for _, s := range a.options.suffixes {
		if strings.HasSuffix(lr, s) {
			lr = strings.ReplaceAll(lr, s, "")
			break
		}
	}

	return strcase.ToSnake(lr), strcase.ToSnake(action)
}

func satisfyContext(t reflect.Type) bool {
	ctxType := reflect.TypeOf((*context.Context)(nil)).Elem()
	return t.AssignableTo(ctxType)
}

func satisfyRequest(t reflect.Type) bool {
	// 值类型 需要先变成指针
	pt := reflect.New(t).Type()
	reqType := reflect.TypeOf((*Request)(nil)).Elem()
	return pt.Implements(reqType)
}

func mustResponse(t reflect.Type) string {
	if t.Kind() != reflect.Pointer || t.Elem().Kind() != reflect.Struct {
		panic("this position type must be a pointer of struct")
	}

	// 指针类型 直接用
	streamType := reflect.TypeOf((*fs.File)(nil)).Elem()
	if t.Implements(streamType) {
		return respTypeStream
	}

	return respTypeJson
}

func mustError(t reflect.Type) {
	errType := reflect.TypeOf((*error)(nil)).Elem()
	if !t.Implements(errType) {
		panic("this position type must be error")
	}
}
