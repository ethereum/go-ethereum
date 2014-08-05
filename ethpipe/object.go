package ethpipe

import (
	"github.com/ethereum/eth-go/ethstate"
	"github.com/ethereum/eth-go/ethutil"
)

type object struct {
	*ethstate.StateObject
}

func (self *object) StorageString(str string) *ethutil.Value {
	if ethutil.IsHex(str) {
		return self.Storage(ethutil.Hex2Bytes(str[2:]))
	} else {
		return self.Storage(ethutil.RightPadBytes([]byte(str), 32))
	}
}

func (self *object) StorageValue(addr *ethutil.Value) *ethutil.Value {
	return self.Storage(addr.Bytes())
}

func (self *object) Storage(addr []byte) *ethutil.Value {
	return self.StateObject.GetStorage(ethutil.BigD(addr))
}
