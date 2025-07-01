package types

import (
	"bytes"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/holiman/uint256"
	"io"
	"slices"
	"sort"
	"strings"
)

//go:generate go run github.com/ferranbt/fastssz/sszgen --path . --objs encodingPerTxAccess,encodingSlotAccess,encodingAccountAccess,encodingAccountAccessList,encodingBlockAccessList,encodingBalanceDelta,encodingBalanceChange,encodingAccountBalanceDiff,encodingCodeChange,encodingAccountNonce,encodingNonceDiffs,encodingAccountNonce,encodingBlockAccessList --output bal_encoding_generated.go

// encoder types

type encodingStorageWrite struct {
	TxIdx      uint64   `ssz-size:"2"`
	ValueAfter [32]byte `ssz-size:"32"`
}

type encodingStorageWrites struct {
	Slot     [32]byte               `ssz-size:"32"`
	Accesses []encodingStorageWrite `ssz-max:"300000"`
}

func (e *encodingStorageWrites) toMap() (map[uint64]common.Hash, error) {
	var prev *uint64

	res := make(map[uint64]common.Hash)

	for _, write := range e.Accesses {
		if prev != nil {
			if *prev >= write.TxIdx {
				return nil, fmt.Errorf("storage write tx indices not in order")
			}
			res[write.TxIdx] = write.ValueAfter
		}
	}
	return res, nil
}

// TODO: implement encoder/decoder manually on this to enforce code size limit
type encodingCodeChange []byte

type encodingAccountAccess struct {
	Address        [20]byte                `ssz-size:"20"`
	StorageWrites  []encodingStorageWrites `ssz-max:"300000"`
	StorageReads   [][32]byte              `ssz-max:"300000"`
	BalanceChanges []encodingBalanceChange `ssz-max:"300000"`
	NonceChanges   []encodingAccountNonce  `ssz-max:"300000"`
	Code           []encodingCodeChange    `ssz-max:"1"`
}

func (e *encodingAccountAccess) toAccountAccess() (*accountAccess, error) {
	res := accountAccess{
		storageWrites:  make(map[common.Hash]storageWrites),
		storageReads:   make(map[common.Hash]struct{}),
		balanceChanges: make(balanceDiff),
		nonceChanges:   make(accountNonceDiffs),
		codeChange:     nil,
	}

	{
		var prevWriteSlot *common.Hash
		for _, write := range e.StorageWrites {
			if prevWriteSlot != nil {
				if bytes.Compare(write.Slot[:], (*prevWriteSlot)[:]) <= 0 {
					return nil, fmt.Errorf("storage writes slots lexicographic order")
				}
			}
			wr, err := write.toMap()
			if err != nil {
				return nil, err
			}

			res.storageWrites[write.Slot] = wr
		}
	}

	{
		var prevReadSlot *common.Hash
		for _, read := range e.StorageReads {
			if prevReadSlot != nil {
				if bytes.Compare(read[:], (*prevReadSlot)[:]) <= 0 {
					return nil, fmt.Errorf("storage read slots not in lexicographic order")
				}
			}
			res.storageReads[read] = struct{}{}
		}
	}

	{
		var prevBalanceChangeIdx *uint64
		for _, balanceChange := range e.BalanceChanges {
			if prevBalanceChangeIdx != nil {
				if *prevBalanceChangeIdx >= balanceChange.TxIdx {
					return nil, fmt.Errorf("balance change tx indices not in ascending order")
				}
			}
			res.balanceChanges[balanceChange.TxIdx] = new(uint256.Int).SetBytes(balanceChange.Delta[:])
		}
	}

	{
		var prevNonceDiffIdx *uint64
		for _, nonceDiff := range e.NonceChanges {
			if prevNonceDiffIdx != nil {
				if *prevNonceDiffIdx >= nonceDiff.TxIdx {
					return nil, fmt.Errorf("nonce diffs not in ascending order by tx index")
				}
			}
			res.nonceChanges[nonceDiff.TxIdx] = nonceDiff.Nonce
		}
	}

	{
		if len(e.Code) == 1 {
			codeChange := bytes.Clone(e.Code[0])
			res.codeChange = &codeChange
		}
	}
	return &res, nil
}

