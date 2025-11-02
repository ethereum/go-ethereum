package state

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/types/bal"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie/trienode"
	"github.com/holiman/uint256"
	"golang.org/x/sync/errgroup"
	"maps"
	"sync"
	"sync/atomic"
	"time"
)

type BALStateTransition struct {
	accessList *BALReader
	db         Database
	reader     Reader
	stateTrie  Trie
	parentRoot common.Hash

	rootHash  common.Hash
	diffs     map[common.Address]*bal.AccountMutations
	prestates sync.Map //map[common.Address]*types.StateAccount

	poststates map[common.Address]*types.StateAccount
	tries      sync.Map //map[common.Address]Trie

	deletions map[common.Address]struct{}

	originStorages   map[common.Address]map[common.Hash]common.Hash
	originStoragesWG sync.WaitGroup

	accountDeleted int64
	accountUpdated int64
	storageDeleted atomic.Int64
	storageUpdated atomic.Int64

	// TODO: maybe package these into their own 'CommitMetrics' struct instead of making them public fields

	stateUpdate *stateUpdate

	metrics BALStateTransitionMetrics
}

func (s *BALStateTransition) Metrics() *BALStateTransitionMetrics {
	return &s.metrics
}

type BALStateTransitionMetrics struct {
	// trie hashing metrics
	AccountUpdate         time.Duration
	StatePrefetch         time.Duration
	StateUpdate           time.Duration
	StateHash             time.Duration
	OriginStorageLoadTime time.Duration

	// commit metrics
	AccountCommits  time.Duration
	StorageCommits  time.Duration
	SnapshotCommits time.Duration
	TrieDBCommits   time.Duration
	TotalCommitTime time.Duration
}

func NewBALStateTransition(accessList *BALReader, db Database, parentRoot common.Hash) (*BALStateTransition, error) {
	reader, err := db.Reader(parentRoot)
	if err != nil {
		panic("OH FUCK")
	}
	stateTrie, err := db.OpenTrie(parentRoot)
	if err != nil {
		return nil, err
	}

	return &BALStateTransition{
		accessList:       accessList,
		db:               db,
		reader:           reader,
		stateTrie:        stateTrie,
		parentRoot:       parentRoot,
		rootHash:         common.Hash{},
		diffs:            make(map[common.Address]*bal.AccountMutations),
		prestates:        sync.Map{},
		poststates:       make(map[common.Address]*types.StateAccount),
		tries:            sync.Map{},
		deletions:        make(map[common.Address]struct{}),
		originStorages:   make(map[common.Address]map[common.Hash]common.Hash),
		originStoragesWG: sync.WaitGroup{},
		stateUpdate:      nil,
	}, nil
}

// TODO: make use of this method return the error from IntermediateRoot or Commit
func (s *BALStateTransition) Error() error {
	return nil
}

// TODO: refresh my knowledge of the storage-clearing EIP and ensure that my assumptions around
// an empty account which contains storage are valid here.
//
// isAccountDeleted checks whether the state account was deleted in this block.  Post selfdestruct-removal,
// deletions can only occur if an account which has a balance becomes the target of a CREATE2 initcode
// which calls SENDALL, clearing the account and marking it for deletion.
func isAccountDeleted(prestate *types.StateAccount, mutations *bal.AccountMutations) bool {
	// TODO: figure out how to simplify this method
	if mutations.Code != nil && len(mutations.Code) != 0 {
		return false
	}
	if mutations.Nonce != nil && *mutations.Nonce != 0 {
		return false
	}
	if mutations.StorageWrites != nil && len(mutations.StorageWrites) > 0 {
		return false
	}
	if mutations.Balance != nil {
		if mutations.Balance.IsZero() {
			if prestate.Nonce != 0 || prestate.Balance.IsZero() || common.BytesToHash(prestate.CodeHash) != types.EmptyCodeHash {
				return false
			}
			// consider an empty account with storage to be deleted, so we don't check root here
			return true
		}
	}
	return false
}

func (s *BALStateTransition) updateAccount(addr common.Address) (*types.StateAccount, []byte) {
	a, _ := s.prestates.Load(addr)
	acct := a.(*types.StateAccount)

	acct, diff := acct.Copy(), s.diffs[addr]
	code := diff.Code

	if diff.Nonce != nil {
		acct.Nonce = *diff.Nonce
	}
	if diff.Balance != nil {
		acct.Balance = new(uint256.Int).Set(diff.Balance)
	}
	if tr, ok := s.tries.Load(addr); ok {
		acct.Root = tr.(Trie).Hash()
	}
	return acct, code
}

