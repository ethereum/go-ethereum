package heimdallapp

import (
	"context"
	"time"

	"github.com/maticnetwork/heimdall/clerk/types"

	"github.com/ethereum/go-ethereum/consensus/bor/clerk"

	abci "github.com/tendermint/tendermint/abci/types"
)

func (h *HeimdallAppClient) StateSyncEvents(ctx context.Context, fromID uint64, to int64) ([]*clerk.EventRecordWithTime, error) {
	totalRecords := make([]*clerk.EventRecordWithTime, 0)

	hCtx := h.hApp.NewContext(true, abci.Header{Height: h.hApp.LastBlockHeight()})

	for {
		fromRecord, err := h.hApp.ClerkKeeper.GetEventRecord(hCtx, fromID)
		if err != nil {
			return nil, err
		}

		events, err := h.hApp.ClerkKeeper.GetEventRecordListWithTime(hCtx, fromRecord.RecordTime, time.Unix(to, 0), 1, stateFetchLimit)
		if err != nil {
			return nil, err
		}

		totalRecords = append(totalRecords, toEvents(events)...)

		if len(events) < stateFetchLimit {
			break
		}

		fromID += uint64(stateFetchLimit)
	}

	return totalRecords, nil
}

func toEvents(hdEvents []types.EventRecord) []*clerk.EventRecordWithTime {
	events := make([]*clerk.EventRecordWithTime, len(hdEvents))

	for i, ev := range hdEvents {
		events[i] = toEvent(ev)
	}

	return events
}

func toEvent(hdEvent types.EventRecord) *clerk.EventRecordWithTime {
	return &clerk.EventRecordWithTime{
		EventRecord: clerk.EventRecord{
			ID:       hdEvent.ID,
			Contract: hdEvent.Contract.EthAddress(),
			Data:     hdEvent.Data.Bytes(),
			TxHash:   hdEvent.TxHash.EthHash(),
			LogIndex: hdEvent.LogIndex,
			ChainID:  hdEvent.ChainID,
		},
		Time: hdEvent.RecordTime,
	}
}
