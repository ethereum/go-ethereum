package state

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/stateless"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/types/bal"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie/utils"
	"github.com/holiman/uint256"
)

type AccessListCreationDB struct {
	idx        uint16
	inner      BlockProcessingDB
	accessList *bal.ConstructionBlockAccessList
}

func NewBlockAccessListBuilder(db BlockProcessingDB) *AccessListCreationDB {
	return &AccessListCreationDB{0, db, bal.NewConstructionBlockAccessList()}
}
func (a *AccessListCreationDB) SetAccessListIndex(idx int) {
	a.idx = uint16(idx)
}

// ConstructedBlockAccessList retrieves the access list that has been constructed
// by the StateDB instance, or nil if BAL construction was not enabled.
func (a *AccessListCreationDB) ConstructedBlockAccessList() *bal.ConstructionBlockAccessList {
	return a.accessList
}

func (a *AccessListCreationDB) CreateAccount(address common.Address) {
	a.inner.CreateAccount(address)
}

func (a *AccessListCreationDB) CreateContract(address common.Address) {
	a.inner.CreateContract(address)
}

func (a *AccessListCreationDB) SubBalance(address common.Address, u *uint256.Int, reason tracing.BalanceChangeReason) uint256.Int {
	return a.inner.SubBalance(address, u, reason)
}

func (a *AccessListCreationDB) AddBalance(address common.Address, u *uint256.Int, reason tracing.BalanceChangeReason) uint256.Int {
	return a.inner.AddBalance(address, u, reason)
}

func (a *AccessListCreationDB) GetBalance(address common.Address) *uint256.Int {
	return a.inner.GetBalance(address)
}

func (a *AccessListCreationDB) GetNonce(address common.Address) uint64 {
	return a.inner.GetNonce(address)
}

func (a *AccessListCreationDB) SetNonce(address common.Address, u uint64, reason tracing.NonceChangeReason) {
	a.inner.SetNonce(address, u, reason)
}

func (a *AccessListCreationDB) GetCodeHash(address common.Address) common.Hash {
	return a.inner.GetCodeHash(address)
}

func (a *AccessListCreationDB) GetCode(address common.Address) []byte {
	return a.inner.GetCode(address)
}

func (a *AccessListCreationDB) SetCode(addr common.Address, code []byte, reason tracing.CodeChangeReason) (prev []byte) {
	return a.inner.SetCode(addr, code, reason)
}

func (a *AccessListCreationDB) GetCodeSize(address common.Address) int {
	return a.inner.GetCodeSize(address)
}

func (a *AccessListCreationDB) AddRefund(u uint64) {
	a.inner.AddRefund(u)
}

func (a *AccessListCreationDB) SubRefund(u uint64) {
	a.inner.SubRefund(u)
}

func (a *AccessListCreationDB) GetRefund() uint64 {
	return a.inner.GetRefund()
}

func (a *AccessListCreationDB) GetStateAndCommittedState(address common.Address, hash common.Hash) (common.Hash, common.Hash) {
	return a.inner.GetStateAndCommittedState(address, hash)
}

func (a *AccessListCreationDB) GetState(address common.Address, hash common.Hash) common.Hash {
	return a.inner.GetState(address, hash)
}

func (a *AccessListCreationDB) SetState(address common.Address, hash common.Hash, hash2 common.Hash) common.Hash {
	return a.inner.SetState(address, hash, hash2)
}

func (a *AccessListCreationDB) GetStorageRoot(addr common.Address) common.Hash {
	return a.inner.GetStorageRoot(addr)
}

func (a *AccessListCreationDB) GetTransientState(addr common.Address, key common.Hash) common.Hash {
	return a.inner.GetTransientState(addr, key)
}

func (a *AccessListCreationDB) SetTransientState(addr common.Address, key, value common.Hash) {
	a.inner.SetTransientState(addr, key, value)
}

func (a *AccessListCreationDB) SelfDestruct(address common.Address) uint256.Int {
	return a.inner.SelfDestruct(address)
}

