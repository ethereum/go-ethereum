package vm

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
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

const ecRecoverInputLength = 128

func ecrecoverFunc(in []byte) []byte {
	// "in" is (hash, v, r, s), each 32 bytes
	// but for ecrecover we want (r, s, v)
	if len(in) < ecRecoverInputLength {
		return nil
	}

	// Treat V as a 256bit integer
	v := new(big.Int).Sub(common.Bytes2Big(in[32:64]), big.NewInt(27))
	// Ethereum requires V to be either 0 or 1 => (27 || 28)
	if !(v.Cmp(Zero) == 0 || v.Cmp(One) == 0) {
		return nil
	}

	// v needs to be moved to the end
	rsv := append(in[64:128], byte(v.Uint64()))
	pubKey := crypto.Ecrecover(in[:32], rsv)
	// make sure the public key is a valid one
	if pubKey == nil || len(pubKey) != 65 {
		return nil
	}

	// the first byte of pubkey is bitcoin heritage
	return common.LeftPadBytes(crypto.Sha3(pubKey[1:])[12:], 32)
}

func memCpy(in []byte) []byte {
	return in
}
