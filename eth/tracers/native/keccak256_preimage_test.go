// Copyright 2021 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package native_test

import (
	"encoding/json"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// mockOpContext implements tracing.OpContext for testing
type mockOpContext struct {
	memory []byte
	stack  []uint256.Int
}

// Ensure mockOpContext implements tracing.OpContext
var _ tracing.OpContext = (*mockOpContext)(nil)

func (m *mockOpContext) MemoryData() []byte {
	return m.memory
}

func (m *mockOpContext) StackData() []uint256.Int {
	return m.stack
}

func (m *mockOpContext) Address() common.Address {
	return common.Address{}
}

func (m *mockOpContext) Caller() common.Address {
	return common.Address{}
}

func (m *mockOpContext) CallValue() *uint256.Int {
	return uint256.NewInt(0)
}

func (m *mockOpContext) CallInput() []byte {
	return []byte{}
}

func (m *mockOpContext) ContractCode() []byte {
	return []byte{}
}

func TestKeccak256PreimageTracerCreation(t *testing.T) {
	tracer, err := tracers.DefaultDirectory.New("keccak256PreimageTracer", &tracers.Context{}, nil, params.MainnetChainConfig)
	require.NoError(t, err)
	require.NotNil(t, tracer)
	require.NotNil(t, tracer.Hooks)
	require.NotNil(t, tracer.Hooks.OnOpcode)
	require.NotNil(t, tracer.GetResult)
}

func TestKeccak256PreimageTracerInitialResult(t *testing.T) {
	tracer, err := tracers.DefaultDirectory.New("keccak256PreimageTracer", &tracers.Context{}, nil, params.MainnetChainConfig)
	require.NoError(t, err)

	result, err := tracer.GetResult()
	require.NoError(t, err)

	var hashes map[common.Hash]hexutil.Bytes
	err = json.Unmarshal(result, &hashes)
	require.NoError(t, err)
	require.Empty(t, hashes)
}

func TestKeccak256PreimageTracerSingleKeccak(t *testing.T) {
	tracer, err := tracers.DefaultDirectory.New("keccak256PreimageTracer", &tracers.Context{}, nil, params.MainnetChainConfig)
	require.NoError(t, err)

	// Test data: "hello world"
	testData := []byte("hello world")
	memory := make([]byte, 32)
	copy(memory, testData)

	// Create stack with offset=0, length=11
	stack := []uint256.Int{
		*uint256.NewInt(11), // length (stack[1])
		*uint256.NewInt(0),  // offset (stack[0])
	}

	mockScope := &mockOpContext{
		memory: memory,
		stack:  stack,
	}

	// Call OnOpcode with KECCAK256
	tracer.OnOpcode(0, byte(vm.KECCAK256), 0, 0, mockScope, nil, 0, nil)

	// Get result
	result, err := tracer.GetResult()
	require.NoError(t, err)

	var hashes map[common.Hash]hexutil.Bytes
	err = json.Unmarshal(result, &hashes)
	require.NoError(t, err)

	// Verify the hash and preimage
	expectedHash := crypto.Keccak256Hash(testData)
	require.Len(t, hashes, 1)
	require.Contains(t, hashes, expectedHash)
	require.Equal(t, hexutil.Bytes(testData), hashes[expectedHash])
}

func TestKeccak256PreimageTracerMultipleKeccak(t *testing.T) {
	tracer, err := tracers.DefaultDirectory.New("keccak256PreimageTracer", &tracers.Context{}, nil, params.MainnetChainConfig)
	require.NoError(t, err)

	testCases := []struct {
		name string
		data []byte
	}{
		{"empty", []byte{}},
		{"hello", []byte("hello")},
		{"world", []byte("world")},
		{"long_data", make([]byte, 100)},
	}

	// Initialize long_data with some pattern
	for i := range testCases[3].data {
		testCases[3].data[i] = byte(i % 256)
	}

	expectedHashes := make(map[common.Hash]hexutil.Bytes)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			memory := make([]byte, max(len(tc.data), 1))
			copy(memory, tc.data)

			stack := []uint256.Int{
				*uint256.NewInt(uint64(len(tc.data))), // length
				*uint256.NewInt(0),                    // offset
			}

			mockScope := &mockOpContext{
				memory: memory,
				stack:  stack,
			}

			// Call OnOpcode with KECCAK256
			tracer.OnOpcode(0, byte(vm.KECCAK256), 0, 0, mockScope, nil, 0, nil)

			expectedHash := crypto.Keccak256Hash(tc.data)
			expectedHashes[expectedHash] = hexutil.Bytes(tc.data)
		})
	}

	// Get final result
	result, err := tracer.GetResult()
	require.NoError(t, err)

	var hashes map[common.Hash]hexutil.Bytes
	err = json.Unmarshal(result, &hashes)
	require.NoError(t, err)

	require.Equal(t, expectedHashes, hashes)
}

