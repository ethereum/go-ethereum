package common

// Database interface
type Database interface {
	Put(key []byte, value []byte)
	Get(key []byte) ([]byte, error)
	Delete(key []byte) error
	Close()
	Flush() error
}
