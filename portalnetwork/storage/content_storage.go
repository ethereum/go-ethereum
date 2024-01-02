package storage

import "fmt"

var ErrContentNotFound = fmt.Errorf("content not found")

type ContentStorage interface {
	Get(contentId []byte) ([]byte, error)

	Put(contentId []byte, content []byte) error
}
