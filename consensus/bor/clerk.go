package bor

import (
	"github.com/maticnetwork/bor/common"
	"github.com/maticnetwork/bor/common/hexutil"
)

// EventRecord represents state record
type EventRecord struct {
	ID       uint64         `json:"id" yaml:"id"`
	Contract common.Address `json:"contract" yaml:"contract"`
	Data     hexutil.Bytes  `json:"data" yaml:"data"`
	TxHash   common.Hash    `json:"tx_hash" yaml:"tx_hash"`
	LogIndex uint64         `json:"log_index" yaml:"log_index"`
	ChainID  string         `json:"bor_chain_id" yaml:"bor_chain_id"`
}
