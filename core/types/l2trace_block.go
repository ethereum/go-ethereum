package types

import (
	"math/big"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/common/hexutil"
	"github.com/scroll-tech/go-ethereum/params"
)

type BlockTrace struct {
	Number      *big.Int            `json:"number"`
	Hash        common.Hash         `json:"hash"`
	GasLimit    uint64              `json:"gasLimit"`
	Difficulty  *big.Int            `json:"difficulty"`
	BaseFee     *big.Int            `json:"baseFee"`
	Coinbase    common.Address      `json:"coinbase"`
	Time        uint64              `json:"time"`
	Transaction []*TransactionTrace `json:"transaction"`
}

type TransactionTrace struct {
	Type     uint8           `json:"type"`
	Nonce    uint64          `json:"nonce"`
	Gas      uint64          `json:"gas"`
	GasPrice *big.Int        `json:"gasPrice"`
	From     common.Address  `json:"from"`
	To       *common.Address `json:"to"`
	ChainId  *big.Int        `json:"chainId"`
	Value    *big.Int        `json:"value"`
	Data     string          `json:"data"`
	IsCreate bool            `json:"isCreate"`
	V        *big.Int        `json:"v"`
	R        *big.Int        `json:"r"`
	S        *big.Int        `json:"s"`
}

// NewTraceBlock supports necessary fields for roller.
func NewTraceBlock(config *params.ChainConfig, block *Block) *BlockTrace {
	txs := make([]*TransactionTrace, block.Transactions().Len())
	for i, tx := range block.Transactions() {
		txs[i] = newTraceTransaction(tx, block.NumberU64(), config)
	}
	return &BlockTrace{
		Number:      block.Number(),
		Hash:        block.Hash(),
		GasLimit:    block.GasLimit(),
		Difficulty:  block.Difficulty(),
		BaseFee:     block.BaseFee(),
		Coinbase:    block.Coinbase(),
		Time:        block.Time(),
		Transaction: txs,
	}
}

// newTraceTransaction returns a transaction that will serialize to the trace
// representation, with the given location metadata set (if available).
func newTraceTransaction(tx *Transaction, blockNumber uint64, config *params.ChainConfig) *TransactionTrace {
	signer := MakeSigner(config, big.NewInt(0).SetUint64(blockNumber))
	from, _ := Sender(signer, tx)
	v, r, s := tx.RawSignatureValues()
	result := &TransactionTrace{
		Type:     tx.Type(),
		Nonce:    tx.Nonce(),
		ChainId:  tx.ChainId(),
		From:     from,
		Gas:      tx.Gas(),
		GasPrice: tx.GasPrice(),
		To:       tx.To(),
		Value:    tx.Value(),
		Data:     hexutil.Encode(tx.Data()),
		IsCreate: tx.To() == nil,
		V:        v,
		R:        r,
		S:        s,
	}
	return result
}
