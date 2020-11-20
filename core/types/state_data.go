package types

import "github.com/ethereum/go-ethereum/common"

// StateData represents state received from Ethereum Blockchain
type StateData struct {
	Did      uint64
	Contract common.Address
	Data     string
	TxHash   common.Hash
}
