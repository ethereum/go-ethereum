// Copyright 2022 The go-ethereum Authors
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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/donovanhide/eventsource"
	"github.com/ethereum/go-ethereum/beacon/merkle"
	"github.com/ethereum/go-ethereum/beacon/params"
	"github.com/ethereum/go-ethereum/beacon/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/protolambda/zrnt/eth2/beacon/capella"
	"github.com/protolambda/zrnt/eth2/configs"
	"github.com/protolambda/ztyp/tree"
)

var (
	ErrNotFound = errors.New("404 Not Found")
	ErrInternal = errors.New("500 Internal Server Error")
)

type CommitteeUpdate struct {
	Version           string
	Update            types.LightClientUpdate
	NextSyncCommittee types.SerializedSyncCommittee
}

// See data structure definition here:
// https://github.com/ethereum/consensus-specs/blob/dev/specs/altair/light-client/sync-protocol.md#lightclientupdate
type committeeUpdateJson struct {
	Version string              `json:"version"`
	Data    committeeUpdateData `json:"data"`
}

type committeeUpdateData struct {
	Header                  jsonBeaconHeader              `json:"attested_header"`
	NextSyncCommittee       types.SerializedSyncCommittee `json:"next_sync_committee"`
	NextSyncCommitteeBranch merkle.Values                 `json:"next_sync_committee_branch"`
	FinalizedHeader         *jsonBeaconHeader             `json:"finalized_header,omitempty"`
	FinalityBranch          merkle.Values                 `json:"finality_branch,omitempty"`
	SyncAggregate           types.SyncAggregate           `json:"sync_aggregate"`
	SignatureSlot           common.Decimal                `json:"signature_slot"`
}

type jsonBeaconHeader struct {
	Beacon types.Header `json:"beacon"`
}

type jsonHeaderWithExecProof struct {
	Beacon          types.Header                    `json:"beacon"`
	Execution       *capella.ExecutionPayloadHeader `json:"execution"`
	ExecutionBranch merkle.Values                   `json:"execution_branch"`
}

// UnmarshalJSON unmarshals from JSON.
func (u *CommitteeUpdate) UnmarshalJSON(input []byte) error {
	var dec committeeUpdateJson
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	u.Version = dec.Version
	u.NextSyncCommittee = dec.Data.NextSyncCommittee
	u.Update = types.LightClientUpdate{
		AttestedHeader: types.SignedHeader{
			Header:        dec.Data.Header.Beacon,
			Signature:     dec.Data.SyncAggregate,
			SignatureSlot: uint64(dec.Data.SignatureSlot),
		},
		NextSyncCommitteeRoot:   u.NextSyncCommittee.Root(),
		NextSyncCommitteeBranch: dec.Data.NextSyncCommitteeBranch,
		FinalityBranch:          dec.Data.FinalityBranch,
	}
	if dec.Data.FinalizedHeader != nil {
		u.Update.FinalizedHeader = &dec.Data.FinalizedHeader.Beacon
	}
	return nil
}

// fetcher is an interface useful for debug-harnessing the http api.
type fetcher interface {
	Do(req *http.Request) (*http.Response, error)
}

// BeaconLightApi requests light client information from a beacon node REST API.
// Note: all required API endpoints are currently only implemented by Lodestar.
type BeaconLightApi struct {
	url           string
	client        fetcher
	customHeaders map[string]string
}

func NewBeaconLightApi(url string, customHeaders map[string]string) *BeaconLightApi {
	return &BeaconLightApi{
		url: url,
		client: &http.Client{
			Timeout: time.Second * 10,
		},
		customHeaders: customHeaders,
	}
}

func (api *BeaconLightApi) httpGet(path string) ([]byte, error) {
	req, err := http.NewRequest("GET", api.url+path, nil)
	if err != nil {
		return nil, err
	}
	for k, v := range api.customHeaders {
		req.Header.Set(k, v)
	}
	resp, err := api.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case 200:
		return io.ReadAll(resp.Body)
	case 404:
		return nil, ErrNotFound
	case 500:
		return nil, ErrInternal
	default:
		return nil, fmt.Errorf("unexpected error from API endpoint \"%s\": status code %d", path, resp.StatusCode)
	}
}

func (api *BeaconLightApi) httpGetf(format string, params ...any) ([]byte, error) {
	return api.httpGet(fmt.Sprintf(format, params...))
}

