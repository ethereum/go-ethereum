package lightclient

import (
	"github.com/protolambda/zrnt/eth2/beacon/common"
	"github.com/protolambda/ztyp/tree"
)

var _ ConsensusAPI = (*MockRpc)(nil)

type MockRpc struct {
	testdataDir string
}

// ChainID implements ConsensusAPI.
func (m *MockRpc) ChainID() uint64 {
	panic("unimplemented")
}

// GetBootstrap implements ConsensusAPI.
func (m *MockRpc) GetBootstrap(blockRoot tree.Root) (common.SpecObj, error) {
	panic("unimplemented")
}

// GetFinalityUpdate implements ConsensusAPI.
func (m *MockRpc) GetFinalityUpdate() (common.SpecObj, error) {
	panic("unimplemented")
}

// GetOptimisticUpdate implements ConsensusAPI.
func (m *MockRpc) GetOptimisticUpdate() (common.SpecObj, error) {
	panic("unimplemented")
}

// GetUpdates implements ConsensusAPI.
func (m *MockRpc) GetUpdates(firstPeriod uint64, count uint64) ([]common.SpecObj, error) {
	panic("unimplemented")
}

// Name implements ConsensusAPI.
func (m *MockRpc) Name() string {
	panic("unimplemented")
}
