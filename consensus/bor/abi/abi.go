package abi

//go:generate mockgen -destination=./abi_mock.go -package=api . ABI
type ABI interface {
	Pack(name string, args ...interface{}) ([]byte, error)
	UnpackIntoInterface(v interface{}, name string, data []byte) error
}
