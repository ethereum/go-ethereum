package types

import "github.com/ethereum/go-ethereum/common"

type DataId [16]byte

// Structs

type BuildBlockArgs struct {
	Slot           uint64
	ProposerPubkey []byte
	Parent         common.Hash
	Timestamp      uint64
	FeeRecipient   common.Address
	GasLimit       uint64
	Random         common.Hash
	Withdrawals    []*Withdrawal
	ParentBeaconBlockRoot common.Hash
	Extra          []byte
	BeaconRoot     common.Hash
	FillPending    bool
}

type DataRecord struct {
	Id                  DataId
	Salt                DataId
	DecryptionCondition uint64
	AllowedPeekers      []common.Address
	AllowedStores       []common.Address
	Version             string
}

type HttpRequest struct {
	Url                    string
	Method                 string
	Headers                []string
	Body                   []byte
	WithFlashbotsSignature bool
}

type SimulateTransactionResult struct {
	Egp     uint64
	Logs    []*SimulatedLog
	Success bool
	Error   string
}

type SimulatedLog struct {
	Data   []byte
	Addr   common.Address
	Topics []common.Hash
}
