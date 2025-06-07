package types

import (
	"maps"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

type AcctPostValues struct {
	Nonce     uint64                      `json:"nonce"`
	Balance   *uint256.Int                `json:"balance"`
	Code      []byte                      `json:"code"`
	StorageKV map[common.Hash]common.Hash `json:"storageKV"`
	Destruct  bool                        `json:"destruct"`
	CodeHash  []byte                      `json:"codeHash"`
	Root      common.Hash                 `json:"root"`
	Cached    bool
}

// For acccount destruct or storage clearing corresponding values would be 0
type TxPostValues map[common.Address]*AcctPostValues

// Balance and StorageKV must be cloned to avoid changing by setting postVals
func (v *AcctPostValues) Clone() *AcctPostValues {
	balance := v.Balance
	if balance != nil {
		balance = v.Balance.Clone()
	}

	return &AcctPostValues{
		Nonce:     v.Nonce,
		Balance:   balance,
		Code:      v.Code,
		StorageKV: maps.Clone(v.StorageKV),
		Destruct:  false,
		CodeHash:  v.CodeHash,
		Root:      v.Root,
		Cached: false,
	}
}
