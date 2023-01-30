// Copyright 2014 The go-ethereum Authors
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

package vm

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/blake2b"
	"github.com/ethereum/go-ethereum/crypto/bls12381"
	"github.com/ethereum/go-ethereum/crypto/bn256"
	"github.com/ethereum/go-ethereum/params"
	big2 "github.com/holiman/big"
	"golang.org/x/crypto/ripemd160"
)

// PrecompiledContract is the basic interface for native Go contracts. The implementation
// requires a deterministic gas count based on the input size of the Run method of the
// contract.
type PrecompiledContract interface {
	ContractRef
	// IsStateful returns true if the precompile contract can execute a state
	// transition or if it can access the StateDB.
	IsStateful() bool
	// RequiredPrice calculates the contract gas used
	RequiredGas(input []byte) uint64
	// Run runs the precompiled contract
	Run(evm *EVM, contract *Contract, readonly bool) ([]byte, error)
}

// PrecompiledContractsHomestead contains the default set of pre-compiled Ethereum
// contracts used in the Frontier and Homestead releases.
var PrecompiledContractsHomestead = map[common.Address]PrecompiledContract{
	ecrecover{}.Address():     &ecrecover{},
	sha256hash{}.Address():    &sha256hash{},
	ripemd160hash{}.Address(): &ripemd160hash{},
	dataCopy{}.Address():      &dataCopy{},
}

// PrecompiledContractsByzantium contains the default set of pre-compiled Ethereum
// contracts used in the Byzantium release.
var PrecompiledContractsByzantium = map[common.Address]PrecompiledContract{
	ecrecover{}.Address():               &ecrecover{},
	sha256hash{}.Address():              &sha256hash{},
	ripemd160hash{}.Address():           &ripemd160hash{},
	dataCopy{}.Address():                &dataCopy{},
	bigModExp{}.Address():               &bigModExp{eip2565: false},
	bn256AddByzantium{}.Address():       &bn256AddByzantium{},
	bn256ScalarMulByzantium{}.Address(): &bn256ScalarMulByzantium{},
	bn256PairingByzantium{}.Address():   &bn256PairingByzantium{},
}

// PrecompiledContractsIstanbul contains the default set of pre-compiled Ethereum
// contracts used in the Istanbul release.
var PrecompiledContractsIstanbul = map[common.Address]PrecompiledContract{
	ecrecover{}.Address():              &ecrecover{},
	sha256hash{}.Address():             &sha256hash{},
	ripemd160hash{}.Address():          &ripemd160hash{},
	dataCopy{}.Address():               &dataCopy{},
	bigModExp{}.Address():              &bigModExp{eip2565: false},
	bn256AddIstanbul{}.Address():       &bn256AddIstanbul{},
	bn256ScalarMulIstanbul{}.Address(): &bn256ScalarMulIstanbul{},
	bn256PairingIstanbul{}.Address():   &bn256PairingIstanbul{},
	blake2F{}.Address():                &blake2F{},
}

// PrecompiledContractsBerlin contains the default set of pre-compiled Ethereum
// contracts used in the Berlin release.
var PrecompiledContractsBerlin = map[common.Address]PrecompiledContract{
	ecrecover{}.Address():              &ecrecover{},
	sha256hash{}.Address():             &sha256hash{},
	ripemd160hash{}.Address():          &ripemd160hash{},
	dataCopy{}.Address():               &dataCopy{},
	bigModExp{}.Address():              &bigModExp{eip2565: true},
	bn256AddIstanbul{}.Address():       &bn256AddIstanbul{},
	bn256ScalarMulIstanbul{}.Address(): &bn256ScalarMulIstanbul{},
	bn256PairingIstanbul{}.Address():   &bn256PairingIstanbul{},
	blake2F{}.Address():                &blake2F{},
}

// PrecompiledContractsBLS contains the set of pre-compiled Ethereum
// contracts specified in EIP-2537. These are exported for testing purposes.
var PrecompiledContractsBLS = map[common.Address]PrecompiledContract{
	bls12381G1Add{}.Address():      &bls12381G1Add{},
	bls12381G1Mul{}.Address():      &bls12381G1Mul{},
	bls12381G1MultiExp{}.Address(): &bls12381G1MultiExp{},
	bls12381G2Add{}.Address():      &bls12381G2Add{},
	bls12381G2Mul{}.Address():      &bls12381G2Mul{},
	bls12381G2MultiExp{}.Address(): &bls12381G2MultiExp{},
	bls12381Pairing{}.Address():    &bls12381Pairing{},
	bls12381MapG1{}.Address():      &bls12381MapG1{},
	bls12381MapG2{}.Address():      &bls12381MapG2{},
}

var (
	// PrecompiledAddressesBerlin defines the default set of pre-compiled
	// Ethereum contract addresses used in the Berlin release.
	PrecompiledAddressesBerlin = []common.Address{
		ecrecover{}.Address(),
		sha256hash{}.Address(),
		ripemd160hash{}.Address(),
		dataCopy{}.Address(),
		bigModExp{}.Address(),
		bn256AddIstanbul{}.Address(),
		bn256ScalarMulIstanbul{}.Address(),
		bn256PairingIstanbul{}.Address(),
		blake2F{}.Address(),
	}
	// PrecompiledAddressesIstanbul defines the default set of pre-compiled
	// Ethereum contract addresses used in the Istanbul release.
	PrecompiledAddressesIstanbul = []common.Address{
		ecrecover{}.Address(),
		sha256hash{}.Address(),
		ripemd160hash{}.Address(),
		dataCopy{}.Address(),
		bigModExp{}.Address(),
		bn256AddIstanbul{}.Address(),
		bn256ScalarMulIstanbul{}.Address(),
		bn256PairingIstanbul{}.Address(),
		blake2F{}.Address(),
	}
	// PrecompiledAddressesByzantium defines the default set of pre-compiled
	// Ethereum contract addresses used in the Byzantium release.
	PrecompiledAddressesByzantium = []common.Address{
		ecrecover{}.Address(),
		sha256hash{}.Address(),
		ripemd160hash{}.Address(),
		dataCopy{}.Address(),
		bigModExp{}.Address(),
		bn256AddByzantium{}.Address(),
		bn256ScalarMulByzantium{}.Address(),
		bn256PairingByzantium{}.Address(),
	}
	// PrecompiledAddressesHomestead defines the default set of pre-compiled
	// Ethereum contract addresses used in the Homestead release.
	PrecompiledAddressesHomestead = []common.Address{
		ecrecover{}.Address(),
		sha256hash{}.Address(),
		ripemd160hash{}.Address(),
		dataCopy{}.Address(),
	}
)

