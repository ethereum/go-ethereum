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

var (
	MaximumExtraDataSize   = big.NewInt(32)     // Maximum size extra data may be after Genesis.
	ExpByteGas             = big.NewInt(10)     // Times ceil(log256(exponent)) for the EXP instruction.
	SloadGas               = big.NewInt(50)     // Multiplied by the number of 32-byte words that are copied (round up) for any *COPY operation and added.
	CallValueTransferGas   = big.NewInt(9000)   // Paid for CALL when the value transfer is non-zero.
	CallNewAccountGas      = big.NewInt(25000)  // Paid for CALL when the destination address didn't exist prior.
	TxGas                  = big.NewInt(21000)  // Per transaction not creating a contract. NOTE: Not payable on data of calls between transactions.
	TxGasContractCreation  = big.NewInt(53000)  // Per transaction that creates a contract. NOTE: Not payable on data of calls between transactions.
	TxDataZeroGas          = big.NewInt(4)      // Per byte of data attached to a transaction that equals zero. NOTE: Not payable on data of calls between transactions.
	DifficultyBoundDivisor = big.NewInt(2048)   // The bound divisor of the difficulty, used in the update calculations.
	QuadCoeffDiv           = big.NewInt(512)    // Divisor for the quadratic particle of the memory cost equation.
	GenesisDifficulty      = big.NewInt(131072) // Difficulty of the Genesis block.
	DurationLimit          = big.NewInt(13)     // The decision boundary on the blocktime duration used to determine whether difficulty should go up or not.
	SstoreSetGas           = big.NewInt(20000)  // Once per SLOAD operation.
	LogDataGas             = big.NewInt(8)      // Per byte in a LOG* operation's data.
	CallStipend            = big.NewInt(2300)   // Free gas given at beginning of call.
	EcrecoverGas           = big.NewInt(3000)   //
	Sha256WordGas          = big.NewInt(12)     //

	MinGasLimit     = big.NewInt(5000)                  // Minimum the gas limit may ever be.
	GenesisGasLimit = big.NewInt(4712388)               // Gas limit of the Genesis block.
	TargetGasLimit  = new(big.Int).Set(GenesisGasLimit) // The artificial target

	Sha3Gas              = big.NewInt(30)     // Once per SHA3 operation.
	Sha256Gas            = big.NewInt(60)     //
	IdentityWordGas      = big.NewInt(3)      //
	Sha3WordGas          = big.NewInt(6)      // Once per word of the SHA3 operation's data.
	SstoreResetGas       = big.NewInt(5000)   // Once per SSTORE operation if the zeroness changes from zero.
	SstoreClearGas       = big.NewInt(5000)   // Once per SSTORE operation if the zeroness doesn't change.
	SstoreRefundGas      = big.NewInt(15000)  // Once per SSTORE operation if the zeroness changes to zero.
	JumpdestGas          = big.NewInt(1)      // Refunded gas, once per SSTORE operation if the zeroness changes to zero.
	IdentityGas          = big.NewInt(15)     //
	GasLimitBoundDivisor = big.NewInt(1024)   // The bound divisor of the gas limit, used in update calculations.
	EpochDuration        = big.NewInt(30000)  // Duration between proof-of-work epochs.
	CallGas              = big.NewInt(40)     // Once per CALL operation & message call transaction.
	CreateDataGas        = big.NewInt(200)    //
	Ripemd160Gas         = big.NewInt(600)    //
	Ripemd160WordGas     = big.NewInt(120)    //
	MinimumDifficulty    = big.NewInt(131072) // The minimum that the difficulty may ever be.
	CallCreateDepth      = big.NewInt(1024)   // Maximum depth of call/create stack.
	ExpGas               = big.NewInt(10)     // Once per EXP instuction.
	LogGas               = big.NewInt(375)    // Per LOG* operation.
	CopyGas              = big.NewInt(3)      //
	StackLimit           = big.NewInt(1024)   // Maximum size of VM stack allowed.
	TierStepGas          = big.NewInt(0)      // Once per operation, for a selection of them.
	LogTopicGas          = big.NewInt(375)    // Multiplied by the * of the LOG*, per LOG transaction. e.g. LOG0 incurs 0 * c_txLogTopicGas, LOG4 incurs 4 * c_txLogTopicGas.
	CreateGas            = big.NewInt(32000)  // Once per CREATE operation & contract-creation transaction.
	SuicideRefundGas     = big.NewInt(24000)  // Refunded following a suicide operation.
	MemoryGas            = big.NewInt(3)      // Times the address of the (highest referenced byte in memory + 1). NOTE: referencing happens on read, write and in instructions such as RETURN and CALL.
	TxDataNonZeroGas     = big.NewInt(68)     // Per byte of data attached to a transaction that is not equal to zero. NOTE: Not payable on data of calls between transactions.

	MaxCodeSize = 24576
)
