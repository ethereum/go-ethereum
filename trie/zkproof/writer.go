package zkproof

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"

	zktrie "github.com/scroll-tech/zktrie/trie"
	zkt "github.com/scroll-tech/zktrie/types"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/common/hexutil"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/ethdb/memorydb"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/trie"
)

type proofList [][]byte

func (n *proofList) Put(key []byte, value []byte) error {
	*n = append(*n, value)
	return nil
}

func (n *proofList) Delete(key []byte) error {
	panic("not supported")
}

func addressToKey(addr common.Address) *zkt.Hash {
	var preImage zkt.Byte32
	copy(preImage[:], addr.Bytes())

	h, err := preImage.Hash()
	if err != nil {
		log.Error("hash failure", "preImage", hexutil.Encode(preImage[:]))
		return nil
	}
	return zkt.NewHashFromBigInt(h)
}

// resume the proof bytes into db and return the leaf node
func resumeProofs(proof []hexutil.Bytes, db *memorydb.Database) *zktrie.Node {
	for _, buf := range proof {

		n, err := zktrie.DecodeSMTProof(buf)
		if err != nil {
			log.Warn("decode proof string fail", "error", err)
		} else if n != nil {
			hash, err := n.NodeHash()
			if err != nil {
				log.Warn("node has no valid node hash", "error", err)
			} else {
				//notice: must consistent with trie/merkletree.go
				bt := hash[:]
				db.Put(bt, buf)
				if n.Type == zktrie.NodeTypeLeaf_New || n.Type == zktrie.NodeTypeEmpty_New {
					return n
				}
			}
		}

	}

	return nil
}

// we have a trick here which suppose the proof array include all middle nodes along the
// whole path in sequence, from root to leaf
func decodeProofForMPTPath(proof proofList, path *SMTPath) {

	var lastNode *zktrie.Node
	keyPath := big.NewInt(0)
	path.KeyPathPart = (*hexutil.Big)(keyPath)

	keyCounter := big.NewInt(1)

	for _, buf := range proof {
		n, err := zktrie.DecodeSMTProof(buf)
		if err != nil {
			log.Warn("decode proof string fail", "error", err)
		} else if n != nil {
			hash, err := n.NodeHash()
			if err != nil {
				log.Warn("node has no valid node hash", "error", err)
				return
			}
			if lastNode == nil {
				// notice: use little-endian represent inside Hash ([:] or Byte32())
				path.Root = hash[:]
			} else {
				if bytes.Equal(hash[:], lastNode.ChildL[:]) {
					path.Path = append(path.Path, SMTPathNode{
						Value:   hash[:],
						Sibling: lastNode.ChildR[:],
					})
				} else if bytes.Equal(hash[:], lastNode.ChildR[:]) {
					path.Path = append(path.Path, SMTPathNode{
						Value:   hash[:],
						Sibling: lastNode.ChildL[:],
					})
					keyPath.Add(keyPath, keyCounter)
				} else {
					panic("Unexpected proof form")
				}
				keyCounter.Mul(keyCounter, big.NewInt(2))
			}
			switch n.Type {
			case zktrie.NodeTypeBranch_0, zktrie.NodeTypeBranch_1, zktrie.NodeTypeBranch_2, zktrie.NodeTypeBranch_3:
				lastNode = n
			case zktrie.NodeTypeLeaf_New:
				vhash, _ := n.ValueHash()
				path.Leaf = &SMTPathNode{
					//here we just return the inner represent of hash (little endian, reversed byte order to common hash)
					Value:   vhash[:],
					Sibling: n.NodeKey[:],
				}
				//sanity check
				keyPart := keyPath.Bytes()
				for i, b := range keyPart {
					ri := len(keyPart) - i
					cb := path.Leaf.Sibling[ri-1] //notice the output is little-endian
					if b&cb != b {
						panic(fmt.Errorf("path key not match: part is %x but key is %x", keyPart, []byte(path.Leaf.Sibling[:])))
					}
				}

				return
			case zktrie.NodeTypeEmpty_New:
				return
			default:
				panic(fmt.Errorf("unknown node type %d", n.Type))
			}
		}
	}

	panic("Unexpected finished here")
}