// DefaultActivePrecompiles returns the set of precompiles enabled with the default configuration.
func DefaultActivePrecompiles(rules params.Rules) []common.Address {
	switch {
	case rules.IsBerlin:
		return PrecompiledAddressesBerlin
	case rules.IsIstanbul:
		return PrecompiledAddressesIstanbul
	case rules.IsByzantium:
		return PrecompiledAddressesByzantium
	default:
		return PrecompiledAddressesHomestead
	}
}

// DefaultPrecompiles define the mapping of address and precompiles from the default configuration
func DefaultPrecompiles(rules params.Rules) (precompiles map[common.Address]PrecompiledContract) {
	switch {
	case rules.IsBerlin:
		precompiles = PrecompiledContractsBerlin
	case rules.IsIstanbul:
		precompiles = PrecompiledContractsIstanbul
	case rules.IsByzantium:
		precompiles = PrecompiledContractsByzantium
	default:
		precompiles = PrecompiledContractsHomestead
	}

	return precompiles
}

// ActivePrecompiles returns the precompiles enabled with the current configuration.
//
// NOTE: The rules argument is ignored as the active precompiles can be set via the WithPrecompiles
// method according to the chain rules from the current block context.
func (evm *EVM) ActivePrecompiles(_ params.Rules) []common.Address {
	return evm.activePrecompiles
}

// Precompile returns a precompiled contract for the given address. This
// function returns false if the address is not a registered precompile.
func (evm *EVM) Precompile(addr common.Address) (PrecompiledContract, bool) {
	p, ok := evm.precompiles[addr]
	return p, ok
}

// WithPrecompiles sets the precompiled contracts and the slice of actives precompiles.
// IMPORTANT: This function does NOT validate the precompiles provided to the EVM. The caller should
// use the ValidatePrecompiles function for this purpose prior to calling WithPrecompiles.
func (evm *EVM) WithPrecompiles(
	precompiles map[common.Address]PrecompiledContract,
	activePrecompiles []common.Address,
) {
	evm.precompiles = precompiles
	evm.activePrecompiles = activePrecompiles
}

// ValidatePrecompiles validates the precompile map against the active
// precompile slice.
// It returns an error if the precompiled contract map has a different length
// than the slice of active contract addresses. This function also checks for
// duplicates, invalid addresses and empty precompile contract instances.
func ValidatePrecompiles(
	precompiles map[common.Address]PrecompiledContract,
	activePrecompiles []common.Address,
) error {
	if len(precompiles) != len(activePrecompiles) {
		return fmt.Errorf("precompiles length mismatch (expected %d, got %d)", len(precompiles), len(activePrecompiles))
	}

	dupActivePrecompiles := make(map[common.Address]bool)

	for _, addr := range activePrecompiles {
		if dupActivePrecompiles[addr] {
			return fmt.Errorf("duplicate active precompile: %s", addr)
		}

		precompile, ok := precompiles[addr]
		if !ok {
			return fmt.Errorf("active precompile address doesn't exist in precompiles map: %s", addr)
		}

		if precompile == nil {
			return fmt.Errorf("precompile contract cannot be nil: %s", addr)
		}

		if bytes.Equal(addr.Bytes(), common.Address{}.Bytes()) {
			return fmt.Errorf("precompile cannot be the zero address: %s", addr)
		}

		dupActivePrecompiles[addr] = true
	}

	return nil
}

// RunPrecompiledContract runs and evaluates the output of a precompiled contract.
// It returns
// - the returned bytes,
// - the _remaining_ gas,
// - any error that occurred
func (evm *EVM) RunPrecompiledContract(
	p PrecompiledContract,
	caller ContractRef,
	input []byte,
	suppliedGas uint64,
	value *big.Int,
	readOnly bool,
) (ret []byte, remainingGas uint64, err error) {
	return runPrecompiledContract(evm, p, caller, input, suppliedGas, value, readOnly)
}

func runPrecompiledContract(
	evm *EVM,
	p PrecompiledContract,
	caller ContractRef,
	input []byte,
	suppliedGas uint64,
	value *big.Int,
	readOnly bool,
) (ret []byte, remainingGas uint64, err error) {
	addrCopy := p.Address()
	inputCopy := make([]byte, len(input))
	copy(inputCopy, input)

	contract := NewPrecompile(caller, AccountRef(addrCopy), value, suppliedGas)
	contract.Input = inputCopy

	gasCost := p.RequiredGas(input)
	if !contract.UseGas(gasCost) {
		return nil, contract.Gas, ErrOutOfGas
	}

	output, err := p.Run(evm, contract, readOnly)
	return output, contract.Gas, err
}

// ECRECOVER implemented as a native contract.
type ecrecover struct{}

// Address defines the precompiled contract address. This MUST match the address
// set in the precompiled contract map.
func (ecrecover) Address() common.Address {
	return common.BytesToAddress([]byte{1})
}

// IsStateful returns false.
func (ecrecover) IsStateful() bool { return false }

func (c *ecrecover) RequiredGas(input []byte) uint64 {
	return params.EcrecoverGas
}

func (c *ecrecover) Run(evm *EVM, contract *Contract, readonly bool) ([]byte, error) {
	const ecRecoverInputLength = 128

	contract.Input = common.RightPadBytes(contract.Input, ecRecoverInputLength)
	// "input" is (hash, v, r, s), each 32 bytes
	// but for ecrecover we want (r, s, v)

	r := new(big.Int).SetBytes(contract.Input[64:96])
	s := new(big.Int).SetBytes(contract.Input[96:128])
	v := contract.Input[63] - 27

	// tighter sig s values input homestead only apply to tx sigs
	if !allZero(contract.Input[32:63]) || !crypto.ValidateSignatureValues(v, r, s, false) {
		return nil, nil
	}
	// We must make sure not to modify the 'input', so placing the 'v' along with
	// the signature needs to be done on a new allocation
	sig := make([]byte, 65)
	copy(sig, contract.Input[64:128])
	sig[64] = v
	// v needs to be at the end for libsecp256k1
	pubKey, err := crypto.Ecrecover(contract.Input[:32], sig)
	// make sure the public key is a valid one
	if err != nil {
		return nil, nil
	}

	// the first byte of pubkey is bitcoin heritage
	return common.LeftPadBytes(crypto.Keccak256(pubKey[1:])[12:], 32), nil
}

