package beacon

import (
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/protolambda/zrnt/eth2/beacon/altair"
	"github.com/protolambda/zrnt/eth2/beacon/capella"
	"github.com/protolambda/zrnt/eth2/beacon/common"
	"github.com/protolambda/zrnt/eth2/configs"
	"github.com/protolambda/zrnt/eth2/util/merkle"

	"github.com/ethereum/go-ethereum/common/hexutil"
	blsu "github.com/protolambda/bls12-381-util"
	"github.com/protolambda/ztyp/tree"
	"github.com/protolambda/ztyp/view"
	"github.com/prysmaticlabs/go-bitfield"
)

var (
	ErrInsufficientParticipation     = errors.New("insufficient participation")
	ErrInvalidTimestamp              = errors.New("invalid timestamp")
	ErrInvalidPeriod                 = errors.New("invalid sync committee period")
	ErrNotRelevant                   = errors.New("update not relevant")
	ErrInvalidFinalityProof          = errors.New("invalid finality proof")
	ErrInvalidNextSyncCommitteeProof = errors.New("invalid next sync committee proof")
	ErrInvalidSignature              = errors.New("invalid sync committee signature")
)

type ConsensusAPI interface {
	GetUpdates(firstPeriod, count uint64) ([]*capella.LightClientUpdate, error)
	GetCheckpointData(checkpointHash common.Root) (*capella.LightClientBootstrap, error)
	GetFinalityData() (*capella.LightClientFinalityUpdate, error)
	GetOptimisticData() (*capella.LightClientOptimisticUpdate, error)
	ChainID() uint64
	Name() string
}

type LightClientStore struct {
	FinalizedHeader               *common.BeaconBlockHeader
	CurrentSyncCommittee          *common.SyncCommittee
	NextSyncCommittee             *common.SyncCommittee
	OptimisticHeader              *common.BeaconBlockHeader
	PreviousMaxActiveParticipants view.Uint64View
	CurrentMaxActiveParticipants  view.Uint64View
}

type ConsensusLightClient struct {
	Store             LightClientStore
	API               ConsensusAPI
	InitialCheckpoint common.Root
	LastCheckpoint    common.Root
	Config            *Config
	Logger            log.Logger
}

