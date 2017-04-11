// Copyright 2015 The go-ethereum Authors
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

package params

import "math/big"

const (
	MaximumExtraDataSize  uint64 = 32    // Maximum size extra data may be after Genesis.
	ExpByteGas            uint64 = 10    // Times ceil(log256(exponent)) for the EXP instruction.
	SloadGas              uint64 = 50    // Multiplied by the number of 32-byte words that are copied (round up) for any *COPY operation and added.
	CallValueTransferGas  uint64 = 9000  // Paid for CALL when the value transfer is non-zero.
	CallNewAccountGas     uint64 = 25000 // Paid for CALL when the destination address didn't exist prior.
	TxGas                 uint64 = 21000 // Per transaction not creating a contract. NOTE: Not payable on data of calls between transactions.
	TxGasContractCreation uint64 = 53000 // Per transaction that creates a contract. NOTE: Not payable on data of calls between transactions.
	TxDataZeroGas         uint64 = 4     // Per byte of data attached to a transaction that equals zero. NOTE: Not payable on data of calls between transactions.
	QuadCoeffDiv          uint64 = 512   // Divisor for the quadratic particle of the memory cost equation.
	SstoreSetGas          uint64 = 20000 // Once per SLOAD operation.
	LogDataGas            uint64 = 8     // Per byte in a LOG* operation's data.
	CallStipend           uint64 = 2300  // Free gas given at beginning of call.
	EcrecoverGas          uint64 = 3000  //
	Sha256WordGas         uint64 = 12    //

	Sha3Gas          uint64 = 30    // Once per SHA3 operation.
	Sha256Gas        uint64 = 60    //
	IdentityWordGas  uint64 = 3     //
	Sha3WordGas      uint64 = 6     // Once per word of the SHA3 operation's data.
	SstoreResetGas   uint64 = 5000  // Once per SSTORE operation if the zeroness changes from zero.
	SstoreClearGas   uint64 = 5000  // Once per SSTORE operation if the zeroness doesn't change.
	SstoreRefundGas  uint64 = 15000 // Once per SSTORE operation if the zeroness changes to zero.
	JumpdestGas      uint64 = 1     // Refunded gas, once per SSTORE operation if the zeroness changes to zero.
	IdentityGas      uint64 = 15    //
	EpochDuration    uint64 = 30000 // Duration between proof-of-work epochs.
	CallGas          uint64 = 40    // Once per CALL operation & message call transaction.
	CreateDataGas    uint64 = 200   //
	Ripemd160Gas     uint64 = 600   //
	Ripemd160WordGas uint64 = 120   //
	CallCreateDepth  uint64 = 1024  // Maximum depth of call/create stack.
	ExpGas           uint64 = 10    // Once per EXP instruction
	LogGas           uint64 = 375   // Per LOG* operation.
	CopyGas          uint64 = 3     //
	StackLimit       uint64 = 1024  // Maximum size of VM stack allowed.
	TierStepGas      uint64 = 0     // Once per operation, for a selection of them.
	LogTopicGas      uint64 = 375   // Multiplied by the * of the LOG*, per LOG transaction. e.g. LOG0 incurs 0 * c_txLogTopicGas, LOG4 incurs 4 * c_txLogTopicGas.
	CreateGas        uint64 = 32000 // Once per CREATE operation & contract-creation transaction.
	SuicideRefundGas uint64 = 24000 // Refunded following a suicide operation.
	MemoryGas        uint64 = 3     // Times the address of the (highest referenced byte in memory + 1). NOTE: referencing happens on read, write and in instructions such as RETURN and CALL.
	TxDataNonZeroGas uint64 = 68    // Per byte of data attached to a transaction that is not equal to zero. NOTE: Not payable on data of calls between transactions.

	MaxCodeSize = 24576
)

var (
	GasLimitBoundDivisor   = big.NewInt(1024)                  // The bound divisor of the gas limit, used in update calculations.
	MinGasLimit            = big.NewInt(5000)                  // Minimum the gas limit may ever be.
	GenesisGasLimit        = big.NewInt(4712388)               // Gas limit of the Genesis block.
	TargetGasLimit         = new(big.Int).Set(GenesisGasLimit) // The artificial target
	DifficultyBoundDivisor = big.NewInt(2048)                  // The bound divisor of the difficulty, used in the update calculations.
	GenesisDifficulty      = big.NewInt(131072)                // Difficulty of the Genesis block.
	MinimumDifficulty      = big.NewInt(131072)                // The minimum that the difficulty may ever be.
	DurationLimit          = big.NewInt(13)                    // The decision boundary on the blocktime duration used to determine whether difficulty should go up or not.
)
