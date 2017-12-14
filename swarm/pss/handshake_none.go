// +build nopsshandshake

package pss

const (
	IsActiveHandshake = false
)

func NewHandshakeParams() interface{} {
	return nil
}
