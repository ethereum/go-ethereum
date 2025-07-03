package types

import (
	"bytes"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
	ssz "github.com/ferranbt/fastssz"
	"github.com/holiman/uint256"
	"io"
	"sort"
)

//go:generate go run github.com/ferranbt/fastssz/sszgen  --output bal_encoding_generated.go --path . --objs encodingStorageWrite,encodingStorageWrites,encodingCodeChange,encodingBalanceChange,encodingAccountNonce,encodingCodeChange,encodingAccountAccess

// These are objects used as input for the access list encoding. They mirror
// the spec format.

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
			wr, err := write.toSlotWrites()
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

// DecodeRLP decodes the access list
func (b *BlockAccessList) DecodeRLP(s *rlp.Stream) error {
	encBytes, err := s.Bytes()
	if err != nil {
		return err
	}
	return b.decodeSSZ(encBytes)
}

var _ rlp.Encoder = &BlockAccessList{}
var _ rlp.Decoder = &BlockAccessList{}

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

func (b *BlockAccessList) encodeSSZ() ([]byte, error) {
	encoderObj := b.toEncodingObj()
	dst, err := encoderObj.MarshalSSZTo(nil)
	if err != nil {
		return nil, err
	}
	return dst, nil
}

func (b *BlockAccessList) decodeSSZ(buf []byte) error {
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

// toSlotWrites returns an instance of the encoding-representation slot writes in
// working representation.
func (e *encodingSlotWrites) toSlotWrites() (slotWrites, error) {
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
