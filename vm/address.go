package vm

import (
	"math/big"

	"github.com/ethereum/go-ethereum/ethcrypto"
	"github.com/ethereum/go-ethereum/ethutil"
)

type Address interface {
	Call(in []byte) []byte
}

type PrecompiledAddress struct {
	Gas *big.Int
	fn  func(in []byte) []byte
}

func (self PrecompiledAddress) Call(in []byte) []byte {
	return self.fn(in)
}

var Precompiled = map[uint64]*PrecompiledAddress{
	1: &PrecompiledAddress{big.NewInt(500), ecrecoverFunc},
	2: &PrecompiledAddress{big.NewInt(100), sha256Func},
	3: &PrecompiledAddress{big.NewInt(100), ripemd160Func},
}

func sha256Func(in []byte) []byte {
	return ethcrypto.Sha256(in)
}

func ripemd160Func(in []byte) []byte {
	return ethutil.RightPadBytes(ethcrypto.Ripemd160(in), 32)
}

func ecrecoverFunc(in []byte) []byte {
	// In case of an invalid sig. Defaults to return nil
	defer func() { recover() }()

	return ethcrypto.Ecrecover(in)
}
