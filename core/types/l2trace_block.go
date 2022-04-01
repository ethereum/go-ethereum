package types

import (
	"math/big"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/common/hexutil"
	"github.com/scroll-tech/go-ethereum/params"
)

type BlockTrace struct {
	Number       string               `json:"number"` // big.Int string
	Hash         common.Hash          `json:"hash"`
	GasLimit     uint64               `json:"gasLimit"`
	Difficulty   string               `json:"difficulty"` // big.Int string
	BaseFee      string               `json:"baseFee"`    // big.Int string
	Coinbase     *AccountProofWrapper `json:"coinbase"`
	Time         uint64               `json:"time"`
	Transactions []*TransactionTrace  `json:"transactions"`
}

type TransactionTrace struct {
	Type     uint8           `json:"type"`
	Nonce    uint64          `json:"nonce"`
	Gas      uint64          `json:"gas"`
	GasPrice string          `json:"gasPrice"` // big.Int string
	From     common.Address  `json:"from"`
	To       *common.Address `json:"to"`
	ChainId  string          `json:"chainId"` // big.Int string
	Value    string          `json:"value"`   // big.Int string
	Data     string          `json:"data"`
	IsCreate bool            `json:"isCreate"`
	V        string          `json:"v"` // big.Int string
	R        string          `json:"r"` // big.Int string
	S        string          `json:"s"` // big.Int string
}

// NewTraceBlock supports necessary fields for roller.
func NewTraceBlock(config *params.ChainConfig, block *Block, coinbase *AccountProofWrapper) *BlockTrace {
	txs := make([]*TransactionTrace, block.Transactions().Len())
	for i, tx := range block.Transactions() {
		txs[i] = newTraceTransaction(tx, block.NumberU64(), config)
	}
	return &BlockTrace{
		Number:       block.Number().String(),
		Hash:         block.Hash(),
		GasLimit:     block.GasLimit(),
		Difficulty:   block.Difficulty().String(),
		BaseFee:      block.BaseFee().String(),
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
		Nonce:    tx.Nonce(),
		ChainId:  tx.ChainId().String(),
		From:     from,
		Gas:      tx.Gas(),
		GasPrice: tx.GasPrice().String(),
		To:       tx.To(),
		Value:    tx.Value().String(),
		Data:     hexutil.Encode(tx.Data()),
		IsCreate: tx.To() == nil,
		V:        v.String(),
		R:        r.String(),
		S:        s.String(),
	}
	return result
}
