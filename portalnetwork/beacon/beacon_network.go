package beacon

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/portalnetwork/portalwire"
	"github.com/ethereum/go-ethereum/portalnetwork/storage"
	ssz "github.com/ferranbt/fastssz"
	"github.com/protolambda/zrnt/eth2/beacon/common"
	"github.com/protolambda/zrnt/eth2/configs"
	"github.com/protolambda/zrnt/eth2/util/merkle"
	"github.com/protolambda/ztyp/codec"
	"github.com/protolambda/ztyp/tree"
)

const (
	LightClientBootstrap        storage.ContentType = 0x10
	LightClientUpdate           storage.ContentType = 0x11
	LightClientFinalityUpdate   storage.ContentType = 0x12
	LightClientOptimisticUpdate storage.ContentType = 0x13
	HistoricalSummaries         storage.ContentType = 0x14
)

type BeaconNetwork struct {
	portalProtocol *portalwire.PortalProtocol
	spec           *common.Spec
	log            log.Logger
	closeCtx       context.Context
	closeFunc      context.CancelFunc
	lightClient    *ConsensusLightClient
}

func NewBeaconNetwork(portalProtocol *portalwire.PortalProtocol) *BeaconNetwork {
	ctx, cancel := context.WithCancel(context.Background())

	return &BeaconNetwork{
		portalProtocol: portalProtocol,
		spec:           configs.Mainnet,
		closeCtx:       ctx,
		closeFunc:      cancel,
		log:            log.New("sub-protocol", "beacon"),
	}
}

func (bn *BeaconNetwork) Start() error {
	err := bn.portalProtocol.Start()
	if err != nil {
		return err
	}
	go bn.processContentLoop(bn.closeCtx)
	bn.log.Debug("beacon network start successfully")
	return nil
}

func (bn *BeaconNetwork) Stop() {
	bn.closeFunc()
	bn.portalProtocol.Stop()
}

func (bn *BeaconNetwork) GetUpdates(firstPeriod, count uint64) ([]common.SpecObj, error) {
	lightClientUpdateKey := &LightClientUpdateKey{
		StartPeriod: firstPeriod,
		Count:       count,
	}

	data, err := bn.getContent(LightClientUpdate, lightClientUpdateKey)
	if err != nil {
		return nil, err
	}
	var lightClientUpdateRange LightClientUpdateRange = make([]ForkedLightClientUpdate, 0)
	err = lightClientUpdateRange.Deserialize(bn.spec, codec.NewDecodingReader(bytes.NewReader(data), uint64(len(data))))
	if err != nil {
		return nil, err
	}
	res := make([]common.SpecObj, len(lightClientUpdateRange))

	for i, item := range lightClientUpdateRange {
		res[i] = item.LightClientUpdate
	}
	return res, nil
}

func (bn *BeaconNetwork) GetCheckpointData(checkpointHash tree.Root) (common.SpecObj, error) {
	bootstrapKey := &LightClientBootstrapKey{
		BlockHash: checkpointHash[:],
	}

	data, err := bn.getContent(LightClientBootstrap, bootstrapKey)
	if err != nil {
		return nil, err
	}

	var forkedLightClientBootstrap *ForkedLightClientBootstrap
	err = forkedLightClientBootstrap.Deserialize(bn.spec, codec.NewDecodingReader(bytes.NewReader(data), uint64(len(data))))
	if err != nil {
		return nil, err
	}
	return forkedLightClientBootstrap.Bootstrap, nil
}

func (bn *BeaconNetwork) GetFinalityUpdate(finalizedSlot uint64) (common.SpecObj, error) {
	finalityUpdateKey := &LightClientFinalityUpdateKey{
		FinalizedSlot: finalizedSlot,
	}
	data, err := bn.getContent(LightClientFinalityUpdate, finalityUpdateKey)
	if err != nil {
		return nil, err
	}

	var forkedLightClientFinalityUpdate *ForkedLightClientFinalityUpdate
	err = forkedLightClientFinalityUpdate.Deserialize(bn.spec, codec.NewDecodingReader(bytes.NewReader(data), uint64(len(data))))
	if err != nil {
		return nil, err
	}

	return forkedLightClientFinalityUpdate.LightClientFinalityUpdate, nil
}

func (bn *BeaconNetwork) GetOptimisticUpdate(optimisticSlot uint64) (common.SpecObj, error) {
	optimisticUpdateKey := &LightClientOptimisticUpdateKey{
		OptimisticSlot: optimisticSlot,
	}

	data, err := bn.getContent(LightClientOptimisticUpdate, optimisticUpdateKey)
	if err != nil {
		return nil, err
	}

	var forkedLightClientOptimisticUpdate *ForkedLightClientOptimisticUpdate
	err = forkedLightClientOptimisticUpdate.Deserialize(bn.spec, codec.NewDecodingReader(bytes.NewReader(data), uint64(len(data))))
	if err != nil {
		return nil, err
	}

	return forkedLightClientOptimisticUpdate.LightClientOptimisticUpdate, nil
}