// GetBestUpdateAndCommittee fetches and validates LightClientUpdate for given
// period and full serialized committee for the next period (committee root hash
// equals update.NextSyncCommitteeRoot).
// Note that the results are validated but the update signature should be verified
// by the caller as its validity depends on the update chain.
func (api *BeaconLightApi) GetBestUpdatesAndCommittees(firstPeriod, count uint64) ([]*types.LightClientUpdate, []*types.SerializedSyncCommittee, error) {
	resp, err := api.httpGetf("/eth/v1/beacon/light_client/updates?start_period=%d&count=%d", firstPeriod, count)
	if err != nil {
		return nil, nil, err
	}

	var data []CommitteeUpdate
	if err := json.Unmarshal(resp, &data); err != nil {
		return nil, nil, err
	}
	if len(data) != int(count) {
		return nil, nil, errors.New("invalid number of committee updates")
	}
	updates := make([]*types.LightClientUpdate, int(count))
	committees := make([]*types.SerializedSyncCommittee, int(count))
	for i, d := range data {
		if d.Update.AttestedHeader.Header.SyncPeriod() != firstPeriod+uint64(i) {
			return nil, nil, errors.New("wrong committee update header period")
		}
		if err := d.Update.Validate(); err != nil {
			return nil, nil, err
		}
		if d.NextSyncCommittee.Root() != d.Update.NextSyncCommitteeRoot {
			return nil, nil, errors.New("wrong sync committee root")
		}
		updates[i], committees[i] = new(types.LightClientUpdate), new(types.SerializedSyncCommittee)
		*updates[i], *committees[i] = d.Update, d.NextSyncCommittee
	}
	return updates, committees, nil
}

// GetOptimisticHeadUpdate fetches a signed header based on the latest available
// optimistic update. Note that the signature should be verified by the caller
// as its validity depends on the update chain.
//
// See data structure definition here:
// https://github.com/ethereum/consensus-specs/blob/dev/specs/altair/light-client/sync-protocol.md#lightclientoptimisticupdate
func (api *BeaconLightApi) GetOptimisticHeadUpdate() (types.SignedHeader, error) {
	resp, err := api.httpGet("/eth/v1/beacon/light_client/optimistic_update")
	if err != nil {
		return types.SignedHeader{}, err
	}
	return decodeOptimisticHeadUpdate(resp)
}

func decodeOptimisticHeadUpdate(enc []byte) (types.SignedHeader, error) {
	var data struct {
		Data struct {
			Header        jsonBeaconHeader    `json:"attested_header"`
			Aggregate     types.SyncAggregate `json:"sync_aggregate"`
			SignatureSlot common.Decimal      `json:"signature_slot"`
		} `json:"data"`
	}
	if err := json.Unmarshal(enc, &data); err != nil {
		return types.SignedHeader{}, err
	}
	if data.Data.Header.Beacon.StateRoot == (common.Hash{}) {
		// workaround for different event encoding format in Lodestar
		if err := json.Unmarshal(enc, &data.Data); err != nil {
			return types.SignedHeader{}, err
		}
	}

	if len(data.Data.Aggregate.Signers) != params.SyncCommitteeBitmaskSize {
		return types.SignedHeader{}, errors.New("invalid sync_committee_bits length")
	}
	if len(data.Data.Aggregate.Signature) != params.BLSSignatureSize {
		return types.SignedHeader{}, errors.New("invalid sync_committee_signature length")
	}
	return types.SignedHeader{
		Header:        data.Data.Header.Beacon,
		Signature:     data.Data.Aggregate,
		SignatureSlot: uint64(data.Data.SignatureSlot),
	}, nil
}

// GetFinalityUpdate fetches the latest available finality update.
//
// See data structure definition here:
// https://github.com/ethereum/consensus-specs/blob/dev/specs/altair/light-client/sync-protocol.md#lightclientfinalityupdate
func (api *BeaconLightApi) GetFinalityUpdate() (types.FinalityUpdate, error) {
	resp, err := api.httpGet("/eth/v1/beacon/light_client/finality_update")
	if err != nil {
		return types.FinalityUpdate{}, err
	}
	return decodeFinalityUpdate(resp)
}

