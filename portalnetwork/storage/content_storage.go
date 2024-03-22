package storage

import "fmt"

var ErrContentNotFound = fmt.Errorf("content not found")

type ContentStorage interface {
	Get(contentKey []byte, contentId []byte) ([]byte, error)

	Put(contentKey []byte, contentId []byte, content []byte) error
}
