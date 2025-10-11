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
	"net/url"
	"strconv"
	"sync/atomic"

	"github.com/donovanhide/eventsource"
	"github.com/ethereum/go-ethereum/beacon/light"
	"github.com/ethereum/go-ethereum/beacon/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/lru"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/restapi"
	"github.com/gorilla/mux"
)

type BeaconApiServer struct {
	checkpointStore *light.CheckpointStore
	committeeChain  *light.CommitteeChain
	headTracker     *light.HeadTracker
	recentBlocks    *lru.Cache[common.Hash, []byte]
	//headValidator   *light.HeadValidator
	eventServer *eventsource.Server
	lastEventId uint64
}

func NewBeaconApiServer(
	checkpointStore *light.CheckpointStore,
	committeeChain *light.CommitteeChain,
	headTracker *light.HeadTracker,
	recentBlocks *lru.Cache[common.Hash, []byte]) *BeaconApiServer {

	eventServer := eventsource.NewServer()
	eventServer.Register("headEvent", eventsource.NewSliceRepository())
	return &BeaconApiServer{
		checkpointStore: checkpointStore,
		committeeChain:  committeeChain,
		headTracker:     headTracker,
		recentBlocks:    recentBlocks,
		eventServer:     eventServer,
	}
}

func (s *BeaconApiServer) RestAPI(server *restapi.Server) restapi.API {
	return func(router *mux.Router) {
		router.HandleFunc("/eth/v1/beacon/light_client/updates", server.WrapHandler(s.handleUpdates, false, false, false)).Methods("GET")
		router.HandleFunc("/eth/v1/beacon/light_client/optimistic_update", server.WrapHandler(s.handleOptimisticUpdate, false, false, false)).Methods("GET")
		router.HandleFunc("/eth/v1/beacon/light_client/finality_update", server.WrapHandler(s.handleFinalityUpdate, false, false, false)).Methods("GET")
		router.HandleFunc("/eth/v1/beacon/headers/head", server.WrapHandler(s.handleHeadHeader, false, false, false)).Methods("GET")
		router.HandleFunc("/eth/v1/beacon/light_client/bootstrap/{checkpointhash}", server.WrapHandler(s.handleBootstrap, false, false, false)).Methods("GET")
		router.HandleFunc("/eth/v1/beacon/blocks/{blockid}", server.WrapHandler(s.handleBlocks, false, false, false)).Methods("GET")
		router.HandleFunc("/eth/v1/events", s.eventServer.Handler("headEvent"))
	}
}

func (s *BeaconApiServer) PublishHeadEvent(slot uint64, blockRoot common.Hash) {
	enc, err := json.Marshal(&jsonHeadEvent{Slot: common.Decimal(slot), Block: blockRoot})
	if err != nil {
		log.Error("Error encoding head event", "error", err)
		return
	}
	s.publishEvent("head", string(enc))
}

func (s *BeaconApiServer) PublishOptimisticHeadUpdate(head types.OptimisticUpdate) {
	enc, err := encodeOptimisticUpdate(head)
	if err != nil {
		log.Error("Error encoding optimistic head update", "error", err)
		return
	}
	s.publishEvent("light_client_optimistic_update", string(enc))
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
	panic("TODO")
}
func (s *BeaconApiServer) handleOptimisticUpdate(ctx context.Context, values url.Values, vars map[string]string, decodeBody func(*any) error) (any, string, int) {
	panic("TODO")
}
func (s *BeaconApiServer) handleFinalityUpdate(ctx context.Context, values url.Values, vars map[string]string, decodeBody func(*any) error) (any, string, int) {
	panic("TODO")
}
func (s *BeaconApiServer) handleHeadHeader(ctx context.Context, values url.Values, vars map[string]string, decodeBody func(*any) error) (any, string, int) {
	panic("TODO")
}
func (s *BeaconApiServer) handleBootstrap(ctx context.Context, values url.Values, vars map[string]string, decodeBody func(*any) error) (any, string, int) {
	panic("TODO")
}
func (s *BeaconApiServer) handleBlocks(ctx context.Context, values url.Values, vars map[string]string, decodeBody func(*any) error) (any, string, int) {
	panic("TODO")
}
