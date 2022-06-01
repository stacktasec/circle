package internal

import (
	"context"
	"go.uber.org/dig"
	"io/fs"
	"reflect"
	"runtime"
	"strings"
)

func LoadConstructors(container *dig.Container, constructors ...any) {
	for _, c := range constructors {
		VerifyConstructor(c)
	}

	for _, item := range constructors {
		if err := container.Provide(item); err != nil {
			panic(err)
		}
	}
}

func LoadGroups(versionGroups map[int]*VersionGroup, groups ...*VersionGroup) {
	for _, g := range groups {
		_, ok := versionGroups[g.MainVersion]
		if ok {
			panic("duplicated main version")
		}
		versionGroups[g.MainVersion] = g
	}
}

type ReflectAction struct {
	// Service 资源名称
	ServiceName string
	// 方法名
	MethodName string
	// 用来绑定的请求数据
	BindData any
	// 用来调用的方法的反射值
	MethodValue reflect.Value
	// 请求 返回类型
	RespType string
}

func MakeReflect(container *dig.Container, constructor any, suffixes []string) []ReflectAction {

	VerifyConstructor(constructor)

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

	if err := container.Invoke(invokerValue.Interface()); err != nil {
		panic(err)
	}

	pointerValue := reflect.ValueOf(rtn)
	pointerType := pointerValue.Type()

	var actions []ReflectAction
	for i := 0; i < pointerType.NumMethod(); i++ {
		// 获得方法
		method := pointerType.Method(i)

		// 必须满足 导出 有 2个入参 2个出参
		// 入参是context.Context Request 则认定为待映射方法
		// 此时 出参 必须是 结构体指针 和 error
		if !method.IsExported() {
			continue
		}

		methodType := method.Type
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

		if !SatisfyContext(in1) {
			continue
		}

		if !SatisfyRequest(in2) {
			continue
		}

		respType := MustResponse(out0)

		MustError(out1)

		svcName, methodName := GenRoute(suffixes, pointerType.Elem().Name(), method.Name)
		action := ReflectAction{
			ServiceName: svcName,
			MethodName:  methodName,
			BindData:    reflect.New(in2).Interface(),
			MethodValue: pointerValue.Method(i),
			RespType:    respType,
		}

		actions = append(actions, action)
	}

	return actions
}

func SatisfyContext(t reflect.Type) bool {
	ctxType := reflect.TypeOf((*context.Context)(nil)).Elem()
	return t.AssignableTo(ctxType)
}

func SatisfyRequest(t reflect.Type) bool {
	// 值类型 需要先变成指针
	pt := reflect.New(t).Type()
	reqType := reflect.TypeOf((*Request)(nil)).Elem()
	return pt.Implements(reqType)
}

func MustResponse(t reflect.Type) string {
	if t.Kind() != reflect.Pointer || t.Elem().Kind() != reflect.Struct {
		panic("this position type must be a pointer of struct")
	}

	// 指针类型 直接用
	streamType := reflect.TypeOf((*fs.File)(nil)).Elem()
	if t.Implements(streamType) {
		return RespTypeStream
	}

	return RespTypeJson
}

func MustError(t reflect.Type) {
	errType := reflect.TypeOf((*error)(nil)).Elem()
	if !t.Implements(errType) {
		panic("this position type must be error")
	}
}

func VerifyConstructor(constructor any) {
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