type encodingBlockAccessList []encodingAccountAccess

func (e *encodingBlockAccessList) toBlockAccessList() (BlockAccessList, error) {
	res := make(BlockAccessList)
	var prevAccount *common.Address
	for _, encAccountAccess := range *e {
		if prevAccount != nil {
			if bytes.Compare(encAccountAccess.Address[:], (*prevAccount)[:]) <= 0 {
				return nil, fmt.Errorf("block access list accounts not in lexicographic order")
			}
		}
		aa, err := encAccountAccess.toAccountAccess()
		if err != nil {
			return nil, err
		}
		res[encAccountAccess.Address] = aa
	}
	return res, nil
}

// TODO: verify that Geth encodes the endianess according to the spec
type encodingBalanceDelta [12]byte

func (b *encodingBalanceDelta) Set(val *uint256.Int) *encodingBalanceDelta {
	valBytes := val.Bytes()
	if len(valBytes) > 12 {
		panic("can't encode value that is greater than 12 bytes in size")
	}
	copy(b[12-len(valBytes):], valBytes[:])
	return b
}

type encodingBalanceChange struct {
	TxIdx uint64 `ssz-size:"2"`
	Delta encodingBalanceDelta
}

type encodingAccountNonce struct {
	TxIdx uint64 `ssz-size:"2"`
	Nonce uint64 `ssz-size:"8"`
}

type BlockAccessList map[common.Address]*accountAccess

func (a *accountAccess) MarkRead(key common.Hash) {
	if _, ok := a.storageWrites[key]; !ok {
		a.storageReads[key] = struct{}{}
	}
}

func (a *accountAccess) MarkWrite(txIdx uint64, key, value common.Hash) {
	if _, ok := a.storageWrites[key]; !ok {
		a.storageWrites[key] = make(storageWrites)
	}

	a.storageWrites[key][txIdx] = value
}

type balanceDiff map[uint64]*uint256.Int

func (b balanceDiff) Copy() balanceDiff {
	res := make(map[uint64]*uint256.Int)
	for idx, balance := range b {
		res[idx] = balance.Clone()
	}
	return res
}

// map of tx index to the prestate nonce
type accountNonceDiffs map[uint64]uint64

type storageWrites map[uint64]common.Hash

func (s storageWrites) toEncoderObj(slot common.Hash) encodingStorageWrites {
	res := encodingStorageWrites{
		Slot: slot,
	}

	var storageWriteIdxs []uint64
	for idx := range s {
		storageWriteIdxs = append(storageWriteIdxs, idx)
	}
	sort.Slice(storageWriteIdxs, func(i, j int) bool {
		return storageWriteIdxs[i] < storageWriteIdxs[j]
	})

	for _, idx := range storageWriteIdxs {
		res.Accesses = append(res.Accesses, encodingStorageWrite{
			TxIdx:      idx,
			ValueAfter: s[idx],
		})
	}

	return res
}

type accountAccess struct {
	storageWrites  map[common.Hash]storageWrites
	storageReads   map[common.Hash]struct{}
	balanceChanges balanceDiff
	nonceChanges   accountNonceDiffs
	codeChange     *[]byte
}

func newAccountAccess() *accountAccess {
	return &accountAccess{
		storageWrites:  make(map[common.Hash]storageWrites),
		storageReads:   make(map[common.Hash]struct{}),
		balanceChanges: make(balanceDiff),
		nonceChanges:   make(accountNonceDiffs),
	}
}

