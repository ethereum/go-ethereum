package types

import "github.com/maticnetwork/bor/common"

// StateData represents state received from Ethereum Blockchain
type StateData struct {
	Did      uint64
	Contract common.Address
	Data     string
	TxHash   common.Hash
}
