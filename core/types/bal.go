package types

import (
	"bytes"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/holiman/uint256"
	"io"
	"maps"
	"sort"
	"strings"
)

//go:generate go run github.com/ferranbt/fastssz/sszgen --path . --objs encodingPerTxAccess,encodingSlotAccess,encodingAccountAccess,encodingAccountAccessList,encodingBlockAccessList,encodingBalanceDelta,encodingBalanceChange,encodingAccountBalanceDiff,encodingCodeChange,encodingAccountNonce,encodingNonceDiffs,encodingAccountNonce,encodingBlockAccessList --output bal_encoding_generated.go

// encoder types

type encodingPerTxAccess struct {
	TxIdx      uint64   `ssz-size:"2"`
	ValueAfter [32]byte `ssz-size:"32"`
}

type encodingSlotAccess struct {
	Slot     [32]byte              `ssz-size:"32"`
	Accesses []encodingPerTxAccess `ssz-max:"300000"`
}

type encodingAccountAccess struct {
	Address  [20]byte             `ssz-size:"20"`
	Accesses []encodingSlotAccess `ssz-max:"300000"`
	Code     []byte               `ssz-max:"24576"`
}

type encodingAccountAccessList []encodingAccountAccess

// TODO: verify that Geth encodes the endianess according to the spec
type encodingBalanceDelta [12]byte

type encodingBalanceChange struct {
	TxIdx uint64 `ssz-size:"2"`
	Delta encodingBalanceDelta
}

type encodingAccountBalanceDiff struct {
	Address [20]byte
	Changes []encodingBalanceChange `ssz-max:"300000"`
}

// TODO: implement encoder/decoder manually on this, as we can't specify tags for a type declaration
// TODO what's the difference between `type encodingBalanceDiffs = []encodingAccountBalanceDiff` and the below?
type encodingBalanceDiffs []encodingAccountBalanceDiff

type encodingAccountCodeDiff struct {
	Address [20]byte
	TxIdx   uint64 `ssz-size:"2"`
	NewCode []byte `ssz-max:"24576"`
}

// TODO: implement encoder/decoder manually on this, as we can't specify tags for a type declaration
type encodingCodeDiffs []encodingAccountCodeDiff

type encodingAccountNonce struct {
	TxIdx uint64 `ssz-size:"2"`
	Nonce uint64 `ssz-size:"8"`
}

type encodingAccountNonces struct {
	Address [20]byte
	Diffs   []encodingAccountNonce `ssz-max:"300000"`
}

// TODO: implement encoder/decoder manually on this, as we can't specify tags for a type declaration
type encodingNonceDiffs []encodingAccountNonces

type encodingBlockAccessList struct {
	AccountAccesses encodingAccountAccessList `ssz-max:"300000"`
	BalanceDiffs    encodingBalanceDiffs      `ssz-max:"300000"`
	CodeDiffs       encodingCodeDiffs         `ssz-max:"300000"`
	NonceDiffs      encodingNonceDiffs        `ssz-max:"30000"`
}

// toMap returns a copy of the code diffs in the working format
func (c encodingCodeDiffs) toMap() (map[common.Address]accountCodeDiff, error) {
	var prevAddr *common.Address
	res := make(map[common.Address]accountCodeDiff)
	for _, diff := range c {
		if prevAddr != nil {
			if bytes.Compare(diff.Address[:], (*prevAddr)[:]) <= 0 {
				return nil, fmt.Errorf("code diffs not in lexicographic order")
			}
		}
		res[diff.Address] = accountCodeDiff{
			diff.TxIdx,
			bytes.Clone(diff.NewCode),
		}
		var p common.Address = diff.Address
		prevAddr = &p
	}
	return res, nil
}

// toMap returns a copy of an account's tx balance diffs in their working form.
func (c *encodingAccountBalanceDiff) toMap() (balanceDiff, error) {
	var prevIdx *uint64
	res := make(balanceDiff)
	for _, diff := range c.Changes {
		if prevIdx != nil {
			if *prevIdx >= diff.TxIdx {
				return nil, fmt.Errorf("not in lexicographic ordering")
			}
		}
		res[diff.TxIdx] = new(uint256.Int).SetBytes(diff.Delta[:])
	}
	return res, nil
}