func (s *BALStateTransition) commitAccount(addr common.Address) (*accountUpdate, *trienode.NodeSet, error) {
	var (
		encode = func(val common.Hash) []byte {
			if val == (common.Hash{}) {
				return nil
			}
			blob, _ := rlp.EncodeToBytes(common.TrimLeftZeroes(val[:]))
			return blob
		}
	)
	op := &accountUpdate{
		address: addr,
		data:    types.SlimAccountRLP(*s.poststates[addr]), // TODO: cache the updated state acocunt somewhere
	}
	if prestate, exist := s.prestates.Load(addr); exist {
		prestate := prestate.(*types.StateAccount)
		op.origin = types.SlimAccountRLP(*prestate)
	}

	if s.diffs[addr].Code != nil {
		op.code = &contractCode{
			crypto.Keccak256Hash(s.diffs[addr].Code),
			s.diffs[addr].Code,
		}
	}

	if len(s.diffs[addr].StorageWrites) == 0 {
		return op, nil, nil
	}

	op.storages = make(map[common.Hash][]byte)
	op.storagesOriginByHash = make(map[common.Hash][]byte)
	op.storagesOriginByKey = make(map[common.Hash][]byte)

	for key, value := range s.diffs[addr].StorageWrites {
		hash := crypto.Keccak256Hash(key.Bytes())
		op.storages[hash] = encode(value)
		origin := encode(s.originStorages[addr][key])
		op.storagesOriginByHash[hash] = origin
		op.storagesOriginByKey[key] = origin
	}
	tr, _ := s.tries.Load(addr)
	root, nodes := tr.(Trie).Commit(false)
	s.poststates[addr].Root = root
	return op, nodes, nil
}