// SHA256 implemented as a native contract.
type sha256hash struct{}

// Address defines the precompiled contract address. This MUST match the address
// set in the precompiled contract map.
func (sha256hash) Address() common.Address {
	return common.BytesToAddress([]byte{2})
}

// IsStateful returns false.
func (sha256hash) IsStateful() bool { return false }

// RequiredGas returns the gas required to execute the pre-compiled contract.
//
// This method does not require any overflow checking as the input size gas costs
// required for anything significant is so high it's impossible to pay for.
func (c *sha256hash) RequiredGas(input []byte) uint64 {
	return uint64(len(input)+31)/32*params.Sha256PerWordGas + params.Sha256BaseGas
}

func (c *sha256hash) Run(evm *EVM, contract *Contract, readonly bool) ([]byte, error) {
	h := sha256.Sum256(contract.Input)
	return h[:], nil
}

// RIPEMD160 implemented as a native contract.
type ripemd160hash struct{}

// Address defines the precompiled contract address. This MUST match the address
// set in the precompiled contract map.
func (ripemd160hash) Address() common.Address {
	return common.BytesToAddress([]byte{3})
}

// IsStateful returns false.
func (ripemd160hash) IsStateful() bool { return false }

// RequiredGas returns the gas required to execute the pre-compiled contract.
//
// This method does not require any overflow checking as the input size gas costs
// required for anything significant is so high it's impossible to pay for.
func (c *ripemd160hash) RequiredGas(input []byte) uint64 {
	return uint64(len(input)+31)/32*params.Ripemd160PerWordGas + params.Ripemd160BaseGas
}

func (c *ripemd160hash) Run(evm *EVM, contract *Contract, readonly bool) ([]byte, error) {
	ripemd := ripemd160.New()
	ripemd.Write(contract.Input)
	return common.LeftPadBytes(ripemd.Sum(nil), 32), nil
}

// data copy implemented as a native contract.
type dataCopy struct{}

// Address defines the precompiled contract address. This MUST match the address
// set in the precompiled contract map.
func (dataCopy) Address() common.Address {
	return common.BytesToAddress([]byte{4})
}

// IsStateful returns false.
func (dataCopy) IsStateful() bool { return false }

// RequiredGas returns the gas required to execute the pre-compiled contract.
//
// This method does not require any overflow checking as the input size gas costs
// required for anything significant is so high it's impossible to pay for.
func (c *dataCopy) RequiredGas(input []byte) uint64 {
	return uint64(len(input)+31)/32*params.IdentityPerWordGas + params.IdentityBaseGas
}

func (c *dataCopy) Run(evm *EVM, contract *Contract, readonly bool) ([]byte, error) {
	return common.CopyBytes(contract.Input), nil
}

// bigModExp implements a native big integer exponential modular operation.
type bigModExp struct {
	eip2565 bool
}

var (
	big0      = big.NewInt(0)
	big1      = big.NewInt(1)
	big3      = big.NewInt(3)
	big4      = big.NewInt(4)
	big7      = big.NewInt(7)
	big8      = big.NewInt(8)
	big16     = big.NewInt(16)
	big20     = big.NewInt(20)
	big32     = big.NewInt(32)
	big64     = big.NewInt(64)
	big96     = big.NewInt(96)
	big480    = big.NewInt(480)
	big1024   = big.NewInt(1024)
	big3072   = big.NewInt(3072)
	big199680 = big.NewInt(199680)
)

// modexpMultComplexity implements bigModexp multComplexity formula, as defined in EIP-198
//
//	def mult_complexity(x):
//		if x <= 64: return x ** 2
//		elif x <= 1024: return x ** 2 // 4 + 96 * x - 3072
//		else: return x ** 2 // 16 + 480 * x - 199680
//
// where is x is max(length_of_MODULUS, length_of_BASE)
func modexpMultComplexity(x *big.Int) *big.Int {
	switch {
	case x.Cmp(big64) <= 0:
		x.Mul(x, x) // x ** 2
	case x.Cmp(big1024) <= 0:
		// (x ** 2 // 4 ) + ( 96 * x - 3072)
		x = new(big.Int).Add(
			new(big.Int).Div(new(big.Int).Mul(x, x), big4),
			new(big.Int).Sub(new(big.Int).Mul(big96, x), big3072),
		)
	default:
		// (x ** 2 // 16) + (480 * x - 199680)
		x = new(big.Int).Add(
			new(big.Int).Div(new(big.Int).Mul(x, x), big16),
			new(big.Int).Sub(new(big.Int).Mul(big480, x), big199680),
		)
	}
	return x
}

// Address defines the precompiled contract address. This MUST match the address
// set in the precompiled contract map.
func (bigModExp) Address() common.Address {
	return common.BytesToAddress([]byte{5})
}

// IsStateful returns false.
func (bigModExp) IsStateful() bool { return false }

