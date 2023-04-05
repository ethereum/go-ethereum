package vm

/*
#cgo LDFLAGS: mina/target/release/libmina.a -ldl
#include "../../mina/target/mina.h"
*/
import "C"
import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/crypto"
)

var sol_bool, _ = abi.NewType("bool", "", nil)
var sol_uint8, _ = abi.NewType("uint8", "", nil)
var sol_bytes32, _ = abi.NewType("bytes32", "", nil)
var sol_bytes32Arr, _ = abi.NewType("bytes32[]", "", nil)

type MinaPoseidon struct{}

func (c *MinaPoseidon) RequiredGas(input []byte) uint64 {
	return 1000
}

// 0x1f831f84
var poseidonHashSignature = crypto.Keccak256([]byte("poseidonHash(uint8,bytes32[])"))[:4]

func (c *MinaPoseidon) Run(input []byte) ([]byte, error) {
	if len(input) < 4+64 || !bytes.Equal(input[:4], poseidonHashSignature) {
		return nil, ErrExecutionReverted
	}

	calldata := input[4:]

	unpacked, err := (abi.Arguments{{
		Type: sol_uint8}, // networkId
		{Type: sol_bytes32Arr}, // fields
	}).Unpack(calldata)

	if err != nil {
		return nil, err
	}

	networkId := unpacked[0].(uint8)
	fields := unpacked[1].([][32]uint8)

	if len(fields) == 0 {
		return nil, ErrExecutionReverted
	}

	output_buffer := [32]byte{}

	if !C.poseidon(
		C.uint8_t(networkId),
		(*C.uint8_t)(&fields[0][0]),
		C.uintptr_t(len(fields)),
		(*C.uint8_t)(&output_buffer[0]),
	) {
		return nil, ErrExecutionReverted
	}

	return output_buffer[:], nil
}

type MinaSigner struct{}

func (c *MinaSigner) RequiredGas(input []byte) uint64 {
	return 1000
}

// 0x462e39d6
var verifySignature = crypto.Keccak256([]byte("verify(uint8,bytes32,bytes32,bytes32,bytes32,bytes32[])"))[:4]

func (c *MinaSigner) Run(input []byte) ([]byte, error) {
	if len(input) < 4+64 || !bytes.Equal(input[:4], verifySignature) {
		return nil, ErrExecutionReverted
	}

	calldata := input[4:]

	unpacked, err := (abi.Arguments{
		{Type: sol_uint8},      // networkId
		{Type: sol_bytes32},    // pubKeyX
		{Type: sol_bytes32},    // pubKeyY
		{Type: sol_bytes32},    // signatureRX
		{Type: sol_bytes32},    // signatureS
		{Type: sol_bytes32Arr}, // fields
	}).Unpack(calldata)

	if err != nil {
		return nil, err
	}

	networkId := unpacked[0].(uint8)
	pubKeyX := unpacked[1].([32]uint8)
	pubKeyY := unpacked[2].([32]uint8)
	signatureRX := unpacked[3].([32]uint8)
	signatureS := unpacked[4].([32]uint8)
	fields := unpacked[5].([][32]uint8)

	if len(fields) == 0 {
		return nil, ErrExecutionReverted
	}

	output_buffer := false

	if !C.verify(
		C.uint8_t(networkId),
		(*C.uint8_t)(&pubKeyX[0]),
		(*C.uint8_t)(&pubKeyY[0]),
		(*C.uint8_t)(&signatureRX[0]),
		(*C.uint8_t)(&signatureS[0]),
		(*C.uint8_t)(&fields[0][0]),
		C.uintptr_t(len(fields)),
		(*C.bool)(&output_buffer),
	) {
		return nil, ErrExecutionReverted
	}

	return abi.Arguments{{Type: sol_bool}}.Pack(output_buffer)
}
