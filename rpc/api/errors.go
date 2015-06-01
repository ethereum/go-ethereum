package api

import "fmt"

type NotImplementedError struct {
	Method string
}

func (e *NotImplementedError) Error() string {
	return fmt.Sprintf("%s method not implemented", e.Method)
}