// TODO: make this a function on the parameter tpye
func (c encodingBalanceDiffs) toMap() (map[common.Address]balanceDiff, error) {
	var prevAddr *common.Address
	res := make(map[common.Address]balanceDiff)
	for _, diff := range c {
		if prevAddr != nil {
			if bytes.Compare(diff.Address[:], (*prevAddr)[:]) <= 0 {
				return nil, fmt.Errorf("balance diffs not in lexicographic order")
			}
		}
		mp, err := diff.toMap()
		if err != nil {
			return nil, err
		}
		res[diff.Address] = mp
		var p common.Address = diff.Address
		prevAddr = &p
	}
	return res, nil
}

func (a *encodingSlotAccess) toSlotAccess() (*slotAccess, error) {
	var prevIdx *uint64
	res := slotAccess{make(map[uint64]common.Hash)}
	for _, diff := range a.Accesses {
		if prevIdx != nil {
			if *prevIdx >= diff.TxIdx {
				return nil, fmt.Errorf("not in lexicographic ordering")
			}
		}
		res.Writes[diff.TxIdx] = diff.ValueAfter
		prevIdx = &diff.TxIdx
	}
	return &res, nil
}

func (a *encodingAccountAccess) toAccountAccess() (*accountAccess, error) {
	res := accountAccess{
		a.Address,
		make(map[common.Hash]slotAccess),
		bytes.Clone(a.Code),
	}
	var prevSlot *[32]byte
	for _, diff := range a.Accesses {
		if prevSlot != nil {
			if bytes.Compare(diff.Slot[:], (*prevSlot)[:]) <= 0 {
				return nil, fmt.Errorf("storage slots not in lexicographic order")
			}
		}
		mp, err := diff.toSlotAccess()
		if err != nil {
			return nil, err
		}
		res.Accesses[diff.Slot] = *mp
		prevSlot = &diff.Slot
	}
	return &res, nil
}

func (al encodingAccountAccessList) toMap() (map[common.Address]*accountAccess, error) {
	var prevAddr *common.Address
	res := make(map[common.Address]*accountAccess)
	for _, diff := range al {
		if prevAddr != nil {
			if bytes.Compare(diff.Address[:], (*prevAddr)[:]) <= 0 {
				return nil, fmt.Errorf("accounts not in lexicographic order")
			}
		}
		mp, err := diff.toAccountAccess()
		if err != nil {
			return nil, err
		}
		res[diff.Address] = mp
		var p common.Address = diff.Address
		prevAddr = &p
	}
	return res, nil
}

// TODO: assert that there were no duplicate keys (tx indices / addresses) in any entries

func (n encodingAccountNonces) toMap() (accountNonceDiffs, error) {
	var prevIdx *uint64
	res := make(accountNonceDiffs)
	for _, diff := range n.Diffs {
		if prevIdx != nil {
			if *prevIdx >= diff.TxIdx {
				return nil, fmt.Errorf("not in lexicographic ordering")
			}
		}
		res[diff.TxIdx] = diff.Nonce
		prevIdx = &diff.TxIdx
	}
	return res, nil
}

func (n encodingNonceDiffs) toMap() (map[common.Address]accountNonceDiffs, error) {
	var prevAddr *common.Address
	res := make(map[common.Address]accountNonceDiffs)
	for _, diff := range n {
		if prevAddr != nil {
			if bytes.Compare(diff.Address[:], (*prevAddr)[:]) <= 0 {
				return nil, fmt.Errorf("nonce diff accounts not in lexicographic order")
			}
		}
		addrDiff, err := diff.toMap()
		if err != nil {
			return nil, err
		}
		res[diff.Address] = addrDiff
		var p common.Address = diff.Address
		prevAddr = &p
	}
	return res, nil
}

func (b *encodingBlockAccessList) ToBlockAccessList() (*BlockAccessList, error) {
	// TODO: ensure that there are no duplicate indexes where appropriate (for tx-idx and addressed keyed maps)
	accountAccesses, err := b.AccountAccesses.toMap()
	if err != nil {
		return nil, err
	}
	balanceChanges, err := b.BalanceDiffs.toMap()
	if err != nil {
		return nil, err
	}
	codeChanges, err := b.CodeDiffs.toMap()
	if err != nil {
		return nil, err
	}
	nonceDiffs, err := b.NonceDiffs.toMap()
	if err != nil {
		return nil, err
	}
	return &BlockAccessList{
		accountAccesses,
		balanceChanges,
		codeChanges,
		nonceDiffs,
		common.Hash{},
	}, nil
}

