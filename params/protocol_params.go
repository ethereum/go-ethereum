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

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

const (
	GasLimitBoundDivisor uint64 = 1024               // The bound divisor of the gas limit, used in update calculations.
	MinGasLimit          uint64 = 5000               // Minimum the gas limit may ever be.
	MaxGasLimit          uint64 = 0x7fffffffffffffff // Maximum the gas limit (2^63-1).
	GenesisGasLimit      uint64 = 4712388            // Gas limit of the Genesis block.

	MaxTxGas uint64 = 1 << 24 // Maximum transaction gas limit after eip-7825 (16,777,216).

	MaximumExtraDataSize  uint64 = 32    // Maximum size extra data may be after Genesis.
	ExpByteGas            uint64 = 10    // Times ceil(log256(exponent)) for the EXP instruction.
	SloadGas              uint64 = 50    //
	CallValueTransferGas  uint64 = 9000  // Paid for CALL when the value transfer is non-zero.
	CallNewAccountGas     uint64 = 25000 // Paid for CALL when the destination address didn't exist prior.
	TxGas                 uint64 = 21000 // Per transaction not creating a contract. NOTE: Not payable on data of calls between transactions.
	TxGasContractCreation uint64 = 53000 // Per transaction that creates a contract. NOTE: Not payable on data of calls between transactions.
	TxDataZeroGas         uint64 = 4     // Per byte of data attached to a transaction that equals zero. NOTE: Not payable on data of calls between transactions.
	QuadCoeffDiv          uint64 = 512   // Divisor for the quadratic particle of the memory cost equation.
	LogDataGas            uint64 = 8     // Per byte in a LOG* operation's data.
	CallStipend           uint64 = 2300  // Free gas given at beginning of call.

	Keccak256Gas     uint64 = 30 // Once per KECCAK256 operation.
	Keccak256WordGas uint64 = 6  // Once per word of the KECCAK256 operation's data.
	InitCodeWordGas  uint64 = 2  // Once per word of the init code when creating a contract.

	SstoreSetGas    uint64 = 20000 // Once per SSTORE operation.
	SstoreResetGas  uint64 = 5000  // Once per SSTORE operation if the zeroness changes from zero.
	SstoreClearGas  uint64 = 5000  // Once per SSTORE operation if the zeroness doesn't change.
	SstoreRefundGas uint64 = 15000 // Once per SSTORE operation if the zeroness changes to zero.

	NetSstoreNoopGas  uint64 = 200   // Once per SSTORE operation if the value doesn't change.
	NetSstoreInitGas  uint64 = 20000 // Once per SSTORE operation from clean zero.
	NetSstoreCleanGas uint64 = 5000  // Once per SSTORE operation from clean non-zero.
	NetSstoreDirtyGas uint64 = 200   // Once per SSTORE operation from dirty.

	NetSstoreClearRefund      uint64 = 15000 // Once per SSTORE operation for clearing an originally existing storage slot
	NetSstoreResetRefund      uint64 = 4800  // Once per SSTORE operation for resetting to the original non-zero value
	NetSstoreResetClearRefund uint64 = 19800 // Once per SSTORE operation for resetting to the original zero value

	SstoreSentryGasEIP2200            uint64 = 2300  // Minimum gas required to be present for an SSTORE call, not consumed
	SstoreSetGasEIP2200               uint64 = 20000 // Once per SSTORE operation from clean zero to non-zero
	SstoreResetGasEIP2200             uint64 = 5000  // Once per SSTORE operation from clean non-zero to something else
	SstoreClearsScheduleRefundEIP2200 uint64 = 15000 // Once per SSTORE operation for clearing an originally existing storage slot

	ColdAccountAccessCostEIP2929 = uint64(2600) // COLD_ACCOUNT_ACCESS_COST
	ColdSloadCostEIP2929         = uint64(2100) // COLD_SLOAD_COST
	WarmStorageReadCostEIP2929   = uint64(100)  // WARM_STORAGE_READ_COST

	// In EIP-2200: SstoreResetGas was 5000.
	// In EIP-2929: SstoreResetGas was changed to '5000 - COLD_SLOAD_COST'.
	// In EIP-3529: SSTORE_CLEARS_SCHEDULE is defined as SSTORE_RESET_GAS + ACCESS_LIST_STORAGE_KEY_COST
	// Which becomes: 5000 - 2100 + 1900 = 4800
	SstoreClearsScheduleRefundEIP3529 uint64 = SstoreResetGasEIP2200 - ColdSloadCostEIP2929 + TxAccessListStorageKeyGas

	JumpdestGas   uint64 = 1     // Once per JUMPDEST operation.
	EpochDuration uint64 = 30000 // Duration between proof-of-work epochs.

	CreateDataGas         uint64 = 200   //
	CallCreateDepth       uint64 = 1024  // Maximum depth of call/create stack.
	ExpGas                uint64 = 10    // Once per EXP instruction
	LogGas                uint64 = 375   // Per LOG* operation.
	CopyGas               uint64 = 3     //	Multiplied by the number of 32-byte words that are copied (round up) for any *COPY operation and added.
	StackLimit            uint64 = 1024  // Maximum size of VM stack allowed.
	TierStepGas           uint64 = 0     // Once per operation, for a selection of them.
	LogTopicGas           uint64 = 375   // Multiplied by the * of the LOG*, per LOG transaction. e.g. LOG0 incurs 0 * c_txLogTopicGas, LOG4 incurs 4 * c_txLogTopicGas.
	CreateGas             uint64 = 32000 // Once per CREATE operation & contract-creation transaction.
	Create2Gas            uint64 = 32000 // Once per CREATE2 operation
	CreateNGasEip4762     uint64 = 1000  // Once per CREATEn operations post-verkle
	SelfdestructRefundGas uint64 = 24000 // Refunded following a selfdestruct operation.
	MemoryGas             uint64 = 3     // Times the address of the (highest referenced byte in memory + 1). NOTE: referencing happens on read, write and in instructions such as RETURN and CALL.

	TxDataNonZeroGasFrontier  uint64 = 68    // Per byte of data attached to a transaction that is not equal to zero. NOTE: Not payable on data of calls between transactions.
	TxDataNonZeroGasEIP2028   uint64 = 16    // Per byte of non zero data attached to a transaction after EIP 2028 (part in Istanbul)
	TxTokenPerNonZeroByte     uint64 = 4     // Token cost per non-zero byte as specified by EIP-7623.
	TxCostFloorPerToken       uint64 = 10    // Cost floor per byte of data as specified by EIP-7623.
	TxAccessListAddressGas    uint64 = 2400  // Per address specified in EIP 2930 access list
	TxAccessListStorageKeyGas uint64 = 1900  // Per storage key specified in EIP 2930 access list
	TxAuthTupleGas            uint64 = 12500 // Per auth tuple code specified in EIP-7702

	// These have been changed during the course of the chain
	CallGasFrontier              uint64 = 40  // Once per CALL operation & message call transaction.
	CallGasEIP150                uint64 = 700 // Static portion of gas for CALL-derivates after EIP 150 (Tangerine)
	BalanceGasFrontier           uint64 = 20  // The cost of a BALANCE operation
	BalanceGasEIP150             uint64 = 400 // The cost of a BALANCE operation after Tangerine
	BalanceGasEIP1884            uint64 = 700 // The cost of a BALANCE operation after EIP 1884 (part of Istanbul)
	ExtcodeSizeGasFrontier       uint64 = 20  // Cost of EXTCODESIZE before EIP 150 (Tangerine)
	ExtcodeSizeGasEIP150         uint64 = 700 // Cost of EXTCODESIZE after EIP 150 (Tangerine)
	SloadGasFrontier             uint64 = 50
	SloadGasEIP150               uint64 = 200
	SloadGasEIP1884              uint64 = 800  // Cost of SLOAD after EIP 1884 (part of Istanbul)
	SloadGasEIP2200              uint64 = 800  // Cost of SLOAD after EIP 2200 (part of Istanbul)
	ExtcodeHashGasConstantinople uint64 = 400  // Cost of EXTCODEHASH (introduced in Constantinople)
	ExtcodeHashGasEIP1884        uint64 = 700  // Cost of EXTCODEHASH after EIP 1884 (part in Istanbul)
	SelfdestructGasEIP150        uint64 = 5000 // Cost of SELFDESTRUCT post EIP 150 (Tangerine)

	// EXP has a dynamic portion depending on the size of the exponent
	ExpByteFrontier uint64 = 10 // was set to 10 in Frontier
	ExpByteEIP158   uint64 = 50 // was raised to 50 during Eip158 (Spurious Dragon)

	// Extcodecopy has a dynamic AND a static cost. This represents only the
	// static portion of the gas. It was changed during EIP 150 (Tangerine)
	ExtcodeCopyBaseFrontier uint64 = 20
	ExtcodeCopyBaseEIP150   uint64 = 700

	// CreateBySelfdestructGas is used when the refunded account is one that does
	// not exist. This logic is similar to call.
	// Introduced in Tangerine Whistle (Eip 150)
	CreateBySelfdestructGas uint64 = 25000

	DefaultBaseFeeChangeDenominator = 8          // Bounds the amount the base fee can change between blocks.
	DefaultElasticityMultiplier     = 2          // Bounds the maximum gas limit an EIP-1559 block may have.
	InitialBaseFee                  = 1000000000 // Initial base fee for EIP-1559 blocks.

	MaxCodeSize     = 24576           // Maximum bytecode to permit for a contract
	MaxInitCodeSize = 2 * MaxCodeSize // Maximum initcode to permit in a creation transaction and create instructions

	// Precompiled contract gas prices

	EcrecoverGas        uint64 = 3000 // Elliptic curve sender recovery gas price
	Sha256BaseGas       uint64 = 60   // Base price for a SHA256 operation
	Sha256PerWordGas    uint64 = 12   // Per-word price for a SHA256 operation
	Ripemd160BaseGas    uint64 = 600  // Base price for a RIPEMD160 operation
	Ripemd160PerWordGas uint64 = 120  // Per-word price for a RIPEMD160 operation
	IdentityBaseGas     uint64 = 15   // Base price for a data copy operation
	IdentityPerWordGas  uint64 = 3    // Per-work price for a data copy operation

	Bn256AddGasByzantium             uint64 = 500    // Byzantium gas needed for an elliptic curve addition
	Bn256AddGasIstanbul              uint64 = 150    // Gas needed for an elliptic curve addition
	Bn256ScalarMulGasByzantium       uint64 = 40000  // Byzantium gas needed for an elliptic curve scalar multiplication
	Bn256ScalarMulGasIstanbul        uint64 = 6000   // Gas needed for an elliptic curve scalar multiplication
	Bn256PairingBaseGasByzantium     uint64 = 100000 // Byzantium base price for an elliptic curve pairing check
	Bn256PairingBaseGasIstanbul      uint64 = 45000  // Base price for an elliptic curve pairing check
	Bn256PairingPerPointGasByzantium uint64 = 80000  // Byzantium per-point price for an elliptic curve pairing check
	Bn256PairingPerPointGasIstanbul  uint64 = 34000  // Per-point price for an elliptic curve pairing check

	Bls12381G1AddGas          uint64 = 375   // Price for BLS12-381 elliptic curve G1 point addition
	Bls12381G1MulGas          uint64 = 12000 // Price for BLS12-381 elliptic curve G1 point scalar multiplication
	Bls12381G2AddGas          uint64 = 600   // Price for BLS12-381 elliptic curve G2 point addition
	Bls12381G2MulGas          uint64 = 22500 // Price for BLS12-381 elliptic curve G2 point scalar multiplication
	Bls12381PairingBaseGas    uint64 = 37700 // Base gas price for BLS12-381 elliptic curve pairing check
	Bls12381PairingPerPairGas uint64 = 32600 // Per-point pair gas price for BLS12-381 elliptic curve pairing check
	Bls12381MapG1Gas          uint64 = 5500  // Gas price for BLS12-381 mapping field element to G1 operation
	Bls12381MapG2Gas          uint64 = 23800 // Gas price for BLS12-381 mapping field element to G2 operation

	P256VerifyGas uint64 = 6900 // secp256r1 elliptic curve signature verifier gas price

	// The Refund Quotient is the cap on how much of the used gas can be refunded. Before EIP-3529,
	// up to half the consumed gas could be refunded. Redefined as 1/5th in EIP-3529
	RefundQuotient        uint64 = 2
	RefundQuotientEIP3529 uint64 = 5

	BlobTxBytesPerFieldElement         = 32      // Size in bytes of a field element
	BlobTxFieldElementsPerBlob         = 4096    // Number of field elements stored in a single data blob
	BlobTxBlobGasPerBlob               = 1 << 17 // Gas consumption of a single data blob (== blob byte size)
	BlobTxMinBlobGasprice              = 1       // Minimum gas price for data blobs
	BlobTxPointEvaluationPrecompileGas = 50000   // Gas price for the point evaluation precompile.
	BlobTxMaxBlobs                     = 6
	BlobBaseCost                       = 1 << 13 // Base execution gas cost for a blob.

	HistoryServeWindow = 8191 // Number of blocks to serve historical block hashes for, EIP-2935.

	MaxBlockSize = 8_388_608 // maximum size of an RLP-encoded block
)

