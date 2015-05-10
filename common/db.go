package common

// Database interface
type Database interface {
	Put(key []byte, value []byte)
	Get(key []byte) ([]byte, error)
	Delete(key []byte) error
	LastKnownTD() []byte
	Close()
	Flush() error
}
