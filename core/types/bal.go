package types

//go:generate go run github.com/ferranbt/fastssz/sszgen --path . --objs PerTxAccess,SlotAccess,AccountAccess,BlockAccessList,BalanceDelta,BalanceChange,AccountBalanceDiff,CodeChange,AccountCodeDiff,AccountNonce,NonceDiffs --output bal_encoding.go

type PerTxAccess struct {
	TxIdx      uint64 `ssz-size:"2"`
	ValueAfter [32]byte
}

type SlotAccess struct {
	Slot     [32]byte      `ssz-size:"32"`
	Accesses []PerTxAccess `ssz-max:"30000"`
}

type AccountAccess struct {
	Address  [20]byte     `ssz-size:"32"`
	Accesses []SlotAccess `ssz-max:"300000"`
	code     []byte       `ssz-max:"24576"` // this is currently a union in the EIP spec, but unions aren't used anywhere in practice so I implement it as a list here.
}

type BalanceDelta [12]byte // {}-endian signed integer

type BalanceChange struct {
	TxIdx uint64 `ssz-size:"2"`
	Delta BalanceDelta
}

type AccountBalanceDiff struct {
	Address [40]byte
	Changes []BalanceChange `ssz-max:"30000"`
}

// TODO: implement encoder/decoder manually on this, as we can't specify tags for a type declaration
type BalanceDiffs = []AccountBalanceDiff

type CodeChange struct {
	TxIdx   uint64 `ssz-size:"2"`
	NewCode []byte `ssz-max:"24576"`
}

type AccountCodeDiff struct {
	Address [40]byte
	Changes []CodeChange `ssz-max:"30000"`
}

// TODO: implement encoder/decoder manually on this, as we can't specify tags for a type declaration
type CodeDiffs []AccountCodeDiff

type AccountNonce struct {
	Address    [40]byte
	NonceAfter uint64
}

// TODO: implement encoder/decoder manually on this, as we can't specify tags for a type declaration
type NonceDiffs []AccountNonce
