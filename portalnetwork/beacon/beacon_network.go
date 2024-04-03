package beacon

import (
	"errors"

	"github.com/ethereum/go-ethereum/beacon/types"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/portalnetwork/storage"
	ssz "github.com/ferranbt/fastssz"
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

func (bn *BeaconNetwork) GetBestUpdatesAndCommittees(firstPeriod, count uint64) ([]*types.LightClientUpdate, []*types.SerializedSyncCommittee, error) {
	lightClientUpdateKey := &LightClientUpdateKey{
		StartPeriod: firstPeriod,
		Count:       count,
	}

	_, err := bn.getContent(LightClientUpdate, lightClientUpdateKey)
	if err != nil {
		return nil, nil, err
	}

	return nil, nil, err
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