// RequiredGas returns the gas required to execute the pre-compiled contract.
func (c *bigModExp) RequiredGas(input []byte) uint64 {
	var (
		baseLen = new(big.Int).SetBytes(getData(input, 0, 32))
		expLen  = new(big.Int).SetBytes(getData(input, 32, 32))
		modLen  = new(big.Int).SetBytes(getData(input, 64, 32))
	)
	if len(input) > 96 {
		input = input[96:]
	} else {
		input = input[:0]
	}
	// Retrieve the head 32 bytes of exp for the adjusted exponent length
	var expHead *big.Int
	if big.NewInt(int64(len(input))).Cmp(baseLen) <= 0 {
		expHead = new(big.Int)
	} else {
		if expLen.Cmp(big32) > 0 {
			expHead = new(big.Int).SetBytes(getData(input, baseLen.Uint64(), 32))
		} else {
			expHead = new(big.Int).SetBytes(getData(input, baseLen.Uint64(), expLen.Uint64()))
		}
	}
	// Calculate the adjusted exponent length
	var msb int
	if bitlen := expHead.BitLen(); bitlen > 0 {
		msb = bitlen - 1
	}
	adjExpLen := new(big.Int)
	if expLen.Cmp(big32) > 0 {
		adjExpLen.Sub(expLen, big32)
		adjExpLen.Mul(big8, adjExpLen)
	}
	adjExpLen.Add(adjExpLen, big.NewInt(int64(msb)))
	// Calculate the gas cost of the operation
	gas := new(big.Int).Set(math.BigMax(modLen, baseLen))
	if c.eip2565 {
		// EIP-2565 has three changes
		// 1. Different multComplexity (inlined here)
		// in EIP-2565 (https://eips.ethereum.org/EIPS/eip-2565):
		//
		// def mult_complexity(x):
		//    ceiling(x/8)^2
		//
		//where is x is max(length_of_MODULUS, length_of_BASE)
		gas = gas.Add(gas, big7)
		gas = gas.Div(gas, big8)
		gas.Mul(gas, gas)

		gas.Mul(gas, math.BigMax(adjExpLen, big1))
		// 2. Different divisor (`GQUADDIVISOR`) (3)
		gas.Div(gas, big3)
		if gas.BitLen() > 64 {
			return math.MaxUint64
		}
		// 3. Minimum price of 200 gas
		if gas.Uint64() < 200 {
			return 200
		}
		return gas.Uint64()
	}
	gas = modexpMultComplexity(gas)
	gas.Mul(gas, math.BigMax(adjExpLen, big1))
	gas.Div(gas, big20)

	if gas.BitLen() > 64 {
		return math.MaxUint64
	}
	return gas.Uint64()
}

func (c *bigModExp) Run(evm *EVM, contract *Contract, readonly bool) ([]byte, error) {
	var (
		baseLen = new(big.Int).SetBytes(getData(contract.Input, 0, 32)).Uint64()
		expLen  = new(big.Int).SetBytes(getData(contract.Input, 32, 32)).Uint64()
		modLen  = new(big.Int).SetBytes(getData(contract.Input, 64, 32)).Uint64()
	)
	if len(contract.Input) > 96 {
		contract.Input = contract.Input[96:]
	} else {
		contract.Input = contract.Input[:0]
	}
	// Handle a special case when both the base and mod length is zero
	if baseLen == 0 && modLen == 0 {
		return []byte{}, nil
	}
	// Retrieve the operands and execute the exponentiation
	var (
		base = new(big2.Int).SetBytes(getData(contract.Input, 0, baseLen))
		exp  = new(big2.Int).SetBytes(getData(contract.Input, baseLen, expLen))
		mod  = new(big2.Int).SetBytes(getData(contract.Input, baseLen+expLen, modLen))
		v    []byte
	)
	switch {
	case mod.BitLen() == 0:
		// Modulo 0 is undefined, return zero
		return common.LeftPadBytes([]byte{}, int(modLen)), nil
	case base.BitLen() == 1: // a bit length of 1 means it's 1 (or -1).
		// If base == 1, then we can just return base % mod (if mod >= 1, which it is)
		v = base.Mod(base, mod).Bytes()
	default:
		v = base.Exp(base, exp, mod).Bytes()
	}
	return common.LeftPadBytes(v, int(modLen)), nil
}

// newCurvePoint unmarshals a binary blob into a bn256 elliptic curve point,
// returning it, or an error if the point is invalid.
func newCurvePoint(blob []byte) (*bn256.G1, error) {
	p := new(bn256.G1)
	if _, err := p.Unmarshal(blob); err != nil {
		return nil, err
	}
	return p, nil
}

// newTwistPoint unmarshals a binary blob into a bn256 elliptic curve point,
// returning it, or an error if the point is invalid.
func newTwistPoint(blob []byte) (*bn256.G2, error) {
	p := new(bn256.G2)
	if _, err := p.Unmarshal(blob); err != nil {
		return nil, err
	}
	return p, nil
}

// runBn256Add implements the Bn256Add precompile, referenced by both
// Byzantium and Istanbul operations.
func runBn256Add(input []byte) ([]byte, error) {
	x, err := newCurvePoint(getData(input, 0, 64))
	if err != nil {
		return nil, err
	}
	y, err := newCurvePoint(getData(input, 64, 64))
	if err != nil {
		return nil, err
	}
	res := new(bn256.G1)
	res.Add(x, y)
	return res.Marshal(), nil
}

// bn256Add implements a native elliptic curve point addition conforming to
// Istanbul consensus rules.
type bn256AddIstanbul struct{}

// Address defines the precompiled contract address. This MUST match the address
// set in the precompiled contract map.
func (bn256AddIstanbul) Address() common.Address {
	return common.BytesToAddress([]byte{6})
}

// IsStateful returns false.
func (bn256AddIstanbul) IsStateful() bool { return false }

// RequiredGas returns the gas required to execute the pre-compiled contract.
func (c *bn256AddIstanbul) RequiredGas(input []byte) uint64 {
	return params.Bn256AddGasIstanbul
}

func (c *bn256AddIstanbul) Run(evm *EVM, contract *Contract, readonly bool) ([]byte, error) {
	return runBn256Add(contract.Input)
}

// bn256AddByzantium implements a native elliptic curve point addition
// conforming to Byzantium consensus rules.
type bn256AddByzantium struct{}

// Address defines the precompiled contract address. This MUST match the address
// set in the precompiled contract map.
func (bn256AddByzantium) Address() common.Address {
	return common.BytesToAddress([]byte{6})
}

// IsStateful returns false.
func (bn256AddByzantium) IsStateful() bool { return false }

// RequiredGas returns the gas required to execute the pre-compiled contract.
func (c *bn256AddByzantium) RequiredGas(input []byte) uint64 {
	return params.Bn256AddGasByzantium
}

func (c *bn256AddByzantium) Run(evm *EVM, contract *Contract, readonly bool) ([]byte, error) {
	return runBn256Add(contract.Input)
}

// runBn256ScalarMul implements the Bn256ScalarMul precompile, referenced by
// both Byzantium and Istanbul operations.
func runBn256ScalarMul(input []byte) ([]byte, error) {
	p, err := newCurvePoint(getData(input, 0, 64))
	if err != nil {
		return nil, err
	}
	res := new(bn256.G1)
	res.ScalarMult(p, new(big.Int).SetBytes(getData(input, 64, 32)))
	return res.Marshal(), nil
}

// bn256ScalarMulIstanbul implements a native elliptic curve scalar
// multiplication conforming to Istanbul consensus rules.
type bn256ScalarMulIstanbul struct{}