func decodeFinalityUpdate(enc []byte) (types.FinalityUpdate, error) {
	var data struct {
		Data struct {
			Attested       jsonHeaderWithExecProof `json:"attested_header"`
			Finalized      jsonHeaderWithExecProof `json:"finalized_header"`
			FinalityBranch merkle.Values           `json:"finality_branch"`
			Aggregate      types.SyncAggregate     `json:"sync_aggregate"`
			SignatureSlot  common.Decimal          `json:"signature_slot"`
		} `json:"data"`
	}
	if err := json.Unmarshal(enc, &data); err != nil {
		return types.FinalityUpdate{}, err
	}

	if len(data.Data.Aggregate.Signers) != params.SyncCommitteeBitmaskSize {
		return types.FinalityUpdate{}, errors.New("invalid sync_committee_bits length")
	}
	if len(data.Data.Aggregate.Signature) != params.BLSSignatureSize {
		return types.FinalityUpdate{}, errors.New("invalid sync_committee_signature length")
	}
	return types.FinalityUpdate{
		Attested: types.HeaderWithExecProof{
			Header:        data.Data.Attested.Beacon,
			PayloadHeader: data.Data.Attested.Execution,
			PayloadBranch: data.Data.Attested.ExecutionBranch,
		},
		Finalized: types.HeaderWithExecProof{
			Header:        data.Data.Finalized.Beacon,
			PayloadHeader: data.Data.Finalized.Execution,
			PayloadBranch: data.Data.Finalized.ExecutionBranch,
		},
		FinalityBranch: data.Data.FinalityBranch,
		Signature:      data.Data.Aggregate,
		SignatureSlot:  uint64(data.Data.SignatureSlot),
	}, nil
}

