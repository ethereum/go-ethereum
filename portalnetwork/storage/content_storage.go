package storage

import "fmt"

var ErrContentNotFound = fmt.Errorf("content not found")

type ContentType byte

type ContentKey struct {
	selector ContentType
	data     []byte
}

func NewContentKey(selector ContentType, hash []byte) *ContentKey {
	return &ContentKey{
		selector: selector,
		data:     hash,
	}
}

func (c *ContentKey) Encode() []byte {
	res := make([]byte, 0, len(c.data)+1)
	res = append(res, byte(c.selector))
	res = append(res, c.data...)
	return res
}

type ContentStorage interface {
	Get(contentKey []byte, contentId []byte) ([]byte, error)

	Put(contentKey []byte, contentId []byte, content []byte) error
}