// non-encoder objects

func nonceDiffsToEncoderObj(nonceDiffs map[common.Address]accountNonceDiffs) (res encodingNonceDiffs) {
	var addrs []common.Address
	for addr, _ := range nonceDiffs {
		addrs = append(addrs, addr)
	}

	sort.Slice(addrs, func(i, j int) bool {
		return bytes.Compare(addrs[i][:], addrs[j][:]) < 0
	})

	for _, addr := range addrs {
		res = append(res, nonceDiffs[addr].toEncoderObj(addr))
	}
	return res
}

type slotAccess struct {
	Writes map[uint64]common.Hash `json:"writes"` // map of tx index to post-tx slot value
}

func (s slotAccess) toEncoderObj(key common.Hash) (res encodingSlotAccess) {
	var (
		slotIdxs []uint64
	)
	res.Slot = key
	for sIdx, _ := range s.Writes {
		slotIdxs = append(slotIdxs, sIdx)
	}
	sort.Slice(slotIdxs, func(i, j int) bool {
		return slotIdxs[i] < slotIdxs[j]
	})
	for _, slotIdx := range slotIdxs {
		res.Accesses = append(res.Accesses, encodingPerTxAccess{
			slotIdx,
			s.Writes[slotIdx],
		})
	}
	return
}

type accountAccess struct {
	Address  common.Address             `json:"address"`
	Accesses map[common.Hash]slotAccess `json:"accesses"` // map of slot key to all post-tx values where that slot was read/written
	Code     []byte                     `json:"code"`
}

func (a *accountAccess) Copy() *accountAccess {
	accesses := make(map[common.Hash]slotAccess)
	for key, access := range a.Accesses {
		accesses[key] = slotAccess{maps.Clone(access.Writes)}
	}

	return &accountAccess{
		a.Address,
		accesses,
		bytes.Clone(a.Code),
	}
}

func (a *accountAccess) MarkRead(key common.Hash) {
	if _, ok := a.Accesses[key]; !ok {
		a.Accesses[key] = slotAccess{
			make(map[uint64]common.Hash),
		}
	}
}