// Bls12381G1MultiExpDiscountTable is the gas discount table for BLS12-381 G1 multi exponentiation operation
var Bls12381G1MultiExpDiscountTable = [128]uint64{1000, 949, 848, 797, 764, 750, 738, 728, 719, 712, 705, 698, 692, 687, 682, 677, 673, 669, 665, 661, 658, 654, 651, 648, 645, 642, 640, 637, 635, 632, 630, 627, 625, 623, 621, 619, 617, 615, 613, 611, 609, 608, 606, 604, 603, 601, 599, 598, 596, 595, 593, 592, 591, 589, 588, 586, 585, 584, 582, 581, 580, 579, 577, 576, 575, 574, 573, 572, 570, 569, 568, 567, 566, 565, 564, 563, 562, 561, 560, 559, 558, 557, 556, 555, 554, 553, 552, 551, 550, 549, 548, 547, 547, 546, 545, 544, 543, 542, 541, 540, 540, 539, 538, 537, 536, 536, 535, 534, 533, 532, 532, 531, 530, 529, 528, 528, 527, 526, 525, 525, 524, 523, 522, 522, 521, 520, 520, 519}

// Bls12381G2MultiExpDiscountTable is the gas discount table for BLS12-381 G2 multi exponentiation operation
var Bls12381G2MultiExpDiscountTable = [128]uint64{1000, 1000, 923, 884, 855, 832, 812, 796, 782, 770, 759, 749, 740, 732, 724, 717, 711, 704, 699, 693, 688, 683, 679, 674, 670, 666, 663, 659, 655, 652, 649, 646, 643, 640, 637, 634, 632, 629, 627, 624, 622, 620, 618, 615, 613, 611, 609, 607, 606, 604, 602, 600, 598, 597, 595, 593, 592, 590, 589, 587, 586, 584, 583, 582, 580, 579, 578, 576, 575, 574, 573, 571, 570, 569, 568, 567, 566, 565, 563, 562, 561, 560, 559, 558, 557, 556, 555, 554, 553, 552, 552, 551, 550, 549, 548, 547, 546, 545, 545, 544, 543, 542, 541, 541, 540, 539, 538, 537, 537, 536, 535, 535, 534, 533, 532, 532, 531, 530, 530, 529, 528, 528, 527, 526, 526, 525, 524, 524}

