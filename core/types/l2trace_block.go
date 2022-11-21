package types

import (
	"encoding/json"
	"math/big"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/params"
)

type TransactionData struct {
	IsCreate bool           `json:"isCreate"`
	From     common.Address `json:"from"`
	*Transaction
}

// TransactionDataAlias just used for UnmarshalJSON.
type TransactionDataAlias TransactionData

func (t *TransactionData) UnmarshalJSON(input []byte) error {
	var jsonConfig struct {
		TransactionDataAlias
		*Transaction
	}
	jsonConfig.Transaction = &Transaction{}
	if err := json.Unmarshal(input, &jsonConfig); err != nil {
		return err
	}
	*t = TransactionData(jsonConfig.TransactionDataAlias)
	t.Transaction = jsonConfig.Transaction
	return nil
}

// NewTraceTransaction returns a transaction that will serialize to the trace
// representation, with the given location metadata set (if available).
func NewTraceTransaction(tx *Transaction, blockNumber uint64, config *params.ChainConfig) *TransactionData {
	signer := MakeSigner(config, big.NewInt(0).SetUint64(blockNumber))
	from, _ := Sender(signer, tx)
	return &TransactionData{
		From:        from,
		IsCreate:    tx.To() == nil,
		Transaction: tx,
	}
}
