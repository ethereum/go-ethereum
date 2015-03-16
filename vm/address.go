package vm

import (
	"math/big"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/common"
)

type Address interface {
	Call(in []byte) []byte
}

type PrecompiledAccount struct {
	Gas func(l int) *big.Int
	fn  func(in []byte) []byte
}

func (self PrecompiledAccount) Call(in []byte) []byte {
	return self.fn(in)
}

var Precompiled = PrecompiledContracts()

// XXX Could set directly. Testing requires resetting and setting of pre compiled contracts.
func PrecompiledContracts() map[string]*PrecompiledAccount {
	return map[string]*PrecompiledAccount{
		// ECRECOVER
		string(common.LeftPadBytes([]byte{1}, 20)): &PrecompiledAccount{func(l int) *big.Int {
			return GasEcrecover
		}, ecrecoverFunc},

		// SHA256
		string(common.LeftPadBytes([]byte{2}, 20)): &PrecompiledAccount{func(l int) *big.Int {
			n := big.NewInt(int64(l+31) / 32)
			n.Mul(n, GasSha256Word)
			return n.Add(n, GasSha256Base)
		}, sha256Func},

		// RIPEMD160
		string(common.LeftPadBytes([]byte{3}, 20)): &PrecompiledAccount{func(l int) *big.Int {
			n := big.NewInt(int64(l+31) / 32)
			n.Mul(n, GasRipemdWord)
			return n.Add(n, GasRipemdBase)
		}, ripemd160Func},

		string(common.LeftPadBytes([]byte{4}, 20)): &PrecompiledAccount{func(l int) *big.Int {
			n := big.NewInt(int64(l+31) / 32)
			n.Mul(n, GasIdentityWord)

			return n.Add(n, GasIdentityBase)
		}, memCpy},
	}
}

func sha256Func(in []byte) []byte {
	return crypto.Sha256(in)
}

func ripemd160Func(in []byte) []byte {
	return common.LeftPadBytes(crypto.Ripemd160(in), 32)
}

func ecrecoverFunc(in []byte) []byte {
	// In case of an invalid sig. Defaults to return nil
	defer func() { recover() }()

	hash := in[:32]
	v := common.BigD(in[32:64]).Bytes()[0] - 27
	sig := append(in[64:], v)

	return common.LeftPadBytes(crypto.Sha3(crypto.Ecrecover(append(hash, sig...))[1:])[12:], 32)
}

func memCpy(in []byte) []byte {
	return in
}
