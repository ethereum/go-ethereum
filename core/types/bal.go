package types

import (
	"bytes"
	"fmt"
	"io"
	"maps"
	"slices"
	"sort"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	ssz "github.com/ferranbt/fastssz"
	"github.com/holiman/uint256"
)

//go:generate go run github.com/ferranbt/fastssz/sszgen  --output bal_encoding_generated.go --path . --objs encodingStorageWrite,encodingStorageWrites,encodingCodeChange,encodingBalanceChange,encodingAccountNonce,encodingCodeChange,encodingAccountAccess

// BlockAccessList contains post-block modified state and some state accessed
// in execution (account addresses and storage keys).
type BlockAccessList struct {
	accounts map[common.Address]*accountAccess
}

// NewBlockAccessList instantiates an empty access list.
func NewBlockAccessList() BlockAccessList {
	return BlockAccessList{
		accounts: make(map[common.Address]*accountAccess),
	}
}

// AccountRead records the address of an account that has been read during execution.
func (b *BlockAccessList) AccountRead(addr common.Address) {
	if _, ok := b.accounts[addr]; !ok {
		b.accounts[addr] = newAccountAccess()
	}
}

// StorageRead records a storage key read during execution.
func (b *BlockAccessList) StorageRead(address common.Address, key common.Hash) {
	if _, ok := b.accounts[address]; !ok {
		b.accounts[address] = newAccountAccess()
	}

	if _, ok := b.accounts[address].StorageWrites[key]; ok {
		return
	}

	b.accounts[address].StorageReads[key] = struct{}{}
}

// StorageWrite records the post-transaction value of a mutated storage slot.
// The storage slot is removed from the list of read slots.
func (b *BlockAccessList) StorageWrite(txIdx uint64, address common.Address, key, value common.Hash) {
	if _, ok := b.accounts[address]; !ok {
		b.accounts[address] = newAccountAccess()
	}

	if _, ok := b.accounts[address].StorageWrites[key]; !ok {
		b.accounts[address].StorageWrites[key] = make(slotWrites)
	}
	b.accounts[address].StorageWrites[key][txIdx] = value
	delete(b.accounts[address].StorageReads, key)
}

// CodeChange records the code of a newly-created contract.
func (b *BlockAccessList) CodeChange(address common.Address, txIndex uint64, code []byte) {
	if _, ok := b.accounts[address]; !ok {
		b.accounts[address] = newAccountAccess()
	}

	b.accounts[address].CodeChange = &codeChange{
		TxIndex: txIndex,
		Code:    slices.Clone(code),
	}
}

func (b *BlockAccessList) encodeSSZ() ([]byte, error) {
	encoderObj := b.toEncodingObj()
	dst, err := encoderObj.MarshalSSZTo(nil)
	if err != nil {
		return nil, err
	}
	return dst, nil
}

// NonceDiff records tx post-state nonce of any contract-like accounts whose nonce was incremented
func (b *BlockAccessList) NonceDiff(address common.Address, txIdx, postNonce uint64) {
	if _, ok := b.accounts[address]; !ok {
		b.accounts[address] = newAccountAccess()
	}

	b.accounts[address].NonceChanges[txIdx] = postNonce
}

// BalanceChange records the post-transaction balance of an account whose
// balance changed.
func (b *BlockAccessList) BalanceChange(txIdx uint64, address common.Address, balance *uint256.Int) {
	if _, ok := b.accounts[address]; !ok {
		b.accounts[address] = newAccountAccess()
	}

	b.accounts[address].BalanceChanges[txIdx] = balance
}

// contains the post-transaction balances of an account, keyed by transaction indices
// where it was changed.
type balanceDiff map[uint64]*uint256.Int

// copy returns a deep copy of the object
func (b balanceDiff) copy() balanceDiff {
	res := make(balanceDiff)
	for idx, balance := range b {
		res[idx] = balance.Clone()
	}
	return res
}

// PrettyPrint returns a human-readable representation of the access list
func (b *BlockAccessList) PrettyPrint() string {
	enc := b.toEncodingObj()
	return enc.prettyPrint()
}

// Hash computes the keccak256 hash of the SSZ encoded access list
func (b *BlockAccessList) Hash() common.Hash {
	enc, _ := b.encodeSSZ()
	return crypto.Keccak256Hash(enc)
}

// EncodeRLP returns the SSZ-encoded access list wrapped into RLP bytes
func (b *BlockAccessList) EncodeRLP(wr io.Writer) error {
	w := rlp.NewEncoderBuffer(wr)
	buf, err := b.encodeSSZ()
	if err != nil {
		return err
	}
	w.WriteBytes(buf)
	return w.Flush()
}

