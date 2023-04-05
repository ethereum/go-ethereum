package vm

/*
#cgo LDFLAGS: mina/target/release/libmina.a -ldl
#include "../../mina/target/mina.h"
*/
import "C"
import (
	"bytes"
	"math/big"

	"github.com/ethereum/go-ethereum/crypto"
)

type MinaPoseidon struct{}

func (c *MinaPoseidon) RequiredGas(input []byte) uint64 {
	return 1000
}

// 0x1f831f84
var minaPoseidonSignature = crypto.Keccak256([]byte("poseidonHash(uint8,bytes32[])"))[:4]

const networkIdIndex = 0
const fieldsHeadIndex = networkIdIndex + 32

func (c *MinaPoseidon) Run(input []byte) ([]byte, error) {
	if !bytes.Equal(input[:4], minaPoseidonSignature) {
		return nil, ErrExecutionReverted
	}

	calldata := input[4:]

	networkId := new(big.Int).SetBytes(getData(calldata, networkIdIndex, 32)).Uint64()

	lenIndex := new(big.Int).SetBytes(getData(calldata, fieldsHeadIndex, 32)).Uint64()
	fieldsLen := new(big.Int).SetBytes(getData(calldata, lenIndex, 32)).Uint64()

	dataIndex := lenIndex + 32
	fields := calldata[dataIndex : dataIndex+fieldsLen*32]

	output_buffer := [32]byte{}

	if !C.poseidon_hash(
		uint32(networkId),
		(*C.uint8_t)(&fields[0]),
		C.uintptr_t(fieldsLen),
		(*C.uint8_t)(&output_buffer[0]),
	) {
		return nil, ErrExecutionReverted
	}

	return output_buffer[:], nil
}
