package biz

import "fmt"

func MakeError(status, message string) error {
	return Error{
		Status:  status,
		Message: message,
	}
}

type Error struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func (e Error) Error() string {
	return fmt.Sprintf("[Status] %s [Message] %s", e.Status, e.Message)
}

func (e Error) Is(err error) bool {
	nErr, ok := err.(Error)
	if !ok {
		return false
	}

	return e.Status == nErr.Status && e.Message == nErr.Message
}
