package engine_v2

import (
	"encoding/json"
	"sort"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS/utils"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/ethdb"
	lru "github.com/hashicorp/golang-lru"
)

// Snapshot is the state of the smart contract validator list
type SnapshotV2 struct {
	sigcache *lru.ARCCache // Cache of recent block signatures to speed up ecrecover

	Round  utils.Round `json:"round"`  // Round number
	Number uint64      `json:"number"` // Block number where the snapshot was created
	Hash   common.Hash `json:"hash"`   // Block hash where the snapshot was created

	// MasterNodes will get assigned on updateM1
	MasterNodes map[common.Address]struct{} `json:"masterNodes"` // Set of authorized master nodes at this moment
}

// newSnapshot creates a new snapshot with the specified startup parameters. This
// method does not initialize the set of recent signers, so only ever use if for
// the genesis block.
func newSnapshot(sigcache *lru.ARCCache, number uint64, hash common.Hash, round utils.Round, qc *utils.QuorumCert, masternodes []common.Address) *SnapshotV2 {
	snap := &SnapshotV2{
		sigcache: sigcache,
		Round:    round,
		Number:   number,
		Hash:     hash,

		MasterNodes: make(map[common.Address]struct{}),
	}
	for _, signer := range masternodes {
		snap.MasterNodes[signer] = struct{}{}
	}
	return snap
}

// loadSnapshot loads an existing snapshot from the database.
func loadSnapshot(sigcache *lru.ARCCache, db ethdb.Database, hash common.Hash) (*SnapshotV2, error) {
	blob, err := db.Get(append([]byte("XDPoS-"), hash[:]...))
	if err != nil {
		return nil, err
	}
	snap := new(SnapshotV2)
	if err := json.Unmarshal(blob, snap); err != nil {
		return nil, err
	}
	snap.sigcache = sigcache

	return snap, nil
}

// store inserts the SnapshotV2 into the database.
func storeSnapshot(s *SnapshotV2, db ethdb.Database) error {
	blob, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return db.Put(append([]byte("XDPoS-"), s.Hash[:]...), blob)
}

// copy creates a deep copy of the SnapshotV2, though not the individual votes.
func (s *SnapshotV2) copy() *SnapshotV2 {
	cpy := &SnapshotV2{
		sigcache:    s.sigcache,
		Round:       s.Round,
		Number:      s.Number,
		Hash:        s.Hash,
		MasterNodes: make(map[common.Address]struct{}),
	}
	for signer := range s.MasterNodes {
		cpy.MasterNodes[signer] = struct{}{}
	}

	return cpy
}

// apply creates a new authorization SnapshotV2 by applying the given headers to
// the original one.
// TODO: XIN-100
func (s *SnapshotV2) apply(headers []*types.Header) (*SnapshotV2, error) {
	return s, nil

	// Allow passing in no headers for cleaner code
	// if len(headers) == 0 {
	// 	return s, nil
	// }
	// // Sanity check that the headers can be applied
	// for i := 0; i < len(headers)-1; i++ {
	// 	if headers[i+1].Number.Uint64() != headers[i].Number.Uint64()+1 {
	// 		return nil, utils.ErrInvalidHeaderOrder
	// 	}
	// }
	// if headers[0].Number.Uint64() != s.Number+1 {
	// 	return nil, utils.ErrInvalidChild
	// }
	// // Iterate through the headers and create a new SnapshotV2
	// snap := s.copy()
	// lastHeader := headers[len(headers)-1]

	// snap.Number += uint64(len(headers))
	// snap.Hash = lastHeader.Hash()

	// extraV2 := new(utils.ExtraFields_v2)
	// err := utils.DecodeBytesExtraFields(lastHeader.Extra, &extraV2)
	// if err != nil {
	// 	return nil, err
	// }
	// snap.Round = extraV2.Round
	// return snap, nil
}

// signers retrieves the list of authorized signers in ascending order, convert into strings then use native sort lib
func (s *SnapshotV2) GetMasterNodes() []common.Address {
	nodes := make([]common.Address, 0, len(s.MasterNodes))
	nodeStrs := make([]string, 0, len(s.MasterNodes))

	for node := range s.MasterNodes {
		nodeStrs = append(nodeStrs, node.Str())
	}
	sort.Strings(nodeStrs)
	for _, str := range nodeStrs {
		nodes = append(nodes, common.StringToAddress(str))
	}

	return nodes
}
