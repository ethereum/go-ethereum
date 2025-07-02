package bal

import (
	"bytes"
	"fmt"
	"io"
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/holiman/uint256"
)

//go:generate go run github.com/ferranbt/fastssz/sszgen  --output bal_encoding_ssz_generated.go --path . --objs encodingStorageWrite,encodingCodeChange,encodingBalanceChange,encodingAccountNonce,encodingAccountAccess,encodingBlockAccessList

//go:generate go run github.com/ethereum/go-ethereum/rlp/rlpgen -out bal_encoding_rlp_storagewrite_generated.go -type encodingStorageWrite -decoder
//go:generate go run github.com/ethereum/go-ethereum/rlp/rlpgen -out bal_encoding_rlp_codechange_generated.go -type encodingCodeChange -decoder
//go:generate go run github.com/ethereum/go-ethereum/rlp/rlpgen -out bal_encoding_rlp_balancechange_generated.go -type encodingBalanceChange -decoder
//go:generate go run github.com/ethereum/go-ethereum/rlp/rlpgen -out bal_encoding_rlp_accountnonce_generated.go -type encodingAccountNonce -decoder
//go:generate go run github.com/ethereum/go-ethereum/rlp/rlpgen -out bal_encoding_rlp_accountaccess_generated.go -type encodingAccountAccess -decoder
//go:generate go run github.com/ethereum/go-ethereum/rlp/rlpgen -out bal_encoding_rlp_blockaccesslist_generated.go -type encodingBlockAccessList -decoder

// These are objects used as input for the access list encoding. They mirror
// the spec format.

type encodingBlockAccessList struct {
	Accesses []encodingAccountAccess `ssz-max:"300000"`
}

// toBlockAccessList converts out of the encoding format, returning an error if
// values in the encoder object are not properly ordered according to the spec.
func (e *encodingBlockAccessList) toBlockAccessList() (*BlockAccessList, error) {
	res := NewBlockAccessList()
	var prevAccount *common.Address
	for _, encAccountAccess := range e.Accesses {
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

type encodingCodeChange struct {
	TxIndex uint16
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
		var prevBalanceChangeIdx *uint16
		for _, balanceChange := range e.BalanceChanges {
			if prevBalanceChangeIdx != nil {
				if *prevBalanceChangeIdx >= balanceChange.TxIdx {
					return nil, fmt.Errorf("balance changes not in ascending order by tx index")
				}
			}
			res.BalanceChanges[balanceChange.TxIdx] = new(uint256.Int).SetBytes(balanceChange.Balance[:])
			prevBalanceChangeIdx = &balanceChange.TxIdx
		}
	}

	{
		var prevNonceDiffIdx *uint16
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

type encodingBalance [16]byte

func (b *encodingBalance) set(val *uint256.Int) *encodingBalance {
	valBytes := val.Bytes()
	if len(valBytes) > 16 {
		panic("can't encode value that is greater than 16 bytes in size")
	}
	copy(b[16-len(valBytes):], valBytes[:])
	return b
}

type encodingBalanceChange struct {
	TxIdx   uint16 `ssz-size:"2"`
	Balance encodingBalance
}

type encodingAccountNonce struct {
	TxIdx uint16 `ssz-size:"2"`
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

	var storageWriteIdxs []uint16
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
		var balanceChangeIdxs []uint16
		for idx := range a.BalanceChanges {
			balanceChangeIdxs = append(balanceChangeIdxs, idx)
		}

		sort.Slice(balanceChangeIdxs, func(i, j int) bool {
			return balanceChangeIdxs[i] < balanceChangeIdxs[j]
		})

		for _, idx := range balanceChangeIdxs {
			res.BalanceChanges = append(res.BalanceChanges, encodingBalanceChange{
				TxIdx:   idx,
				Balance: *new(encodingBalance).set(a.BalanceChanges[idx]),
			})
		}
	}

	{
		var nonceChangeIdxs []uint16
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
func (b *BlockAccessList) toEncodingObj() *encodingBlockAccessList {
	var res encodingBlockAccessList
	var addrs []common.Address
	for addr := range b.accounts {
		addrs = append(addrs, addr)
	}
	sort.Slice(addrs, func(i, j int) bool {
		return bytes.Compare(addrs[i][:], addrs[j][:]) < 0
	})

	for _, addr := range addrs {
		res.Accesses = append(res.Accesses, b.accounts[addr].toEncodingObj(addr))
	}
	return &res
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
	TxIdx      uint16
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
	var prev *uint16

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
