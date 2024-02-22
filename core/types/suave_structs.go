package types

import "github.com/ethereum/go-ethereum/common"

// Structs

type BuildBlockArgs struct {
	Slot                  uint64
	ProposerPubkey        []byte
	Parent                common.Hash
	Timestamp             uint64
	FeeRecipient          common.Address
	GasLimit              uint64
	Random                common.Hash
	Withdrawals           []*Withdrawal
	ParentBeaconBlockRoot common.Hash
	Extra                 []byte
	BeaconRoot            common.Hash
	FillPending           bool
}