type Config struct {
	ConsensusAPI         string
	Port                 uint64
	DefaultCheckpoint    common.Root
	Checkpoint           common.Root
	DataDir              string
	Chain                ChainConfig
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

type GenericUpdate struct {
	AttestedHeader          *common.BeaconBlockHeader
	SyncAggregate           *altair.SyncAggregate
	SignatureSlot           common.Slot
	NextSyncCommittee       *common.SyncCommittee
	NextSyncCommitteeBranch *altair.SyncCommitteeProofBranch
	FinalizedHeader         *common.BeaconBlockHeader
	FinalityBranch          *altair.FinalizedRootProofBranch
}

func NewConsensusLightClient(api ConsensusAPI, config *Config, checkpointBlockRoot common.Root, logger log.Logger) (*ConsensusLightClient, error) {
	client := &ConsensusLightClient{
		API:               api,
		Config:            config,
		Logger:            logger,
		InitialCheckpoint: checkpointBlockRoot,
	}

	err := client.bootstrap()
	if err != nil {
		return nil, err
	}

	return client, nil
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
		FinalizedHeader:               &bootstrap.Header.Beacon,
		CurrentSyncCommittee:          &bootstrap.CurrentSyncCommittee,
		OptimisticHeader:              &bootstrap.Header.Beacon,
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

func (c *ConsensusLightClient) VerifyGenericUpdate(update *GenericUpdate) error {
	bits := bitfield.Bitlist(update.SyncAggregate.SyncCommitteeBits).Count()
	if bits == 0 {
		return ErrInsufficientParticipation
	}
	updateFinalizedSlot := update.FinalizedHeader.Slot
	validTime := uint64(c.expectedCurrentSlot()) >= uint64(update.SignatureSlot) && update.SignatureSlot > update.AttestedHeader.Slot && update.AttestedHeader.Slot >= updateFinalizedSlot
	if !validTime {
		return ErrInvalidTimestamp
	}

	storePeriod := CalcSyncPeriod(uint64(c.Store.FinalizedHeader.Slot))
	updateSigPeriod := CalcSyncPeriod(uint64(update.SignatureSlot))
	validPeriod := false
	if c.Store.NextSyncCommittee != nil {
		validPeriod = updateSigPeriod == storePeriod || updateSigPeriod == storePeriod+1
	} else {
		validPeriod = updateSigPeriod == storePeriod
	}
	if !validPeriod {
		return ErrInvalidPeriod
	}

	updateAttestedPeriod := CalcSyncPeriod(uint64(update.AttestedHeader.Slot))
	updateHasNextCommittee := c.Store.NextSyncCommittee == nil && update.NextSyncCommittee != nil && updateAttestedPeriod == storePeriod

	if update.AttestedHeader.Slot <= c.Store.FinalizedHeader.Slot && !updateHasNextCommittee {
		return ErrNotRelevant
	}
	if update.FinalizedHeader != nil && update.FinalityBranch != nil {
		isValid := IsFinalityProofValid(*update.AttestedHeader, *update.FinalizedHeader, *update.FinalityBranch)
		if !isValid {
			return ErrInvalidFinalityProof
		}
	}
	if update.NextSyncCommittee != nil && update.NextSyncCommitteeBranch != nil {
		isValid := IsNextCommitteeProofValid(*update.AttestedHeader, *update.NextSyncCommittee, *update.NextSyncCommitteeBranch)
		if !isValid {
			return ErrInvalidNextSyncCommitteeProof
		}
	}
	var syncCommittee *common.SyncCommittee

	if updateSigPeriod == storePeriod {
		syncCommittee = c.Store.CurrentSyncCommittee
	} else {
		syncCommittee = c.Store.NextSyncCommittee
	}

	pks := GetParticipatingKeys(*syncCommittee, update.SyncAggregate.SyncCommitteeBits)

	isValidSig, err := c.VerifySyncCommitteeSignature(pks, *update.AttestedHeader, update.SyncAggregate.SyncCommitteeSignature, update.SignatureSlot)
	if err != nil {
		return err
	}
	if !isValidSig {
		return ErrInvalidSignature
	}
	return nil
}

func (c *ConsensusLightClient) VerifyUpdate(update *capella.LightClientUpdate) error {
	genericUpdate := FromLightClientUpdate(update)
	return c.VerifyGenericUpdate(genericUpdate)
}

func (c *ConsensusLightClient) VerifyFinalityUpdate(update *capella.LightClientFinalityUpdate) error {
	genericUpdate := FromLightClientFinalityUpdate(update)
	return c.VerifyGenericUpdate(genericUpdate)
}

func (c *ConsensusLightClient) VerifyOptimisticUpdate(update *capella.LightClientOptimisticUpdate) error {
	genericUpdate := FromLightClientOptimisticUpdate(update)
	return c.VerifyGenericUpdate(genericUpdate)
}

func (c *ConsensusLightClient) VerifySyncCommitteeSignature(pks []common.BLSPubkey, attestedHeader common.BeaconBlockHeader, signature common.BLSSignature, signatureSlot common.Slot) (bool, error) {
	headerRoot := attestedHeader.HashTreeRoot(tree.GetHashFn())
	signingRoot := c.ComputeCommitteeSignRoot(headerRoot, signatureSlot)
	blsuPubKeys := make([]*blsu.Pubkey, 0, len(pks))
	for _, p := range pks {
		blsuPubKey, err := p.Pubkey()
		if err != nil {
			return false, err
		}
		blsuPubKeys = append(blsuPubKeys, blsuPubKey)
	}
	blsuSig, err := signature.Signature()
	if err != nil {
		return false, err
	}
	return blsu.FastAggregateVerify(blsuPubKeys, signingRoot[:], blsuSig), nil
}

func (c *ConsensusLightClient) ComputeCommitteeSignRoot(headerRoot tree.Root, slot common.Slot) common.Root {
	genesisRoot := c.Config.Chain.GenesisRoot
	domainType := hexutil.MustDecode("0x07000000")
	forkVersion := c.Config.Spec.ForkVersion(slot)
	domain := common.ComputeDomain(common.BLSDomainType(domainType), forkVersion, genesisRoot)
	return ComputeSigningRoot(headerRoot, domain)
}

func (c *ConsensusLightClient) expectedCurrentSlot() common.Slot {
	return c.Config.Spec.TimeToSlot(common.Timestamp(time.Now().Unix()), common.Timestamp(c.Config.Chain.GenesisTime))
}

func (c *ConsensusLightClient) slotTimestamp(slot common.Slot) (common.Timestamp, error) {
	atSlot, err := c.Config.Spec.TimeAtSlot(slot, common.Timestamp(c.Config.Chain.GenesisTime))
	if err != nil {
		return 0, err
	}

	return atSlot, nil
}

func (c *ConsensusLightClient) isCurrentCommitteeProofValid(attestedHeader common.BeaconBlockHeader, currentCommittee common.SyncCommittee, currentCommitteeBranch altair.SyncCommitteeProofBranch) bool {
	return merkle.VerifyMerkleBranch(currentCommittee.HashTreeRoot(c.Config.Spec, tree.GetHashFn()), currentCommitteeBranch[:], 5, 22, attestedHeader.StateRoot)
}

func FromLightClientUpdate(update *capella.LightClientUpdate) *GenericUpdate {
	return &GenericUpdate{
		AttestedHeader:          &update.AttestedHeader.Beacon,
		SyncAggregate:           &update.SyncAggregate,
		SignatureSlot:           update.SignatureSlot,
		NextSyncCommittee:       &update.NextSyncCommittee,
		NextSyncCommitteeBranch: &update.NextSyncCommitteeBranch,
		FinalizedHeader:         &update.FinalizedHeader.Beacon,
		FinalityBranch:          &update.FinalityBranch,
	}
}

func FromLightClientFinalityUpdate(update *capella.LightClientFinalityUpdate) *GenericUpdate {
	return &GenericUpdate{
		AttestedHeader:  &update.AttestedHeader.Beacon,
		SyncAggregate:   &update.SyncAggregate,
		SignatureSlot:   update.SignatureSlot,
		FinalizedHeader: &update.FinalizedHeader.Beacon,
		FinalityBranch:  &update.FinalityBranch,
	}
}

func FromLightClientOptimisticUpdate(update *capella.LightClientOptimisticUpdate) *GenericUpdate {
	return &GenericUpdate{
		AttestedHeader: &update.AttestedHeader.Beacon,
		SyncAggregate:  &update.SyncAggregate,
		SignatureSlot:  update.SignatureSlot,
	}
}

func ComputeSigningRoot(root common.Root, domain common.BLSDomain) common.Root {
	data := common.SigningData{
		ObjectRoot: root,
		Domain:     domain,
	}
	return data.HashTreeRoot(tree.GetHashFn())
}

func CalcSyncPeriod(slot uint64) uint64 {
	epoch := slot / 32 // 32 slots per epoch
	return epoch / 256 // 256 epochs per sync committee
}

func IsFinalityProofValid(attestedHeader common.BeaconBlockHeader, finalityHeader common.BeaconBlockHeader, finalityBranch altair.FinalizedRootProofBranch) bool {
	leaf := finalityHeader.HashTreeRoot(tree.GetHashFn())
	root := attestedHeader.StateRoot
	return merkle.VerifyMerkleBranch(leaf, finalityBranch[:], 6, 41, root)
}

func IsNextCommitteeProofValid(attestedHeader common.BeaconBlockHeader, nextCommittee common.SyncCommittee, nextCommitteeBranch altair.SyncCommitteeProofBranch) bool {
	leaf := nextCommittee.HashTreeRoot(configs.Mainnet, tree.GetHashFn())
	root := attestedHeader.StateRoot
	return merkle.VerifyMerkleBranch(leaf, nextCommitteeBranch[:], 5, 23, root)
}

func GetParticipatingKeys(committee common.SyncCommittee, syncBits altair.SyncCommitteeBits) []common.BLSPubkey {
	bits := bitfield.Bitlist(syncBits)
	res := make([]common.BLSPubkey, 0, bits.Count())
	for i := 0; i < int(bits.Len()); i++ {
		if bits.BitAt(uint64(i)) {
			res = append(res, committee.Pubkeys[i])
		}
	}
	return res
}