// GetHead fetches and validates the beacon header with the given blockRoot.
// If blockRoot is null hash then the latest head header is fetched.
func (api *BeaconLightApi) GetHeader(blockRoot common.Hash) (types.Header, error) {
	var blockId string
	if blockRoot == (common.Hash{}) {
		blockId = "head"
	} else {
		blockId = blockRoot.Hex()
	}
	resp, err := api.httpGetf("/eth/v1/beacon/headers/%s", blockId)
	if err != nil {
		return types.Header{}, err
	}

	var data struct {
		Data struct {
			Root      common.Hash `json:"root"`
			Canonical bool        `json:"canonical"`
			Header    struct {
				Message   types.Header  `json:"message"`
				Signature hexutil.Bytes `json:"signature"`
			} `json:"header"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp, &data); err != nil {
		return types.Header{}, err
	}
	header := data.Data.Header.Message
	if blockRoot == (common.Hash{}) {
		blockRoot = data.Data.Root
	}
	if header.Hash() != blockRoot {
		return types.Header{}, errors.New("retrieved beacon header root does not match")
	}
	return header, nil
}

// GetCheckpointData fetches and validates bootstrap data belonging to the given checkpoint.
func (api *BeaconLightApi) GetCheckpointData(checkpointHash common.Hash) (*types.BootstrapData, error) {
	resp, err := api.httpGetf("/eth/v1/beacon/light_client/bootstrap/0x%x", checkpointHash[:])
	if err != nil {
		return nil, err
	}

	// See data structure definition here:
	// https://github.com/ethereum/consensus-specs/blob/dev/specs/altair/light-client/sync-protocol.md#lightclientbootstrap
	type bootstrapData struct {
		Data struct {
			Header          jsonBeaconHeader               `json:"header"`
			Committee       *types.SerializedSyncCommittee `json:"current_sync_committee"`
			CommitteeBranch merkle.Values                  `json:"current_sync_committee_branch"`
		} `json:"data"`
	}

	var data bootstrapData
	if err := json.Unmarshal(resp, &data); err != nil {
		return nil, err
	}
	if data.Data.Committee == nil {
		return nil, errors.New("sync committee is missing")
	}
	header := data.Data.Header.Beacon
	if header.Hash() != checkpointHash {
		return nil, fmt.Errorf("invalid checkpoint block header, have %v want %v", header.Hash(), checkpointHash)
	}
	checkpoint := &types.BootstrapData{
		Header:          header,
		CommitteeBranch: data.Data.CommitteeBranch,
		CommitteeRoot:   data.Data.Committee.Root(),
		Committee:       data.Data.Committee,
	}
	if err := checkpoint.Validate(); err != nil {
		return nil, fmt.Errorf("invalid checkpoint: %w", err)
	}
	if checkpoint.Header.Hash() != checkpointHash {
		return nil, errors.New("wrong checkpoint hash")
	}
	return checkpoint, nil
}

func (api *BeaconLightApi) GetBeaconBlock(blockRoot common.Hash) (*capella.BeaconBlock, error) {
	resp, err := api.httpGetf("/eth/v2/beacon/blocks/0x%x", blockRoot)
	if err != nil {
		return nil, err
	}

	var beaconBlockMessage struct {
		Data struct {
			Message capella.BeaconBlock `json:"message"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp, &beaconBlockMessage); err != nil {
		return nil, fmt.Errorf("invalid block json data: %v", err)
	}
	beaconBlock := new(capella.BeaconBlock)
	*beaconBlock = beaconBlockMessage.Data.Message
	root := common.Hash(beaconBlock.HashTreeRoot(configs.Mainnet, tree.GetHashFn()))
	if root != blockRoot {
		return nil, fmt.Errorf("Beacon block root hash mismatch (expected: %x, got: %x)", blockRoot, root)
	}
	return beaconBlock, nil
}

func decodeHeadEvent(enc []byte) (uint64, common.Hash, error) {
	var data struct {
		Slot  common.Decimal `json:"slot"`
		Block common.Hash    `json:"block"`
	}
	if err := json.Unmarshal(enc, &data); err != nil {
		return 0, common.Hash{}, err
	}
	return uint64(data.Slot), data.Block, nil
}

type HeadEventListener struct {
	OnNewHead    func(slot uint64, blockRoot common.Hash)
	OnSignedHead func(head types.SignedHeader)
	OnFinality   func(head types.FinalityUpdate)
	OnError      func(err error)
}

// StartHeadListener creates an event subscription for heads and signed (optimistic)
// head updates and calls the specified callback functions when they are received.
// The callbacks are also called for the current head and optimistic head at startup.
// They are never called concurrently.
func (api *BeaconLightApi) StartHeadListener(listener HeadEventListener) func() {
	closeCh := make(chan struct{})   // initiate closing the stream
	closedCh := make(chan struct{})  // stream closed (or failed to create)
	stoppedCh := make(chan struct{}) // sync loop stopped
	streamCh := make(chan *eventsource.Stream, 1)
	go func() {
		defer close(closedCh)
		// when connected to a Lodestar node the subscription blocks until the
		// first actual event arrives; therefore we create the subscription in
		// a separate goroutine while letting the main goroutine sync up to the
		// current head
		req, err := http.NewRequest("GET", api.url+
			"/eth/v1/events?topics=head&topics=light_client_optimistic_update&topics=light_client_finality_update", nil)
		if err != nil {
			listener.OnError(fmt.Errorf("error creating event subscription request: %v", err))
			return
		}
		for k, v := range api.customHeaders {
			req.Header.Set(k, v)
		}
		stream, err := eventsource.SubscribeWithRequest("", req)
		if err != nil {
			listener.OnError(fmt.Errorf("error creating event subscription: %v", err))
			close(streamCh)
			return
		}
		streamCh <- stream
		<-closeCh
		stream.Close()
	}()

	go func() {
		defer close(stoppedCh)

		if head, err := api.GetHeader(common.Hash{}); err == nil {
			listener.OnNewHead(head.Slot, head.Hash())
		}
		if signedHead, err := api.GetOptimisticHeadUpdate(); err == nil {
			listener.OnSignedHead(signedHead)
		}
		if finalityUpdate, err := api.GetFinalityUpdate(); err == nil {
			listener.OnFinality(finalityUpdate)
		}
		stream := <-streamCh
		if stream == nil {
			return
		}

		for {
			select {
			case event, ok := <-stream.Events:
				if !ok {
					break
				}
				switch event.Event() {
				case "head":
					if slot, blockRoot, err := decodeHeadEvent([]byte(event.Data())); err == nil {
						listener.OnNewHead(slot, blockRoot)
					} else {
						listener.OnError(fmt.Errorf("error decoding head event: %v", err))
					}
				case "light_client_optimistic_update":
					if signedHead, err := decodeOptimisticHeadUpdate([]byte(event.Data())); err == nil {
						listener.OnSignedHead(signedHead)
					} else {
						listener.OnError(fmt.Errorf("error decoding optimistic update event: %v", err))
					}
				case "light_client_finality_update":
					if finalityUpdate, err := decodeFinalityUpdate([]byte(event.Data())); err == nil {
						listener.OnFinality(finalityUpdate)
					} else {
						listener.OnError(fmt.Errorf("error decoding finality update event: %v", err))
					}
				default:
					listener.OnError(fmt.Errorf("unexpected event: %s", event.Event()))
				}
			case err, ok := <-stream.Errors:
				if !ok {
					break
				}
				listener.OnError(err)
			}
		}
	}()
	return func() {
		close(closeCh)
		<-closedCh
		<-stoppedCh
	}
}