func (a *AccessListCreationDB) HasSelfDestructed(address common.Address) bool {
	return a.inner.HasSelfDestructed(address)
}

func (a *AccessListCreationDB) SelfDestruct6780(address common.Address) (uint256.Int, bool) {
	return a.inner.SelfDestruct6780(address)
}

func (a *AccessListCreationDB) Exist(address common.Address) bool {
	return a.inner.Exist(address)
}

func (a *AccessListCreationDB) Empty(address common.Address) bool {
	return a.inner.Empty(address)
}

func (a *AccessListCreationDB) AddressInAccessList(addr common.Address) bool {
	return a.inner.AddressInAccessList(addr)
}

func (a *AccessListCreationDB) SlotInAccessList(addr common.Address, slot common.Hash) (addressOk bool, slotOk bool) {
	return a.inner.SlotInAccessList(addr, slot)
}

func (a *AccessListCreationDB) AddAddressToAccessList(addr common.Address) {
	a.inner.AddAddressToAccessList(addr)
}

func (a *AccessListCreationDB) AddSlotToAccessList(addr common.Address, slot common.Hash) {
	a.inner.AddSlotToAccessList(addr, slot)
}

func (a *AccessListCreationDB) PointCache() *utils.PointCache {
	return a.inner.PointCache()
}

func (a *AccessListCreationDB) Prepare(rules params.Rules, sender, coinbase common.Address, dest *common.Address, precompiles []common.Address, txAccesses types.AccessList) {
	a.inner.Prepare(rules, sender, coinbase, dest, precompiles, txAccesses)
}

func (a *AccessListCreationDB) RevertToSnapshot(i int) {
	a.inner.RevertToSnapshot(i)
}

func (a *AccessListCreationDB) Snapshot() int {
	return a.inner.Snapshot()
}

func (a *AccessListCreationDB) AddLog(log *types.Log) {
	a.inner.AddLog(log)
}

func (a *AccessListCreationDB) AddPreimage(hash common.Hash, bytes []byte) {
	a.inner.AddPreimage(hash, bytes)
}

func (a *AccessListCreationDB) Witness() *stateless.Witness {
	return a.inner.Witness()
}

func (a *AccessListCreationDB) AccessEvents() *AccessEvents {
	return a.inner.AccessEvents()
}

func (a *AccessListCreationDB) TxIndex() int {
	return a.inner.TxIndex()
}

func (a *AccessListCreationDB) Finalise(b bool) (*bal.StateDiff, *bal.StateAccesses) {
	diff, accesses := a.inner.Finalise(b)
	a.accessList.ApplyDiff(uint(a.idx), diff)
	a.accessList.ApplyAccesses(*accesses) // TODO: can remove the pointer on accesses (map is already a reference type)
	return nil, nil                       // TODO: not sure what to do here.  The diff has been applied to the access list so it is "owned" by the access list, not sure why a caller would need it...
}

func (a *AccessListCreationDB) GetLogs(hash common.Hash, blockNumber uint64, blockHash common.Hash, blockTime uint64) []*types.Log {
	return a.inner.GetLogs(hash, blockNumber, blockHash, blockTime)
}

func (a *AccessListCreationDB) IntermediateRoot(deleteEmpty bool) common.Hash {
	return a.inner.IntermediateRoot(deleteEmpty)
}

func (a *AccessListCreationDB) Database() Database {
	return a.inner.Database()
}
func (a *AccessListCreationDB) GetTrie() Trie {
	return a.inner.GetTrie()
}
func (s *AccessListCreationDB) SetTxContext(thash common.Hash, ti int) {
	s.inner.SetTxContext(thash, ti)
}

func (s *AccessListCreationDB) Error() error {
	return s.inner.Error()
}

func (s *AccessListCreationDB) Copy() BlockProcessingDB {
	return &AccessListCreationDB{s.idx, s.inner.Copy(), s.accessList.Copy()}
}

var _ BlockProcessingDB = &AccessListCreationDB{}
