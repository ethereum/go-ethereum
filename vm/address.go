package vm

import (
	"math/big"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethutil"
)

type Address interface {
	Call(in []byte) []byte
}

type PrecompiledAddress struct {
	Gas func(l int) *big.Int
	fn  func(in []byte) []byte
}

func (self PrecompiledAddress) Call(in []byte) []byte {
	return self.fn(in)
}

var Precompiled = map[uint64]*PrecompiledAddress{
	1: &PrecompiledAddress{func(l int) *big.Int {
		return GasEcrecover
	}, ecrecoverFunc},
	2: &PrecompiledAddress{func(l int) *big.Int {
		n := big.NewInt(int64(l+31)/32 + 1)
		n.Mul(n, GasSha256)
		return n
	}, sha256Func},
	3: &PrecompiledAddress{func(l int) *big.Int {
		n := big.NewInt(int64(l+31)/32 + 1)
		n.Mul(n, GasRipemd)
		return n
	}, ripemd160Func},
}

func sha256Func(in []byte) []byte {
	return crypto.Sha256(in)
}

func ripemd160Func(in []byte) []byte {
	return ethutil.LeftPadBytes(crypto.Ripemd160(in), 32)
}

func ecrecoverFunc(in []byte) []byte {
	// In case of an invalid sig. Defaults to return nil
	defer func() { recover() }()

	hash := in[:32]
	v := ethutil.BigD(in[32:64]).Bytes()[0] - 27
	sig := append(in[64:], v)

	return ethutil.LeftPadBytes(crypto.Sha3(crypto.Ecrecover(append(hash, sig...))[1:])[12:], 32)
}
