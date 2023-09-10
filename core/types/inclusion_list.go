package types

import "github.com/ethereum/go-ethereum/common"

// InclusionList represents pairs of transaction summary and the transaction data itself
type InclusionList struct {
	Summary      []*InclusionListEntry `json:"summary"`
	Transactions []*Transaction        `json:"transactions"`
}

// InclusionListEntry denotes a summary entry of (address, gasLimit)
type InclusionListEntry struct {
	Address  common.Address `json:"address"`
	GasLimit uint32         `json:"gasLimit"` // TODO(manav): change to uint8
}
