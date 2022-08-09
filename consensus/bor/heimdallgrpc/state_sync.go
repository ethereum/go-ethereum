package heimdallgrpc

import (
	"context"

	proto "github.com/maticnetwork/polyproto/heimdall"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/bor/clerk"
)

func (h *HeimdallGRPCClient) StateSyncEvents(ctx context.Context, fromID uint64, to int64) ([]*clerk.EventRecordWithTime, error) {
	eventRecords := make([]*clerk.EventRecordWithTime, 0)

	req := &proto.StateSyncEventsRequest{
		FromID: fromID,
		ToTime: uint64(to),
		Limit:  uint64(stateFetchLimit),
	}

	var (
		res    proto.Heimdall_StateSyncEventsClient
		events *proto.StateSyncEventsResponse
		err    error
	)

	res, err = h.client.StateSyncEvents(ctx, req)
	if err != nil {
		return nil, err
	}

	for {
		events, err = res.Recv()
		if err != nil {
			break
		}

		for _, event := range events.Result {
			eventRecord := &clerk.EventRecordWithTime{
				EventRecord: clerk.EventRecord{
					ID:       event.ID,
					Contract: common.HexToAddress(event.Contract),
					Data:     common.Hex2Bytes(event.Data[2:]),
					TxHash:   common.HexToHash(event.TxHash),
					LogIndex: event.LogIndex,
					ChainID:  event.ChainID,
				},
				Time: event.Time.AsTime(),
			}
			eventRecords = append(eventRecords, eventRecord)
		}
	}

	return eventRecords, err
}
