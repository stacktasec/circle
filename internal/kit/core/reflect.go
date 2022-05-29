package core

import (
	"context"
	"github.com/iancoleman/strcase"
	"io/fs"
	"reflect"
	"strings"
)

const serviceSuffix = "service"

// 获取该结构体里的所有receiver method
func makeActions(impl any) []reflectAction {

	rawType := reflect.TypeOf(impl)
	if rawType.Kind() != reflect.Struct {
		panic("impl should be struct")
	}

	rawTypeName := strings.ToLower(rawType.Name())
	if !strings.HasSuffix(rawTypeName, serviceSuffix) {
		panic("struct must have suffix [Service]")
	}
	implName := strings.ReplaceAll(rawTypeName, serviceSuffix, "")

	implValue := reflect.New(reflect.TypeOf(impl))
	implType := implValue.Type()

	serviceType := reflect.TypeOf((*Service)(nil)).Elem()
	if !implType.Implements(serviceType) {
		panic("service type must implement interface Service")
	}

	numMethods := implType.NumMethod()
	if numMethods == 0 {
		panic("service type must have a method")
	}

	var actions []reflectAction
	for i := 0; i < numMethods; i++ {
		// 获得方法
		methodType := implType.Method(i)

		// 必须满足 导出 有 2个入参 2个出参
		// 入参是context.Context Request 则认定为待映射方法
		// 此时 出参 必须是 结构体指针 和 error
		if !methodType.IsExported() {
			continue
		}

		// 检查参数是否符合规定格式
		inParams := methodType.Type.NumIn()
		outParams := methodType.Type.NumOut()
		if inParams != 3 || outParams != 2 {
			continue
		}

		// 必须满足 如下 四元组
		in1 := methodType.Type.In(1)
		in2 := methodType.Type.In(2)
		out0 := methodType.Type.Out(0)
		out1 := methodType.Type.Out(1)

		if !satisfyContext(in1) {
			continue
		}

		if !satisfyRequest(in2) {
			continue
		}

		respType := mustResponse(out0)

		mustError(out1)

		methodValue := implValue.Method(i)
		action := reflectAction{
			serviceName: strcase.ToSnake(implName),
			methodName:  strcase.ToSnake(methodType.Name),
			bindData:    reflect.New(in2).Interface(),
			methodData:  methodValue,
			respType:    respType,
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
