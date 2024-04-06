package beacon

import (
	"bytes"
	"errors"

	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/portalnetwork/storage"
	ssz "github.com/ferranbt/fastssz"
	"github.com/protolambda/zrnt/eth2/beacon/capella"
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
	portalProtocol *discover.PortalProtocol
}

func (bn *BeaconNetwork) GetBestUpdatesAndCommittees(firstPeriod, count uint64) (LightClientUpdateRange, error) {
	lightClientUpdateKey := &LightClientUpdateKey{
		StartPeriod: firstPeriod,
		Count:       count,
	}

	lightClientUpdateRangeContent, err := bn.getContent(LightClientUpdate, lightClientUpdateKey)
	if err != nil {
		return nil, err
	}

	var lightClientUpdateRange LightClientUpdateRange = make([]ForkedLightClientUpdate, 0)
	err = lightClientUpdateRange.Deserialize(codec.NewDecodingReader(bytes.NewReader(lightClientUpdateRangeContent), uint64(len(lightClientUpdateRangeContent))))
	if err != nil {
		return nil, err
	}

	return lightClientUpdateRange, nil
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
	err = forkedLightClientBootstrap.Deserialize(codec.NewDecodingReader(bytes.NewReader(bootstrapValue), uint64(len(bootstrapValue))))
	if err != nil {
		return nil, err
	}

	if forkedLightClientBootstrap.ForkDigest != Capella {
		return nil, errors.New("unknown fork digest")
	}

	return forkedLightClientBootstrap.Bootstrap.(*capella.LightClientBootstrap), nil
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
