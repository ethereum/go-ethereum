package types

import (
	"bytes"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
	"maps"
	"sort"
)

//go:generate go run github.com/ferranbt/fastssz/sszgen --path . --objs encodingPerTxAccess,encodingSlotAccess,encodingAccountAccess,encodingBlockAccessList,encodingBalanceDelta,encodingBalanceChange,encodingAccountBalanceDiff,encodingCodeChange,encodingAccountCodeDiff,encodingAccountNonce,encodingNonceDiffs,encodingBlockAccessList --output bal_encoding_generated.go

// encoder types

type encodingPerTxAccess struct {
	TxIdx      uint64 `ssz-size:"2"`
	ValueAfter [32]byte
}

type encodingSlotAccess struct {
	Slot     [32]byte              `ssz-size:"32"`
	Accesses []encodingPerTxAccess `ssz-max:"30000"`
}

type encodingAccountAccess struct {
	Address  [20]byte             `ssz-size:"32"`
	Accesses []encodingSlotAccess `ssz-max:"300000"`
	Code     []byte               `ssz-max:"24576"`
}

type encodingAccountAccessList []encodingAccountAccess

type encodingBalanceDelta [12]byte // {}-endian signed integer

type encodingBalanceChange struct {
	TxIdx uint64 `ssz-size:"2"`
	Delta encodingBalanceDelta
}

type encodingAccountBalanceDiff struct {
	Address [20]byte
	Changes []encodingBalanceChange `ssz-max:"30000"`
}

// TODO: implement encoder/decoder manually on this, as we can't specify tags for a type declaration
type encodingBalanceDiffs = []encodingAccountBalanceDiff

type encodingCodeChange struct {
	TxIdx   uint64 `ssz-size:"2"`
	NewCode []byte `ssz-max:"24576"`
}

type encodingAccountCodeDiff struct {
	Address [20]byte
	Changes []encodingCodeChange `ssz-max:"30000"`
}

// TODO: implement encoder/decoder manually on this, as we can't specify tags for a type declaration
type encodingCodeDiffs []encodingAccountCodeDiff

type encodingAccountNonce struct {
	Address     [20]byte
	NonceBefore uint64
}

// TODO: implement encoder/decoder manually on this, as we can't specify tags for a type declaration
type encodingNonceDiffs []encodingAccountNonce

type encodingBlockAccessList struct {
	AccountAccesses encodingAccountAccessList `ssz-max:"100"`
	BalanceDiffs    encodingBalanceDiffs      `ssz-max:"100"`
	CodeDiffs       encodingCodeDiffs         `ssz-max:"100"`
	NonceDiffs      encodingNonceDiffs        `ssz-max:"100"`
}

// non-encoder objects

func nonceDiffsToEncoderObj(nonceDiffs map[common.Address]uint64) (res encodingNonceDiffs) {
	var addrs []common.Address
	for addr, _ := range nonceDiffs {
		addrs = append(addrs, addr)
	}
	sort.Slice(addrs, func(i, j int) bool {
		return bytes.Compare(addrs[i][:], addrs[j][:]) > 0
	})

	for _, addr := range addrs {
		res = append(res, encodingAccountNonce{
			Address:     addr,
			NonceBefore: nonceDiffs[addr],
		})
	}
	return
}

type slotAccess struct {
	writes map[uint64]common.Hash // map of tx index to post-tx slot value
}

func (s slotAccess) toEncoderObj(key common.Hash) (res encodingSlotAccess) {
	var (
		slotIdxs []uint64
	)
	res.Slot = key
	for sIdx, _ := range s.writes {
		slotIdxs = append(slotIdxs, sIdx)
	}
	sort.Slice(slotIdxs, func(i, j int) bool {
		return slotIdxs[i] < slotIdxs[j]
	})
	for _, slotIdx := range slotIdxs {
		res.Accesses = append(res.Accesses, encodingPerTxAccess{
			slotIdx,
			s.writes[slotIdx],
		})
	}
	return
}

type accountAccess struct {
	address  common.Address
	accesses map[common.Hash]slotAccess // map of slot key to all post-tx values where that slot was read/written
	code     []byte
}

func (a *accountAccess) MarkRead(key common.Hash) {
	if _, ok := a.accesses[key]; !ok {
		a.accesses[key] = slotAccess{
			make(map[uint64]common.Hash),
		}
	}
}

func (a *accountAccess) MarkWrite(txIdx uint64, key, value common.Hash) {
	if _, ok := a.accesses[key]; !ok {
		a.accesses[key] = slotAccess{
			make(map[uint64]common.Hash),
		}
	}

	a.accesses[key].writes[txIdx] = value
}

// map of transaction idx to the new code
type codeDiff map[uint64][]byte

