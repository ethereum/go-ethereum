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
	BlockNumber *big.Int
	Logs        []TheIndex_rlpLog
	Code        []byte
	States      []TheIndex_rlpState
}
