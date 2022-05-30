package core

import (
	"fmt"
)

type Request interface {
	Validate() error
}

type knownError struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func (k knownError) Error() string {
	return fmt.Sprintf("[Status] %s [Message] %s", k.Status, k.Message)
}

func (k knownError) Is(err error) bool {
	nErr, ok := err.(knownError)
	if !ok {
		return false
	}

	return k.Status == nErr.Status && k.Message == nErr.Message
}

func MakeKnownError(status, message string) error {
	return knownError{
		Status:  status,
		Message: message,
	}
}
