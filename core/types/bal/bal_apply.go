package bal

import (
	"fmt"
	"slices"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/holiman/uint256"
)

// DecodeApplyRLP decodes only the consequences consumed by snap/2's
// applyAccessList path. It preserves account addresses, storage slots and their
// final values, and the final balance/nonce/code values. Storage reads,
// intermediate changes, and change indices that do not affect the final state
// are not materialized.
//
// The input is still the canonical EIP-7928 BAL RLP. This decoder is not a
// replacement for consensus validation or BAL hash verification.
func DecodeApplyRLP(input []byte) (*BlockAccessList, error) {
	root, rest, err := rlp.SplitList(input)
	if err != nil {
		return nil, err
	}
	if len(rest) != 0 {
		return nil, fmt.Errorf("BAL has trailing bytes")
	}
	var out BlockAccessList
	for len(root) > 0 {
		accountRaw, next, err := rlp.SplitList(root)
		if err != nil {
			return nil, err
		}
		root = next

		addr, accountRaw, err := rlp.SplitString(accountRaw)
		if err != nil {
			return nil, err
		}
		if len(addr) != common.AddressLength {
			return nil, fmt.Errorf("invalid BAL address length %d", len(addr))
		}
		entry := AccountAccess{Address: common.BytesToAddress(addr)}

		storageRaw, accountRaw, err := rlp.SplitList(accountRaw)
		if err != nil {
			return nil, err
		}
		for len(storageRaw) > 0 {
			slotRaw, next, err := rlp.SplitList(storageRaw)
			if err != nil {
				return nil, err
			}
			storageRaw = next
			slot, slotRaw, err := rlp.SplitString(slotRaw)
			if err != nil {
				return nil, err
			}
			changes, slotRaw, err := rlp.SplitList(slotRaw)
			if err != nil {
				return nil, err
			}
			if len(slotRaw) != 0 {
				return nil, fmt.Errorf("storage slot has trailing fields")
			}
			idx, value, ok, err := finalIndexedValue(changes)
			if err != nil {
				return nil, err
			}
			if !ok {
				return nil, fmt.Errorf("storage slot has no changes")
			}
			if idx > uint64(^uint32(0)) {
				return nil, fmt.Errorf("BAL index overflows uint32")
			}
			entry.StorageChanges = append(entry.StorageChanges, encodingSlotChanges{
				Slot: new(uint256.Int).SetBytes(slot),
				SlotChanges: []encodingStorageWrite{{
					BlockAccessIndex: uint32(idx),
					PostValue:        new(uint256.Int).SetBytes(value),
				}},
			})
		}

		// Storage reads are required for BAL validity and dependency analysis,
		// but applyAccessList does not consume them. Validation has already
		// happened before catch-up applies this view.
		if _, accountRaw, err = rlp.SplitList(accountRaw); err != nil {
			return nil, err
		}

		balances, accountRaw, err := rlp.SplitList(accountRaw)
		if err != nil {
			return nil, err
		}
		if idx, value, ok, err := finalIndexedValue(balances); err != nil {
			return nil, err
		} else if ok {
			if idx > uint64(^uint32(0)) {
				return nil, fmt.Errorf("BAL index overflows uint32")
			}
			entry.BalanceChanges = []encodingBalanceChange{{
				BlockAccessIndex: uint32(idx),
				PostBalance:      new(uint256.Int).SetBytes(value),
			}}
		}

		nonces, accountRaw, err := rlp.SplitList(accountRaw)
		if err != nil {
			return nil, err
		}
		if idx, value, ok, err := finalIndexedValue(nonces); err != nil {
			return nil, err
		} else if ok {
			if idx > uint64(^uint32(0)) {
				return nil, fmt.Errorf("BAL index overflows uint32")
			}
			nonce, err := bytesUint64(value)
			if err != nil {
				return nil, err
			}
			entry.NonceChanges = []encodingAccountNonce{{
				BlockAccessIndex: uint32(idx),
				PostNonce:        nonce,
			}}
		}

		codes, accountRaw, err := rlp.SplitList(accountRaw)
		if err != nil {
			return nil, err
		}
		if idx, value, ok, err := finalIndexedValue(codes); err != nil {
			return nil, err
		} else if ok {
			if idx > uint64(^uint32(0)) {
				return nil, fmt.Errorf("BAL index overflows uint32")
			}
			entry.CodeChanges = []encodingCodeChange{{
				BlockAccessIndex: uint32(idx),
				NewCode:          slices.Clone(value),
			}}
		}
		if len(accountRaw) != 0 {
			return nil, fmt.Errorf("BAL account has trailing fields")
		}
		out = append(out, entry)
	}
	return &out, nil
}

func finalIndexedValue(list []byte) (uint64, []byte, bool, error) {
	var (
		lastIndex uint64
		lastValue []byte
		ok        bool
	)
	for len(list) > 0 {
		pair, rest, err := rlp.SplitList(list)
		if err != nil {
			return 0, nil, false, err
		}
		list = rest
		index, pair, err := rlp.SplitUint64(pair)
		if err != nil {
			return 0, nil, false, err
		}
		value, pair, err := rlp.SplitString(pair)
		if err != nil {
			return 0, nil, false, err
		}
		if len(pair) != 0 {
			return 0, nil, false, fmt.Errorf("indexed change has trailing fields")
		}
		lastIndex, lastValue, ok = index, value, true
	}
	return lastIndex, lastValue, ok, nil
}

func bytesUint64(value []byte) (uint64, error) {
	if len(value) > 8 {
		return 0, fmt.Errorf("uint64 overflow")
	}
	var n uint64
	for _, b := range value {
		n = n<<8 | uint64(b)
	}
	return n, nil
}