func (b *BlockAccessList) DecodeSSZ(buf []byte) error {
	var enc encodingBlockAccessList
	if err := enc.UnmarshalSSZ(buf); err != nil {
		return err
	}
	res, err := enc.toBlockAccessList()
	if err != nil {
		return err
	}
	*b = *res
	return nil
}

// DecodeRLP decodes the access list
func (b *BlockAccessList) DecodeRLP(s *rlp.Stream) error {
	encBytes, err := s.Bytes()
	if err != nil {
		return err
	}
	return b.DecodeSSZ(encBytes)
}

var _ rlp.Encoder = &BlockAccessList{}
var _ rlp.Decoder = &BlockAccessList{}

// the post-state nonce values of a contract account keyed by tx index
type accountNonceDiffs map[uint64]uint64

// the post-state values of a storage slot, keyed by tx index
type slotWrites map[uint64]common.Hash

// toEncoderObj returns an instance of the slot writes which will be used as
// the input for encoding.
func (s slotWrites) toEncoderObj(slot common.Hash) encodingSlotWrites {
	res := encodingSlotWrites{
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

// codeChange contains the code deployed at an address and the transaction
// index where the deployment took place.
type codeChange struct {
	TxIndex uint64 `json:"txIndex,omitempty"`
	Code    []byte `json:"code,omitempty"`
}

// post-state values of an account's storage slots modified in a block, keyed
// by slot key
type storageWrites map[common.Hash]slotWrites

func (s storageWrites) copy() storageWrites {
	res := make(storageWrites)
	for slot, writes := range s {
		res[slot] = maps.Clone(writes)
	}
	return res
}

// accountAccess contains post-block account state for mutations as well as
// all storage keys that were read during execution.
type accountAccess struct {
	StorageWrites  storageWrites            `json:"storageWrites,omitempty"`
	StorageReads   map[common.Hash]struct{} `json:"storageReads,omitempty"`
	BalanceChanges balanceDiff              `json:"balanceChanges,omitempty"`
	NonceChanges   accountNonceDiffs        `json:"nonceChanges,omitempty"`

	// only set for contract accounts which were deployed in the block
	CodeChange *codeChange `json:"codeChange,omitempty"`
}

func newAccountAccess() *accountAccess {
	return &accountAccess{
		StorageWrites:  make(map[common.Hash]slotWrites),
		StorageReads:   make(map[common.Hash]struct{}),
		BalanceChanges: make(balanceDiff),
		NonceChanges:   make(accountNonceDiffs),
	}
}

// toEncodingObj creates an instance of the accountAccess of the type that is
// used as input for the encoding.
func (a *accountAccess) toEncodingObj(addr common.Address) encodingAccountAccess {
	res := encodingAccountAccess{
		Address:        addr,
		StorageWrites:  make([]encodingSlotWrites, 0),
		StorageReads:   make([][32]byte, 0),
		BalanceChanges: make([]encodingBalanceChange, 0),
		NonceChanges:   make([]encodingAccountNonce, 0),
		Code:           nil,
	}

	{
		var writeSlots []common.Hash

		for slot := range a.StorageWrites {
			writeSlots = append(writeSlots, slot)
		}
		sort.Slice(writeSlots, func(i, j int) bool {
			return bytes.Compare(writeSlots[i][:], writeSlots[j][:]) < 0
		})

		for _, slot := range writeSlots {
			res.StorageWrites = append(res.StorageWrites, a.StorageWrites[slot].toEncoderObj(slot))
		}
	}

	{
		var readSlots []common.Hash
		for slot := range a.StorageReads {
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
		for idx := range a.BalanceChanges {
			balanceChangeIdxs = append(balanceChangeIdxs, idx)
		}

		sort.Slice(balanceChangeIdxs, func(i, j int) bool {
			return balanceChangeIdxs[i] < balanceChangeIdxs[j]
		})

		for _, idx := range balanceChangeIdxs {
			res.BalanceChanges = append(res.BalanceChanges, encodingBalanceChange{
				TxIdx: idx,
				Delta: *new(encodingBalanceDelta).set(a.BalanceChanges[idx]),
			})
		}
	}

	{
		var nonceChangeIdxs []uint64
		for idx := range a.NonceChanges {
			nonceChangeIdxs = append(nonceChangeIdxs, idx)
		}
		sort.Slice(nonceChangeIdxs, func(i, j int) bool {
			return nonceChangeIdxs[i] < nonceChangeIdxs[j]
		})

		for _, idx := range nonceChangeIdxs {
			res.NonceChanges = append(res.NonceChanges, encodingAccountNonce{
				TxIdx: idx,
				Nonce: a.NonceChanges[idx],
			})
		}
	}

	if a.CodeChange != nil {
		res.Code = []encodingCodeChange{
			{
				a.CodeChange.TxIndex,
				bytes.Clone(a.CodeChange.Code),
			},
		}
	}

	return res
}

// toEncodingObj returns an instance of the access list expressed as the type
// which is used as input for the decoding.
func (b *BlockAccessList) toEncodingObj() (res encodingBlockAccessList) {
	var addrs []common.Address
	for addr := range b.accounts {
		addrs = append(addrs, addr)
	}
	sort.Slice(addrs, func(i, j int) bool {
		return bytes.Compare(addrs[i][:], addrs[j][:]) < 0
	})

	for _, addr := range addrs {
		res = append(res, b.accounts[addr].toEncodingObj(addr))
	}
	return res
}

// Copy returns a deep copy of the access list.
func (b *BlockAccessList) Copy() *BlockAccessList {
	res := new(BlockAccessList)
	for addr, aa := range b.accounts {
		var aaCopy accountAccess
		aaCopy.StorageReads = maps.Clone(aa.StorageReads)
		aaCopy.StorageWrites = aa.StorageWrites.copy()
		aaCopy.NonceChanges = maps.Clone(aa.NonceChanges)
		aaCopy.BalanceChanges = aa.BalanceChanges.copy()
		if aa.CodeChange != nil {
			aaCopy.CodeChange = &codeChange{
				TxIndex: aa.CodeChange.TxIndex,
				Code:    slices.Clone(aa.CodeChange.Code),
			}
		}
		res.accounts[addr] = &aaCopy
	}
	return res
}

func (e *encodingBlockAccessList) prettyPrint() string {
	var res bytes.Buffer
	printWithIndent := func(indent int, text string) {
		fmt.Fprintf(&res, "%s%s\n", strings.Repeat("    ", indent), text)
	}
	for _, accountDiff := range *e {
		printWithIndent(0, fmt.Sprintf("%x:", accountDiff.Address))
		if len(accountDiff.Code) > 0 {
			printWithIndent(1, "code:")
			printWithIndent(2, fmt.Sprintf("%d: %x", accountDiff.Code[0].TxIndex, accountDiff.Code[0].Code))
		}

		printWithIndent(1, "storage writes:")
		for _, sWrite := range accountDiff.StorageWrites {
			printWithIndent(2, fmt.Sprintf("%x:", sWrite.Slot))
			for _, access := range sWrite.Accesses {
				printWithIndent(3, fmt.Sprintf("%d: %x", access.TxIdx, access.ValueAfter))
			}
		}

		printWithIndent(1, "storage reads:")
		for _, slot := range accountDiff.StorageReads {
			printWithIndent(2, fmt.Sprintf("%x", slot))
		}

		printWithIndent(1, "balance changes:")
		for _, change := range accountDiff.BalanceChanges {
			balance := new(uint256.Int).SetBytes(change.Delta[:]).String()
			printWithIndent(2, fmt.Sprintf("%d: %s", change.TxIdx, balance))
		}

		printWithIndent(1, "nonce changes:")
		for _, change := range accountDiff.NonceChanges {
			printWithIndent(2, fmt.Sprintf("%d: %d", change.TxIdx, change.Nonce))
		}
	}

	return res.String()
}

// used as input for encoding.
type encodingStorageWrite struct {
	TxIdx      uint64   `ssz-size:"2"`
	ValueAfter [32]byte `ssz-size:"32"`
}

// used as input for encoding.  Storage writes are expected to be sorted
// lexicographically by their storage key.
type encodingSlotWrites struct {
	Slot     [32]byte               `ssz-size:"32"`
	Accesses []encodingStorageWrite `ssz-max:"300000"`
}

// toMap returns an instance of the encoding-representation slot writes in
// working representation.
func (e *encodingSlotWrites) toMap() (slotWrites, error) {
	var prev *uint64

	res := make(slotWrites)

	for _, write := range e.Accesses {
		if prev != nil {
			if *prev >= write.TxIdx {
				return nil, fmt.Errorf("storage write tx indices not in order")
			}
		}
		res[write.TxIdx] = write.ValueAfter
		prev = &write.TxIdx
	}
	return res, nil
}

// encoding objects:  These are used as input for the encoding.
// They mirror the spec format.

type encodingCodeChange struct {
	TxIndex uint64 `ssz-size:"2"`
	Code    []byte `ssz-max:"24576"`
}

type encodingAccountAccess struct {
	Address        [20]byte                `ssz-size:"20"`
	StorageWrites  []encodingSlotWrites    `ssz-max:"300000"`
	StorageReads   [][32]byte              `ssz-max:"300000"`
	BalanceChanges []encodingBalanceChange `ssz-max:"300000"`
	NonceChanges   []encodingAccountNonce  `ssz-max:"300000"`
	Code           []encodingCodeChange    `ssz-max:"1"`
}

// toAccountAccess converts the account accesses out of encoding format.
// If any of the keys in the encoding object are not ordered according to the
// spec, an error is returned.
func (e *encodingAccountAccess) toAccountAccess() (*accountAccess, error) {
	res := accountAccess{
		StorageWrites:  make(map[common.Hash]slotWrites),
		StorageReads:   make(map[common.Hash]struct{}),
		BalanceChanges: make(balanceDiff),
		NonceChanges:   make(accountNonceDiffs),
		CodeChange:     nil,
	}

	{
		var prevWriteSlot *[32]byte
		for _, write := range e.StorageWrites {
			if prevWriteSlot != nil {
				if bytes.Compare((*prevWriteSlot)[:], write.Slot[:]) >= 0 {
					return nil, fmt.Errorf("storage writes slots lexicographic order")
				}
			}
			wr, err := write.toMap()
			if err != nil {
				return nil, err
			}

			res.StorageWrites[write.Slot] = wr
			prevWriteSlot = &write.Slot
		}
	}

	{
		var prevReadSlot *[32]byte
		for _, read := range e.StorageReads {
			if prevReadSlot != nil {
				if bytes.Compare((*prevReadSlot)[:], read[:]) >= 0 {
					return nil, fmt.Errorf("storage read slots not in lexicographic order")
				}
			}
			res.StorageReads[read] = struct{}{}
			prevReadSlot = &read
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
			res.BalanceChanges[balanceChange.TxIdx] = new(uint256.Int).SetBytes(balanceChange.Delta[:])
			prevBalanceChangeIdx = &balanceChange.TxIdx
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
			res.NonceChanges[nonceDiff.TxIdx] = nonceDiff.Nonce
			prevNonceDiffIdx = &nonceDiff.TxIdx
		}
	}

	{
		if len(e.Code) == 1 {
			codeChange := codeChange{e.Code[0].TxIndex, bytes.Clone(e.Code[0].Code)}
			res.CodeChange = &codeChange
		}
	}
	return &res, nil
}

type encodingBlockAccessList []encodingAccountAccess

// toBlockAccessList converts out of the encoding format, returning an error if
// values in the encoder object are not properly ordered according to the spec.
func (e *encodingBlockAccessList) toBlockAccessList() (*BlockAccessList, error) {
	res := NewBlockAccessList()
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
		res.accounts[encAccountAccess.Address] = aa
	}
	return &res, nil
}

// SSZ encoding/decoding methods implemented manually for encodingBlockAccessList
// because defined types cannot have tags, so fastssz isn't able to generate
// methods on them.

func (e *encodingBlockAccessList) MarshalSSZTo(buf []byte) (dst []byte, err error) {
	if len(*e) > 300000 {
		return nil, fmt.Errorf("oversized")
	}

	offset := 4 * len(*e)
	for ii := 0; ii < len(*e); ii++ {
		dst = ssz.WriteOffset(dst, offset)
		sz := (*e)[ii].SizeSSZ()
		offset += sz
	}
	for ii := 0; ii < len(*e); ii++ {
		if dst, err = (*e)[ii].MarshalSSZTo(dst); err != nil {
			return
		}
	}

	return
}

func (e *encodingBlockAccessList) UnmarshalSSZ(buf []byte) error {
	num, err := ssz.DecodeDynamicLength(buf, 300000)
	if err != nil {
		return err
	}
	res := make([]encodingAccountAccess, num)
	err = ssz.UnmarshalDynamic(buf, num, func(indx int, buf []byte) (err error) {
		if err = res[indx].UnmarshalSSZ(buf); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	*e = res
	return nil
}

func (e *encodingBlockAccessList) MarshalSSZ() ([]byte, error) {
	return ssz.MarshalSSZ(e)
}

func (e *encodingBlockAccessList) SizeSSZ() (size int) {
	for i := 0; i < len(*e); i++ {
		size += 4
		size += (*e)[i].SizeSSZ()
	}
	return
}

var _ ssz.Marshaler = &encodingBlockAccessList{}
var _ ssz.Unmarshaler = &encodingBlockAccessList{}

// TODO: verify that Geth encodes the endianess according to the spec
type encodingBalanceDelta [16]byte

func (b *encodingBalanceDelta) set(val *uint256.Int) *encodingBalanceDelta {
	valBytes := val.Bytes()
	if len(valBytes) > 16 {
		panic("can't encode value that is greater than 12 bytes in size")
	}
	copy(b[16-len(valBytes):], valBytes[:])
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