func (s *BALStateTransition) CommitWithUpdate(block uint64, deleteEmptyObjects bool, noStorageWiping bool) (common.Hash, *stateUpdate, error) {
	// 1) create a stateUpdate object
	// Commit objects to the trie, measuring the elapsed time
	var (
		commitStart             = time.Now()
		accountTrieNodesUpdated int
		accountTrieNodesDeleted int
		storageTrieNodesUpdated int
		storageTrieNodesDeleted int

		lock    sync.Mutex                                           // protect two maps below
		nodes   = trienode.NewMergedNodeSet()                        // aggregated trie nodes
		updates = make(map[common.Hash]*accountUpdate, len(s.diffs)) // aggregated account updates

		// merge aggregates the dirty trie nodes into the global set.
		//
		// Given that some accounts may be destroyed and then recreated within
		// the same block, it's possible that a node set with the same owner
		// may already exist. In such cases, these two sets are combined, with
		// the later one overwriting the previous one if any nodes are modified
		// or deleted in both sets.
		//
		// merge run concurrently across  all the state objects and account trie.
		merge = func(set *trienode.NodeSet) error {
			if set == nil {
				return nil
			}
			lock.Lock()
			defer lock.Unlock()

			updates, deletes := set.Size()
			if set.Owner == (common.Hash{}) {
				accountTrieNodesUpdated += updates
				accountTrieNodesDeleted += deletes
			} else {
				storageTrieNodesUpdated += updates
				storageTrieNodesDeleted += deletes
			}
			return nodes.Merge(set)
		}
	)

	destructedPrestates := make(map[common.Address]*types.StateAccount)
	s.prestates.Range(func(key, value any) bool {
		addr := key.(common.Address)
		acct := value.(*types.StateAccount)
		destructedPrestates[addr] = acct
		return true
	})

	deletes, delNodes, err := handleDestruction(s.db, s.stateTrie, noStorageWiping, maps.Keys(s.deletions), destructedPrestates)
	if err != nil {
		return common.Hash{}, nil, err
	}
	for _, set := range delNodes {
		if err := merge(set); err != nil {
			return common.Hash{}, nil, err
		}
	}

	// Handle all state updates afterwards, concurrently to one another to shave
	// off some milliseconds from the commit operation. Also accumulate the code
	// writes to run in parallel with the computations.
	var (
		start   = time.Now()
		root    common.Hash
		workers errgroup.Group
	)
	// Schedule the account trie first since that will be the biggest, so give
	// it the most time to crunch.
	//
	// TODO(karalabe): This account trie commit is *very* heavy. 5-6ms at chain
	// heads, which seems excessive given that it doesn't do hashing, it just
	// shuffles some data. For comparison, the *hashing* at chain head is 2-3ms.
	// We need to investigate what's happening as it seems something's wonky.
	// Obviously it's not an end of the world issue, just something the original
	// code didn't anticipate for.
	workers.Go(func() error {
		// Write the account trie changes, measuring the amount of wasted time
		newroot, set := s.stateTrie.Commit(true)
		root = newroot

		if err := merge(set); err != nil {
			return err
		}
		s.metrics.AccountCommits = time.Since(start)
		return nil
	})

	s.originStoragesWG.Wait()

	// Schedule each of the storage tries that need to be updated, so they can
	// run concurrently to one another.
	//
	// TODO(karalabe): Experimentally, the account commit takes approximately the
	// same time as all the storage commits combined, so we could maybe only have
	// 2 threads in total. But that kind of depends on the account commit being
	// more expensive than it should be, so let's fix that and revisit this todo.
	for addr, _ := range s.diffs {
		if _, isDeleted := s.deletions[addr]; isDeleted {
			continue
		}

		address := addr
		// Run the storage updates concurrently to one another
		workers.Go(func() error {
			// Write any storage changes in the state object to its storage trie
			update, set, err := s.commitAccount(address)
			if err != nil {
				return err
			}

			if err := merge(set); err != nil {
				return err
			}
			lock.Lock()
			updates[crypto.Keccak256Hash(address[:])] = update
			s.metrics.StorageCommits = time.Since(start) // overwrite with the longest storage commit runtime
			lock.Unlock()
			return nil
		})
	}
	// Wait for everything to finish and update the metrics
	if err := workers.Wait(); err != nil {
		return common.Hash{}, nil, err
	}

	accountUpdatedMeter.Mark(s.accountUpdated)
	storageUpdatedMeter.Mark(s.storageUpdated.Load())
	accountDeletedMeter.Mark(s.accountDeleted)
	storageDeletedMeter.Mark(s.storageDeleted.Load())
	accountTrieUpdatedMeter.Mark(int64(accountTrieNodesUpdated))
	accountTrieDeletedMeter.Mark(int64(accountTrieNodesDeleted))
	storageTriesUpdatedMeter.Mark(int64(storageTrieNodesUpdated))
	storageTriesDeletedMeter.Mark(int64(storageTrieNodesDeleted))

	ret := newStateUpdate(noStorageWiping, s.parentRoot, root, block, deletes, updates, nodes)

	snapshotCommits, trieDBCommits, err := flushStateUpdate(s.db, block, ret)
	if err != nil {
		return common.Hash{}, nil, err
	}

	s.metrics.SnapshotCommits, s.metrics.TrieDBCommits = snapshotCommits, trieDBCommits
	s.metrics.TotalCommitTime = time.Since(commitStart)
	return root, ret, nil
}

