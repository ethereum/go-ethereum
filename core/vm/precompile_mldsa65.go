package vm

import (
	"github.com/cloudflare/circl/sign/mldsa/mldsa65"
	"github.com/ethereum/go-ethereum/common"
)

// Address: 0x0000000000000000000000000000000000000101.
var mldsa65VerifyAddr = common.HexToAddress("0101")

const (
	mldsa65GasBase    uint64 = 250_000
	mldsa65GasPerWord uint64 = 12
)

type mldsa65VerifyPrecompile struct{}

func (p *mldsa65VerifyPrecompile) RequiredGas(input []byte) uint64 {
	pkLen := mldsa65.PublicKeySize
	sigLen := mldsa65.SignatureSize
	if len(input) <= pkLen+sigLen {
		return mldsa65GasBase
	}
	msgLen := len(input) - pkLen - sigLen
	words := uint64((msgLen + 31) / 32)
	return mldsa65GasBase + mldsa65GasPerWord*words
}

func (p *mldsa65VerifyPrecompile) Run(input []byte) ([]byte, error) {
	one := common.LeftPadBytes([]byte{1}, 32)
	zero := make([]byte, 32)

	pkLen := mldsa65.PublicKeySize
	sigLen := mldsa65.SignatureSize
	if len(input) < pkLen+sigLen {
		return zero, nil
	}
	pkBytes := input[:pkLen]
	sig := input[pkLen : pkLen+sigLen]
	msg := input[pkLen+sigLen:]

	var pk mldsa65.PublicKey
	if err := pk.UnmarshalBinary(pkBytes); err != nil {
		return zero, nil
	}
	if mldsa65.Verify(&pk, msg, nil, sig) {
		return one, nil
	}
	return zero, nil
}

func (p *mldsa65VerifyPrecompile) Name() string {
	return "MLDSA65V"
}
