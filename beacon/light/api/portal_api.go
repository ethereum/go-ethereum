package api

import (
	"errors"

	"github.com/ethereum/go-ethereum/beacon/types"
)

type PortalLightApi struct {
}

func NewPortalLightApi() *PortalLightApi {
	return &PortalLightApi{}
}

func (api *PortalLightApi) GetBestUpdatesAndCommittees(firstPeriod, count uint64) ([]*types.LightClientUpdate, []*types.SerializedSyncCommittee, error) {
	//contentKey := &beacon.LightClientUpdateKey{
	//	StartPeriod: firstPeriod,
	//	Count:       count,
	//}

	//resp, err := api.httpGetf("/eth/v1/beacon/light_client/updates?start_period=%d&count=%d", firstPeriod, count)
	//if err != nil {
	//	return nil, nil, err
	//}
	//
	//var data []CommitteeUpdate
	//if err := json.Unmarshal(resp, &data); err != nil {
	//	return nil, nil, err
	//}
	//if len(data) != int(count) {
	//	return nil, nil, errors.New("invalid number of committee updates")
	//}
	//updates := make([]*types.LightClientUpdate, int(count))
	//committees := make([]*types.SerializedSyncCommittee, int(count))
	//for i, d := range data {
	//	if d.Update.AttestedHeader.Header.SyncPeriod() != firstPeriod+uint64(i) {
	//		return nil, nil, errors.New("wrong committee update header period")
	//	}
	//	if err := d.Update.Validate(); err != nil {
	//		return nil, nil, err
	//	}
	//	if d.NextSyncCommittee.Root() != d.Update.NextSyncCommitteeRoot {
	//		return nil, nil, errors.New("wrong sync committee root")
	//	}
	//	updates[i], committees[i] = new(types.LightClientUpdate), new(types.SerializedSyncCommittee)
	//	*updates[i], *committees[i] = d.Update, d.NextSyncCommittee
	//}
	//return updates, committees, nil

	return nil, nil, errors.New("not implemented")
}
