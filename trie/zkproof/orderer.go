package zkproof

import (
	"math/big"
	"sort"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/common/hexutil"
	"github.com/scroll-tech/go-ethereum/core/types"
)

type opIterator interface {
	next() *types.AccountWrapper
}

type opOrderer interface {
	readonly(bool)
	absorb(*types.AccountWrapper)
	absorbStorage(*types.AccountWrapper, *types.StorageWrapper)
	end_absorb() opIterator
}

type iterateOp []*types.AccountWrapper

func (ops *iterateOp) next() *types.AccountWrapper {

	sl := *ops

	if len(sl) == 0 {
		return nil
	}

	*ops = sl[1:]
	return sl[0]
}

type simpleOrderer struct {
	readOnly int
	savedOp  []*types.AccountWrapper
}

func (od *simpleOrderer) SavedOp() []*types.AccountWrapper { return od.savedOp }

func (od *simpleOrderer) readonly(mode bool) {
	if mode {
		od.readOnly += 1
	} else if od.readOnly == 0 {
		panic("unexpected readonly mode stack pop")
	} else {
		od.readOnly -= 1
	}
}

func (od *simpleOrderer) absorb(st *types.AccountWrapper) {
	if od.readOnly > 0 {
		return
	}
	od.savedOp = append(od.savedOp, st)
}

func (od *simpleOrderer) absorbStorage(st *types.AccountWrapper, _ *types.StorageWrapper) {
	od.absorb(st)
}

func (od *simpleOrderer) end_absorb() opIterator {
	ret := iterateOp(od.savedOp)
	return &ret
}

type multiOpIterator []opIterator

func (opss *multiOpIterator) next() *types.AccountWrapper {

	sl := *opss
	if len(sl) == 0 {
		return nil
	}

	op := sl[0].next()

	for op == nil {

		sl = sl[1:]
		*opss = sl
		if len(sl) == 0 {
			return nil
		}
		op = sl[0].next()
	}
	return op
}

type rwTblOrderer struct {
	readOnly         int
	readOnlySnapshot struct {
		accounts map[string]*types.AccountWrapper
		storages map[string]map[string]*types.StorageWrapper
	}
	initedData map[common.Address]*types.AccountWrapper

	// help to track all accounts being touched, and provide the
	// completed account status for storage updating
	traced map[string]*types.AccountWrapper

	opAccNonce    map[string]*types.AccountWrapper
	opAccBalance  map[string]*types.AccountWrapper
	opAccCodeHash map[string]*types.AccountWrapper
	opStorage     map[string]map[string]*types.StorageWrapper
}

func NewSimpleOrderer() *simpleOrderer { return &simpleOrderer{} }

func NewRWTblOrderer(inited map[common.Address]*types.StateAccount) *rwTblOrderer {

	initedAcc := make(map[common.Address]*types.AccountWrapper)
	for addr, data := range inited {
		if data == nil {
			initedAcc[addr] = &types.AccountWrapper{
				Address: addr,
				Balance: (*hexutil.Big)(big.NewInt(0)),
			}
		} else {
			bl := data.Balance
			if bl == nil {
				bl = big.NewInt(0)
			}

			initedAcc[addr] = &types.AccountWrapper{
				Address:          addr,
				Nonce:            data.Nonce,
				Balance:          (*hexutil.Big)(bl),
				KeccakCodeHash:   common.BytesToHash(data.KeccakCodeHash),
				PoseidonCodeHash: common.BytesToHash(data.PoseidonCodeHash),
				CodeSize:         data.CodeSize,
			}
		}

	}

	return &rwTblOrderer{
		initedData:    initedAcc,
		traced:        make(map[string]*types.AccountWrapper),
		opAccNonce:    make(map[string]*types.AccountWrapper),
		opAccBalance:  make(map[string]*types.AccountWrapper),
		opAccCodeHash: make(map[string]*types.AccountWrapper),
		opStorage:     make(map[string]map[string]*types.StorageWrapper),
	}
}

func (od *rwTblOrderer) readonly(mode bool) {
	if mode {
		if od.readOnly == 0 {
			od.readOnlySnapshot.accounts = make(map[string]*types.AccountWrapper)
			od.readOnlySnapshot.storages = make(map[string]map[string]*types.StorageWrapper)
		}
		od.readOnly += 1
	} else if od.readOnly == 0 {
		panic("unexpected readonly mode stack pop")
	} else {
		od.readOnly -= 1
		if od.readOnly == 0 {
			for addrS, st := range od.readOnlySnapshot.accounts {
				od.absorb(st)
				if m, existed := od.readOnlySnapshot.storages[addrS]; existed {
					for _, stg := range m {
						st.Storage = stg
						od.absorbStorage(st, nil)
					}
				}
			}
		}
	}
}

