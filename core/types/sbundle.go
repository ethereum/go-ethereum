package types

import (
	"encoding/json"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// Simplified Share Bundle Type for PoC

type SBundle struct {
	BlockNumber     *big.Int      `json:"blockNumber,omitempty"` // if BlockNumber is set it must match DecryptionCondition!
	MaxBlock        *big.Int      `json:"maxBlock,omitempty"`
	Txs             Transactions  `json:"txs"`
	RevertingHashes []common.Hash `json:"revertingHashes,omitempty"`
	RefundPercent   *int          `json:"percent,omitempty"`
}

type RpcSBundle struct {
	BlockNumber     *hexutil.Big    `json:"blockNumber,omitempty"`
	MaxBlock        *hexutil.Big    `json:"maxBlock,omitempty"`
	Txs             []hexutil.Bytes `json:"txs"`
	RevertingHashes []common.Hash   `json:"revertingHashes,omitempty"`
	RefundPercent   *int            `json:"percent,omitempty"`
}

func (s *SBundle) MarshalJSON() ([]byte, error) {
	txs := []hexutil.Bytes{}
	for _, tx := range s.Txs {
		txBytes, err := tx.MarshalBinary()
		if err != nil {
			return nil, err
		}
		txs = append(txs, txBytes)
	}

	var blockNumber *hexutil.Big
	if s.BlockNumber != nil {
		blockNumber = new(hexutil.Big)
		*blockNumber = hexutil.Big(*s.BlockNumber)
	}

	return json.Marshal(&RpcSBundle{
		BlockNumber:     blockNumber,
		Txs:             txs,
		RevertingHashes: s.RevertingHashes,
		RefundPercent:   s.RefundPercent,
	})
}

func (s *SBundle) UnmarshalJSON(data []byte) error {
	var rpcSBundle RpcSBundle
	if err := json.Unmarshal(data, &rpcSBundle); err != nil {
		return err
	}

	var txs Transactions
	for _, txBytes := range rpcSBundle.Txs {
		var tx Transaction
		err := tx.UnmarshalBinary(txBytes)
		if err != nil {
			return err
		}

		txs = append(txs, &tx)
	}

	s.BlockNumber = (*big.Int)(rpcSBundle.BlockNumber)
	s.MaxBlock = (*big.Int)(rpcSBundle.MaxBlock)
	s.Txs = txs
	s.RevertingHashes = rpcSBundle.RevertingHashes
	s.RefundPercent = rpcSBundle.RefundPercent

	return nil
}

type RPCMevShareBundle struct {
	Version   string `json:"version"`
	Inclusion struct {
		Block    string `json:"block"`
		MaxBlock string `json:"maxBlock"`
	} `json:"inclusion"`
	Body []struct {
		Tx        string `json:"tx"`
		CanRevert bool   `json:"canRevert"`
	} `json:"body"`
	Validity struct {
		Refund []struct {
			BodyIdx int `json:"bodyIdx"`
			Percent int `json:"percent"`
		} `json:"refund"`
	} `json:"validity"`
}