type zktrieProofWriter struct {
	db                  *trie.ZktrieDatabase
	tracingZktrie       *trie.ZkTrie
	tracingStorageTries map[common.Address]*trie.ZkTrie
	tracingAccounts     map[common.Address]*types.StateAccount
}

func (wr *zktrieProofWriter) TracingAccounts() map[common.Address]*types.StateAccount {
	return wr.tracingAccounts
}

func NewZkTrieProofWriter(storage *types.StorageTrace) (*zktrieProofWriter, error) {

	underlayerDb := memorydb.New()
	zkDb := trie.NewZktrieDatabase(underlayerDb)

	accounts := make(map[common.Address]*types.StateAccount)

	// resuming proof bytes to underlayerDb
	for addrs, proof := range storage.Proofs {
		if n := resumeProofs(proof, underlayerDb); n != nil {
			addr := common.HexToAddress(addrs)
			if n.Type == zktrie.NodeTypeEmpty_New {
				accounts[addr] = nil
			} else if acc, err := types.UnmarshalStateAccount(n.Data()); err == nil {
				if bytes.Equal(n.NodeKey[:], addressToKey(addr)[:]) {
					accounts[addr] = acc
				} else {
					// should still mark the address as being trace (data not existed yet)
					accounts[addr] = nil
				}

			} else {
				return nil, fmt.Errorf("decode account bytes fail: %s, raw data [%x]", err, n.Data())
			}

		} else {
			return nil, fmt.Errorf("can not resume proof for address %s", addrs)
		}
	}

	storages := make(map[common.Address]*trie.ZkTrie)

	for addrs, stgLists := range storage.StorageProofs {

		addr := common.HexToAddress(addrs)
		accState, existed := accounts[addr]
		if !existed {
			// trace is malformed but currently we just warn about that
			log.Warn("no account state found for this addr, mal records", "address", addrs)
			continue
		} else if accState == nil {
			// create an empty zktrie for uninit address
			storages[addr], _ = trie.NewZkTrie(common.Hash{}, zkDb)
			continue
		}

		for keys, proof := range stgLists {

			if n := resumeProofs(proof, underlayerDb); n != nil {
				var err error
				storages[addr], err = trie.NewZkTrie(accState.Root, zkDb)
				if err != nil {
					return nil, fmt.Errorf("zktrie create failure for storage in addr <%s>: %s, (root %s)", addrs, err, accState.Root)
				}

			} else {
				return nil, fmt.Errorf("can not resume proof for storage %s@%s", keys, addrs)
			}

		}
	}

	for _, delProof := range storage.DeletionProofs {

		n, err := zktrie.DecodeSMTProof(delProof)
		if err != nil {
			log.Warn("decode delproof string fail", "error", err, "node", delProof)
		} else if n != nil {
			hash, err := n.NodeHash()
			if err != nil {
				log.Warn("node has no valid node hash", "error", err)
			} else {
				//notice: must consistent with trie/merkletree.go
				bt := hash[:]
				underlayerDb.Put(bt, delProof)
			}
		}
	}

	zktrie, err := trie.NewZkTrie(
		storage.RootBefore,
		trie.NewZktrieDatabase(underlayerDb),
	)
	if err != nil {
		return nil, fmt.Errorf("zktrie create failure: %s", err)
	}

	// sanity check
	if !bytes.Equal(zktrie.Hash().Bytes(), storage.RootBefore.Bytes()) {
		return nil, fmt.Errorf("unmatch init trie hash: expected %x but has %x", storage.RootBefore.Bytes(), zktrie.Hash().Bytes())
	}

	return &zktrieProofWriter{
		db:                  zkDb,
		tracingZktrie:       zktrie,
		tracingAccounts:     accounts,
		tracingStorageTries: storages,
	}, nil
}

const (
	posSSTOREBefore = 0
	posCREATE       = 0
	posCREATEAfter  = 1
	posCALL         = 2
	posSTATICCALL   = 0

	// posSELFDESTRUCT = 2
)

func getAccountState(l *types.StructLogRes, pos int) *types.AccountWrapper {
	if exData := l.ExtraData; exData == nil {
		return nil
	} else if len(exData.StateList) < pos {
		return nil
	} else {
		return exData.StateList[pos]
	}
}

