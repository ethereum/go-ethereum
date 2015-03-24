package common

// Database interface
type Database interface {
	Put(key []byte, value []byte)
	Get(key []byte) ([]byte, error)
	//GetKeys() []*Key
	Delete(key []byte) error
	LastKnownTD() []byte
	Close()
	Print()
}
