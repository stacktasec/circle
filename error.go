package circle

import "fmt"

func NewKnownError(status, message string) error {
	return knownError{
		Status:  status,
		Message: message,
	}
}

type knownError struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func (k knownError) Error() string {
	return fmt.Sprintf("[Status] %s [Message] %s", k.Status, k.Message)
}

func (k knownError) Is(err error) bool {
	knownErr, ok := err.(knownError)
	if !ok {
		return false
	}

	return k.Status == knownErr.Status && k.Message == knownErr.Message
}