func copyAccountState(st *types.AccountWrapper) *types.AccountWrapper {

	var stg *types.StorageWrapper
	if st.Storage != nil {
		stg = &types.StorageWrapper{
			Key:   st.Storage.Key,
			Value: st.Storage.Value,
		}
	}

	return &types.AccountWrapper{
		Nonce:            st.Nonce,
		Balance:          (*hexutil.Big)(big.NewInt(0).Set(st.Balance.ToInt())),
		KeccakCodeHash:   st.KeccakCodeHash,
		PoseidonCodeHash: st.PoseidonCodeHash,
		CodeSize:         st.CodeSize,
		Address:          st.Address,
		Storage:          stg,
	}
}

func isDeletedAccount(state *types.AccountWrapper) bool {
	return state.Nonce == 0 && bytes.Equal(state.KeccakCodeHash.Bytes(), common.Hash{}.Bytes())
}

func getAccountDataFromLogState(state *types.AccountWrapper) *types.StateAccount {

	if isDeletedAccount(state) {
		return nil
	}

	return &types.StateAccount{
		Nonce:            state.Nonce,
		Balance:          (*big.Int)(state.Balance),
		KeccakCodeHash:   state.KeccakCodeHash.Bytes(),
		PoseidonCodeHash: state.PoseidonCodeHash.Bytes(),
		CodeSize:         state.CodeSize,
		// Root omitted intentionally
	}
}

// for sanity check
func verifyAccount(addr common.Address, data *types.StateAccount, leaf *SMTPathNode) error {

	if leaf == nil {
		if data != nil {
			return fmt.Errorf("path has no corresponding leaf for account")
		} else {
			return nil
		}
	}

	addrKey := addressToKey(addr)
	if !bytes.Equal(addrKey[:], leaf.Sibling) {
		if data != nil {
			return fmt.Errorf("unmatch leaf node in address: %s", addr)
		}
	} else if data != nil {
		arr, flag := data.MarshalFields()
		h, err := zkt.HandlingElemsAndByte32(flag, arr)
		//log.Info("sanity check acc before", "addr", addr.String(), "key", leaf.Sibling.Text(16), "hash", h.Text(16))

		if err != nil {
			return fmt.Errorf("fail to hash account: %v", err)
		}
		if !bytes.Equal(h[:], leaf.Value) {
			return fmt.Errorf("unmatch data in leaf for address %s", addr)
		}
	}
	return nil
}

// for sanity check
func verifyStorage(key *zkt.Byte32, data *zkt.Byte32, leaf *SMTPathNode) error {

	emptyData := bytes.Equal(data[:], common.Hash{}.Bytes())

	if leaf == nil {
		if !emptyData {
			return fmt.Errorf("path has no corresponding leaf for storage")
		} else {
			return nil
		}
	}

	keyHash, err := key.Hash()
	if err != nil {
		return err
	}

	if !bytes.Equal(zkt.NewHashFromBigInt(keyHash)[:], leaf.Sibling) {
		if !emptyData {
			return fmt.Errorf("unmatch leaf node in storage: %x", key[:])
		}
	} else {
		h, err := data.Hash()
		//log.Info("sanity check acc before", "addr", addr.String(), "key", leaf.Sibling.Text(16), "hash", h.Text(16))

		if err != nil {
			return fmt.Errorf("fail to hash data: %v", err)
		}
		if !bytes.Equal(zkt.NewHashFromBigInt(h)[:], leaf.Value) {
			return fmt.Errorf("unmatch data in leaf for storage %x", key[:])
		}
	}
	return nil
}

