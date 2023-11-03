package discover

import "fmt"

var ContentNotFound = fmt.Errorf("content not found")

type Storage interface {
	ContentId(contentKey []byte) []byte

	Get(contentKey []byte, contentId []byte) ([]byte, error)

	Put(contentKey []byte, content []byte) error
}
