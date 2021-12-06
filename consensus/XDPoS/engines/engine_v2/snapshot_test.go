package engine_v2

import (
	"fmt"
	"io/ioutil"
	"math/big"
	"testing"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS/utils"
	"github.com/XinFinOrg/XDPoSChain/core/rawdb"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/ethdb/leveldb"
	"github.com/stretchr/testify/assert"
)

func TestGetMasterNodes(t *testing.T) {
	masterNodes := []common.Address{
		{4}, {3}, {2}, {1},
	}
	snap := newSnapshot(nil, 1, common.Hash{}, utils.Round(1), nil, masterNodes)
	sortedNodes := snap.GetMasterNodes()
	for i := range masterNodes {
		if masterNodes[i] != sortedNodes[3-i] {
			t.Error("should get sorted master nodes list", i, sortedNodes[i])
			return
		}
	}
}
func TestApplyNewSnapshot(t *testing.T) {
	t.Skip("apply has been temporary commented out")
	snap := newSnapshot(nil, 1, common.Hash{}, utils.Round(1), nil, nil)
	extra := utils.ExtraFields_v2{
		Round: 10,
		QuorumCert: &utils.QuorumCert{
			ProposedBlockInfo: &utils.BlockInfo{},
		},
	}
	extraBytes, err := extra.EncodeToBytes()
	assert.Nil(t, err)

	headers := []*types.Header{
		{Number: big.NewInt(2)},
		{Number: big.NewInt(3)},
		{Number: big.NewInt(4)},
		{
			Number: big.NewInt(5),
			Extra:  extraBytes,
		},
	}
	newSnap, err := snap.apply(headers)
	assert.Nil(t, err)
	if newSnap.Number != 5 {
		t.Error("newSnapshot number should have last header number")
	}
	if newSnap.Hash != headers[3].Hash() {
		t.Error("newSnapshot hash should equal the last header given")
	}
	if newSnap.Round != 10 {
		t.Error("newSnapshot round number should also have last header round number")
	}
}

func TestApplyWithWrongHeader(t *testing.T) {
	t.Skip("apply has been temporary commented out")
	snap := newSnapshot(nil, 1, common.Hash{}, utils.Round(1), nil, nil)
	headers := []*types.Header{
		{Number: big.NewInt(3)},
	}
	_, err := snap.apply(headers)
	assert.Equal(t, err, utils.ErrInvalidChild)

	snap = newSnapshot(nil, 1, common.Hash{}, utils.Round(1), nil, nil)
	headers = []*types.Header{
		{Number: big.NewInt(2)},
		{Number: big.NewInt(4)},
	}
	_, err = snap.apply(headers)
	assert.Equal(t, err, utils.ErrInvalidHeaderOrder)
}

// Should perform deep copy
func TestCopySnapshot(t *testing.T) {
	masterNodes := []common.Address{
		{4}, {3}, {2}, {1},
	}
	snap := newSnapshot(nil, 1, common.Hash{}, utils.Round(1), nil, masterNodes)

	newSnapshot := snap.copy()
	if newSnapshot == snap {
		t.Error("should return given different memory address")
	}

	for node := range snap.MasterNodes {
		if _, ok := newSnapshot.MasterNodes[node]; !ok {
			t.Error("snapshot masternodes should copy to new object")
		}
	}
}

func TestStoreLoadSnapshot(t *testing.T) {
	snap := newSnapshot(nil, 1, common.Hash{0x1}, utils.Round(1), nil, nil)
	dir, err := ioutil.TempDir("", "snapshot-test")
	if err != nil {
		panic(fmt.Sprintf("can't create temporary directory: %v", err))
	}
	db, err := leveldb.New(dir, 256, 0, "")
	if err != nil {
		panic(fmt.Sprintf("can't create temporary database: %v", err))
	}
	lddb := rawdb.NewDatabase(db)

	err = storeSnapshot(snap, lddb)
	if err != nil {
		t.Error("store snapshot failed", err)
	}

	restoredSnapshot, err := loadSnapshot(nil, lddb, snap.Hash)
	if err != nil || restoredSnapshot.Hash != snap.Hash {
		t.Error("load snapshot failed", err)
	}
}