// update traced account state, and return the corresponding trace object which
// is still opened for more infos
// the updated accData state is obtained by a closure which enable it being derived from current status
func (w *zktrieProofWriter) traceAccountUpdate(addr common.Address, updateAccData func(*types.StateAccount) *types.StateAccount) (*StorageTrace, error) {

	out := new(StorageTrace)
	//account trie
	out.Address = addr.Bytes()
	out.AccountPath = [2]*SMTPath{{}, {}}
	//fill dummy
	out.AccountUpdate = [2]*StateAccount{}

	accDataBefore, existed := w.tracingAccounts[addr]
	if !existed {
		//sanity check
		panic(fmt.Errorf("code do not add initialized status for account %s", addr))
	}

	var proof proofList
	s_key, _ := zkt.ToSecureKeyBytes(addr.Bytes())
	if err := w.tracingZktrie.Prove(s_key.Bytes(), 0, &proof); err != nil {
		return nil, fmt.Errorf("prove BEFORE state fail: %s", err)
	}

	decodeProofForMPTPath(proof, out.AccountPath[0])
	if err := verifyAccount(addr, accDataBefore, out.AccountPath[0].Leaf); err != nil {
		panic(fmt.Errorf("code fail to trace account status correctly: %s", err))
	}
	if accDataBefore != nil {
		// we have ensured the nBefore has a key corresponding to the query one
		out.AccountKey = out.AccountPath[0].Leaf.Sibling
		out.AccountUpdate[0] = &StateAccount{
			Nonce:            int(accDataBefore.Nonce),
			Balance:          (*hexutil.Big)(big.NewInt(0).Set(accDataBefore.Balance)),
			KeccakCodeHash:   accDataBefore.KeccakCodeHash,
			PoseidonCodeHash: accDataBefore.PoseidonCodeHash,
			CodeSize:         accDataBefore.CodeSize,
		}
	}

	accData := updateAccData(accDataBefore)
	if accData != nil {
		out.AccountUpdate[1] = &StateAccount{
			Nonce:            int(accData.Nonce),
			Balance:          (*hexutil.Big)(big.NewInt(0).Set(accData.Balance)),
			KeccakCodeHash:   accData.KeccakCodeHash,
			PoseidonCodeHash: accData.PoseidonCodeHash,
			CodeSize:         accData.CodeSize,
		}
	}

	if accData != nil {
		if err := w.tracingZktrie.TryUpdateAccount(addr.Bytes32(), accData); err != nil {
			return nil, fmt.Errorf("update zktrie account state fail: %s", err)
		}
		w.tracingAccounts[addr] = accData
	} else if accDataBefore != nil {
		if err := w.tracingZktrie.TryDelete(addr.Bytes32()); err != nil {
			return nil, fmt.Errorf("delete zktrie account state fail: %s", err)
		}
		w.tracingAccounts[addr] = nil
	} // notice if both before/after is nil, we do not touch zktrie

	proof = proofList{}
	if err := w.tracingZktrie.Prove(s_key.Bytes(), 0, &proof); err != nil {
		return nil, fmt.Errorf("prove AFTER state fail: %s", err)
	}

	decodeProofForMPTPath(proof, out.AccountPath[1])
	if err := verifyAccount(addr, accData, out.AccountPath[1].Leaf); err != nil {
		panic(fmt.Errorf("state AFTER has no valid account: %s", err))
	}
	if accData != nil {
		if out.AccountKey == nil {
			out.AccountKey = out.AccountPath[1].Leaf.Sibling[:]
		}
		//now accountKey must has been filled
	}

	// notice we have change that no leaf (account data) exist in either before or after,
	// for that case we had to calculate the nodeKey here
	if out.AccountKey == nil {
		word := zkt.NewByte32FromBytesPaddingZero(addr.Bytes())
		k, err := word.Hash()
		if err != nil {
			panic(fmt.Errorf("unexpected hash error for address: %s", err))
		}
		kHash := zkt.NewHashFromBigInt(k)
		out.AccountKey = hexutil.Bytes(kHash[:])
	}

	return out, nil
}