func (c codeDiff) toEncoderObj(addr common.Address) (res encodingAccountCodeDiff) {
	res.Address = addr
	var diffIdxs []uint64
	for idx, _ := range c {
		diffIdxs = append(diffIdxs, idx)
	}
	sort.Slice(diffIdxs, func(i, j int) bool {
		return diffIdxs[i] < diffIdxs[j]
	})
	for _, idx := range diffIdxs {
		res.Changes = append(res.Changes, encodingCodeChange{
			TxIdx:   idx,
			NewCode: bytes.Clone(c[idx]),
		})
	}
	return
}

func (b *encodingBalanceDelta) Set(val *uint256.Int) *encodingBalanceDelta {
	valBytes := val.Bytes()
	if len(valBytes) > 12 {
		panic("can't encode value that is greater than 12 bytes in size")
	}
	copy(b[12-len(valBytes):], valBytes[:])
	return b
}

type balanceDiff map[uint64]*uint256.Int

func (b balanceDiff) toEncoderObj(addr common.Address) (res encodingAccountBalanceDiff) {
	res.Address = addr
	var diffIdxs []uint64
	for txIdx, _ := range b {
		diffIdxs = append(diffIdxs, txIdx)
	}
	sort.Slice(diffIdxs, func(i, j int) bool {
		return diffIdxs[i] < diffIdxs[j]
	})

	for _, idx := range diffIdxs {
		res.Changes = append(res.Changes, encodingBalanceChange{
			TxIdx: idx,
			Delta: *new(encodingBalanceDelta).Set(b[idx]),
		})
	}
	return res
}

type BlockAccessList struct {
	accountAccesses map[common.Address]*accountAccess
	balanceChanges  map[common.Address]balanceDiff
	codeChanges     map[common.Address]codeDiff
	prestateNonces  map[common.Address]uint64
	hash            common.Hash
}

func codeDiffsToEncoderObj(codeChanges map[common.Address]codeDiff) (res encodingCodeDiffs) {
	var codeChangeAddrs []common.Address

	for addr, _ := range codeChanges {
		codeChangeAddrs = append(codeChangeAddrs, addr)
	}
	sort.Slice(codeChangeAddrs, func(i, j int) bool {
		return bytes.Compare(codeChangeAddrs[i][:], codeChangeAddrs[j][:]) < 0
	})

	for _, addr := range codeChangeAddrs {
		res = append(res, codeChanges[addr].toEncoderObj(addr))
	}
	return
}

func NewBlockAccessList() *BlockAccessList {
	return &BlockAccessList{
		make(map[common.Address]*accountAccess),
		make(map[common.Address]balanceDiff),
		make(map[common.Address]codeDiff),
		make(map[common.Address]uint64),
		common.Hash{},
	}
}

func (b *BlockAccessList) Eq(other *BlockAccessList) bool {

	// check that the account accesses are equal (consider moving this into its own function)

	if len(b.accountAccesses) != len(other.accountAccesses) {
		return false
	}
	for address, aa := range b.accountAccesses {
		otherAA, ok := other.accountAccesses[address]
		if !ok {
			return false
		}
		if len(aa.accesses) != len(otherAA.accesses) {
			return false
		}
		for key, vals := range aa.accesses {
			otherAccesses, ok := otherAA.accesses[key]
			if !ok {
				return false
			}
			if len(vals.writes) != len(otherAccesses.writes) {
				return false
			}

			for i, writeVal := range vals.writes {
				otherWriteVal, ok := otherAccesses.writes[i]
				if !ok {
					return false
				}
				if writeVal != otherWriteVal {
					return false
				}
			}
		}
	}

	// check that the code changes are equal

	if len(b.codeChanges) != len(other.codeChanges) {
		return false
	}
	for addr, codeCh := range b.codeChanges {
		otherCodeCh, ok := other.codeChanges[addr]
		if !ok {
			return false
		}
		equal := maps.EqualFunc(codeCh, otherCodeCh, func(b1, b2 []byte) bool {
			return bytes.Equal(b1, b2)
		})
		if !equal {
			return false
		}
	}

	if !maps.Equal(b.prestateNonces, other.prestateNonces) {
		return false
	}

	if len(b.balanceChanges) != len(other.balanceChanges) {
		return false
	}

	for addr, balanceChanges := range b.balanceChanges {
		otherBalanceChanges, ok := other.balanceChanges[addr]
		if !ok {
			return false
		}

		if len(balanceChanges) != len(otherBalanceChanges) {
			return false
		}

		for txIdx, balanceCh := range balanceChanges {
			otherBalanceCh, ok := otherBalanceChanges[txIdx]
			if !ok {
				return false
			}

			if balanceCh != otherBalanceCh {
				return false
			}
		}
	}
	return true
}

