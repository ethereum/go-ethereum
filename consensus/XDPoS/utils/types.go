// Copyright (c) 2018 XDPoSChain
// XDPoS types and interfaces

package utils

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/lru"
	"github.com/ethereum/go-ethereum/core/types"
)

type Masternode struct {
	Address common.Address
	Stake   *big.Int
}

type PublicApiSnapshot struct {
	Number  uint64                      `json:"number"`  // Block number where the snapshot was created
	Hash    common.Hash                 `json:"hash"`    // Block hash where the snapshot was created
	Signers map[common.Address]struct{} `json:"signers"` // Set of authorized signers at this moment
	Recents map[uint64]common.Address   `json:"recents"` // Set of recent signers for spam protections
}

type MissedRoundInfo struct {
	Round            types.Round
	Miner            common.Address
	CurrentBlockHash common.Hash
	CurrentBlockNum  *big.Int
	ParentBlockHash  common.Hash
	ParentBlockNum   *big.Int
}

type PublicApiMissedRoundsMetadata struct {
	EpochRound       types.Round
	EpochBlockNumber *big.Int
	MissedRounds     []MissedRoundInfo
}

// Given an epoch number, this struct records the epoch switch block (first block in epoch) infos such as block number
type EpochNumInfo struct {
	EpochBlockHash        common.Hash `json:"hash"`
	EpochRound            types.Round `json:"round"`
	EpochFirstBlockNumber *big.Int    `json:"firstBlock"`
	EpochLastBlockNumber  *big.Int    `json:"lastBlock"`
}

type SigLRU = lru.Cache[common.Hash, common.Address]
