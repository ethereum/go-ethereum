package lightclient

import (
	"bytes"
	"errors"
	"time"

	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/portalnetwork/beacon"
	"github.com/ethereum/go-ethereum/portalnetwork/storage"
	"github.com/protolambda/zrnt/eth2/beacon/common"
	"github.com/protolambda/ztyp/codec"
	"github.com/protolambda/ztyp/tree"
)

var _ ConsensusAPI = &PortalRpc{}

type PortalRpc struct {
	portalProtocol *discover.PortalProtocol
	spec           *common.Spec
}

// ChainID implements ConsensusAPI.
func (p *PortalRpc) ChainID() uint64 {
	return 1
}

// GetCheckpointData implements ConsensusAPI.
func (p *PortalRpc) GetBootstrap(blockRoot tree.Root) (common.SpecObj, error) {
	bootstrapKey := &beacon.LightClientBootstrapKey{
		BlockHash: blockRoot[:],
	}
	contentKeyBytes, err := bootstrapKey.MarshalSSZ()
	if err != nil {
		return nil, err
	}
	contentKey := storage.NewContentKey(beacon.LightClientBootstrap, contentKeyBytes).Encode()
	// Get from local
	contentId := p.portalProtocol.ToContentId(contentKey)
	res, err := p.getContent(contentKey, contentId)
	if err != nil {
		return nil, err
	}
	forkedLightClientBootstrap := &beacon.ForkedLightClientBootstrap{}
	err = forkedLightClientBootstrap.Deserialize(p.spec, codec.NewDecodingReader(bytes.NewReader(res), uint64(len(res))))
	if err != nil {
		return nil, err
	}
	return forkedLightClientBootstrap.Bootstrap, nil
}

// GetFinalityData implements ConsensusAPI.
func (p *PortalRpc) GetFinalityUpdate() (common.SpecObj, error) {
	// Get the finality update for the most recent finalized epoch. We use 0 as the finalized
    // slot because the finalized slot is not known at this point and the protocol is
    // designed to return the most recent which is > 0
	finUpdateKey := &beacon.LightClientFinalityUpdateKey{
		FinalizedSlot: 0,
	}
	contentKeyBytes, err := finUpdateKey.MarshalSSZ()
	if err != nil {
		return nil, err
	}
	contentKey := storage.NewContentKey(beacon.LightClientFinalityUpdate, contentKeyBytes).Encode()
	// Get from local
	contentId := p.portalProtocol.ToContentId(contentKey)
	res, err := p.getContent(contentKey, contentId)
	if err != nil {
		return nil, err
	}
	finalityUpdate := &beacon.ForkedLightClientFinalityUpdate{}
	err = finalityUpdate.Deserialize(p.spec, codec.NewDecodingReader(bytes.NewReader(res), uint64(len(res))))
	if err != nil {
		return nil, err
	}
	return finalityUpdate.LightClientFinalityUpdate, nil
}

// GetOptimisticData implements ConsensusAPI.
func (p *PortalRpc) GetOptimisticUpdate() (common.SpecObj, error) {
	currentSlot := p.spec.TimeToSlot(common.Timestamp(time.Now().Unix()), common.Timestamp(beacon.BeaconGenesisTime))
	optimisticUpdateKey := &beacon.LightClientOptimisticUpdateKey{
		OptimisticSlot: uint64(currentSlot),
	}
	contentKeyBytes, err := optimisticUpdateKey.MarshalSSZ()
	if err != nil {
		return nil, err
	}
	contentKey := storage.NewContentKey(beacon.LightClientOptimisticUpdate, contentKeyBytes).Encode()
	// Get from local
	contentId := p.portalProtocol.ToContentId(contentKey)
	res, err := p.getContent(contentKey, contentId)
	if err != nil {
		return nil, err
	}
	optimisticUpdate := &beacon.ForkedLightClientOptimisticUpdate{}
	err = optimisticUpdate.Deserialize(p.spec, codec.NewDecodingReader(bytes.NewReader(res), uint64(len(res))))
	if err != nil {
		return nil, err
	}
	return optimisticUpdate.LightClientOptimisticUpdate, nil
}

// GetUpdates implements ConsensusAPI.
func (p *PortalRpc) GetUpdates(firstPeriod uint64, count uint64) ([]common.SpecObj, error) {
	lightClientUpdateKey := &beacon.LightClientUpdateKey{
		StartPeriod: firstPeriod,
		Count:       count,
	}
	contentKeyBytes, err := lightClientUpdateKey.MarshalSSZ()
	if err != nil {
		return nil, err
	}
	contentKey := storage.NewContentKey(beacon.LightClientUpdate, contentKeyBytes).Encode()
	// Get from local
	contentId := p.portalProtocol.ToContentId(contentKey)
	data, err := p.getContent(contentKey, contentId)
	if err != nil {
		return nil, err
	}
	var lightClientUpdateRange beacon.LightClientUpdateRange = make([]beacon.ForkedLightClientUpdate, 0)
	err = lightClientUpdateRange.Deserialize(p.spec, codec.NewDecodingReader(bytes.NewReader(data), uint64(len(data))))
	if err != nil {
		return nil, err
	}
	res := make([]common.SpecObj, len(lightClientUpdateRange))

	for i, item := range lightClientUpdateRange {
		res[i] = item.LightClientUpdate
	}
	return res, nil
}

// Name implements ConsensusAPI.
func (p *PortalRpc) Name() string {
	return "portal"
}

func (p *PortalRpc) getContent(contentKey, contentId []byte) ([]byte, error) {
	res, err := p.portalProtocol.Get(contentKey, contentId)
	// other error
	if err != nil && !errors.Is(err, storage.ErrContentNotFound) {
		return nil, err
	}
	if res == nil {
		// Get from remote
		res, _, err = p.portalProtocol.ContentLookup(contentKey, contentId)
		if err != nil {
			return nil, err
		}
	} 
	return res, nil
}
