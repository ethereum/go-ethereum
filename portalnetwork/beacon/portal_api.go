package beacon

import (
	"time"

	zrntcommon "github.com/protolambda/zrnt/eth2/beacon/common"
	"github.com/protolambda/ztyp/tree"
)

var _ ConsensusAPI = &PortalLightApi{}

type PortalLightApi struct {
	bn *BeaconNetwork
}

func NewPortalLightApi() *PortalLightApi {
	return &PortalLightApi{}
}

func (api *PortalLightApi) GetUpdates(firstPeriod, count uint64) ([]zrntcommon.SpecObj, error) {
	return api.bn.GetUpdates(firstPeriod, count)
}

func (api *PortalLightApi) GetCheckpointData(checkpointHash tree.Root) (zrntcommon.SpecObj, error) {
	return api.bn.GetCheckpointData(checkpointHash)
}

func (api *PortalLightApi) GetFinalityData() (zrntcommon.SpecObj, error) {
	expectedCurrentSlot := api.bn.spec.TimeToSlot(zrntcommon.Timestamp(time.Now().Unix()), zrntcommon.Timestamp(BeaconGenesisTime))
	recentEpochStart := expectedCurrentSlot - (expectedCurrentSlot % api.bn.spec.SLOTS_PER_EPOCH) + 1
	return api.bn.GetFinalityUpdate(uint64(recentEpochStart))
}

func (api *PortalLightApi) GetOptimisticData() (zrntcommon.SpecObj, error) {
	expectedCurrentSlot := api.bn.spec.TimeToSlot(zrntcommon.Timestamp(time.Now().Unix()), zrntcommon.Timestamp(BeaconGenesisTime))
	return api.bn.GetOptimisticUpdate(uint64(expectedCurrentSlot))
}

func (api *PortalLightApi) ChainID() uint64 {
	return 1
}

func (api *PortalLightApi) Name() string {
	return "portal"
}
