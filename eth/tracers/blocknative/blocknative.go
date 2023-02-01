package blocknative

import (
	"encoding/json"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"math/big"
	"strconv"
	"strings"
)

var Tracers = map[string]func() (Tracer, error){
	"txnOpCodeTracer": NewTxnOpCodeTracer,
}

type Tracer interface {
	vm.EVMLogger
	GetResult() (json.RawMessage, error)
	Stop(err error)
}

func bytesToHex(s []byte) string {
	return "0x" + common.Bytes2Hex(s)
}

func bigToHex(n *big.Int) string {
	if n == nil {
		return ""
	}
	return "0x" + n.Text(16)
}

func uintToHex(n uint64) string {
	return "0x" + strconv.FormatUint(n, 16)
}

func addrToHex(a common.Address) string {
	return strings.ToLower(a.Hex())
}