func (a *accountAccess) toEncodingObj(addr common.Address) encodingAccountAccess {
	res := encodingAccountAccess{
		Address:        addr,
		StorageWrites:  make([]encodingStorageWrites, 0),
		StorageReads:   make([][32]byte, 0),
		BalanceChanges: make([]encodingBalanceChange, 0),
		NonceChanges:   make([]encodingAccountNonce, 0),
		Code:           nil,
	}

	{
		var writeSlots []common.Hash

		for slot := range a.storageWrites {
			writeSlots = append(writeSlots, slot)
		}
		sort.Slice(writeSlots, func(i, j int) bool {
			return bytes.Compare(writeSlots[i][:], writeSlots[j][:]) < 0
		})

		for _, slot := range writeSlots {
			res.StorageWrites = append(res.StorageWrites, a.storageWrites[slot].toEncoderObj(slot))
		}
	}

	{
		var readSlots []common.Hash
		for slot := range a.storageReads {
			readSlots = append(readSlots, slot)
		}
		sort.Slice(readSlots, func(i, j int) bool {
			return bytes.Compare(readSlots[i][:], readSlots[j][:]) < 0
		})
		for _, slot := range readSlots {
			res.StorageReads = append(res.StorageReads, slot)
		}
	}

	{
		var balanceChangeIdxs []uint64
		for idx := range a.balanceChanges {
			balanceChangeIdxs = append(balanceChangeIdxs, idx)
		}

		sort.Slice(balanceChangeIdxs, func(i, j int) bool {
			return balanceChangeIdxs[i] < balanceChangeIdxs[j]
		})

		for _, idx := range balanceChangeIdxs {
			res.BalanceChanges = append(res.BalanceChanges, encodingBalanceChange{
				TxIdx: idx,
				Delta: *new(encodingBalanceDelta).Set(a.balanceChanges[idx]),
			})
		}
	}

	{
		var nonceChangeIdxs []uint64
		for idx := range a.nonceChanges {
			nonceChangeIdxs = append(nonceChangeIdxs, idx)
		}
		sort.Slice(nonceChangeIdxs, func(i, j int) bool {
			return nonceChangeIdxs[i] < nonceChangeIdxs[j]
		})

		for _, idx := range nonceChangeIdxs {
			res.NonceChanges = append(res.NonceChanges, encodingAccountNonce{
				TxIdx: idx,
				Nonce: a.nonceChanges[idx],
			})
		}
	}

	if a.codeChange != nil {
		res.Code = []encodingCodeChange{
			slices.Clone(*a.codeChange),
		}
	}

	return res
}

func (b BlockAccessList) toEncodingObj() (res encodingBlockAccessList) {
	for addr, acct := range b {
		res = append(res, acct.toEncodingObj(addr))
	}
	return res
}

func (b BlockAccessList) Copy() BlockAccessList {
	panic("not implemented")
}

func (b *BlockAccessList) Eq(other *BlockAccessList) bool {
	panic("not implemented")
}

// NonceDiff records tx post-state nonce of any contract-like accounts whose nonce was incremented
func (b BlockAccessList) NonceDiff(address common.Address, txIdx, postNonce uint64) {
	if _, ok := b[address]; !ok {
		b[address] = newAccountAccess()
	}

	b[address].nonceChanges[txIdx] = postNonce
}

// BalanceChange records the transaction post-state balance of an account that changed its balance
// TODO: for the first transaction in the block, should this consider balances before any system contracts
// were executed?
// TODO: for the final transaction in the block, should this consider the balance change from block reward?
func (b BlockAccessList) BalanceChange(txIdx uint64, address common.Address, balance *uint256.Int) {
	if _, ok := b[address]; !ok {
		b[address] = newAccountAccess()
	}

	b[address].balanceChanges[txIdx] = balance
}

// TODO for eip:  specify that storage slots which are read/modified for accounts that are created/selfdestructed
// in same transaction aren't included in teh BAL (?)

// TODO for eip:  specify that storage slots of newly-created accounts which are only read are not included in the BAL (?)

// called during tx execution every time a storage slot is read
func (b BlockAccessList) StorageRead(address common.Address, key common.Hash) {
	if _, ok := b[address]; !ok {
		b[address] = newAccountAccess()
	}

	if _, ok := b[address].storageWrites[key]; ok {
		return
	}

	b[address].storageReads[key] = struct{}{}
}

