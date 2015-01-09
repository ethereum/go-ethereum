package qwhisper

import (
	"fmt"
	"unsafe"
)

type Watch struct {
}

func (self *Watch) Arrived(v unsafe.Pointer) {
	fmt.Println(v)
}
