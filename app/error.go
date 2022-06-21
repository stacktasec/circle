package app

import "fmt"

func NewError(status, message string, details ...any) error {
	return internalError{
		Status:  status,
		Message: message,
		Details: details,
	}
}

type internalError struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Details []any  `json:"details"`
}

func (i internalError) Error() string {
	return fmt.Sprintf("[Status] %s [Message] %s", i.Status, i.Message)
}

func (i internalError) Is(err error) bool {
	knownErr, ok := err.(internalError)
	if !ok {
		return false
	}

	return i.Status == knownErr.Status && i.Message == knownErr.Message
}
