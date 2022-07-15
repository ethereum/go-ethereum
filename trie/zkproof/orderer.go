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
	absorb(*types.AccountWrapper)
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
	savedOp []*types.AccountWrapper
}

func (od *simpleOrderer) absorb(st *types.AccountWrapper) {
	od.savedOp = append(od.savedOp, st)
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
	traced map[string]*types.AccountWrapper

	opAccNonce    map[string]*types.AccountWrapper
	opAccBalance  map[string]*types.AccountWrapper
	opAccCodeHash map[string]*types.AccountWrapper
	opStorage     map[string]map[string]*types.StorageWrapper
}

func newRWTblOrderer(inited map[common.Address]*types.StateAccount) *rwTblOrderer {

	traced := make(map[string]*types.AccountWrapper)

	for addr, data := range inited {
		if data == nil {
			continue
		}

		bl := data.Balance
		if bl == nil {
			bl = big.NewInt(0)
		}

		traced[addr.String()] = &types.AccountWrapper{
			Address:  addr,
			Nonce:    data.Nonce,
			Balance:  (*hexutil.Big)(bl),
			CodeHash: common.BytesToHash(data.CodeHash),
		}
	}

	return &rwTblOrderer{
		traced:        traced,
		opAccNonce:    make(map[string]*types.AccountWrapper),
		opAccBalance:  make(map[string]*types.AccountWrapper),
		opAccCodeHash: make(map[string]*types.AccountWrapper),
		opStorage:     make(map[string]map[string]*types.StorageWrapper),
	}
}

func (od *rwTblOrderer) absorb(st *types.AccountWrapper) {

	addrStr := st.Address.String()

	start, existed := od.traced[addrStr]
	if !existed {
		start = &types.AccountWrapper{
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
		traced.Balance = start.Balance
		traced.CodeHash = start.CodeHash
		traced.Storage = nil
		od.opAccNonce[addrStr] = traced
	} else {
		traced.Nonce = st.Nonce
	}

	if traced, existed := od.opAccBalance[addrStr]; !existed {
		traced = copyAccountState(st)
		traced.CodeHash = start.CodeHash
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
		traced.CodeHash = st.CodeHash
	}

	if stg := st.Storage; stg != nil {
		m, existed := od.opStorage[addrStr]
		if !existed {
			m = make(map[string]*types.StorageWrapper)
			od.opStorage[addrStr] = m
		}

		// key must be unified into 32 bytes
		keyBytes := hexutil.MustDecode(stg.Key)
		m[common.BytesToHash(keyBytes).String()] = stg
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