// called every time a mutated storage value is committed upon transaction finalization
func (b BlockAccessList) StorageWrite(txIdx uint64, address common.Address, key, value common.Hash) {
	if _, ok := b[address]; !ok {
		b[address] = newAccountAccess()
	}

	if _, ok := b[address].storageWrites[key]; !ok {
		b[address].storageWrites[key] = make(storageWrites)
	}
	b[address].storageWrites[key][txIdx] = value
	delete(b[address].storageReads, key)
}

// TODO: include these in the PR to the EIP
// arguments for post-transaction nonces, which include nonces from tx senders:
// * we can parallelize block execution and state root computation:
//     - pre state + post-diffs gives us everything we need to update the tree
// * delegated EOAs can call code that does multiple creations, bumping the delegated acct nonce by more than 1 per tx
// * simpler implementation current spec: just accumulate modified nonces at transaction finalisation.

// called during tx finalisation for each dirty account with mutated code
func (b BlockAccessList) CodeChange(address common.Address, code []byte) {
	if _, ok := b[address]; !ok {
		b[address] = newAccountAccess()
	}

	cc := slices.Clone(code)
	b[address].codeChange = &cc
}

func (b *BlockAccessList) encodeSSZ() ([]byte, error) {
	encoderObj := b.toEncodingObj()
	dst, err := encoderObj.MarshalSSZTo(nil)
	if err != nil {
		return nil, err
	}
	return dst, nil
}

func (e encodingBlockAccessList) PrettyPrint() string {
	var res bytes.Buffer
	printWithIndent := func(indent int, text string) {
		fmt.Fprintf(&res, "%s%s\n", strings.Repeat("    ", indent), text)
	}
	for _, accountDiff := range e {
		printWithIndent(0, fmt.Sprintf("%x:", accountDiff.Address))
		printWithIndent(1, fmt.Sprintf("code:    %x", accountDiff.Code)) // TODO: code shouldn't be in account accesses (?)

		printWithIndent(1, "storage writes:")
		for _, slot := range accountDiff.StorageWrites {
			printWithIndent(2, fmt.Sprintf("%x:", slot))
			for _, access := range slot.Accesses {
				printWithIndent(3, fmt.Sprintf("idx: %d", access.TxIdx))
				printWithIndent(3, fmt.Sprintf("post: %x", access.ValueAfter))
			}
		}

		printWithIndent(1, "storage reads:")
		for _, slot := range accountDiff.StorageReads {
			printWithIndent(2, fmt.Sprintf("%x", slot))
		}

		printWithIndent(1, "balance changes:")
		for _, change := range accountDiff.BalanceChanges {
			printWithIndent(2, fmt.Sprintf("index: %d", change.TxIdx))
			printWithIndent(2, fmt.Sprintf("balance: %s", new(uint256.Int).SetBytes(change.Delta[:]).String()))
		}

		printWithIndent(1, "nonce changes:")
		for _, change := range accountDiff.NonceChanges {
			printWithIndent(2, fmt.Sprintf("index: %d", change.TxIdx))
			printWithIndent(2, fmt.Sprintf("nonce: %d", change.Nonce))
		}
	}

	return res.String()
}

// human-readable representation
func (b BlockAccessList) PrettyPrint() string {
	enc := b.toEncodingObj()
	return enc.PrettyPrint()
}

func (b *BlockAccessList) Hash() common.Hash {
	panic("not implemented")
}

func (b *BlockAccessList) EncodeRLP(wr io.Writer) error {
	w := rlp.NewEncoderBuffer(wr)
	buf, err := b.encodeSSZ()
	if err != nil {
		return err
	}
	w.WriteBytes(buf)
	return w.Flush()
}

func (b *BlockAccessList) DecodeRLP(s *rlp.Stream) error {
	var enc encodingBlockAccessList
	encBytes, err := s.Bytes()
	if err != nil {
		return err
	}
	if err := enc.UnmarshalSSZ(encBytes); err != nil {
		return err
	}
	res, err := enc.toBlockAccessList()
	if err != nil {
		return err
	}
	*b = res
	return nil
}

var _ rlp.Encoder = &BlockAccessList{}
var _ rlp.Decoder = &BlockAccessList{}