// Address defines the precompiled contract address. This MUST match the address
// set in the precompiled contract map.
func (bn256ScalarMulIstanbul) Address() common.Address {
	return common.BytesToAddress([]byte{7})
}

// IsStateful returns false.
func (bn256ScalarMulIstanbul) IsStateful() bool { return false }

// RequiredGas returns the gas required to execute the pre-compiled contract.
func (c *bn256ScalarMulIstanbul) RequiredGas(input []byte) uint64 {
	return params.Bn256ScalarMulGasIstanbul
}

func (c *bn256ScalarMulIstanbul) Run(evm *EVM, contract *Contract, readonly bool) ([]byte, error) {
	return runBn256ScalarMul(contract.Input)
}

// bn256ScalarMulByzantium implements a native elliptic curve scalar
// multiplication conforming to Byzantium consensus rules.
type bn256ScalarMulByzantium struct{}

// Address defines the precompiled contract address. This MUST match the address
// set in the precompiled contract map.
func (bn256ScalarMulByzantium) Address() common.Address {
	return common.BytesToAddress([]byte{7})
}

// IsStateful returns false.
func (bn256ScalarMulByzantium) IsStateful() bool { return false }

// RequiredGas returns the gas required to execute the pre-compiled contract.
func (c *bn256ScalarMulByzantium) RequiredGas(input []byte) uint64 {
	return params.Bn256ScalarMulGasByzantium
}

func (c *bn256ScalarMulByzantium) Run(evm *EVM, contract *Contract, readonly bool) ([]byte, error) {
	return runBn256ScalarMul(contract.Input)
}

var (
	// true32Byte is returned if the bn256 pairing check succeeds.
	true32Byte = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}

	// false32Byte is returned if the bn256 pairing check fails.
	false32Byte = make([]byte, 32)

	// errBadPairingInput is returned if the bn256 pairing input is invalid.
	errBadPairingInput = errors.New("bad elliptic curve pairing size")
)

// runBn256Pairing implements the Bn256Pairing precompile, referenced by both
// Byzantium and Istanbul operations.
func runBn256Pairing(input []byte) ([]byte, error) {
	// Handle some corner cases cheaply
	if len(input)%192 > 0 {
		return nil, errBadPairingInput
	}
	// Convert the input into a set of coordinates
	var (
		cs []*bn256.G1
		ts []*bn256.G2
	)
	for i := 0; i < len(input); i += 192 {
		c, err := newCurvePoint(input[i : i+64])
		if err != nil {
			return nil, err
		}
		t, err := newTwistPoint(input[i+64 : i+192])
		if err != nil {
			return nil, err
		}
		cs = append(cs, c)
		ts = append(ts, t)
	}
	// Execute the pairing checks and return the results
	if bn256.PairingCheck(cs, ts) {
		return true32Byte, nil
	}
	return false32Byte, nil
}

// bn256PairingIstanbul implements a pairing pre-compile for the bn256 curve
// conforming to Istanbul consensus rules.
type bn256PairingIstanbul struct{}

// Address defines the precompiled contract address. This MUST match the address
// set in the precompiled contract map.
func (bn256PairingIstanbul) Address() common.Address {
	return common.BytesToAddress([]byte{8})
}

// IsStateful returns false.
func (bn256PairingIstanbul) IsStateful() bool { return false }

// RequiredGas returns the gas required to execute the pre-compiled contract.
func (c *bn256PairingIstanbul) RequiredGas(input []byte) uint64 {
	return params.Bn256PairingBaseGasIstanbul + uint64(len(input)/192)*params.Bn256PairingPerPointGasIstanbul
}

func (c *bn256PairingIstanbul) Run(evm *EVM, contract *Contract, readonly bool) ([]byte, error) {
	return runBn256Pairing(contract.Input)
}

// bn256PairingByzantium implements a pairing pre-compile for the bn256 curve
// conforming to Byzantium consensus rules.
type bn256PairingByzantium struct{}

// Address defines the precompiled contract address. This MUST match the address
// set in the precompiled contract map.
func (bn256PairingByzantium) Address() common.Address {
	return common.BytesToAddress([]byte{8})
}

// IsStateful returns false.
func (bn256PairingByzantium) IsStateful() bool { return false }

// RequiredGas returns the gas required to execute the pre-compiled contract.
func (c *bn256PairingByzantium) RequiredGas(input []byte) uint64 {
	return params.Bn256PairingBaseGasByzantium + uint64(len(input)/192)*params.Bn256PairingPerPointGasByzantium
}

func (c *bn256PairingByzantium) Run(evm *EVM, contract *Contract, readonly bool) ([]byte, error) {
	return runBn256Pairing(contract.Input)
}

type blake2F struct{}

// Address defines the precompiled contract address. This MUST match the address
// set in the precompiled contract map.
func (blake2F) Address() common.Address {
	return common.BytesToAddress([]byte{9})
}

// IsStateful returns false.
func (blake2F) IsStateful() bool { return false }

func (c *blake2F) RequiredGas(input []byte) uint64 {
	// If the input is malformed, we can't calculate the gas, return 0 and let the
	// actual call choke and fault.
	if len(input) != blake2FInputLength {
		return 0
	}
	return uint64(binary.BigEndian.Uint32(input[0:4]))
}

const (
	blake2FInputLength        = 213
	blake2FFinalBlockBytes    = byte(1)
	blake2FNonFinalBlockBytes = byte(0)
)

var (
	errBlake2FInvalidInputLength = errors.New("invalid input length")
	errBlake2FInvalidFinalFlag   = errors.New("invalid final flag")
)

func (c *blake2F) Run(evm *EVM, contract *Contract, readonly bool) ([]byte, error) {
	// Make sure the input is valid (correct length and final flag)
	if len(contract.Input) != blake2FInputLength {
		return nil, errBlake2FInvalidInputLength
	}
	if contract.Input[212] != blake2FNonFinalBlockBytes && contract.Input[212] != blake2FFinalBlockBytes {
		return nil, errBlake2FInvalidFinalFlag
	}
	// Parse the input into the Blake2b call parameters
	var (
		rounds = binary.BigEndian.Uint32(contract.Input[0:4])
		final  = contract.Input[212] == blake2FFinalBlockBytes

		h [8]uint64
		m [16]uint64
		t [2]uint64
	)
	for i := 0; i < 8; i++ {
		offset := 4 + i*8
		h[i] = binary.LittleEndian.Uint64(contract.Input[offset : offset+8])
	}
	for i := 0; i < 16; i++ {
		offset := 68 + i*8
		m[i] = binary.LittleEndian.Uint64(contract.Input[offset : offset+8])
	}
	t[0] = binary.LittleEndian.Uint64(contract.Input[196:204])
	t[1] = binary.LittleEndian.Uint64(contract.Input[204:212])

	// Execute the compression function, extract and return the result
	blake2b.F(&h, m, t, final, rounds)

	output := make([]byte, 64)
	for i := 0; i < 8; i++ {
		offset := i * 8
		binary.LittleEndian.PutUint64(output[offset:offset+8], h[i])
	}
	return output, nil
}

