package bor

import (
	"github.com/maticnetwork/bor/common"
)

// EventRecord represents state record
type EventRecord struct {
	ID       uint64         `json:"id" yaml:"id"`
	Contract common.Address `json:"contract" yaml:"contract"`
	Data     []byte         `json:"data" yaml:"data"`
	TxHash   common.Hash    `json:"tx_hash" yaml:"tx_hash"`
	LogIndex uint64         `json:"log_index" yaml:"log_index"`
}
