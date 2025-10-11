// Copyright 2025 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more detaiapi.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"sync/atomic"

	"github.com/donovanhide/eventsource"
	"github.com/ethereum/go-ethereum/beacon/light"
	"github.com/ethereum/go-ethereum/beacon/light/request"
	"github.com/ethereum/go-ethereum/beacon/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/lru"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/restapi"
	"github.com/gorilla/mux"
)

type BeaconApiServer struct {
	scheduler       *request.Scheduler
	checkpointStore *light.CheckpointStore
	committeeChain  *light.CommitteeChain
	headTracker     *light.HeadTracker
	getBeaconBlock  func(common.Hash) *types.BeaconBlock
	execBlocks      *lru.Cache[common.Hash, struct{}] // execution block root -> processed flag
	eventServer     *eventsource.Server
	closeCh         chan struct{}

	lastEventId    uint64
	lastHeadInfo   types.HeadInfo
	lastOptimistic types.OptimisticUpdate
	lastFinality   types.FinalityUpdate
}

type ExecChain interface {
	SubscribeChainEvent(ch chan<- core.ChainEvent) event.Subscription
}

func NewBeaconApiServer(
	scheduler *request.Scheduler,
	checkpointStore *light.CheckpointStore,
	committeeChain *light.CommitteeChain,
	headTracker *light.HeadTracker,
	getBeaconBlock func(common.Hash) *types.BeaconBlock,
	execChain ExecChain) *BeaconApiServer {

	eventServer := eventsource.NewServer()
	eventServer.Register("headEvent", eventsource.NewSliceRepository())
	s := &BeaconApiServer{
		scheduler:       scheduler,
		checkpointStore: checkpointStore,
		committeeChain:  committeeChain,
		headTracker:     headTracker,
		getBeaconBlock:  getBeaconBlock,
		eventServer:     eventServer,
		closeCh:         make(chan struct{}),
	}
	if execChain != nil {
		s.execBlocks = lru.NewCache[common.Hash, struct{}](100)
		ch := make(chan core.ChainEvent, 1)
		sub := execChain.SubscribeChainEvent(ch)
		go func() {
			defer sub.Unsubscribe()
			for {
				select {
				case ev := <-ch:
					s.execBlocks.Add(ev.Header.Hash(), struct{}{})
					s.scheduler.Trigger()
				case <-s.closeCh:
					return
				}
			}
		}()
	}
	return s
}

func (s *BeaconApiServer) Stop() {
	close(s.closeCh)
}

func (s *BeaconApiServer) RestAPI(server *restapi.Server) restapi.API {
	return func(router *mux.Router) {
		router.HandleFunc("/eth/v1/beacon/light_client/updates", server.WrapHandler(s.handleUpdates, false, false, false)).Methods("GET")
		router.HandleFunc("/eth/v1/beacon/light_client/optimistic_update", server.WrapHandler(s.handleOptimisticUpdate, false, false, false)).Methods("GET")
		router.HandleFunc("/eth/v1/beacon/light_client/finality_update", server.WrapHandler(s.handleFinalityUpdate, false, false, false)).Methods("GET")
		router.HandleFunc("/eth/v1/beacon/headers/head", server.WrapHandler(s.handleHeadHeader, false, false, false)).Methods("GET")
		router.HandleFunc("/eth/v1/beacon/light_client/bootstrap/{checkpointhash}", server.WrapHandler(s.handleBootstrap, false, false, false)).Methods("GET")
		router.HandleFunc("/eth/v2/beacon/blocks/{blockhash}", server.WrapHandler(s.handleBlocks, false, false, false)).Methods("GET")
		router.HandleFunc("/eth/v1/events", s.eventServer.Handler("headEvent"))
	}
}

func (s *BeaconApiServer) Process(requester request.Requester, events []request.Event) {
	if head := s.headTracker.PrefetchHead(); head != s.lastHeadInfo && s.getBeaconBlock(head.BlockRoot) != nil {
		s.lastHeadInfo = head
		s.publishHeadEvent(head)
	}
	if vh, ok := s.headTracker.ValidatedOptimistic(); ok && vh.Attested.Header != s.lastOptimistic.Attested.Header && s.canPublish(vh.Attested) {
		s.lastOptimistic = vh
		s.publishOptimisticUpdate(vh)
	}
	if fh, ok := s.headTracker.ValidatedFinality(); ok && fh.Finalized.Header != s.lastFinality.Finalized.Header && s.canPublish(fh.Attested) {
		s.lastFinality = fh
		s.publishFinalityUpdate(fh)
	}
}

func (s *BeaconApiServer) canPublish(header types.HeaderWithExecProof) bool {
	if s.getBeaconBlock(header.Hash()) == nil {
		return false
	}
	if s.execBlocks != nil {
		if _, ok := s.execBlocks.Get(header.PayloadHeader.BlockHash()); !ok {
			return false
		}
	}
	return true
}

func (s *BeaconApiServer) publishHeadEvent(headInfo types.HeadInfo) {
	enc, err := json.Marshal(&jsonHeadEvent{Slot: common.Decimal(headInfo.Slot), Block: headInfo.BlockRoot})
	if err != nil {
		log.Error("Error encoding head event", "error", err)
		return
	}
	s.publishEvent("head", string(enc))
}