func (a *accountAccess) MarkWrite(txIdx uint64, key, value common.Hash) {
	if _, ok := a.Accesses[key]; !ok {
		a.Accesses[key] = slotAccess{
			make(map[uint64]common.Hash),
		}
	}

	a.Accesses[key].Writes[txIdx] = value
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

func (b balanceDiff) Copy() balanceDiff {
	res := make(map[uint64]*uint256.Int)
	for idx, balance := range b {
		res[idx] = balance.Clone()
	}
	return res
}

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

type accountCodeDiff struct {
	TxIdx uint64 `json:"txIdx"`
	Code  []byte `json:"Code"`
}

func (c *accountCodeDiff) Copy() accountCodeDiff {
	return accountCodeDiff{
		c.TxIdx,
		bytes.Clone(c.Code),
	}
}

// map of tx index to the prestate nonce
type accountNonceDiffs map[uint64]uint64

func (a accountNonceDiffs) toEncoderObj(addr common.Address) encodingAccountNonces {
	res := encodingAccountNonces{
		Address: addr,
	}
	var (
		diffIdxs []uint64
	)
	for sIdx, _ := range a {
		diffIdxs = append(diffIdxs, sIdx)
	}
	sort.Slice(diffIdxs, func(i, j int) bool {
		return diffIdxs[i] < diffIdxs[j]
	})

	for _, txIdx := range a {
		res.Diffs = append(res.Diffs, encodingAccountNonce{
			TxIdx: txIdx,
			Nonce: a[txIdx],
		})
	}
	return res
}

type accountDiffs map[common.Address]*accountAccess
type balanceDiffs map[common.Address]balanceDiff
type codeDiffs map[common.Address]accountCodeDiff
type nonceDiffs map[common.Address]accountNonceDiffs

type BlockAccessList struct {
	AccountDiffs accountDiffs `json:"accountDiffs"`
	BalanceDiffs balanceDiffs `json:"balanceDiffs"`
	CodeDiffs    codeDiffs    `json:"codeDiffs"`
	NonceDiffs   nonceDiffs   `json:"nonceDiffs"`
	hash         common.Hash  `json:"hash"`
}

// Copy deep-copies the access list
func (b *BlockAccessList) Copy() *BlockAccessList {
	accountAccesses := make(map[common.Address]*accountAccess)
	balanceChanges := make(map[common.Address]balanceDiff)
	codeChanges := make(map[common.Address]accountCodeDiff)

	for addr, aa := range b.AccountDiffs {
		accountAccesses[addr] = aa.Copy()
	}
	for addr, bd := range b.BalanceDiffs {
		balanceChanges[addr] = bd.Copy()
	}
	for addr, cd := range b.CodeDiffs {
		codeChanges[addr] = cd.Copy()
	}

	return &BlockAccessList{
		accountAccesses,
		balanceChanges,
		codeChanges,
		maps.Clone(b.NonceDiffs),
		b.hash,
	}
}

func (c codeDiffs) toEncoderObj() (res encodingCodeDiffs) {
	var codeChangeAddrs []common.Address

	for addr, _ := range c {
		codeChangeAddrs = append(codeChangeAddrs, addr)
	}
	sort.Slice(codeChangeAddrs, func(i, j int) bool {
		return bytes.Compare(codeChangeAddrs[i][:], codeChangeAddrs[j][:]) < 0
	})

	for _, addr := range codeChangeAddrs {
		res = append(res, encodingAccountCodeDiff{
			addr,
			c[addr].TxIdx,
			bytes.Clone(c[addr].Code),
		})
	}
	return
}

func NewBlockAccessList() *BlockAccessList {
	return &BlockAccessList{
		make(accountDiffs),
		make(balanceDiffs),
		make(codeDiffs),
		make(nonceDiffs),
		common.Hash{},
	}
}

func (b *BlockAccessList) Eq(other *BlockAccessList) bool {

	// check that the account accesses are equal (consider moving this into its own function)

	if len(b.AccountDiffs) != len(other.AccountDiffs) {
		return false
	}
	for address, aa := range b.AccountDiffs {
		otherAA, ok := other.AccountDiffs[address]
		if !ok {
			return false
		}
		if len(aa.Accesses) != len(otherAA.Accesses) {
			return false
		}
		for key, vals := range aa.Accesses {
			otherAccesses, ok := otherAA.Accesses[key]
			if !ok {
				return false
			}
			if len(vals.Writes) != len(otherAccesses.Writes) {
				return false
			}

			for i, writeVal := range vals.Writes {
				otherWriteVal, ok := otherAccesses.Writes[i]
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

	if len(b.CodeDiffs) != len(other.CodeDiffs) {
		return false
	}
	for addr, codeCh := range b.CodeDiffs {
		otherCodeCh, ok := other.CodeDiffs[addr]
		if !ok {
			return false
		}
		if bytes.Compare(codeCh.Code, otherCodeCh.Code) != 0 {
			return false
		}
		if codeCh.TxIdx != otherCodeCh.TxIdx {
			return false
		}
	}

	if len(b.NonceDiffs) != len(other.NonceDiffs) {
		return false
	}
	for addr, prestateNonces := range b.NonceDiffs {
		otherPrestateNonces, ok := other.NonceDiffs[addr]
		if !ok {
			return false
		}
		if !maps.Equal(prestateNonces, otherPrestateNonces) {
			return false
		}
	}

	if len(b.BalanceDiffs) != len(other.BalanceDiffs) {
		return false
	}

	for addr, balanceChanges := range b.BalanceDiffs {
		otherBalanceChanges, ok := other.BalanceDiffs[addr]
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

// NonceDiff records tx post-state nonce of any contract-like accounts whose nonce was incremented
func (b *BlockAccessList) NonceDiff(address common.Address, txIdx, originNonce uint64) {
	if _, ok := b.NonceDiffs[address]; ok {
		return
	}
	if _, ok := b.NonceDiffs[address]; !ok {
		b.NonceDiffs[address] = make(accountNonceDiffs)
	}
	b.NonceDiffs[address][txIdx] = originNonce
}

// BalanceChange records the transaction post-state balance of an account that changed its balance
// TODO: for the first transaction in the block, should this consider balances before any system contracts
// were executed?
// TODO: for the final transaction in the block, should this consider the balance change from block reward?
func (b *BlockAccessList) BalanceChange(txIdx uint64, address common.Address, balance *uint256.Int) {
	if _, ok := b.BalanceDiffs[address]; !ok {
		b.BalanceDiffs[address] = make(balanceDiff)
	}
	b.BalanceDiffs[address][txIdx] = balance.Clone()
}

// TODO for eip:  specify that storage slots which are read/modified for accounts that are created/selfdestructed
// in same transaction aren't included in teh BAL (?)

// TODO for eip:  specify that storage slots of newly-created accounts which are only read are not included in the BAL (?)

// called during tx execution every time a storage slot is read
func (b *BlockAccessList) StorageRead(address common.Address, key common.Hash) {
	if _, ok := b.AccountDiffs[address]; !ok {
		b.AccountDiffs[address] = &accountAccess{
			address,
			make(map[common.Hash]slotAccess),
			nil,
		}
	}
	b.AccountDiffs[address].MarkRead(key)
}

// called every time a mutated storage value is committed upon transaction finalization
func (b *BlockAccessList) StorageWrite(txIdx uint64, address common.Address, key, value common.Hash) {
	if _, ok := b.AccountDiffs[address]; !ok {
		b.AccountDiffs[address] = &accountAccess{
			address,
			make(map[common.Hash]slotAccess),
			nil,
		}
	}
	b.AccountDiffs[address].MarkWrite(txIdx, key, value)
}

// TODO: is there a way to bump the EOA nonce more than 1 in a transaction?
// ^ delegated EOA can execute code which calls CREATE multiple times

// TODO: include these in the PR to the EIP
// arguments for post-transaction nonces, which include nonces from tx senders:
// * we can parallelize block execution and state root computation:
//     - pre state + post-diffs gives us everything we need to update the tree
// * delegated EOAs can call code that does multiple creations, bumping the delegated acct nonce by more than 1 per tx
// * simpler implementation current spec: just accumulate modified nonces at transaction finalisation.

// called during tx finalisation for each dirty account with mutated code
func (b *BlockAccessList) CodeChange(txIdx uint64, address common.Address, code []byte) {
	if _, ok := b.CodeDiffs[address]; !ok {
		b.CodeDiffs[address] = accountCodeDiff{}
	}
	b.CodeDiffs[address] = accountCodeDiff{
		txIdx,
		bytes.Clone(code),
	}
}

func (b *BlockAccessList) toEncoderObj() *encodingBlockAccessList {
	var (
		accountAccessesAddrs   []common.Address
		encoderAccountAccesses encodingAccountAccessList

		balanceDiffsAddrs   []common.Address
		encoderBalanceDiffs encodingBalanceDiffs
	)

	for addr, _ := range b.AccountDiffs {
		accountAccessesAddrs = append(accountAccessesAddrs, addr)
	}
	sort.Slice(accountAccessesAddrs, func(i, j int) bool {
		return bytes.Compare(accountAccessesAddrs[i][:], accountAccessesAddrs[j][:]) < 0
	})
	for _, addr := range accountAccessesAddrs {
		encoderAccountAccesses = append(encoderAccountAccesses, encodingAccountAccess{
			Address:  addr,
			Accesses: nil,
			Code:     b.AccountDiffs[addr].Code,
		})
		// sort the accesses lexicographically by key, and the occurance of each key ascending by tx idx
		// then encode them
		var storageAccessKeys []common.Hash
		for key, _ := range b.AccountDiffs[addr].Accesses {
			storageAccessKeys = append(storageAccessKeys, key)
		}
		sort.Slice(storageAccessKeys, func(i, j int) bool {
			return bytes.Compare(storageAccessKeys[i][:], storageAccessKeys[j][:]) < 0
		})
		var accesses []encodingSlotAccess
		for _, accessSlot := range storageAccessKeys {
			accesses = append(accesses, b.AccountDiffs[addr].Accesses[accessSlot].toEncoderObj(accessSlot))
		}
		encoderAccountAccesses = append(encoderAccountAccesses, encodingAccountAccess{
			Address:  addr,
			Accesses: accesses,
			Code:     b.AccountDiffs[addr].Code,
		})
	}

	// encode balance diffs
	for addr, _ := range b.BalanceDiffs {
		balanceDiffsAddrs = append(balanceDiffsAddrs, addr)
	}
	sort.Slice(balanceDiffsAddrs, func(i, j int) bool {
		return bytes.Compare(balanceDiffsAddrs[i][:], balanceDiffsAddrs[j][:]) < 0
	})

	for _, addr := range balanceDiffsAddrs {
		encoderBalanceDiffs = append(encoderBalanceDiffs, b.BalanceDiffs[addr].toEncoderObj(addr))
	}

	encoderObj := encodingBlockAccessList{
		AccountAccesses: encoderAccountAccesses,
		BalanceDiffs:    encoderBalanceDiffs,
		CodeDiffs:       b.CodeDiffs.toEncoderObj(),
		NonceDiffs:      nonceDiffsToEncoderObj(b.NonceDiffs),
	}
	return &encoderObj
}

func (b *BlockAccessList) encodeSSZ() ([]byte, error) {
	encoderObj := b.toEncoderObj()
	dst, err := encoderObj.MarshalSSZTo(nil)
	if err != nil {
		return nil, err
	}
	return dst, nil
}

func (e *encodingBlockAccessList) PrettyPrint() string {
	var res bytes.Buffer
	printWithIndent := func(indent int, text string) {
		fmt.Fprintf(&res, "%s%s\n", strings.Repeat("    ", indent), text)
	}
	fmt.Fprintf(&res, "accounts:\n")
	for _, accountDiff := range e.AccountAccesses {
		printWithIndent(1, fmt.Sprintf("address: %x", accountDiff.Address))
		printWithIndent(1, fmt.Sprintf("code:    %x", accountDiff.Code)) // TODO: code shouldn't be in account accesses (?)

		printWithIndent(1, "slots:")
		for _, slot := range accountDiff.Accesses {
			printWithIndent(2, fmt.Sprintf("%x", slot))
			printWithIndent(2, "accesses:")
			for _, access := range slot.Accesses {
				printWithIndent(3, fmt.Sprintf("idx: %d", access.TxIdx))
				printWithIndent(3, fmt.Sprintf("post: %x", access.ValueAfter))
			}
		}
	}
	printWithIndent(0, "code:")
	for _, codeDiff := range e.CodeDiffs {
		printWithIndent(1, fmt.Sprintf("address: %x", codeDiff.Address))
		printWithIndent(1, fmt.Sprintf("index:   %x", codeDiff.TxIdx))
		printWithIndent(1, fmt.Sprintf("code:    %x", codeDiff.NewCode))
	}
	printWithIndent(0, "balances:")
	for _, b := range e.BalanceDiffs {
		printWithIndent(1, fmt.Sprintf("%x:", b.Address))
		for _, change := range b.Changes {
			printWithIndent(2, fmt.Sprintf("index: %d", change.TxIdx))
			printWithIndent(2, fmt.Sprintf("balance: %s", new(uint256.Int).SetBytes(change.Delta[:]).String()))
		}
	}

	printWithIndent(0, "nonces:")
	for _, n := range e.NonceDiffs {
		printWithIndent(1, fmt.Sprintf("%x:", n.Address))
		for _, nonceDiff := range n.Diffs {
			printWithIndent(2, fmt.Sprintf("index: %d", nonceDiff.TxIdx))
			printWithIndent(2, fmt.Sprintf("nonce: %d", nonceDiff.Nonce))
		}
	}

	return res.String()
}

// human-readable representation
func (b *BlockAccessList) PrettyPrint() string {
	enc := b.toEncoderObj()
	return enc.PrettyPrint()
}

func (b *BlockAccessList) Hash() common.Hash {
	if b.hash == (common.Hash{}) {
		// TODO: cache the encoded bal
		encoded, err := b.encodeSSZ()
		if err != nil {
			panic(err)
		}
		b.hash = common.BytesToHash(crypto.Keccak256(encoded))
	}
	return b.hash
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
	res, err := enc.ToBlockAccessList()
	if err != nil {
		return err
	}
	*b = *res
	return nil
}

var _ rlp.Encoder = &BlockAccessList{}
var _ rlp.Decoder = &BlockAccessList{}