// update traced storage state, and return the corresponding trace object
func (w *zktrieProofWriter) traceStorageUpdate(addr common.Address, key, value []byte) (*StorageTrace, error) {

	trie := w.tracingStorageTries[addr]
	if trie == nil {
		return nil, fmt.Errorf("no trace storage trie for %s", addr)
	}

	statePath := [2]*SMTPath{{}, {}}
	stateUpdate := [2]*StateStorage{}

	storeKey := zkt.NewByte32FromBytesPaddingZero(common.BytesToHash(key).Bytes())
	storeValueBefore := trie.Get(storeKey[:])
	storeValue := zkt.NewByte32FromBytes(value)
	valZero := zkt.Byte32{}

	if storeValueBefore != nil && !bytes.Equal(storeValueBefore[:], common.Hash{}.Bytes()) {
		stateUpdate[0] = &StateStorage{
			Key:   storeKey.Bytes(),
			Value: storeValueBefore,
		}
	}

	var storageBeforeProof, storageAfterProof proofList
	s_key, _ := zkt.ToSecureKeyBytes(storeKey.Bytes())
	if err := trie.Prove(s_key.Bytes(), 0, &storageBeforeProof); err != nil {
		return nil, fmt.Errorf("prove BEFORE storage state fail: %s", err)
	}

	decodeProofForMPTPath(storageBeforeProof, statePath[0])
	if err := verifyStorage(storeKey, zkt.NewByte32FromBytes(storeValueBefore), statePath[0].Leaf); err != nil {
		panic(fmt.Errorf("storage BEFORE has no valid data: %s (%v)", err, statePath[0]))
	}

	if !bytes.Equal(storeValue.Bytes(), common.Hash{}.Bytes()) {
		if err := trie.TryUpdate(storeKey.Bytes(), storeValue.Bytes()); err != nil {
			return nil, fmt.Errorf("update zktrie storage fail: %s", err)
		}
		stateUpdate[1] = &StateStorage{
			Key:   storeKey.Bytes(),
			Value: storeValue.Bytes(),
		}
	} else {
		if err := trie.TryDelete(storeKey.Bytes()); err != nil {
			return nil, fmt.Errorf("delete zktrie storage fail: %s", err)
		}
	}

	if err := trie.Prove(s_key.Bytes(), 0, &storageAfterProof); err != nil {
		return nil, fmt.Errorf("prove AFTER storage state fail: %s", err)
	}
	decodeProofForMPTPath(storageAfterProof, statePath[1])
	if err := verifyStorage(storeKey, storeValue, statePath[1].Leaf); err != nil {
		panic(fmt.Errorf("storage AFTER has no valid data: %s (%v)", err, statePath[1]))
	}

	out, err := w.traceAccountUpdate(addr,
		func(acc *types.StateAccount) *types.StateAccount {
			if acc == nil {
				// in case we read an unexist account
				if !bytes.Equal(valZero.Bytes(), value) {
					panic(fmt.Errorf("write to an unexist account [%s] which is not allowed", addr))
				}
				return nil
			}

			//sanity check
			if accRootFromState := zkt.ReverseByteOrder(statePath[0].Root); !bytes.Equal(acc.Root[:], accRootFromState) {
				panic(fmt.Errorf("unexpected storage root before: [%s] vs [%x]", acc.Root, accRootFromState))
			}
			return &types.StateAccount{
				Nonce:            acc.Nonce,
				Balance:          acc.Balance,
				Root:             common.BytesToHash(zkt.ReverseByteOrder(statePath[1].Root)),
				KeccakCodeHash:   acc.KeccakCodeHash,
				PoseidonCodeHash: acc.PoseidonCodeHash,
				CodeSize:         acc.CodeSize,
			}
		})
	if err != nil {
		return nil, fmt.Errorf("update account %s in SSTORE fail: %s", addr, err)
	}

	if stateUpdate[1] != nil {
		out.StateKey = statePath[1].Leaf.Sibling
	} else if stateUpdate[0] != nil {
		out.StateKey = statePath[0].Leaf.Sibling
	} else {
		// it occurs when we are handling SLOAD with non-exist value
		// still no pretty idea, had to touch the internal behavior in zktrie ....
		if h, err := storeKey.Hash(); err != nil {
			return nil, fmt.Errorf("hash storekey fail: %s", err)
		} else {
			out.StateKey = zkt.NewHashFromBigInt(h)[:]
		}
		stateUpdate[1] = &StateStorage{
			Key:   storeKey.Bytes(),
			Value: valZero.Bytes(),
		}
		stateUpdate[0] = stateUpdate[1]
	}

	out.StatePath = statePath
	out.StateUpdate = stateUpdate
	return out, nil
}