var (
	errBLS12381InvalidInputLength          = errors.New("invalid input length")
	errBLS12381InvalidFieldElementTopBytes = errors.New("invalid field element top bytes")
	errBLS12381G1PointSubgroup             = errors.New("g1 point is not on correct subgroup")
	errBLS12381G2PointSubgroup             = errors.New("g2 point is not on correct subgroup")
)

// bls12381G1Add implements EIP-2537 G1Add precompile.
type bls12381G1Add struct{}

// Address defines the precompiled contract address. This MUST match the address
// set in the precompiled contract map.
func (bls12381G1Add) Address() common.Address {
	return common.BytesToAddress([]byte{10})
}

// IsStateful returns false.
func (bls12381G1Add) IsStateful() bool { return false }

// RequiredGas returns the gas required to execute the pre-compiled contract.
func (c *bls12381G1Add) RequiredGas(input []byte) uint64 {
	return params.Bls12381G1AddGas
}

func (c *bls12381G1Add) Run(evm *EVM, contract *Contract, readonly bool) ([]byte, error) {
	// Implements EIP-2537 G1Add precompile.
	// > G1 addition call expects `256` bytes as an input that is interpreted as byte concatenation of two G1 points (`128` bytes each).
	// > Output is an encoding of addition operation result - single G1 point (`128` bytes).
	if len(contract.Input) != 256 {
		return nil, errBLS12381InvalidInputLength
	}
	var err error
	var p0, p1 *bls12381.PointG1

	// Initialize G1
	g := bls12381.NewG1()

	// Decode G1 point p_0
	if p0, err = g.DecodePoint(contract.Input[:128]); err != nil {
		return nil, err
	}
	// Decode G1 point p_1
	if p1, err = g.DecodePoint(contract.Input[128:]); err != nil {
		return nil, err
	}

	// Compute r = p_0 + p_1
	r := g.New()
	g.Add(r, p0, p1)

	// Encode the G1 point result into 128 bytes
	return g.EncodePoint(r), nil
}

// bls12381G1Mul implements EIP-2537 G1Mul precompile.
type bls12381G1Mul struct{}

// Address defines the precompiled contract address. This MUST match the address
// set in the precompiled contract map.
func (bls12381G1Mul) Address() common.Address {
	return common.BytesToAddress([]byte{11})
}

// IsStateful returns false.
func (bls12381G1Mul) IsStateful() bool { return false }

// RequiredGas returns the gas required to execute the pre-compiled contract.
func (c *bls12381G1Mul) RequiredGas(input []byte) uint64 {
	return params.Bls12381G1MulGas
}

func (c *bls12381G1Mul) Run(evm *EVM, contract *Contract, readonly bool) ([]byte, error) {
	// Implements EIP-2537 G1Mul precompile.
	// > G1 multiplication call expects `160` bytes as an input that is interpreted as byte concatenation of encoding of G1 point (`128` bytes) and encoding of a scalar value (`32` bytes).
	// > Output is an encoding of multiplication operation result - single G1 point (`128` bytes).
	if len(contract.Input) != 160 {
		return nil, errBLS12381InvalidInputLength
	}
	var err error
	var p0 *bls12381.PointG1

	// Initialize G1
	g := bls12381.NewG1()

	// Decode G1 point
	if p0, err = g.DecodePoint(contract.Input[:128]); err != nil {
		return nil, err
	}
	// Decode scalar value
	e := new(big.Int).SetBytes(contract.Input[128:])

	// Compute r = e * p_0
	r := g.New()
	g.MulScalar(r, p0, e)

	// Encode the G1 point into 128 bytes
	return g.EncodePoint(r), nil
}

// bls12381G1MultiExp implements EIP-2537 G1MultiExp precompile.
type bls12381G1MultiExp struct{}

// Address defines the precompiled contract address. This MUST match the address
// set in the precompiled contract map.
func (bls12381G1MultiExp) Address() common.Address {
	return common.BytesToAddress([]byte{12})
}

// IsStateful returns false.
func (bls12381G1MultiExp) IsStateful() bool { return false }

// RequiredGas returns the gas required to execute the pre-compiled contract.
func (c *bls12381G1MultiExp) RequiredGas(input []byte) uint64 {
	// Calculate G1 point, scalar value pair length
	k := len(input) / 160
	if k == 0 {
		// Return 0 gas for small input length
		return 0
	}
	// Lookup discount value for G1 point, scalar value pair length
	var discount uint64
	if dLen := len(params.Bls12381MultiExpDiscountTable); k < dLen {
		discount = params.Bls12381MultiExpDiscountTable[k-1]
	} else {
		discount = params.Bls12381MultiExpDiscountTable[dLen-1]
	}
	// Calculate gas and return the result
	return (uint64(k) * params.Bls12381G1MulGas * discount) / 1000
}

func (c *bls12381G1MultiExp) Run(evm *EVM, contract *Contract, readonly bool) ([]byte, error) {
	// Implements EIP-2537 G1MultiExp precompile.
	// G1 multiplication call expects `160*k` bytes as an input that is interpreted as byte concatenation of `k` slices each of them being a byte concatenation of encoding of G1 point (`128` bytes) and encoding of a scalar value (`32` bytes).
	// Output is an encoding of multiexponentiation operation result - single G1 point (`128` bytes).
	k := len(contract.Input) / 160
	if len(contract.Input) == 0 || len(contract.Input)%160 != 0 {
		return nil, errBLS12381InvalidInputLength
	}
	var err error
	points := make([]*bls12381.PointG1, k)
	scalars := make([]*big.Int, k)

	// Initialize G1
	g := bls12381.NewG1()

	// Decode point scalar pairs
	for i := 0; i < k; i++ {
		off := 160 * i
		t0, t1, t2 := off, off+128, off+160
		// Decode G1 point
		if points[i], err = g.DecodePoint(contract.Input[t0:t1]); err != nil {
			return nil, err
		}
		// Decode scalar value
		scalars[i] = new(big.Int).SetBytes(contract.Input[t1:t2])
	}

	// Compute r = e_0 * p_0 + e_1 * p_1 + ... + e_(k-1) * p_(k-1)
	r := g.New()
	g.MultiExp(r, points, scalars)

	// Encode the G1 point to 128 bytes
	return g.EncodePoint(r), nil
}

