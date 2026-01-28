// Copyright (c) 2018 XDPoSChain
// XDPoS delegated-proof-of-stake protocol constants.

package utils

import (
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

var (
	EpochLength = uint64(900) // Default number of blocks after which to checkpoint and reset the pending votes

	ExtraVanity = 32 // Fixed number of extra-data prefix bytes reserved for signer vanity
	ExtraSeal   = 65 // Fixed number of extra-data suffix bytes reserved for signer seal

	NonceAuthVote = hexutil.MustDecode("0xffffffffffffffff") // Magic nonce number to vote on adding a new signer
	NonceDropVote = hexutil.MustDecode("0x0000000000000000") // Magic nonce number to vote on removing a signer.

	UncleHash      = types.CalcUncleHash(nil) // Always Keccak256(RLP([])) as uncles are meaningless outside of PoW.
	InmemoryEpochs = 5 * EpochLength          // Number of mapping from block to epoch switch infos to keep in memory

	InmemoryRound2Epochs = 65536 // Number of mapping of epoch switch blocks for quickly locating epoch switch block.
)

const (
	InmemorySnapshots      = 128 // Number of recent vote snapshots to keep in memory
	BlockSignersCacheLimit = 9000
	M2ByteLength           = 4
)

const (
	PeriodicJobPeriod = 60
	PoolHygieneRound  = 10
)
