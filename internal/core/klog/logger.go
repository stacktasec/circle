package klog

import (
	"fmt"
)

var logger *zapLogger

func init() {
	InitLogger()
}

func Debug(format any, a ...any) {
	msg := fmt.Sprintf(fmt.Sprintf("%+v", format), a...)
	logger.Debug(msg)
}

func Info(format any, a ...any) {
	msg := fmt.Sprintf(fmt.Sprintf("%+v", format), a...)
	logger.Info(msg)
}

func Warn(format any, a ...any) {
	msg := fmt.Sprintf(fmt.Sprintf("%+v", format), a...)
	logger.Warn(msg)
}

func Error(format any, a ...any) {
	msg := fmt.Sprintf(fmt.Sprintf("%+v", format), a...)
	logger.Error(msg)
}

func Fatal(format any, a ...any) {
	msg := fmt.Sprintf(fmt.Sprintf("%+v", format), a...)
	logger.Fatal(msg)
}

func SyncLogger() error {
	return logger.Sync()
}
