package abi

type ABI interface {
	Pack(name string, args ...interface{}) ([]byte, error)
	UnpackIntoInterface(v interface{}, name string, data []byte) error
}