func (w *zktrieProofWriter) HandleNewState(accountState *types.AccountWrapper) (*StorageTrace, error) {

	if accountState.Storage != nil {
		storeAddr := hexutil.MustDecode(accountState.Storage.Key)
		storeValue := hexutil.MustDecode(accountState.Storage.Value)
		return w.traceStorageUpdate(accountState.Address, storeAddr, storeValue)
	} else {

		var stateRoot common.Hash
		accData := getAccountDataFromLogState(accountState)

		out, err := w.traceAccountUpdate(accountState.Address, func(accBefore *types.StateAccount) *types.StateAccount {
			if accBefore != nil {
				stateRoot = accBefore.Root
			}
			// we need to restore stateRoot from before
			if accData != nil {
				accData.Root = stateRoot
			}
			return accData
		})
		if err != nil {
			return nil, fmt.Errorf("update account state %s fail: %s", accountState.Address, err)
		}

		hash := zkt.NewHashFromBytes(stateRoot[:])
		out.CommonStateRoot = hash[:]
		return out, nil
	}

}

func handleLogs(od opOrderer, currentContract common.Address, logs []*types.StructLogRes) {
	logStack := []int{0}
	contractStack := map[int]common.Address{}
	callEnterAddress := currentContract

	// now trace every OP which could cause changes on state:
	for i, sLog := range logs {

		//trace log stack by depth rather than scanning specified op
		if sl := len(logStack); sl < sLog.Depth {
			logStack = append(logStack, i)
			//update currentContract according to previous op
			contractStack[sl] = currentContract
			currentContract = callEnterAddress

		} else if sl > sLog.Depth {
			logStack = logStack[:sl-1]
			currentContract = contractStack[sLog.Depth]
			resumePos := logStack[len(logStack)-1]
			calledLog := logs[resumePos]

			//no need to handle fail calling
			if calledLog.ExtraData != nil {
				if !calledLog.ExtraData.CallFailed {
					//reentry the last log which "cause" the calling, some handling may needed
					switch calledLog.Op {
					case "CREATE", "CREATE2":
						//addr, accDataBefore := getAccountDataFromProof(calledLog, posCALLBefore)
						od.absorb(getAccountState(calledLog, posCREATEAfter))
					}
				} else {
					od.readonly(false)
				}
			}

		} else {
			logStack[sl-1] = i
		}
		//sanity check
		if len(logStack) != sLog.Depth {
			panic("tracking log stack failure")
		}
		callEnterAddress = currentContract

		//check extra status for current op if it is a call
		if extraData := sLog.ExtraData; extraData != nil {
			if extraData.CallFailed || len(sLog.ExtraData.Caller) < 2 {
				// no enough caller data (2) is being capture indicate we are in an immediate failure
				// i.e. it fail before stack entry (like no enough balance for a "call with value"),
				// or we just not handle this calling op correctly yet

				// for a failed option, now we just purpose nothing happens (FIXME: it is inconsentent with mpt_table)
				// except for CREATE, for which the callee's nonce would be increased
				switch sLog.Op {
				case "CREATE", "CREATE2":
					st := copyAccountState(extraData.Caller[0])
					st.Nonce += 1
					od.absorb(st)
				}
			}

			if extraData.CallFailed {
				od.readonly(true)
			}
			// now trace caller's status first
			if caller := extraData.Caller; len(caller) >= 2 {
				od.absorb(caller[1])
			}
		}

		switch sLog.Op {
		case "SELFDESTRUCT":
			// NOTE: this op code has been disabled so we treat it as nothing now

			//in SELFDESTRUCT, a call on target address is made so the balance would be updated
			//in the last item
			//stateTarget := getAccountState(sLog, posSELFDESTRUCT)
			//od.absorb(stateTarget)
			//then build an "deleted state", only address and other are default
			//od.absorb(&types.AccountWrapper{Address: currentContract})

		case "CREATE", "CREATE2":
			// notice in immediate failure we have no enough tracing in extraData
			if len(sLog.ExtraData.StateList) >= 2 {
				state := getAccountState(sLog, posCREATE)
				od.absorb(state)
				//update contract to CREATE addr
				callEnterAddress = state.Address
			}

		case "CALL", "CALLCODE":
			// notice in immediate failure we have no enough tracing in extraData
			if len(sLog.ExtraData.StateList) >= 3 {
				state := getAccountState(sLog, posCALL)
				od.absorb(state)
				callEnterAddress = state.Address
			}
		case "STATICCALL":
			//static call has no update on target address (and no immediate failure?)
			callEnterAddress = getAccountState(sLog, posSTATICCALL).Address
		case "DELEGATECALL":

		case "SLOAD":
			accountState := getAccountState(sLog, posSSTOREBefore)
			od.absorbStorage(accountState, nil)
		case "SSTORE":
			log.Debug("build SSTORE", "pc", sLog.Pc, "key", sLog.Stack[len(sLog.Stack)-1])
			accountState := copyAccountState(getAccountState(sLog, posSSTOREBefore))
			// notice the log only provide the value BEFORE store and it is not suitable for our protocol,
			// here we change it into value AFTER update
			before := accountState.Storage
			accountState.Storage = &types.StorageWrapper{
				Key:   sLog.Stack[len(sLog.Stack)-1],
				Value: sLog.Stack[len(sLog.Stack)-2],
			}
			od.absorbStorage(accountState, before)

		default:
		}
	}
}

