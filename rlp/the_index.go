// Copyright 2021 orbs-network
// No license

package rlp

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type TheIndex_rlpBlock struct {
	BlockNumber *big.Int
	Time        uint64
	Hash        common.Hash
	Coinbase    common.Address
	Difficulty  *big.Int
	GasLimit    uint64
}

type TheIndex_rlpLog struct {
	Topics []common.Hash
	Data   []byte
}

type TheIndex_rlpState struct {
	Key   common.Hash
	Value common.Hash
}

type TheIndex_rlpContract struct {
	Address common.Address
	Logs    []TheIndex_rlpLog
	Code    []byte
	States  []TheIndex_rlpState
	Balance *big.Int
}

type TheIndex_rlpContractsForBlock struct {
	BlockNumber *big.Int
	Contracts   []TheIndex_rlpContract
}

type TheIndex_rlpAccount struct {
	Address  common.Address
	Balance  *big.Int
	CodeHash []byte
}

type TheIndex_rlpAccountsForBlock struct {
	BlockNumber *big.Int
	Accounts    []TheIndex_rlpAccount
}