// Difficulty parameters.
var (
	DifficultyBoundDivisor = big.NewInt(2048)   // The bound divisor of the difficulty, used in the update calculations.
	GenesisDifficulty      = big.NewInt(131072) // Difficulty of the Genesis block.
	MinimumDifficulty      = big.NewInt(131072) // The minimum that the difficulty may ever be.
	DurationLimit          = big.NewInt(13)     // The decision boundary on the blocktime duration used to determine whether difficulty should go up or not.
)

// System contracts.
var (
	// SystemAddress is where the system-transaction is sent from as per EIP-4788
	SystemAddress = common.HexToAddress("0xfffffffffffffffffffffffffffffffffffffffe")

	// EIP-4788 - Beacon block root in the EVM
	BeaconRootsAddress = common.HexToAddress("0x000F3df6D732807Ef1319fB7B8bB8522d0Beac02")
	BeaconRootsCode    = common.FromHex("3373fffffffffffffffffffffffffffffffffffffffe14604d57602036146024575f5ffd5b5f35801560495762001fff810690815414603c575f5ffd5b62001fff01545f5260205ff35b5f5ffd5b62001fff42064281555f359062001fff015500")

	// EIP-2935 - Serve historical block hashes from state
	HistoryStorageAddress = common.HexToAddress("0x0000F90827F1C53a10cb7A02335B175320002935")
	HistoryStorageCode    = common.FromHex("3373fffffffffffffffffffffffffffffffffffffffe14604657602036036042575f35600143038111604257611fff81430311604257611fff9006545f5260205ff35b5f5ffd5b5f35611fff60014303065500")

	// EIP-7002 - Execution layer triggerable withdrawals
	WithdrawalQueueAddress = common.HexToAddress("0x00000961Ef480Eb55e80D19ad83579A64c007002")
	WithdrawalQueueCode    = common.FromHex("3373fffffffffffffffffffffffffffffffffffffffe1460cb5760115f54807fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff146101f457600182026001905f5b5f82111560685781019083028483029004916001019190604d565b909390049250505036603814608857366101f457346101f4575f5260205ff35b34106101f457600154600101600155600354806003026004013381556001015f35815560010160203590553360601b5f5260385f601437604c5fa0600101600355005b6003546002548082038060101160df575060105b5f5b8181146101835782810160030260040181604c02815460601b8152601401816001015481526020019060020154807fffffffffffffffffffffffffffffffff00000000000000000000000000000000168252906010019060401c908160381c81600701538160301c81600601538160281c81600501538160201c81600401538160181c81600301538160101c81600201538160081c81600101535360010160e1565b910180921461019557906002556101a0565b90505f6002555f6003555b5f54807fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff14156101cd57505f5b6001546002828201116101e25750505f6101e8565b01600290035b5f555f600155604c025ff35b5f5ffd")

	// EIP-7251 - Increase the MAX_EFFECTIVE_BALANCE
	ConsolidationQueueAddress = common.HexToAddress("0x0000BBdDc7CE488642fb579F8B00f3a590007251")
	ConsolidationQueueCode    = common.FromHex("3373fffffffffffffffffffffffffffffffffffffffe1460d35760115f54807fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff1461019a57600182026001905f5b5f82111560685781019083028483029004916001019190604d565b9093900492505050366060146088573661019a573461019a575f5260205ff35b341061019a57600154600101600155600354806004026004013381556001015f358155600101602035815560010160403590553360601b5f5260605f60143760745fa0600101600355005b6003546002548082038060021160e7575060025b5f5b8181146101295782810160040260040181607402815460601b815260140181600101548152602001816002015481526020019060030154905260010160e9565b910180921461013b5790600255610146565b90505f6002555f6003555b5f54807fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff141561017357505f5b6001546001828201116101885750505f61018e565b01600190035b5f555f6001556074025ff35b5f5ffd")

	// Blob tickets (random address for test)
	BlobTicketAllocationAddress = common.HexToAddress("0x8fd501b55bf41f51460815f0a1f00b541fc161")
	BlobTicketAllocationCode    = common.FromHex("608060405234801561000f575f5ffd5b506004361061008a575f3560e01c8063afbd178211610059578063afbd1782146109dd578063bda4fec5146109fb578063e3d670d714610a2b578063f56f48f214610a5b5761008b565b80630309cc7b14610930578063442871131461094c5780635f5152261461097d5780639977c78a146109ad5761008b565b5b5f36606073fffffffffffffffffffffffffffffffffffffffe73ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff1614610108576040517fdd169cfb00000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b6060805f85859050111561012e5784848101906101259190611092565b80925081935050505b5f5f90505b600380549050811015610702575f6003828154811061015557610154611108565b5b905f5260205f20015f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1690505f5f5f8373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f2090505f60015f8473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f205490505f5f90505f5f90505b87518161ffff16101561029f578473ffffffffffffffffffffffffffffffffffffffff16888261ffff168151811061024357610242611108565b5b602002602001015173ffffffffffffffffffffffffffffffffffffffff160361028c57868161ffff168151811061027d5761027c611108565b5b6020026020010151915061029f565b808061029790611162565b915050610208565b505b8280549050821015610560575f8383815481106102c1576102c0611108565b5b905f5260205f20906002020190505f815f015f9054906101000a900461ffff1661ffff16036102fe5782806102f590611194565b935050506102a1565b43600261ffff16826001015461031491906111db565b101561042357805f015f9054906101000a900461ffff1660025f8773ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f205f8282829054906101000a900461ffff16610383919061120e565b92506101000a81548161ffff021916908361ffff1602179055505f8261ffff1611156103f3575f815f015f9054906101000a900461ffff1661ffff168361ffff16116103cf57826103e1565b815f015f9054906101000a900461ffff165b905080836103ef919061120e565b9250505b5f815f015f6101000a81548161ffff021916908361ffff160217905550828061041b90611194565b93505061055a565b5f8261ffff161115610553575f815f015f9054906101000a900461ffff1661ffff168361ffff16116104555782610467565b815f015f9054906101000a900461ffff165b905080825f015f8282829054906101000a900461ffff16610488919061120e565b92506101000a81548161ffff021916908361ffff1602179055508060025f8873ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f205f8282829054906101000a900461ffff166104fb919061120e565b92506101000a81548161ffff021916908361ffff1602179055508083610521919061120e565b92505f825f015f9054906101000a900461ffff1661ffff160361054d57838061054990611194565b9450505b50610559565b50610560565b5b506102a1565b8160015f8673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f20819055505f60025f8673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f205f9054906101000a900461ffff1661ffff16036106ea576003600160038054905061060b9190611243565b8154811061061c5761061b611108565b5b905f5260205f20015f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff166003868154811061065857610657611108565b5b905f5260205f20015f6101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff16021790555060038054806106af576106ae611276565b5b600190038181905f5260205f20015f6101000a81549073ffffffffffffffffffffffffffffffffffffffff02191690559055505050506106fd565b84806106f590611194565b955050505050505b610133565b505f60038054905067ffffffffffffffff81111561072357610722610e05565b5b6040519080825280602002602001820160405280156107515781602001602082028036833780820191505090505b5090505f60038054905067ffffffffffffffff81111561077457610773610e05565b5b6040519080825280602002602001820160405280156107a25781602001602082028036833780820191505090505b5090505f5f90505b6003805490508110156108fc57600381815481106107cb576107ca611108565b5b905f5260205f20015f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1683828151811061080657610805611108565b5b602002602001019073ffffffffffffffffffffffffffffffffffffffff16908173ffffffffffffffffffffffffffffffffffffffff168152505060025f6003838154811061085757610856611108565b5b905f5260205f20015f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f205f9054906101000a900461ffff168282815181106108d9576108d8611108565b5b602002602001019061ffff16908161ffff168152505080806001019150506107aa565b508181604051602001610910929190611411565b604051602081830303815290604052945050505050915050805190602001f35b61094a60048036038101906109459190611446565b610a79565b005b610966600480360381019061096191906114ae565b610ccb565b60405161097492919061150a565b60405180910390f35b61099760048036038101906109929190611531565b610d11565b6040516109a4919061155c565b60405180910390f35b6109c760048036038101906109c29190611575565b610d68565b6040516109d491906115af565b60405180910390f35b6109e5610da3565b6040516109f291906115c8565b60405180910390f35b610a156004803603810190610a109190611531565b610da8565b604051610a22919061155c565b60405180910390f35b610a456004803603810190610a409190611531565b610dbd565b604051610a5291906115c8565b60405180910390f35b610a63610ddb565b604051610a7091906115c8565b60405180910390f35b5f4390506005548114610a9a57601561ffff16600481905550806005819055505b6004548261ffff161115610ada576040517fa8d8f34900000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b5f60025f8573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f205f9054906101000a900461ffff1661ffff1603610b9057600383908060018154018082558091505060019003905f5260205f20015f9091909190916101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055505b5f5f8473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f2060405180604001604052808461ffff16815260200183815250908060018154018082558091505060019003905f5260205f2090600202015f909190919091505f820151815f015f6101000a81548161ffff021916908361ffff1602179055506020820151816001015550508160025f8573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f205f8282829054906101000a900461ffff16610c9091906115e1565b92506101000a81548161ffff021916908361ffff1602179055508161ffff1660045f828254610cbf9190611243565b92505081905550505050565b5f602052815f5260405f208181548110610ce3575f80fd5b905f5260205f2090600202015f9150915050805f015f9054906101000a900461ffff16908060010154905082565b5f60025f8373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f205f9054906101000a900461ffff1661ffff169050919050565b60038181548110610d77575f80fd5b905f5260205f20015f915054906101000a900473ffffffffffffffffffffffffffffffffffffffff1681565b601581565b6001602052805f5260405f205f915090505481565b6002602052805f5260405f205f915054906101000a900461ffff1681565b600281565b5f604051905090565b5f5ffd5b5f5ffd5b5f5ffd5b5f601f19601f8301169050919050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52604160045260245ffd5b610e3b82610df5565b810181811067ffffffffffffffff82111715610e5a57610e59610e05565b5b80604052505050565b5f610e6c610de0565b9050610e788282610e32565b919050565b5f67ffffffffffffffff821115610e9757610e96610e05565b5b602082029050602081019050919050565b5f5ffd5b5f73ffffffffffffffffffffffffffffffffffffffff82169050919050565b5f610ed582610eac565b9050919050565b610ee581610ecb565b8114610eef575f5ffd5b50565b5f81359050610f0081610edc565b92915050565b5f610f18610f1384610e7d565b610e63565b90508083825260208201905060208402830185811115610f3b57610f3a610ea8565b5b835b81811015610f645780610f508882610ef2565b845260208401935050602081019050610f3d565b5050509392505050565b5f82601f830112610f8257610f81610df1565b5b8135610f92848260208601610f06565b91505092915050565b5f67ffffffffffffffff821115610fb557610fb4610e05565b5b602082029050602081019050919050565b5f61ffff82169050919050565b610fdc81610fc6565b8114610fe6575f5ffd5b50565b5f81359050610ff781610fd3565b92915050565b5f61100f61100a84610f9b565b610e63565b9050808382526020820190506020840283018581111561103257611031610ea8565b5b835b8181101561105b57806110478882610fe9565b845260208401935050602081019050611034565b5050509392505050565b5f82601f83011261107957611078610df1565b5b8135611089848260208601610ffd565b91505092915050565b5f5f604083850312156110a8576110a7610de9565b5b5f83013567ffffffffffffffff8111156110c5576110c4610ded565b5b6110d185828601610f6e565b925050602083013567ffffffffffffffff8111156110f2576110f1610ded565b5b6110fe85828601611065565b9150509250929050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52603260045260245ffd5b7f4e487b71000000000000000000000000000000000000000000000000000000005f52601160045260245ffd5b5f61116c82610fc6565b915061ffff82036111805761117f611135565b5b600182019050919050565b5f819050919050565b5f61119e8261118b565b91507fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff82036111d0576111cf611135565b5b600182019050919050565b5f6111e58261118b565b91506111f08361118b565b925082820190508082111561120857611207611135565b5b92915050565b5f61121882610fc6565b915061122383610fc6565b9250828203905061ffff81111561123d5761123c611135565b5b92915050565b5f61124d8261118b565b91506112588361118b565b92508282039050818111156112705761126f611135565b5b92915050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52603160045260245ffd5b5f81519050919050565b5f82825260208201905092915050565b5f819050602082019050919050565b6112d581610ecb565b82525050565b5f6112e683836112cc565b60208301905092915050565b5f602082019050919050565b5f611308826112a3565b61131281856112ad565b935061131d836112bd565b805f5b8381101561134d57815161133488826112db565b975061133f836112f2565b925050600181019050611320565b5085935050505092915050565b5f81519050919050565b5f82825260208201905092915050565b5f819050602082019050919050565b61138c81610fc6565b82525050565b5f61139d8383611383565b60208301905092915050565b5f602082019050919050565b5f6113bf8261135a565b6113c98185611364565b93506113d483611374565b805f5b838110156114045781516113eb8882611392565b97506113f6836113a9565b9250506001810190506113d7565b5085935050505092915050565b5f6040820190508181035f83015261142981856112fe565b9050818103602083015261143d81846113b5565b90509392505050565b5f5f6040838503121561145c5761145b610de9565b5b5f61146985828601610ef2565b925050602061147a85828601610fe9565b9150509250929050565b61148d8161118b565b8114611497575f5ffd5b50565b5f813590506114a881611484565b92915050565b5f5f604083850312156114c4576114c3610de9565b5b5f6114d185828601610ef2565b92505060206114e28582860161149a565b9150509250929050565b6114f581610fc6565b82525050565b6115048161118b565b82525050565b5f60408201905061151d5f8301856114ec565b61152a60208301846114fb565b9392505050565b5f6020828403121561154657611545610de9565b5b5f61155384828501610ef2565b91505092915050565b5f60208201905061156f5f8301846114fb565b92915050565b5f6020828403121561158a57611589610de9565b5b5f6115978482850161149a565b91505092915050565b6115a981610ecb565b82525050565b5f6020820190506115c25f8301846115a0565b92915050565b5f6020820190506115db5f8301846114ec565b92915050565b5f6115eb82610fc6565b91506115f683610fc6565b9250828201905061ffff8111156116105761160f611135565b5b9291505056fea2646970667358221220025e215254a4422cde6449804ceea203d39ea9932929ad229bc6b740ae44d67964736f6c63430008210033")

	BlobTicketSenderSlot  = big.NewInt(3)
	BlobTicketBalanceSlot = big.NewInt(2)
)