func HandleTx(od opOrderer, txResult *types.ExecutionResult) {

	// the from state is read before tx is handled and nonce is added, we combine both
	preTxSt := copyAccountState(txResult.From)
	preTxSt.Nonce += 1
	od.absorb(preTxSt)

	if txResult.Failed {
		od.readonly(true)
	}

	var toAddr common.Address
	if state := txResult.AccountCreated; state != nil {
		od.absorb(state)
		toAddr = state.Address
	} else {
		toAddr = txResult.To.Address
	}

	handleLogs(od, toAddr, txResult.StructLogs)
	if txResult.Failed {
		od.readonly(false)
	}

	for _, state := range txResult.AccountsAfter {
		// special case: for suicide, the state has been captured in SELFDESTRUCT
		// and we skip it here
		if isDeletedAccount(state) {
			log.Debug("skip suicide address", "address", state.Address)
			continue
		}

		od.absorb(state)
	}

}

const defaultOrdererScheme = MPTWitnessRWTbl

var usedOrdererScheme = defaultOrdererScheme

func SetOrderScheme(t MPTWitnessType) { usedOrdererScheme = t }

// HandleBlockTrace only for backward compatibility
func HandleBlockTrace(block *types.BlockTrace) ([]*StorageTrace, error) {
	return HandleBlockTraceEx(block, usedOrdererScheme)
}

func HandleBlockTraceEx(block *types.BlockTrace, ordererScheme MPTWitnessType) ([]*StorageTrace, error) {

	writer, err := NewZkTrieProofWriter(block.StorageTrace)
	if err != nil {
		return nil, err
	}

	var od opOrderer
	switch ordererScheme {
	case MPTWitnessNothing:
		panic("should not come here when scheme is 0")
	case MPTWitnessNatural:
		od = &simpleOrderer{}
	case MPTWitnessRWTbl:
		od = NewRWTblOrderer(writer.tracingAccounts)
	default:
		return nil, fmt.Errorf("unrecognized scheme %d", ordererScheme)
	}

	for _, tx := range block.ExecutionResults {
		HandleTx(od, tx)
	}

	// notice some coinbase addr (like all zero) is in fact not exist and should not be update
	// TODO: not a good solution, just for patch ...
	if coinbaseData := writer.tracingAccounts[block.Coinbase.Address]; coinbaseData != nil {
		od.absorb(block.Coinbase)
	}

	opDisp := od.end_absorb()
	var outTrace []*StorageTrace

	for op := opDisp.next(); op != nil; op = opDisp.next() {
		trace, err := writer.HandleNewState(op)
		if err != nil {
			return nil, err
		}
		outTrace = append(outTrace, trace)
	}

	finalHash := writer.tracingZktrie.Hash()
	if !bytes.Equal(finalHash.Bytes(), block.StorageTrace.RootAfter.Bytes()) {
		return outTrace, fmt.Errorf("unmatch hash: [%x] vs [%x]", finalHash.Bytes(), block.StorageTrace.RootAfter.Bytes())
	}

	return outTrace, nil

}

func FillBlockTraceForMPTWitness(order MPTWitnessType, block *types.BlockTrace) error {

	if order == MPTWitnessNothing {
		return nil
	}

	trace, err := HandleBlockTraceEx(block, order)
	if err != nil {
		return err
	}

	msg, err := json.Marshal(trace)
	if err != nil {
		return err
	}

	rawmsg := json.RawMessage(msg)

	block.MPTWitness = &rawmsg
	return nil
}