func (s *BALStateTransition) IntermediateRoot(_ bool) common.Hash {
	if s.rootHash != (common.Hash{}) {
		return s.rootHash
	}

	start := time.Now()
	lastIdx := len(s.accessList.block.Transactions()) + 1

	s.originStoragesWG.Add(1)
	go func() {
		defer s.originStoragesWG.Done()
		for _, addr := range s.accessList.ModifiedAccounts() {
			diff := s.accessList.readAccountDiff(addr, lastIdx)
			if len(diff.StorageWrites) > 0 {
				s.originStorages[addr] = make(map[common.Hash]common.Hash)
				for key := range diff.StorageWrites {
					val, err := s.reader.Storage(addr, key)
					if err != nil {
						panic("TODO: wat do?")
					}
					s.originStorages[addr][key] = val
				}
			}
		}
		s.metrics.OriginStorageLoadTime = time.Since(start)
	}()

	var wg sync.WaitGroup

	// 1. resolve the entire state object for the updated addrs, in parallel prefetch them in the account trie
	// 1. in parallel:
	//		* load the prestate of mutated state objects from the snapshot, update their tries.
	//		* prefetch all mutated account in the account trie

	for _, addr := range s.accessList.ModifiedAccounts() {
		diff := s.accessList.readAccountDiff(addr, lastIdx)
		s.diffs[addr] = diff
	}

	for _, addr := range s.accessList.ModifiedAccounts() {
		wg.Add(1)
		address := addr
		go func() {
			acct := s.accessList.prestateReader.account(address)
			diff := s.diffs[address]
			if acct == nil {
				acct = types.NewEmptyStateAccount()
			}
			s.prestates.Store(address, acct)

			if len(diff.StorageWrites) > 0 {
				tr, err := s.db.OpenStorageTrie(s.parentRoot, address, acct.Root, s.stateTrie)
				if err != nil {
					panic("FUCK")
				}
				s.tries.Store(address, tr)

				var (
					updateKeys, updateValues [][]byte
					deleteKeys               [][]byte
				)
				for key, val := range diff.StorageWrites {
					if val != (common.Hash{}) {
						updateKeys = append(updateKeys, key[:])
						updateValues = append(updateValues, common.TrimLeftZeroes(val[:]))

						s.storageUpdated.Add(1)
					} else {
						deleteKeys = append(deleteKeys, key[:])

						s.storageDeleted.Add(1)
					}
				}
				if err := tr.UpdateStorageBatch(address, updateKeys, updateValues); err != nil {
					panic("FUCK1")
				}

				for _, key := range deleteKeys {
					if err := tr.DeleteStorage(address, key); err != nil {
						panic("SHITTT")
					}
				}

				hashStart := time.Now()
				tr.Hash()
				s.metrics.StateHash = time.Since(hashStart)
				/*
					var objTrieData string
					it, err := tr.NodeIterator([]byte{})
					if err != nil {
						panic(err)
					}
					for it.Next(true) {
						if it.Leaf() {
							objTrieData += fmt.Sprintf("%x: %x\n", it.Path(), it.LeafBlob())
						} else {
							objTrieData += fmt.Sprintf("%x: %x\n", it.Path(), it.Hash())
						}
					}
					fmt.Printf("acct hash %x. hash=%x,  updated storage: %v, deleted storage: %v trie:\n%s\n", crypto.Keccak256Hash(address[:]), tr.Hash(), updateKeys, deleteKeys, objTrieData)
				*/
			}

			wg.Done()
		}()
	}

	wg.Add(1)
	// prefetch all modified accounts in the main account trie.
	go func() {
		prefetchStart := time.Now()
		if err := s.stateTrie.PrefetchAccount(s.accessList.ModifiedAccounts()); err != nil {
			panic("FUCK2")
		}
		s.metrics.StatePrefetch = time.Since(prefetchStart)
		wg.Done()
	}()

	wg.Wait()
	s.metrics.AccountUpdate = time.Since(start)

	// stage 2: update the main account trie

	stateUpdateStart := time.Now()
	for mutatedAddr, _ := range s.diffs {
		p, _ := s.prestates.Load(mutatedAddr)
		prestate := p.(*types.StateAccount)

		isDeleted := isAccountDeleted(prestate, s.diffs[mutatedAddr])
		if isDeleted {
			if err := s.stateTrie.DeleteAccount(mutatedAddr); err != nil {
				panic("FUCK3")
			}
			s.deletions[mutatedAddr] = struct{}{}
		} else {
			acct, code := s.updateAccount(mutatedAddr)

			if code != nil {
				codeHash := crypto.Keccak256Hash(code)
				acct.CodeHash = codeHash.Bytes()
				if err := s.stateTrie.UpdateContractCode(mutatedAddr, codeHash, code); err != nil {
					panic("FUCK4")
				}
			}
			if err := s.stateTrie.UpdateAccount(mutatedAddr, acct, len(code)); err != nil {
				panic("FUCK4")
			}
			s.poststates[mutatedAddr] = acct
		}
	}

	s.metrics.StateUpdate = time.Since(stateUpdateStart)

	stateTrieHashStart := time.Now()
	s.rootHash = s.stateTrie.Hash()
	s.metrics.StateHash = time.Since(stateTrieHashStart)

	/*
		it, err := s.stateTrie.NodeIterator([]byte{})
		if err != nil {
			panic(err)
		}
		fmt.Println("state trie")
		for it.Next(true) {
			if it.Leaf() {
				fmt.Printf("%x: %x\n", it.Path(), it.LeafBlob())
			} else {
				fmt.Printf("%x: %x\n", it.Path(), it.Hash())
			}
		}
	*/
	return s.rootHash
}

func (s *BALStateTransition) Preimages() map[common.Hash][]byte {
	// TODO: implement this
	return make(map[common.Hash][]byte)
}