func TestKeccak256PreimageTracerNonKeccakOpcodes(t *testing.T) {
	tracer, err := tracers.DefaultDirectory.New("keccak256PreimageTracer", &tracers.Context{}, nil, params.MainnetChainConfig)
	require.NoError(t, err)

	testData := []byte("should not be recorded")
	memory := make([]byte, 32)
	copy(memory, testData)

	stack := []uint256.Int{
		*uint256.NewInt(uint64(len(testData))),
		*uint256.NewInt(0),
	}

	mockScope := &mockOpContext{
		memory: memory,
		stack:  stack,
	}

	// Test various non-KECCAK256 opcodes
	nonKeccakOpcodes := []vm.OpCode{
		vm.ADD, vm.MUL, vm.SUB, vm.DIV, vm.SDIV, vm.MOD, vm.SMOD,
		vm.ADDMOD, vm.MULMOD, vm.EXP, vm.SIGNEXTEND, vm.SLOAD,
		vm.SSTORE, vm.MLOAD, vm.MSTORE, vm.CALL, vm.RETURN,
	}

	for _, opcode := range nonKeccakOpcodes {
		tracer.OnOpcode(0, byte(opcode), 0, 0, mockScope, nil, 0, nil)
	}

	// Get result - should be empty
	result, err := tracer.GetResult()
	require.NoError(t, err)

	var hashes map[common.Hash]hexutil.Bytes
	err = json.Unmarshal(result, &hashes)
	require.NoError(t, err)
	require.Empty(t, hashes)
}

func TestKeccak256PreimageTracerMemoryOffset(t *testing.T) {
	tracer, err := tracers.DefaultDirectory.New("keccak256PreimageTracer", &tracers.Context{}, nil, params.MainnetChainConfig)
	require.NoError(t, err)

	// Test data at different memory offset
	prefix := []byte("prefix_data_")
	testData := []byte("target_data")
	memory := make([]byte, len(prefix)+len(testData)+10)
	copy(memory, prefix)
	copy(memory[len(prefix):], testData)

	// Stack: offset=len(prefix), length=len(testData)
	stack := []uint256.Int{
		*uint256.NewInt(uint64(len(testData))), // length
		*uint256.NewInt(uint64(len(prefix))),   // offset
	}

	mockScope := &mockOpContext{
		memory: memory,
		stack:  stack,
	}

	// Call OnOpcode with KECCAK256
	tracer.OnOpcode(0, byte(vm.KECCAK256), 0, 0, mockScope, nil, 0, nil)

	// Get result
	result, err := tracer.GetResult()
	require.NoError(t, err)

	var hashes map[common.Hash]hexutil.Bytes
	err = json.Unmarshal(result, &hashes)
	require.NoError(t, err)

	// Verify the hash matches the target data, not the prefix
	expectedHash := crypto.Keccak256Hash(testData)
	require.Len(t, hashes, 1)
	require.Contains(t, hashes, expectedHash)
	require.Equal(t, hexutil.Bytes(testData), hashes[expectedHash])
}

func TestKeccak256PreimageTracerMemoryPadding(t *testing.T) {
	tracer, err := tracers.DefaultDirectory.New("keccak256PreimageTracer", &tracers.Context{}, nil, params.MainnetChainConfig)
	require.NoError(t, err)

	// Test data that extends beyond memory bounds (should be zero-padded)
	testData := []byte("short")
	memory := make([]byte, len(testData))
	copy(memory, testData)

	// Request more data than available in memory
	requestedLength := len(testData) + 5
	stack := []uint256.Int{
		*uint256.NewInt(uint64(requestedLength)), // length > memory size
		*uint256.NewInt(0),                       // offset
	}

	mockScope := &mockOpContext{
		memory: memory,
		stack:  stack,
	}

	// Call OnOpcode with KECCAK256
	tracer.OnOpcode(0, byte(vm.KECCAK256), 0, 0, mockScope, nil, 0, nil)

	// Get result
	result, err := tracer.GetResult()
	require.NoError(t, err)

	var hashes map[common.Hash]hexutil.Bytes
	err = json.Unmarshal(result, &hashes)
	require.NoError(t, err)

	// Verify the hash includes zero padding
	expectedData := make([]byte, requestedLength)
	copy(expectedData, testData)
	// Rest is zero-padded by default

	expectedHash := crypto.Keccak256Hash(expectedData)
	require.Len(t, hashes, 1)
	require.Contains(t, hashes, expectedHash)
	require.Equal(t, hexutil.Bytes(expectedData), hashes[expectedHash])
}