// bls12381G2Add implements EIP-2537 G2Add precompile.
type bls12381G2Add struct{}

// Address defines the precompiled contract address. This MUST match the address
// set in the precompiled contract map.
func (bls12381G2Add) Address() common.Address {
	return common.BytesToAddress([]byte{13})
}

// IsStateful returns false.
func (bls12381G2Add) IsStateful() bool { return false }

// RequiredGas returns the gas required to execute the pre-compiled contract.
func (c *bls12381G2Add) RequiredGas(input []byte) uint64 {
	return params.Bls12381G2AddGas
}

func (c *bls12381G2Add) Run(evm *EVM, contract *Contract, readonly bool) ([]byte, error) {
	// Implements EIP-2537 G2Add precompile.
	// > G2 addition call expects `512` bytes as an input that is interpreted as byte concatenation of two G2 points (`256` bytes each).
	// > Output is an encoding of addition operation result - single G2 point (`256` bytes).
	if len(contract.Input) != 512 {
		return nil, errBLS12381InvalidInputLength
	}
	var err error
	var p0, p1 *bls12381.PointG2

	// Initialize G2
	g := bls12381.NewG2()
	r := g.New()

	// Decode G2 point p_0
	if p0, err = g.DecodePoint(contract.Input[:256]); err != nil {
		return nil, err
	}
	// Decode G2 point p_1
	if p1, err = g.DecodePoint(contract.Input[256:]); err != nil {
		return nil, err
	}

	// Compute r = p_0 + p_1
	g.Add(r, p0, p1)

	// Encode the G2 point into 256 bytes
	return g.EncodePoint(r), nil
}

// bls12381G2Mul implements EIP-2537 G2Mul precompile.
type bls12381G2Mul struct{}

// Address defines the precompiled contract address. This MUST match the address
// set in the precompiled contract map.
func (bls12381G2Mul) Address() common.Address {
	return common.BytesToAddress([]byte{14})
}

// IsStateful returns false.
func (bls12381G2Mul) IsStateful() bool { return false }

// RequiredGas returns the gas required to execute the pre-compiled contract.
func (c *bls12381G2Mul) RequiredGas(input []byte) uint64 {
	return params.Bls12381G2MulGas
}

func (c *bls12381G2Mul) Run(evm *EVM, contract *Contract, readonly bool) ([]byte, error) {
	// Implements EIP-2537 G2MUL precompile logic.
	// > G2 multiplication call expects `288` bytes as an input that is interpreted as byte concatenation of encoding of G2 point (`256` bytes) and encoding of a scalar value (`32` bytes).
	// > Output is an encoding of multiplication operation result - single G2 point (`256` bytes).
	if len(contract.Input) != 288 {
		return nil, errBLS12381InvalidInputLength
	}
	var err error
	var p0 *bls12381.PointG2

	// Initialize G2
	g := bls12381.NewG2()

	// Decode G2 point
	if p0, err = g.DecodePoint(contract.Input[:256]); err != nil {
		return nil, err
	}
	// Decode scalar value
	e := new(big.Int).SetBytes(contract.Input[256:])

	// Compute r = e * p_0
	r := g.New()
	g.MulScalar(r, p0, e)

	// Encode the G2 point into 256 bytes
	return g.EncodePoint(r), nil
}

// bls12381G2MultiExp implements EIP-2537 G2MultiExp precompile.
type bls12381G2MultiExp struct{}

// Address defines the precompiled contract address. This MUST match the address
// set in the precompiled contract map.
func (bls12381G2MultiExp) Address() common.Address {
	return common.BytesToAddress([]byte{15})
}

// IsStateful returns false.
func (bls12381G2MultiExp) IsStateful() bool { return false }

// RequiredGas returns the gas required to execute the pre-compiled contract.
func (c *bls12381G2MultiExp) RequiredGas(input []byte) uint64 {
	// Calculate G2 point, scalar value pair length
	k := len(input) / 288
	if k == 0 {
		// Return 0 gas for small input length
		return 0
	}
	// Lookup discount value for G2 point, scalar value pair length
	var discount uint64
	if dLen := len(params.Bls12381MultiExpDiscountTable); k < dLen {
		discount = params.Bls12381MultiExpDiscountTable[k-1]
	} else {
		discount = params.Bls12381MultiExpDiscountTable[dLen-1]
	}
	// Calculate gas and return the result
	return (uint64(k) * params.Bls12381G2MulGas * discount) / 1000
}

func (c *bls12381G2MultiExp) Run(evm *EVM, contract *Contract, readonly bool) ([]byte, error) {
	// Implements EIP-2537 G2MultiExp precompile logic
	// > G2 multiplication call expects `288*k` bytes as an input that is interpreted as byte concatenation of `k` slices each of them being a byte concatenation of encoding of G2 point (`256` bytes) and encoding of a scalar value (`32` bytes).
	// > Output is an encoding of multiexponentiation operation result - single G2 point (`256` bytes).
	k := len(contract.Input) / 288
	if len(contract.Input) == 0 || len(contract.Input)%288 != 0 {
		return nil, errBLS12381InvalidInputLength
	}
	var err error
	points := make([]*bls12381.PointG2, k)
	scalars := make([]*big.Int, k)

	// Initialize G2
	g := bls12381.NewG2()

	// Decode point scalar pairs
	for i := 0; i < k; i++ {
		off := 288 * i
		t0, t1, t2 := off, off+256, off+288
		// Decode G1 point
		if points[i], err = g.DecodePoint(contract.Input[t0:t1]); err != nil {
			return nil, err
		}
		// Decode scalar value
		scalars[i] = new(big.Int).SetBytes(contract.Input[t1:t2])
	}

	// Compute r = e_0 * p_0 + e_1 * p_1 + ... + e_(k-1) * p_(k-1)
	r := g.New()
	g.MultiExp(r, points, scalars)

	// Encode the G2 point to 256 bytes.
	return g.EncodePoint(r), nil
}

