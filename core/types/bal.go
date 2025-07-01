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

type encodingStorageWrite struct {
	TxIdx      uint64   `ssz-size:"2"`
	ValueAfter [32]byte `ssz-size:"32"`
}

type encodingStorageRead struct {
	TxIdx uint64   `ssz-size:"2"`
	Key   [32]byte `ssz-size:"32"`
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
		res[encAccountAccess.Address] = *aa
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

type BlockAccessList map[common.Address]accountAccess

func (a *accountAccess) MarkRead(key common.Hash) {
}

func (a *accountAccess) MarkWrite(txIdx uint64, key, value common.Hash) {
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

	for txIdx, postNonce := range a {
		res.Diffs = append(res.Diffs, encodingAccountNonce{
			TxIdx: txIdx,
			Nonce: postNonce,
		})
	}
	return res
}

type storageWrites map[uint64]common.Hash

type accountAccess struct {
	storageWrites  map[common.Hash]storageWrites
	storageReads   map[common.Hash]struct{}
	balanceChanges balanceDiff
	nonceChanges   accountNonceDiffs
	codeChange     *[]byte
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
func (b *BlockAccessList) NonceDiff(address common.Address, txIdx, postNonce uint64) {
	if _, ok := b.NonceDiffs[address]; ok {
		return
	}
	if _, ok := b.NonceDiffs[address]; !ok {
		b.NonceDiffs[address] = make(accountNonceDiffs)
	}
	b.NonceDiffs[address][txIdx] = postNonce
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
		// sort the accesses lexicographically by key, and the occurance of each key ascending by tx idx
		// then encode them
		var storageAccessKeys []common.Hash
		for key, _ := range b.AccountDiffs[addr].Accesses {
			storageAccessKeys = append(storageAccessKeys, key)
		}
		sort.Slice(storageAccessKeys, func(i, j int) bool {
			return bytes.Compare(storageAccessKeys[i][:], storageAccessKeys[j][:]) < 0
		})
		var accesses []encodingStorageWrites
		for _, accessSlot := range storageAccessKeys {
			accesses = append(accesses, b.AccountDiffs[addr].Accesses[accessSlot].toEncoderObj(accessSlot))
		}
		encoderAccountAccesses = append(encoderAccountAccesses, encodingAccountAccess{
			Address:       addr,
			StorageWrites: accesses,
			Code:          b.AccountDiffs[addr].Code,
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
		for _, slot := range accountDiff.StorageWrites {
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
