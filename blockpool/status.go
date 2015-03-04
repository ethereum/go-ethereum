package blockpool

import (
	"fmt"
	"sync"
)

type statusValues struct {
	BlockHashes       int    // number of hashes fetched this session
	BlockHashesInPool int    // number of hashes currently in  the pool
	Blocks            int    // number of blocks fetched this session
	BlocksInPool      int    // number of blocks currently in  the pool
	BlocksInChain     int    // number of blocks inserted/connected to the blockchain this session
	NewBlocks         int    // number of new blocks (received with new blocks msg) this session
	Forks             int    // number of chain forks in the blockchain (poolchain) this session
	LongestChain      int    // the longest chain inserted since the start of session (aka session blockchain height)
	BestPeer          []byte //Pubkey
	Syncing           bool   // requesting, updating etc
	Peers             int    // cumulative number of all different registered peers since the start of this session
	ActivePeers       int    // cumulative number of all different peers that contributed a hash or block since the start of this session
	LivePeers         int    // number of live peers registered with the block pool (supposed to be redundant but good sanity check
	BestPeers         int    // cumulative number of all peers that at some point were promoted as best peer (peer with highest TD status) this session
	BadPeers          int    // cumulative number of all peers that violated the protocol (invalid block or pow, unrequested hash or block, etc)
}

type status struct {
	lock        sync.Mutex
	values      statusValues
	chain       map[string]int
	peers       map[string]int
	bestPeers   map[string]int
	badPeers    map[string]int
	activePeers map[string]int
}

func newStatus() *status {
	return &status{
		chain:       make(map[string]int),
		peers:       make(map[string]int),
		bestPeers:   make(map[string]int),
		badPeers:    make(map[string]int),
		activePeers: make(map[string]int),
	}
}

type Status struct {
	statusValues
}

// blockpool status for reporting
func (self *BlockPool) Status() *Status {
	self.status.lock.Lock()
	defer self.status.lock.Unlock()
	self.status.values.ActivePeers = len(self.status.activePeers)
	self.status.values.BestPeers = len(self.status.bestPeers)
	self.status.values.BadPeers = len(self.status.badPeers)
	self.status.values.LivePeers = len(self.peers.peers)
	self.status.values.Peers = len(self.status.peers)
	self.status.values.BlockHashesInPool = len(self.pool)
	return &Status{self.status.values}
}

func (self *Status) String() string {
	return fmt.Sprintf(`
  Syncing:            %v
  BlockHashes:        %v
  BlockHashesInPool:  %v
  Blocks:             %v
  BlocksInPool:       %v
  BlocksInChain:      %v
  NewBlocks:          %v
  Forks:              %v
  LongestChain:       %v
  Peers:              %v
  LivePeers:          %v
  ActivePeers:        %v
  BestPeers:          %v
  BadPeers:           %v
`,
		self.Syncing,
		self.BlockHashes,
		self.BlockHashesInPool,
		self.Blocks,
		self.BlocksInPool,
		self.BlocksInChain,
		self.NewBlocks,
		self.Forks,
		self.LongestChain,
		self.Peers,
		self.LivePeers,
		self.ActivePeers,
		self.BestPeers,
		self.BadPeers,
	)
}

func (self *BlockPool) syncing() {
	self.status.lock.Lock()
	defer self.status.lock.Unlock()
	if !self.status.values.Syncing {
		self.status.values.Syncing = true
		go func() {
			self.wg.Wait()
			self.status.lock.Lock()
			self.status.values.Syncing = false
			self.status.lock.Unlock()
		}()
	}
}
