package beacon

import (
	"github.com/protolambda/zrnt/eth2/beacon/capella"
	"github.com/protolambda/zrnt/eth2/beacon/common"
	"github.com/protolambda/ztyp/view"
)

type ConsensusAPI interface {
	GetUpdates(firstPeriod, count uint64) (LightClientUpdateRange, error)
	GetCheckpointData(checkpointHash common.Root) (*capella.LightClientBootstrap, error)
	GetFinalityData() (*capella.LightClientFinalityUpdate, error)
	GetOptimisticData() (*capella.LightClientOptimisticUpdate, error)
	ChainID() uint64
	Name() string
}

type LightClientStore struct {
	FinalizedHeader               common.BeaconBlockHeader
	CurrentSyncCommittee          common.SyncCommittee
	NextSyncCommittee             common.SyncCommittee
	OptimisticHeader              common.BeaconBlockHeader
	PreviousMaxActiveParticipants view.Uint64View
	CurrentMaxActiveParticipants  view.Uint64View
}

type ConsensusLightClient struct {
	Store             LightClientStore
	API               ConsensusAPI
	InitialCheckpoint common.Root
	LastCheckpoint    common.Root
	Config            Config
}

type Config struct {
	ConsensusAPI         string
	Port                 uint64
	DefaultCheckpoint    common.Root
	Checkpoint           common.Root
	DataDir              string
	ChainConfig          ChainConfig
	Spec                 *common.Spec
	MaxCheckpointAge     uint64
	Fallback             string
	LoadExternalFallback bool
	StrictCheckpointAge  bool
}

type ChainConfig struct {
	ChainID     uint64
	GenesisTime uint64
	GenesisRoot common.Root
}
