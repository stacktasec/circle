package internal

import "fmt"

type KnownError struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func (k KnownError) Error() string {
	return fmt.Sprintf("[Status] %s [Message] %s", k.Status, k.Message)
}

func (k KnownError) Is(err error) bool {
	nErr, ok := err.(KnownError)
	if !ok {
		return false
	}

	return k.Status == nErr.Status && k.Message == nErr.Message
}