func (bn *BeaconNetwork) getContent(contentType storage.ContentType, beaconContentKey ssz.Marshaler) ([]byte, error) {
	contentKeyBytes, err := beaconContentKey.MarshalSSZ()
	if err != nil {
		return nil, err
	}

	contentKey := storage.NewContentKey(contentType, contentKeyBytes).Encode()
	contentId := bn.portalProtocol.ToContentId(contentKey)

	res, err := bn.portalProtocol.Get(contentKey, contentId)
	// other error
	if err != nil && !errors.Is(err, storage.ErrContentNotFound) {
		return nil, err
	}

	if res != nil {
		return res, nil
	}

	content, _, err := bn.portalProtocol.ContentLookup(contentKey, contentId)
	if err != nil {
		return nil, err
	}

	return content, nil
}

func (bn *BeaconNetwork) validateContent(contentKey []byte, content []byte) error {
	switch storage.ContentType(contentKey[0]) {
	case LightClientUpdate:
		var lightClientUpdateRange LightClientUpdateRange = make([]ForkedLightClientUpdate, 0)
		err := lightClientUpdateRange.Deserialize(bn.spec, codec.NewDecodingReader(bytes.NewReader(content), uint64(len(content))))
		if err != nil {
			return err
		}
		lightClientUpdateKey := &LightClientUpdateKey{}
		err = lightClientUpdateKey.UnmarshalSSZ(contentKey[1:])
		if err != nil {
			return err
		}
		if lightClientUpdateKey.Count != uint64(len(lightClientUpdateRange)) {
			return fmt.Errorf("light client updates count does not match the content key count: %d != %d", len(lightClientUpdateRange), lightClientUpdateKey.Count)
		}
		return nil
	case LightClientBootstrap:
		var forkedLightClientBootstrap ForkedLightClientBootstrap
		err := forkedLightClientBootstrap.Deserialize(bn.spec, codec.NewDecodingReader(bytes.NewReader(content), uint64(len(content))))
		if err != nil {
			return err
		}
		currentSlot := bn.spec.TimeToSlot(common.Timestamp(time.Now().Unix()), common.Timestamp(BeaconGenesisTime))

		genericBootstrap, err := FromBootstrap(forkedLightClientBootstrap.Bootstrap)
		if err != nil {
			return err
		}
		fourMonth := time.Hour * 24 * 30 * 4
		fourMonthInSlots := common.Timestamp(fourMonth.Seconds()) / (bn.spec.SECONDS_PER_SLOT)
		fourMonthAgoSlot := currentSlot - common.Slot(fourMonthInSlots)

		if genericBootstrap.Header.Slot < fourMonthAgoSlot {
			return fmt.Errorf("light client bootstrap slot is too old: %d", genericBootstrap.Header.Slot)
		}
		return nil
	case LightClientFinalityUpdate:
		lightClientFinalityUpdateKey := &LightClientFinalityUpdateKey{}
		err := lightClientFinalityUpdateKey.UnmarshalSSZ(contentKey[1:])
		if err != nil {
			return err
		}
		var forkedLightClientFinalityUpdate ForkedLightClientFinalityUpdate
		err = forkedLightClientFinalityUpdate.Deserialize(bn.spec, codec.NewDecodingReader(bytes.NewReader(content), uint64(len(content))))
		if err != nil {
			return err
		}
		if forkedLightClientFinalityUpdate.ForkDigest != Deneb {
			return fmt.Errorf("light client finality update is not from the recent fork. Expected deneb, got %v", forkedLightClientFinalityUpdate.ForkDigest)
		}
		finalizedSlot := lightClientFinalityUpdateKey.FinalizedSlot
		genericUpdate, err := FromLightClientFinalityUpdate(forkedLightClientFinalityUpdate.LightClientFinalityUpdate)
		if err != nil {
			return err
		}
		if finalizedSlot != uint64(genericUpdate.FinalizedHeader.Slot) {
			return fmt.Errorf("light client finality update finalized slot does not match the content key finalized slot: %d != %d", genericUpdate.FinalizedHeader.Slot, finalizedSlot)
		}
		return nil
	case LightClientOptimisticUpdate:
		lightClientOptimisticUpdateKey := &LightClientOptimisticUpdateKey{}
		err := lightClientOptimisticUpdateKey.UnmarshalSSZ(contentKey[1:])
		if err != nil {
			return err
		}
		var forkedLightClientOptimisticUpdate ForkedLightClientOptimisticUpdate
		err = forkedLightClientOptimisticUpdate.Deserialize(bn.spec, codec.NewDecodingReader(bytes.NewReader(content), uint64(len(content))))
		if err != nil {
			return err
		}
		if forkedLightClientOptimisticUpdate.ForkDigest != Deneb {
			return fmt.Errorf("light client optimistic update is not from the recent fork. Expected deneb, got %v", forkedLightClientOptimisticUpdate.ForkDigest)
		}
		genericUpdate, err := FromLightClientOptimisticUpdate(forkedLightClientOptimisticUpdate.LightClientOptimisticUpdate)
		if err != nil {
			return err
		}
		// Check if key signature slot matches the light client optimistic update signature slot
		if lightClientOptimisticUpdateKey.OptimisticSlot != uint64(genericUpdate.SignatureSlot) {
			return fmt.Errorf("light client optimistic update signature slot does not match the content key signature slot: %d != %d", genericUpdate.SignatureSlot, lightClientOptimisticUpdateKey.OptimisticSlot)
		}
		return nil
	// TODO: VERIFY
	case HistoricalSummaries:
		forkedHistoricalSummariesWithProof, err := bn.generalSummariesValidation(contentKey, content)
		if err != nil {
			return err
		}
		// TODO get root from light client
		header := bn.lightClient.GetFinalityHeader()
		latestFinalizedRoot := header.StateRoot

		valid := bn.stateSummariesValidation(*forkedHistoricalSummariesWithProof, latestFinalizedRoot)
		if !valid {
			return errors.New("merkle proof validation failed for HistoricalSummariesProof")
		}
		return nil
	default:
		return fmt.Errorf("unknown content type %v", contentKey[0])
	}
}