func (s *BeaconApiServer) publishOptimisticUpdate(update types.OptimisticUpdate) {
	enc, err := encodeOptimisticUpdate(update)
	if err != nil {
		log.Error("Error encoding optimistic head update", "error", err)
		return
	}
	s.publishEvent("light_client_optimistic_update", string(enc))
}

func (s *BeaconApiServer) publishFinalityUpdate(update types.FinalityUpdate) {
	enc, err := encodeFinalityUpdate(update)
	if err != nil {
		log.Error("Error encoding optimistic head update", "error", err)
		return
	}
	s.publishEvent("light_client_finality_update", string(enc))
}

type serverEvent struct {
	id, event, data string
}

func (e *serverEvent) Id() string    { return e.id }
func (e *serverEvent) Event() string { return e.event }
func (e *serverEvent) Data() string  { return e.data }

func (s *BeaconApiServer) publishEvent(event, data string) {
	id := atomic.AddUint64(&s.lastEventId, 1)
	s.eventServer.Publish([]string{"headEvent"}, &serverEvent{
		id:    strconv.FormatUint(id, 10),
		event: event,
		data:  data,
	})
}

func (s *BeaconApiServer) handleUpdates(ctx context.Context, values url.Values, vars map[string]string, decodeBody func(*any) error) (any, string, int) {
	startStr, countStr := values.Get("start_period"), values.Get("count")
	start, err := strconv.ParseUint(startStr, 10, 64)
	if err != nil {
		return nil, "invalid start_period parameter", http.StatusBadRequest
	}
	var count uint64
	if countStr != "" {
		count, err = strconv.ParseUint(countStr, 10, 64)
		if err != nil {
			return nil, "invalid count parameter", http.StatusBadRequest
		}
	} else {
		count = 1
	}
	var updates []CommitteeUpdate
	for period := start; period < start+count; period++ {
		update := s.committeeChain.GetUpdate(period)
		if update == nil {
			continue
		}
		committee := s.committeeChain.GetCommittee(period + 1)
		if committee == nil {
			continue
		}
		updates = append(updates, CommitteeUpdate{
			Update:            *update,
			NextSyncCommittee: *committee,
		})
	}
	return updates, "", 0
}

func (s *BeaconApiServer) handleOptimisticUpdate(ctx context.Context, values url.Values, vars map[string]string, decodeBody func(*any) error) (any, string, int) {
	if s.lastOptimistic.Attested.Header == (types.Header{}) {
		return nil, "no optimistic update available", http.StatusNotFound
	}
	update, err := encodeOptimisticUpdate(s.lastOptimistic)
	if err != nil {
		return nil, "error encoding optimistic update", http.StatusInternalServerError
	}
	return json.RawMessage(update), "", 0
}

func (s *BeaconApiServer) handleFinalityUpdate(ctx context.Context, values url.Values, vars map[string]string, decodeBody func(*any) error) (any, string, int) {
	if s.lastFinality.Attested.Header == (types.Header{}) {
		return nil, "no finality update available", http.StatusNotFound
	}
	update, err := encodeFinalityUpdate(s.lastFinality)
	if err != nil {
		return nil, "error encoding finality update", http.StatusInternalServerError
	}
	return json.RawMessage(update), "", 0
}

func (s *BeaconApiServer) handleHeadHeader(ctx context.Context, values url.Values, vars map[string]string, decodeBody func(*any) error) (any, string, int) {
	block := s.getBeaconBlock(s.lastHeadInfo.BlockRoot)
	if block == nil {
		return nil, "unknown head block", http.StatusNotFound
	}
	header := block.Header()
	var headerData jsonHeaderData
	headerData.ExecutionOptimistic = block.ExecutionOptimistic
	headerData.Finalized = block.Finalized
	headerData.Data.Canonical = true
	headerData.Data.Header.Message = header
	headerData.Data.Header.Signature = block.Data.Signature
	headerData.Data.Root = header.Hash()
	return headerData, "", 0
}

func (s *BeaconApiServer) handleBootstrap(ctx context.Context, values url.Values, vars map[string]string, decodeBody func(*any) error) (any, string, int) {
	hex, err := hexutil.Decode(vars["checkpointhash"])
	if err != nil || len(hex) != common.HashLength {
		return nil, "invalid checkpoint hash", http.StatusBadRequest
	}
	var checkpointHash common.Hash
	copy(checkpointHash[:], hex)
	checkpoint := s.checkpointStore.Get(checkpointHash)
	if checkpoint == nil {
		return nil, "unknown checkpoint", http.StatusNotFound
	}
	var response jsonBootstrapData
	response.Version = checkpoint.Version
	response.Data.Header.Beacon = checkpoint.Header
	response.Data.CommitteeBranch = checkpoint.CommitteeBranch
	response.Data.Committee = checkpoint.Committee
	return response, "", 0
}

func (s *BeaconApiServer) handleBlocks(ctx context.Context, values url.Values, vars map[string]string, decodeBody func(*any) error) (any, string, int) {
	hex, err := hexutil.Decode(vars["blockhash"])
	if err != nil || len(hex) != common.HashLength {
		return nil, "invalid block hash", http.StatusBadRequest
	}
	var blockHash common.Hash
	copy(blockHash[:], hex)
	block := s.getBeaconBlock(blockHash)
	if block == nil {
		return nil, "unknown beacon block", http.StatusNotFound
	}
	return block, "", 0
}
