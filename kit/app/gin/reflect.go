package gin

import (
	"context"
	"github.com/stacktasec/circle/kit/app/internal"
	"io/fs"
	"reflect"
	"runtime"
	"strings"
)

func satisfyContext(t reflect.Type) bool {
	ctxType := reflect.TypeOf((*context.Context)(nil)).Elem()
	return t.AssignableTo(ctxType)
}

func satisfyRequest(t reflect.Type) bool {
	// 值类型 需要先变成指针
	pt := reflect.New(t).Type()
	reqType := reflect.TypeOf((*internal.Request)(nil)).Elem()
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

func verifyConstructor(constructor any) {
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
