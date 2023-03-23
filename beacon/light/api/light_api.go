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
	"github.com/ethereum/go-ethereum/beacon/light"
	"github.com/ethereum/go-ethereum/beacon/light/types"
	"github.com/ethereum/go-ethereum/beacon/merkle"
	"github.com/ethereum/go-ethereum/beacon/params"
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

// BeaconLightApi requests light client information from a beacon node REST API.
// Note: all required API endpoints are currently only implemented by Lodestar.
type BeaconLightApi struct {
	url           string
	client        *http.Client
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
		return nil, fmt.Errorf("Unexpected error from API endpoint \"%s\": status code %d", path, resp.StatusCode)
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
//TODO handle valid partial results
func (api *BeaconLightApi) GetBestUpdatesAndCommittees(firstPeriod, count uint64) ([]*types.LightClientUpdate, []*types.SerializedCommittee, error) {
	resp, err := api.httpGetf("/eth/v1/beacon/light_client/updates?start_period=%d&count=%d", firstPeriod, count)
	if err != nil {
		return nil, nil, err
	}

	var data []types.CommitteeUpdate
	if err := json.Unmarshal(resp, &data); err != nil {
		return nil, nil, err
	}
	if len(data) != int(count) {
		return nil, nil, errors.New("invalid number of committee updates")
	}
	updates := make([]*types.LightClientUpdate, int(count))
	committees := make([]*types.SerializedCommittee, int(count))
	for i, d := range data {
		if d.Update.Header.SyncPeriod() != firstPeriod+uint64(i) {
			return nil, nil, errors.New("wrong committee update header period")
		}
		if err := d.Update.Validate(); err != nil {
			return nil, nil, err
		}
		if d.NextSyncCommittee.Root() != d.Update.NextSyncCommitteeRoot {
			return nil, nil, errors.New("wrong sync committee root")
		}
		updates[i], committees[i] = d.Update, d.NextSyncCommittee
	}
	return updates, committees, nil
}

// GetOptimisticHeadUpdate fetches a signed header based on the latest available
// optimistic update. Note that the signature should be verified by the caller
// as its validity depends on the update chain.
//
// See data structure definition here:
// https://github.com/ethereum/consensus-specs/blob/dev/specs/altair/light-client/sync-protocol.md#lightclientoptimisticupdate
func (api *BeaconLightApi) GetOptimisticHeadUpdate() (types.SignedHead, error) {
	resp, err := api.httpGet("/eth/v1/beacon/light_client/optimistic_update")
	if err != nil {
		return types.SignedHead{}, err
	}
	return decodeOptimisticHeadUpdate(resp)
}

func decodeOptimisticHeadUpdate(enc []byte) (types.SignedHead, error) {
	var data struct {
		Data struct {
			Header        types.JsonBeaconHeader `json:"attested_header"`
			Aggregate     types.SyncAggregate    `json:"sync_aggregate"`
			SignatureSlot common.Decimal         `json:"signature_slot"`
		} `json:"data"`
	}
	if err := json.Unmarshal(enc, &data); err != nil {
		return types.SignedHead{}, err
	}
	if data.Data.Header.Beacon.StateRoot == (common.Hash{}) {
		// workaround for different event encoding format in Lodestar
		if err := json.Unmarshal(enc, &data.Data); err != nil {
			return types.SignedHead{}, err
		}
	}

	if len(data.Data.Aggregate.BitMask) != params.SyncCommitteeBitmaskSize {
		return types.SignedHead{}, errors.New("invalid sync_committee_bits length")
	}
	if len(data.Data.Aggregate.Signature) != params.BlsSignatureSize {
		return types.SignedHead{}, errors.New("invalid sync_committee_signature length")
	}
	return types.SignedHead{
		Header:        data.Data.Header.Beacon,
		SyncAggregate: data.Data.Aggregate,
		SignatureSlot: uint64(data.Data.SignatureSlot),
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

// does not verify state root
//TODO ...
/*func (api *BeaconLightApi) GetHeadStateProof(format merkle.ProofFormat) (merkle.MultiProof, error) {
	encFormat, bitLength := EncodeCompactProofFormat(format) //TODO cache encoding?
	return api.getStateProof("head", format, encFormat, bitLength)
}*/

func (api *BeaconLightApi) GetStateProof(stateRoot common.Hash, format merkle.ProofFormat) (merkle.MultiProof, error) {
	encFormat, bitLength := EncodeCompactProofFormat(format) //TODO cache encoding?
	proof, err := api.getStateProof(stateRoot.Hex(), format, encFormat, bitLength)
	if err != nil {
		return merkle.MultiProof{}, err
	}
	if proof.RootHash() != stateRoot {
		return merkle.MultiProof{}, errors.New("Received proof has incorrect state root")
	}
	return proof, nil
}

func (api *BeaconLightApi) getStateProof(stateId string, format merkle.ProofFormat, encFormat []byte, bitLength int) (merkle.MultiProof, error) {
	resp, err := api.httpGetf("/eth/v0/beacon/proof/state/%s?format=0x%x", stateId, encFormat)
	if err != nil {
		return merkle.MultiProof{}, err
	}
	valueCount := (bitLength + 1) / 2
	if len(resp) != valueCount*32 {
		return merkle.MultiProof{}, errors.New("Invalid state proof length")
	}
	values := make(merkle.Values, valueCount)
	for i := range values {
		copy(values[i][:], resp[i*32:(i+1)*32])
	}
	return merkle.MultiProof{Format: format, Values: values}, nil
}

// EncodeCompactProofFormat encodes a merkle.ProofFormat into a binary compact
// proof format. See description here:
// https://github.com/ChainSafe/consensus-specs/blob/feat/multiproof/ssz/merkle-proofs.md#compact-multiproofs
func EncodeCompactProofFormat(format merkle.ProofFormat) ([]byte, int) {
	target := make([]byte, 0, 64)
	var bitLength int
	encodeProofFormatSubtree(format, &target, &bitLength)
	return target, bitLength
}

// encodeProofFormatSubtree recursively encodes a subtree of a proof format into
// binary compact format.
func encodeProofFormatSubtree(format merkle.ProofFormat, target *[]byte, bitLength *int) {
	bytePtr, bitMask := *bitLength>>3, byte(128)>>(*bitLength&7)
	*bitLength++
	if bytePtr == len(*target) {
		*target = append(*target, byte(0))
	}
	if left, right := format.Children(); left == nil {
		(*target)[bytePtr] += bitMask
	} else {
		encodeProofFormatSubtree(left, target, bitLength)
		encodeProofFormatSubtree(right, target, bitLength)
	}
}

// GetCheckpointData fetches and validates bootstrap data belonging to the given checkpoint.
func (api *BeaconLightApi) GetCheckpointData(checkpointHash common.Hash) (*light.CheckpointData, error) {
	resp, err := api.httpGetf("/eth/v1/beacon/light_client/bootstrap/0x%x", checkpointHash[:])
	if err != nil {
		return nil, err
	}

	// See data structure definition here:
	// https://github.com/ethereum/consensus-specs/blob/dev/specs/altair/light-client/sync-protocol.md#lightclientbootstrap
	type bootstrapData struct {
		Data struct {
			Header          types.JsonBeaconHeader     `json:"header"`
			Committee       *types.SerializedCommittee `json:"current_sync_committee"`
			CommitteeBranch merkle.Values              `json:"current_sync_committee_branch"`
		} `json:"data"`
	}

	var data bootstrapData
	if err := json.Unmarshal(resp, &data); err != nil {
		return nil, err
	}
	header := data.Data.Header.Beacon
	if header.Hash() != checkpointHash {
		return nil, errors.New("invalid checkpoint block header")
	}
	checkpoint := &light.CheckpointData{
		Header:          header,
		CommitteeBranch: data.Data.CommitteeBranch,
		CommitteeRoot:   data.Data.Committee.Root(),
		Committee:       data.Data.Committee,
	}
	if !checkpoint.Validate() {
		return nil, errors.New("invalid sync committee Merkle proof")
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

// StartHeadListener creates an event subscription for heads and signed (optimistic)
// head updates and calls the specified callback functions when they are received.
// The callbacks are also called for the current head and optimistic head at startup.
// They are never called concurrently.
func (api *BeaconLightApi) StartHeadListener(headFn func(slot uint64, blockRoot common.Hash), signedFn func(head types.SignedHead), errFn func(err error)) func() {
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
		stream, err := eventsource.Subscribe(api.url+"/eth/v1/events?topics=head&topics=light_client_optimistic_update", "")
		if err != nil {
			errFn(fmt.Errorf("Error creating event subscription: %v", err))
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
			headFn(head.Slot, head.Hash())
		}
		if signedHead, err := api.GetOptimisticHeadUpdate(); err == nil {
			signedFn(signedHead)
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
						headFn(slot, blockRoot)
					} else {
						errFn(fmt.Errorf("Error decoding head event: %v", err))
					}
				case "light_client_optimistic_update":
					if signedHead, err := decodeOptimisticHeadUpdate([]byte(event.Data())); err == nil {
						signedFn(signedHead)
					} else {
						errFn(fmt.Errorf("Error decoding optimistic update event: %v", err))
					}
				default:
					errFn(fmt.Errorf("Unexpected event: %s", event.Event()))
				}
			case err, ok := <-stream.Errors:
				if !ok {
					break
				}
				errFn(err)
			}
		}
	}()
	return func() {
		close(closeCh)
		<-closedCh
		<-stoppedCh
	}
}