func TestKeccak256PreimageTracerDuplicateHashes(t *testing.T) {
	tracer, err := tracers.DefaultDirectory.New("keccak256PreimageTracer", &tracers.Context{}, nil, params.MainnetChainConfig)
	require.NoError(t, err)

	testData := []byte("duplicate_test")
	memory := make([]byte, len(testData))
	copy(memory, testData)

	stack := []uint256.Int{
		*uint256.NewInt(uint64(len(testData))),
		*uint256.NewInt(0),
	}

	mockScope := &mockOpContext{
		memory: memory,
		stack:  stack,
	}

	// Call OnOpcode with KECCAK256 multiple times with same data
	for i := 0; i < 3; i++ {
		tracer.OnOpcode(0, byte(vm.KECCAK256), 0, 0, mockScope, nil, 0, nil)
	}

	// Get result
	result, err := tracer.GetResult()
	require.NoError(t, err)

	var hashes map[common.Hash]hexutil.Bytes
	err = json.Unmarshal(result, &hashes)
	require.NoError(t, err)

	// Should only have one entry (duplicates overwrite)
	expectedHash := crypto.Keccak256Hash(testData)
	require.Len(t, hashes, 1)
	require.Contains(t, hashes, expectedHash)
	require.Equal(t, hexutil.Bytes(testData), hashes[expectedHash])
}

func TestKeccak256PreimageTracerWithExecutionError(t *testing.T) {
	tracer, err := tracers.DefaultDirectory.New("keccak256PreimageTracer", &tracers.Context{}, nil, params.MainnetChainConfig)
	require.NoError(t, err)

	testData := []byte("error_test")
	memory := make([]byte, len(testData))
	copy(memory, testData)

	stack := []uint256.Int{
		*uint256.NewInt(uint64(len(testData))),
		*uint256.NewInt(0),
	}

	mockScope := &mockOpContext{
		memory: memory,
		stack:  stack,
	}

	// Call OnOpcode with KECCAK256 and an execution error
	tracer.OnOpcode(0, byte(vm.KECCAK256), 0, 0, mockScope, nil, 0, vm.ErrOutOfGas)

	// Get result - should still record the hash even with execution error
	result, err := tracer.GetResult()
	require.NoError(t, err)

	var hashes map[common.Hash]hexutil.Bytes
	err = json.Unmarshal(result, &hashes)
	require.NoError(t, err)

	expectedHash := crypto.Keccak256Hash(testData)
	require.Len(t, hashes, 1)
	require.Contains(t, hashes, expectedHash)
	require.Equal(t, hexutil.Bytes(testData), hashes[expectedHash])
}

func TestKeccak256PreimageTracerInsufficientStack(t *testing.T) {
	tracer, err := tracers.DefaultDirectory.New("keccak256PreimageTracer", &tracers.Context{}, nil, params.MainnetChainConfig)
	require.NoError(t, err)

	// Test with insufficient stack items (should cause panic, but we test it doesn't crash)
	testData := []byte("test")
	memory := make([]byte, len(testData))
	copy(memory, testData)

	// Stack with only one item (need 2 for KECCAK256)
	stack := []uint256.Int{
		*uint256.NewInt(0), // only offset, missing length
	}

	mockScope := &mockOpContext{
		memory: memory,
		stack:  stack,
	}

	// This should not panic due to insufficient stack
	tracer.OnOpcode(0, byte(vm.KECCAK256), 0, 0, mockScope, nil, 0, nil)
}

func TestKeccak256PreimageTracerLargeData(t *testing.T) {
	tracer, err := tracers.DefaultDirectory.New("keccak256PreimageTracer", &tracers.Context{}, nil, params.MainnetChainConfig)
	require.NoError(t, err)

	// Test with large data
	largeData := make([]byte, 1024)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	memory := make([]byte, len(largeData))
	copy(memory, largeData)

	stack := []uint256.Int{
		*uint256.NewInt(uint64(len(largeData))),
		*uint256.NewInt(0),
	}

	mockScope := &mockOpContext{
		memory: memory,
		stack:  stack,
	}

	// Call OnOpcode with KECCAK256
	tracer.OnOpcode(0, byte(vm.KECCAK256), 0, 0, mockScope, nil, 0, nil)

	// Get result
	result, err := tracer.GetResult()
	require.NoError(t, err)

	var hashes map[common.Hash]hexutil.Bytes
	err = json.Unmarshal(result, &hashes)
	require.NoError(t, err)

	expectedHash := crypto.Keccak256Hash(largeData)
	require.Len(t, hashes, 1)
	require.Contains(t, hashes, expectedHash)
	require.Equal(t, hexutil.Bytes(largeData), hashes[expectedHash])
}
