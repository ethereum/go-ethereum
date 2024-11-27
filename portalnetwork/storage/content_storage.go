package storage

import (
	"fmt"

	"github.com/holiman/uint256"
)

var ErrContentNotFound = fmt.Errorf("content not found")
var ErrInsufficientRadius = fmt.Errorf("insufficient radius")

var MaxDistance = uint256.MustFromHex("0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")

type ContentType byte

var SizeKey = []byte("size")

type ContentKey struct {
	selector ContentType
	data     []byte
}

func NewContentKey(selector ContentType, data []byte) *ContentKey {
	return &ContentKey{
		selector: selector,
		data:     data,
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

	Radius() *uint256.Int
}

type MockStorage struct {
	Db map[string][]byte
}

func NewMockStorage() ContentStorage {
	return &MockStorage{
		Db: make(map[string][]byte),
	}
}

func (m *MockStorage) Get(contentKey []byte, contentId []byte) ([]byte, error) {
	if content, ok := m.Db[string(contentId)]; ok {
		return content, nil
	}
	return nil, ErrContentNotFound
}

func (m *MockStorage) Put(contentKey []byte, contentId []byte, content []byte) error {
	m.Db[string(contentId)] = content
	return nil
}

func (m *MockStorage) Radius() *uint256.Int {
	return uint256.MustFromHex("0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")
}