// TODO: this should be called once per account per block for every account that sent txs in that block.
// the value is the prestate nonce before the start of the first tx execution from that account in the block.
func (b *BlockAccessList) NonceDiff(address common.Address, originNonce uint64) {
	b.prestateNonces[address] = originNonce
}

// called during tx finalisation for each
func (b *BlockAccessList) BalanceChange(txIdx uint64, address common.Address, balance *uint256.Int) {
	if _, ok := b.balanceChanges[address]; !ok {
		b.balanceChanges[address] = make(balanceDiff)
	}
	b.balanceChanges[address][txIdx] = balance.Clone()
}

// TODO for eip:  specify that storage slots which are read/modified for accounts that are created/selfdestructed
// in same transaction aren't included in teh BAL (?)

// TODO for eip:  specify that storage slots of newly-created accounts which are only read are not included in the BAL (?)

// called during tx execution every time a storage slot is read
func (b *BlockAccessList) StorageRead(address common.Address, key common.Hash) {
	if _, ok := b.accountAccesses[address]; !ok {
		b.accountAccesses[address] = &accountAccess{
			address,
			make(map[common.Hash]slotAccess),
			nil,
		}
	}
	b.accountAccesses[address].MarkRead(key)
}

// called every time a mutated storage value is committed upon transaction finalization
func (b *BlockAccessList) StorageWrite(txIdx uint64, address common.Address, key, value common.Hash) {
	if _, ok := b.accountAccesses[address]; !ok {
		b.accountAccesses[address] = &accountAccess{
			address,
			make(map[common.Hash]slotAccess),
			nil,
		}
	}
	b.accountAccesses[address].MarkWrite(txIdx, key, value)
}

// called during tx finalisation for each dirty account with mutated code
func (b *BlockAccessList) CodeChange(txIdx uint64, address common.Address, code []byte) {
	if _, ok := b.codeChanges[address]; !ok {
		b.codeChanges[address] = codeDiff{}
	}
	b.codeChanges[address][txIdx] = bytes.Clone(code)
}

func (b *BlockAccessList) encodeSSZ() []byte {
	var (
		accountAccessesAddrs   []common.Address
		encoderAccountAccesses encodingAccountAccessList

		balanceDiffsAddrs   []common.Address
		encoderBalanceDiffs encodingBalanceDiffs
	)

	for addr, _ := range b.accountAccesses {
		accountAccessesAddrs = append(accountAccessesAddrs, addr)
	}
	sort.Slice(accountAccessesAddrs, func(i, j int) bool {
		return bytes.Compare(accountAccessesAddrs[i][:], accountAccessesAddrs[j][:]) < 0
	})
	for _, addr := range accountAccessesAddrs {
		encoderAccountAccesses = append(encoderAccountAccesses, encodingAccountAccess{
			Address:  addr,
			Accesses: nil,
			Code:     b.accountAccesses[addr].code,
		})
		// sort the accesses lexicographically by key, and the occurance of each key ascending by tx idx
		// then encode them
		var storageAccessKeys []common.Hash
		for key, _ := range b.accountAccesses[addr].accesses {
			storageAccessKeys = append(storageAccessKeys, key)
		}
		sort.Slice(storageAccessKeys, func(i, j int) bool {
			return bytes.Compare(storageAccessKeys[i][:], storageAccessKeys[j][:]) < 0
		})
		var accesses []encodingSlotAccess
		for _, accessSlot := range storageAccessKeys {
			accesses = append(accesses, b.accountAccesses[addr].accesses[accessSlot].toEncoderObj(accessSlot))
		}
		encoderAccountAccesses = append(encoderAccountAccesses, encodingAccountAccess{
			Address:  addr,
			Accesses: accesses,
			Code:     b.accountAccesses[addr].code,
		})
	}

	// encode balance diffs
	for addr, _ := range b.balanceChanges {
		balanceDiffsAddrs = append(balanceDiffsAddrs, addr)
	}
	sort.Slice(balanceDiffsAddrs, func(i, j int) bool {
		return bytes.Compare(balanceDiffsAddrs[i][:], balanceDiffsAddrs[j][:]) < 0
	})

	for _, addr := range balanceDiffsAddrs {
		encoderBalanceDiffs = append(encoderBalanceDiffs, b.balanceChanges[addr].toEncoderObj(addr))
	}

	encoderObj := encodingBlockAccessList{
		AccountAccesses: encoderAccountAccesses,
		BalanceDiffs:    encoderBalanceDiffs,
		CodeDiffs:       codeDiffsToEncoderObj(b.codeChanges),
		NonceDiffs:      nonceDiffsToEncoderObj(b.prestateNonces),
	}
	_ = encoderObj
	return nil
}

func (b *BlockAccessList) Hash() common.Hash {
	if b.hash == (common.Hash{}) {
		b.hash = common.BytesToHash(crypto.Keccak256(b.encodeSSZ()))
	}
	return b.hash
}
