package klog

import (
	"fmt"
	"github.com/stacktasec/circle/kit/klog/zap"
)

var builtinLogger Logger

func init() {
	builtinLogger = zap.NewLogger(internalWithSkip(2))
}

func Debug(format any, a ...any) {
	msg := fmt.Sprintf(fmt.Sprintf("%+v", format), a...)
	builtinLogger.Debug(msg)
}

func Info(format any, a ...any) {
	msg := fmt.Sprintf(fmt.Sprintf("%+v", format), a...)
	builtinLogger.Info(msg)
}

func Warn(format any, a ...any) {
	msg := fmt.Sprintf(fmt.Sprintf("%+v", format), a...)
	builtinLogger.Warn(msg)
}

func Error(format any, a ...any) {
	msg := fmt.Sprintf(fmt.Sprintf("%+v", format), a...)
	builtinLogger.Error(msg)
}

func Fatal(format any, a ...any) {
	msg := fmt.Sprintf(fmt.Sprintf("%+v", format), a...)
	builtinLogger.Fatal(msg)
}

func SyncLogger() error {
	return builtinLogger.Sync()
}
