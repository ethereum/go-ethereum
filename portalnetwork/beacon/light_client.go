package beacon

import (
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/protolambda/zrnt/eth2/beacon/altair"
	"github.com/protolambda/zrnt/eth2/beacon/capella"
	"github.com/protolambda/zrnt/eth2/beacon/common"
	"github.com/protolambda/zrnt/eth2/util/merkle"
	"github.com/protolambda/ztyp/tree"
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
	Logger            log.Logger
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

//lint:ignore U1000 placeholder function
func (c *ConsensusLightClient) bootstrap() error {
	bootstrap, err := c.API.GetCheckpointData(c.InitialCheckpoint)
	if err != nil {
		return err
	}

	isValid := c.isValidCheckpoint(bootstrap.Header.Beacon.Slot)
	if !isValid {
		if c.Config.StrictCheckpointAge {
			return errors.New("checkpoint is too old")
		} else {
			c.Logger.Warn("checkpoint is too old")
		}
	}

	committeeValid := c.isCurrentCommitteeProofValid(bootstrap.Header.Beacon, bootstrap.CurrentSyncCommittee, bootstrap.CurrentSyncCommitteeBranch)

	headerHash := bootstrap.Header.Beacon.HashTreeRoot(tree.GetHashFn()).String()
	expectedHash := c.InitialCheckpoint.String()

	headerValid := headerHash == expectedHash

	if !headerValid {
		return fmt.Errorf("header hash %s does not match expected hash %s", headerHash, expectedHash)
	}

	if !committeeValid {
		return errors.New("committee proof is invalid")
	}

	c.Store = LightClientStore{
		FinalizedHeader:               bootstrap.Header.Beacon,
		CurrentSyncCommittee:          bootstrap.CurrentSyncCommittee,
		OptimisticHeader:              bootstrap.Header.Beacon,
		PreviousMaxActiveParticipants: view.Uint64View(0),
		CurrentMaxActiveParticipants:  view.Uint64View(0),
	}

	return nil
}

func (c *ConsensusLightClient) isValidCheckpoint(blockHashSlot common.Slot) bool {
	currentSlot := c.expectedCurrentSlot()
	currentSlotTimestamp, err := c.slotTimestamp(currentSlot)
	if err != nil {
		return false
	}
	blockHashSlotTimestamp, err := c.slotTimestamp(blockHashSlot)
	if err != nil {
		return false
	}

	slotAge := currentSlotTimestamp - blockHashSlotTimestamp

	return uint64(slotAge) < c.Config.MaxCheckpointAge
}

func (c *ConsensusLightClient) expectedCurrentSlot() common.Slot {
	return c.Config.Spec.TimeToSlot(common.Timestamp(time.Now().Unix()), common.Timestamp(c.Config.ChainConfig.GenesisTime))
}

func (c *ConsensusLightClient) slotTimestamp(slot common.Slot) (common.Timestamp, error) {
	atSlot, err := c.Config.Spec.TimeAtSlot(slot, common.Timestamp(c.Config.ChainConfig.GenesisTime))
	if err != nil {
		return 0, err
	}

	return atSlot, nil
}

func (c *ConsensusLightClient) isCurrentCommitteeProofValid(attestedHeader common.BeaconBlockHeader, currentCommittee common.SyncCommittee, currentCommitteeBranch altair.SyncCommitteeProofBranch) bool {
	return merkle.VerifyMerkleBranch(currentCommittee.HashTreeRoot(c.Config.Spec, tree.GetHashFn()), currentCommitteeBranch[:], 5, 22, attestedHeader.StateRoot)
}
