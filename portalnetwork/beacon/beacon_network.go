package beacon

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/portalnetwork/storage"
	ssz "github.com/ferranbt/fastssz"
	"github.com/protolambda/zrnt/eth2/beacon/capella"
	"github.com/protolambda/zrnt/eth2/beacon/common"
	"github.com/protolambda/zrnt/eth2/configs"
	"github.com/protolambda/ztyp/codec"
	"github.com/protolambda/ztyp/tree"
)

const (
	LightClientBootstrap        storage.ContentType = 0x10
	LightClientUpdate           storage.ContentType = 0x11
	LightClientFinalityUpdate   storage.ContentType = 0x12
	LightClientOptimisticUpdate storage.ContentType = 0x13
	HistoricalSummaries         storage.ContentType = 0x14
	BeaconGenesisTime           uint64              = 1606824023
)

type BeaconNetwork struct {
	portalProtocol *discover.PortalProtocol
	spec           *common.Spec
	log            log.Logger
	closeCtx       context.Context
	closeFunc      context.CancelFunc
}

func NewBeaconNetwork(portalProtocol *discover.PortalProtocol) *BeaconNetwork {
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

func (bn *BeaconNetwork) GetUpdates(firstPeriod, count uint64) ([]*capella.LightClientUpdate, error) {
	lightClientUpdateKey := &LightClientUpdateKey{
		StartPeriod: firstPeriod,
		Count:       count,
	}

	lightClientUpdateRangeContent, err := bn.getContent(LightClientUpdate, lightClientUpdateKey)
	if err != nil {
		return nil, err
	}

	var lightClientUpdateRange LightClientUpdateRange = make([]ForkedLightClientUpdate, 0)
	err = lightClientUpdateRange.Deserialize(bn.spec, codec.NewDecodingReader(bytes.NewReader(lightClientUpdateRangeContent), uint64(len(lightClientUpdateRangeContent))))
	if err != nil {
		return nil, err
	}

	updates := make([]*capella.LightClientUpdate, len(lightClientUpdateRange))
	for i, update := range lightClientUpdateRange {
		if update.ForkDigest != Capella {
			return nil, errors.New("unknown fork digest")
		}
		updates[i] = update.LightClientUpdate.(*capella.LightClientUpdate)
	}
	return updates, nil
}

func (bn *BeaconNetwork) GetCheckpointData(checkpointHash tree.Root) (*capella.LightClientBootstrap, error) {
	bootstrapKey := &LightClientBootstrapKey{
		BlockHash: checkpointHash[:],
	}

	bootstrapValue, err := bn.getContent(LightClientBootstrap, bootstrapKey)
	if err != nil {
		return nil, err
	}

	var forkedLightClientBootstrap ForkedLightClientBootstrap
	err = forkedLightClientBootstrap.Deserialize(bn.spec, codec.NewDecodingReader(bytes.NewReader(bootstrapValue), uint64(len(bootstrapValue))))
	if err != nil {
		return nil, err
	}

	if forkedLightClientBootstrap.ForkDigest != Capella {
		return nil, errors.New("unknown fork digest")
	}

	return forkedLightClientBootstrap.Bootstrap.(*capella.LightClientBootstrap), nil
}

func (bn *BeaconNetwork) GetFinalityUpdate(finalizedSlot uint64) (*capella.LightClientFinalityUpdate, error) {
	finalityUpdateKey := &LightClientFinalityUpdateKey{
		FinalizedSlot: finalizedSlot,
	}

	finalityUpdateValue, err := bn.getContent(LightClientFinalityUpdate, finalityUpdateKey)
	if err != nil {
		return nil, err
	}

	var forkedLightClientFinalityUpdate ForkedLightClientFinalityUpdate
	err = forkedLightClientFinalityUpdate.Deserialize(bn.spec, codec.NewDecodingReader(bytes.NewReader(finalityUpdateValue), uint64(len(finalityUpdateValue))))
	if err != nil {
		return nil, err
	}

	if forkedLightClientFinalityUpdate.ForkDigest != Capella {
		return nil, errors.New("unknown fork digest")
	}

	return forkedLightClientFinalityUpdate.LightClientFinalityUpdate.(*capella.LightClientFinalityUpdate), nil
}

func (bn *BeaconNetwork) GetOptimisticUpdate(optimisticSlot uint64) (*capella.LightClientOptimisticUpdate, error) {
	optimisticUpdateKey := &LightClientOptimisticUpdateKey{
		OptimisticSlot: optimisticSlot,
	}

	optimisticUpdateValue, err := bn.getContent(LightClientOptimisticUpdate, optimisticUpdateKey)
	if err != nil {
		return nil, err
	}

	var forkedLightClientOptimisticUpdate ForkedLightClientOptimisticUpdate
	err = forkedLightClientOptimisticUpdate.Deserialize(bn.spec, codec.NewDecodingReader(bytes.NewReader(optimisticUpdateValue), uint64(len(optimisticUpdateValue))))
	if err != nil {
		return nil, err
	}

	if forkedLightClientOptimisticUpdate.ForkDigest != Capella {
		return nil, errors.New("unknown fork digest")
	}

	return forkedLightClientOptimisticUpdate.LightClientOptimisticUpdate.(*capella.LightClientOptimisticUpdate), nil
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
		return lightClientUpdateRange.Deserialize(bn.spec, codec.NewDecodingReader(bytes.NewReader(content), uint64(len(content))))
	case LightClientBootstrap:
		var forkedLightClientBootstrap ForkedLightClientBootstrap
		return forkedLightClientBootstrap.Deserialize(bn.spec, codec.NewDecodingReader(bytes.NewReader(content), uint64(len(content))))
	// TODO: IF WE NEED LIGHT CLIENT VERIFY
	case LightClientFinalityUpdate:
		var forkedLightClientFinalityUpdate ForkedLightClientFinalityUpdate
		return forkedLightClientFinalityUpdate.Deserialize(bn.spec, codec.NewDecodingReader(bytes.NewReader(content), uint64(len(content))))
	// TODO: IF WE NEED LIGHT CLIENT VERIFY
	case LightClientOptimisticUpdate:
		var forkedLightClientOptimisticUpdate ForkedLightClientOptimisticUpdate
		return forkedLightClientOptimisticUpdate.Deserialize(bn.spec, codec.NewDecodingReader(bytes.NewReader(content), uint64(len(content))))
	// TODO: VERIFY
	case HistoricalSummaries:
		var historicalSummaries HistoricalSummariesProof
		return historicalSummaries.Deserialize(codec.NewDecodingReader(bytes.NewReader(content), uint64(len(content))))
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
