package types

import (
	"math/big"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/common/hexutil"
	"github.com/scroll-tech/go-ethereum/params"
)

type BlockTrace struct {
	Number       *hexutil.Big        `json:"number"`
	Hash         common.Hash         `json:"hash"`
	GasLimit     uint64              `json:"gasLimit"`
	Difficulty   *hexutil.Big        `json:"difficulty"`
	BaseFee      *hexutil.Big        `json:"baseFee"`
	Coinbase     *AccountWrapper     `json:"coinbase"`
	Time         uint64              `json:"time"`
	Transactions []*TransactionTrace `json:"transactions"`
}

type TransactionTrace struct {
	Type     uint8           `json:"type"`
	Nonce    uint64          `json:"nonce"`
	TxHash   string          `json:"txHash"`
	Gas      uint64          `json:"gas"`
	GasPrice *hexutil.Big    `json:"gasPrice"`
	From     common.Address  `json:"from"`
	To       *common.Address `json:"to"`
	ChainId  *hexutil.Big    `json:"chainId"`
	Value    *hexutil.Big    `json:"value"`
	Data     string          `json:"data"`
	IsCreate bool            `json:"isCreate"`
	V        *hexutil.Big    `json:"v"`
	R        *hexutil.Big    `json:"r"`
	S        *hexutil.Big    `json:"s"`
}

// NewTraceBlock supports necessary fields for roller.
func NewTraceBlock(config *params.ChainConfig, block *Block, coinbase *AccountWrapper) *BlockTrace {
	txs := make([]*TransactionTrace, block.Transactions().Len())
	for i, tx := range block.Transactions() {
		txs[i] = newTraceTransaction(tx, block.NumberU64(), config)
	}

	return &BlockTrace{
		Number:       (*hexutil.Big)(block.Number()),
		Hash:         block.Hash(),
		GasLimit:     block.GasLimit(),
		Difficulty:   (*hexutil.Big)(block.Difficulty()),
		BaseFee:      (*hexutil.Big)(block.BaseFee()),
		Coinbase:     coinbase,
		Time:         block.Time(),
		Transactions: txs,
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
		TxHash:   tx.Hash().String(),
		Nonce:    tx.Nonce(),
		ChainId:  (*hexutil.Big)(tx.ChainId()),
		From:     from,
		Gas:      tx.Gas(),
		GasPrice: (*hexutil.Big)(tx.GasPrice()),
		To:       tx.To(),
		Value:    (*hexutil.Big)(tx.Value()),
		Data:     hexutil.Encode(tx.Data()),
		IsCreate: tx.To() == nil,
		V:        (*hexutil.Big)(v),
		R:        (*hexutil.Big)(r),
		S:        (*hexutil.Big)(s),
	}
	return result
}
