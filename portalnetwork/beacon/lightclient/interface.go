package lightclient

import "github.com/protolambda/zrnt/eth2/beacon/common"

type ConsensusAPI interface {
	GetBootstrap(blockRoot common.Root) (common.SpecObj, error)
	GetUpdates(firstPeriod, count uint64) ([]common.SpecObj, error)
	GetFinalityUpdate() (common.SpecObj, error)
	GetOptimisticUpdate() (common.SpecObj, error)
	ChainID() uint64
	Name() string
}