func (od *rwTblOrderer) absorbStorage(st *types.AccountWrapper, before *types.StorageWrapper) {
	if st.Storage == nil {
		panic("do not call absorbStorage ")
	}

	od.absorb(st)
	addrStr := st.Address.String()

	if stg := st.Storage; stg != nil {
		m, existed := od.opStorage[addrStr]
		if !existed {
			m = make(map[string]*types.StorageWrapper)
			od.opStorage[addrStr] = m
		}

		// key must be unified into 32 bytes
		keyBytes := hexutil.MustDecode(stg.Key)
		keyStr := common.BytesToHash(keyBytes).String()

		// trace every "touched" status for readOnly
		if od.readOnly > 0 {
			m, existed := od.readOnlySnapshot.storages[addrStr]
			if !existed {
				m = make(map[string]*types.StorageWrapper)
				od.readOnlySnapshot.storages[addrStr] = m
			}
			if _, hashTraced := m[keyStr]; !hashTraced {
				if before != nil {
					m[keyStr] = before
				} else {
					m[keyStr] = stg
				}

			}
		}

		m[keyStr] = stg
	}

}

func (od *rwTblOrderer) absorb(st *types.AccountWrapper) {

	initedRef, existed := od.initedData[st.Address]
	if !existed {
		panic("encounter unprepared status")
	}

	addrStr := st.Address.String()

	// trace every "touched" status for readOnly
	if od.readOnly > 0 {
		snapShot, existed := od.traced[addrStr]
		if !existed {
			snapShot = initedRef
		}

		if _, hasTraced := od.readOnlySnapshot.accounts[addrStr]; !hasTraced {
			od.readOnlySnapshot.accounts[addrStr] = copyAccountState(snapShot)
		}
	}

	if isDeletedAccount(st) {
		// for account delete, made a safer data for status
		st = &types.AccountWrapper{
			Address: st.Address,
			Balance: (*hexutil.Big)(big.NewInt(0)),
		}
	}

	od.traced[addrStr] = st

	// notice there would be at least one entry for all 3 fields when accessing an address
	// this may caused extract "read" op in mpt circuit which has no corresponding one in rwtable
	// we can avoid it unless obtaining more tips from the understanding of opcode
	// but it would be ok if we have adopted the new lookup way (root_prev, root_cur) under discussion:
	// https://github.com/privacy-scaling-explorations/zkevm-specs/issues/217

	if traced, existed := od.opAccNonce[addrStr]; !existed {
		traced = copyAccountState(st)
		traced.Balance = initedRef.Balance
		traced.KeccakCodeHash = initedRef.KeccakCodeHash
		traced.PoseidonCodeHash = initedRef.PoseidonCodeHash
		traced.CodeSize = initedRef.CodeSize
		traced.Storage = nil
		od.opAccNonce[addrStr] = traced
	} else {
		traced.Nonce = st.Nonce
	}

	if traced, existed := od.opAccBalance[addrStr]; !existed {
		traced = copyAccountState(st)
		traced.KeccakCodeHash = initedRef.KeccakCodeHash
		traced.PoseidonCodeHash = initedRef.PoseidonCodeHash
		traced.CodeSize = initedRef.CodeSize
		traced.Storage = nil
		od.opAccBalance[addrStr] = traced
	} else {
		traced.Nonce = st.Nonce
		traced.Balance = st.Balance
	}

	if traced, existed := od.opAccCodeHash[addrStr]; !existed {
		traced = copyAccountState(st)
		traced.Storage = nil
		od.opAccCodeHash[addrStr] = traced
	} else {
		traced.Nonce = st.Nonce
		traced.Balance = st.Balance
		traced.KeccakCodeHash = st.KeccakCodeHash
		traced.PoseidonCodeHash = st.PoseidonCodeHash
		traced.CodeSize = st.CodeSize
	}

}

func (od *rwTblOrderer) end_absorb() opIterator {
	// now sort every map by address / key
	// inited has collected all address, just sort address once
	sortedAddrs := make([]string, 0, len(od.traced))
	for addrs := range od.traced {
		sortedAddrs = append(sortedAddrs, addrs)
	}
	sort.Strings(sortedAddrs)

	var iterNonce []*types.AccountWrapper
	var iterBalance []*types.AccountWrapper
	var iterCodeHash []*types.AccountWrapper
	var iterStorage []*types.AccountWrapper

	for _, addrStr := range sortedAddrs {

		if v, existed := od.opAccNonce[addrStr]; existed {
			iterNonce = append(iterNonce, v)
		}

		if v, existed := od.opAccBalance[addrStr]; existed {
			iterBalance = append(iterBalance, v)
		}

		if v, existed := od.opAccCodeHash[addrStr]; existed {
			iterCodeHash = append(iterCodeHash, v)
		}

		if stgM, existed := od.opStorage[addrStr]; existed {

			tracedStatus := od.traced[addrStr]
			if tracedStatus == nil {
				panic("missed traced status found in storage slot")
			}

			sortedKeys := make([]string, 0, len(stgM))
			for key := range stgM {
				sortedKeys = append(sortedKeys, key)
			}
			sort.Strings(sortedKeys)

			for _, key := range sortedKeys {
				st := copyAccountState(tracedStatus)
				st.Storage = stgM[key]
				iterStorage = append(iterStorage, st)
			}
		}

	}

	var finalRet []opIterator
	for _, arr := range [][]*types.AccountWrapper{iterNonce, iterBalance, iterCodeHash, iterStorage} {
		wrappedIter := iterateOp(arr)
		finalRet = append(finalRet, &wrappedIter)
	}

	wrappedRet := multiOpIterator(finalRet)
	return &wrappedRet
}