func (bn *BeaconNetwork) validateContents(contentKeys [][]byte, contents [][]byte) error {
	for i, content := range contents {
		contentKey := contentKeys[i]
		err := bn.validateContent(contentKey, content)
		if err != nil {
			bn.log.Error("content validate failed", "contentKey", hexutil.Encode(contentKey), "content", hexutil.Encode(content), "err", err)
			return fmt.Errorf("content validate failed with content key %x and content %x", contentKey, content)
		}
		contentId := bn.portalProtocol.ToContentId(contentKey)
		err = bn.portalProtocol.Put(contentKey, contentId, content)
		if err != nil {
			bn.log.Error("put content failed", "contentKey", hexutil.Encode(contentKey), "content", hexutil.Encode(content), "err", err)
			return err
		}
	}
	return nil
}

func (bn *BeaconNetwork) processContentLoop(ctx context.Context) {
	contentChan := bn.portalProtocol.GetContent()
	for {
		select {
		case <-ctx.Done():
			return
		case contentElement := <-contentChan:
			err := bn.validateContents(contentElement.ContentKeys, contentElement.Contents)
			if err != nil {
				bn.log.Error("validate content failed", "err", err)
				continue
			}
			go func(ctx context.Context) {
				select {
				case <-ctx.Done():
					return
				default:
					var gossippedNum int
					gossippedNum, err = bn.portalProtocol.Gossip(&contentElement.Node, contentElement.ContentKeys, contentElement.Contents)
					bn.log.Trace("gossippedNum", "gossippedNum", gossippedNum)
					if err != nil {
						bn.log.Error("gossip failed", "err", err)
						return
					}
				}
			}(ctx)
		}
	}
}

func (bn *BeaconNetwork) generalSummariesValidation(contentKey, content []byte) (*ForkedHistoricalSummariesWithProof, error) {
	key := &HistoricalSummariesWithProofKey{}
	err := key.Deserialize(codec.NewDecodingReader(bytes.NewReader(contentKey[1:]), uint64(len(contentKey[1:]))))
	if err != nil {
		return nil, err
	}
	forkedHistoricalSummariesWithProof := &ForkedHistoricalSummariesWithProof{}
	err = forkedHistoricalSummariesWithProof.Deserialize(bn.spec, codec.NewDecodingReader(bytes.NewReader(content), uint64(len(content))))
	if err != nil {
		return nil, err
	}
	if forkedHistoricalSummariesWithProof.HistoricalSummariesWithProof.EPOCH != common.Epoch(key.Epoch) {
		return nil, fmt.Errorf("historical summaries with proof epoch does not match the content key epoch: %d != %d", forkedHistoricalSummariesWithProof.HistoricalSummariesWithProof.EPOCH, key.Epoch)
	}
	return forkedHistoricalSummariesWithProof, nil
}

func (bn *BeaconNetwork) stateSummariesValidation(f ForkedHistoricalSummariesWithProof, latestFinalizedRoot common.Root) bool {
	proof := f.HistoricalSummariesWithProof.Proof
	summariesRoot := f.HistoricalSummariesWithProof.HistoricalSummaries.HashTreeRoot(bn.spec, tree.GetHashFn())

	gIndex := 59
	return merkle.VerifyMerkleBranch(summariesRoot, proof.Proof[:], 5, uint64(gIndex), latestFinalizedRoot)
}