// bls12381Pairing implements EIP-2537 Pairing precompile.
type bls12381Pairing struct{}

// Address defines the precompiled contract address. This MUST match the address
// set in the precompiled contract map.
func (bls12381Pairing) Address() common.Address {
	return common.BytesToAddress([]byte{16})
}

// IsStateful returns false.
func (bls12381Pairing) IsStateful() bool { return false }

// RequiredGas returns the gas required to execute the pre-compiled contract.
func (c *bls12381Pairing) RequiredGas(input []byte) uint64 {
	return params.Bls12381PairingBaseGas + uint64(len(input)/384)*params.Bls12381PairingPerPairGas
}

func (c *bls12381Pairing) Run(evm *EVM, contract *Contract, readonly bool) ([]byte, error) {
	// Implements EIP-2537 Pairing precompile logic.
	// > Pairing call expects `384*k` bytes as an inputs that is interpreted as byte concatenation of `k` slices. Each slice has the following structure:
	// > - `128` bytes of G1 point encoding
	// > - `256` bytes of G2 point encoding
	// > Output is a `32` bytes where last single byte is `0x01` if pairing result is equal to multiplicative identity in a pairing target field and `0x00` otherwise
	// > (which is equivalent of Big Endian encoding of Solidity values `uint256(1)` and `uin256(0)` respectively).
	k := len(contract.Input) / 384
	if len(contract.Input) == 0 || len(contract.Input)%384 != 0 {
		return nil, errBLS12381InvalidInputLength
	}

	// Initialize BLS12-381 pairing engine
	e := bls12381.NewPairingEngine()
	g1, g2 := e.G1, e.G2

	// Decode pairs
	for i := 0; i < k; i++ {
		off := 384 * i
		t0, t1, t2 := off, off+128, off+384

		// Decode G1 point
		p1, err := g1.DecodePoint(contract.Input[t0:t1])
		if err != nil {
			return nil, err
		}
		// Decode G2 point
		p2, err := g2.DecodePoint(contract.Input[t1:t2])
		if err != nil {
			return nil, err
		}

		// 'point is on curve' check already done,
		// Here we need to apply subgroup checks.
		if !g1.InCorrectSubgroup(p1) {
			return nil, errBLS12381G1PointSubgroup
		}
		if !g2.InCorrectSubgroup(p2) {
			return nil, errBLS12381G2PointSubgroup
		}

		// Update pairing engine with G1 and G2 points
		e.AddPair(p1, p2)
	}
	// Prepare 32 byte output
	out := make([]byte, 32)

	// Compute pairing and set the result
	if e.Check() {
		out[31] = 1
	}
	return out, nil
}

// decodeBLS12381FieldElement decodes BLS12-381 elliptic curve field element.
// Removes top 16 bytes of 64 byte input.
func decodeBLS12381FieldElement(in []byte) ([]byte, error) {
	if len(in) != 64 {
		return nil, errors.New("invalid field element length")
	}
	// check top bytes
	for i := 0; i < 16; i++ {
		if in[i] != byte(0x00) {
			return nil, errBLS12381InvalidFieldElementTopBytes
		}
	}
	out := make([]byte, 48)
	copy(out[:], in[16:])
	return out, nil
}

// bls12381MapG1 implements EIP-2537 MapG1 precompile.
type bls12381MapG1 struct{}

// Address defines the precompiled contract address. This MUST match the address
// set in the precompiled contract map.
func (bls12381MapG1) Address() common.Address {
	return common.BytesToAddress([]byte{17})
}

// IsStateful returns false.
func (bls12381MapG1) IsStateful() bool { return false }

// RequiredGas returns the gas required to execute the pre-compiled contract.
func (c *bls12381MapG1) RequiredGas(input []byte) uint64 {
	return params.Bls12381MapG1Gas
}

func (c *bls12381MapG1) Run(evm *EVM, contract *Contract, readonly bool) ([]byte, error) {
	// Implements EIP-2537 Map_To_G1 precompile.
	// > Field-to-curve call expects `64` bytes an an input that is interpreted as a an element of the base field.
	// > Output of this call is `128` bytes and is G1 point following respective encoding rules.
	if len(contract.Input) != 64 {
		return nil, errBLS12381InvalidInputLength
	}

	// Decode input field element
	fe, err := decodeBLS12381FieldElement(contract.Input)
	if err != nil {
		return nil, err
	}

	// Initialize G1
	g := bls12381.NewG1()

	// Compute mapping
	r, err := g.MapToCurve(fe)
	if err != nil {
		return nil, err
	}

	// Encode the G1 point to 128 bytes
	return g.EncodePoint(r), nil
}

// bls12381MapG2 implements EIP-2537 MapG2 precompile.
type bls12381MapG2 struct{}

// Address defines the precompiled contract address. This MUST match the address
// set in the precompiled contract map.
func (bls12381MapG2) Address() common.Address {
	return common.BytesToAddress([]byte{18})
}

// IsStateful returns false.
func (bls12381MapG2) IsStateful() bool { return false }

// RequiredGas returns the gas required to execute the pre-compiled contract.
func (c *bls12381MapG2) RequiredGas(input []byte) uint64 {
	return params.Bls12381MapG2Gas
}

func (c *bls12381MapG2) Run(evm *EVM, contract *Contract, readonly bool) ([]byte, error) {
	// Implements EIP-2537 Map_FP2_TO_G2 precompile logic.
	// > Field-to-curve call expects `128` bytes an an input that is interpreted as a an element of the quadratic extension field.
	// > Output of this call is `256` bytes and is G2 point following respective encoding rules.
	if len(contract.Input) != 128 {
		return nil, errBLS12381InvalidInputLength
	}

	// Decode input field element
	fe := make([]byte, 96)
	c0, err := decodeBLS12381FieldElement(contract.Input[:64])
	if err != nil {
		return nil, err
	}
	copy(fe[48:], c0)
	c1, err := decodeBLS12381FieldElement(contract.Input[64:])
	if err != nil {
		return nil, err
	}
	copy(fe[:48], c1)

	// Initialize G2
	g := bls12381.NewG2()

	// Compute mapping
	r, err := g.MapToCurve(fe)
	if err != nil {
		return nil, err
	}

	// Encode the G2 point to 256 bytes
	return g.EncodePoint(r), nil
}
