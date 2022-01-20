package engine_v2

import (
	"encoding/json"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS/utils"
	"github.com/XinFinOrg/XDPoSChain/ethdb"
	lru "github.com/hashicorp/golang-lru"
)

// Snapshot is the state of the smart contract validator list
// The validator list is used on next epoch master nodes
// If we don't have the snapshot, then we have to trace back the gap block smart contract state which is very costly
type SnapshotV2 struct {
	Round  utils.Round `json:"round"`  // Round number
	Number uint64      `json:"number"` // Block number where the snapshot was created
	Hash   common.Hash `json:"hash"`   // Block hash where the snapshot was created

	// MasterNodes will get assigned on updateM1
	NextEpochMasterNodes []common.Address `json:"masterNodes"` // Set of authorized master nodes at this moment for next epoch
}

// create new snapshot for next epoch to use
func newSnapshot(number uint64, hash common.Hash, round utils.Round, qc *utils.QuorumCert, masternodes []common.Address) *SnapshotV2 {
	snap := &SnapshotV2{
		Round:                round,
		Number:               number,
		Hash:                 hash,
		NextEpochMasterNodes: masternodes,
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

// retrieves master nodes list in map type
func (s *SnapshotV2) GetMappedMasterNodes() map[common.Address]struct{} {
	ms := make(map[common.Address]struct{})
	for _, n := range s.NextEpochMasterNodes {
		ms[n] = struct{}{}
	}
	return ms
}
