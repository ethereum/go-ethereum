package xeth

/*
import "github.com/ethereum/go-ethereum/ethutil"

var cnfCtr = ethutil.Hex2Bytes("661005d2720d855f1d9976f88bb10c1a3398c77f")

type Config struct {
	pipe *XEth
}

func (self *Config) Get(name string) *Object {
	configCtrl := self.pipe.World().safeGet(cnfCtr)
	var addr []byte

	switch name {
	case "NameReg":
		addr = []byte{0}
	case "DnsReg":
		objectAddr := configCtrl.GetStorage(ethutil.BigD([]byte{0}))
		domainAddr := (&Object{self.pipe.World().safeGet(objectAddr.Bytes())}).StorageString("DnsReg").Bytes()
		return &Object{self.pipe.World().safeGet(domainAddr)}
	case "MergeMining":
		addr = []byte{4}
	default:
		addr = ethutil.RightPadBytes([]byte(name), 32)
	}

	objectAddr := configCtrl.GetStorage(ethutil.BigD(addr))

	return &Object{self.pipe.World().safeGet(objectAddr.Bytes())}
}

func (self *Config) Exist() bool {
	return self.pipe.World().Get(cnfCtr) != nil
}
*/
