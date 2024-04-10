package beacon

import (
	"bytes"
	"errors"

	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/portalnetwork/storage"
	ssz "github.com/ferranbt/fastssz"
	"github.com/protolambda/zrnt/eth2/beacon/capella"
	"github.com/protolambda/zrnt/eth2/beacon/common"
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
	PortalProtocol *discover.PortalProtocol
	Spec           *common.Spec
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
	err = lightClientUpdateRange.Deserialize(bn.Spec, codec.NewDecodingReader(bytes.NewReader(lightClientUpdateRangeContent), uint64(len(lightClientUpdateRangeContent))))
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
	err = forkedLightClientBootstrap.Deserialize(bn.Spec, codec.NewDecodingReader(bytes.NewReader(bootstrapValue), uint64(len(bootstrapValue))))
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
	err = forkedLightClientFinalityUpdate.Deserialize(bn.Spec, codec.NewDecodingReader(bytes.NewReader(finalityUpdateValue), uint64(len(finalityUpdateValue))))
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
	err = forkedLightClientOptimisticUpdate.Deserialize(bn.Spec, codec.NewDecodingReader(bytes.NewReader(optimisticUpdateValue), uint64(len(optimisticUpdateValue))))
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
	contentId := bn.PortalProtocol.ToContentId(contentKey)

	res, err := bn.PortalProtocol.Get(contentKey, contentId)
	// other error
	if err != nil && !errors.Is(err, storage.ErrContentNotFound) {
		return nil, err
	}

	if res != nil {
		return res, nil
	}

	content, _, err := bn.PortalProtocol.ContentLookup(contentKey, contentId)
	if err != nil {
		return nil, err
	}

	return content, nil
}
