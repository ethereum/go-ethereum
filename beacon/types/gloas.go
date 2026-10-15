// Copyright 2026 The go-ethereum Authors
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
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package types

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/attestantio/go-eth2-client/spec/deneb"
	"github.com/attestantio/go-eth2-client/spec/electra"
	"github.com/attestantio/go-eth2-client/spec/gloas"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	coretypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/types/bal"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
)

// gloasBeaconBlock implements beaconBlockData for Gloas blocks. Gloas moves the
// execution payload out of the beacon block, into a separately retrieved
// execution payload envelope.
type gloasBeaconBlock struct {
	block    *gloas.BeaconBlock
	root     common.Hash
	payload  *coretypes.Block
	requests [][]byte
}

// Slot returns the slot number of the block.
func (g *gloasBeaconBlock) Slot() uint64 {
	return uint64(g.block.Slot)
}

// Header returns the block's header data.
func (g *gloasBeaconBlock) Header() Header {
	return makeHeader(uint64(g.block.Slot), uint64(g.block.ProposerIndex), g.block.ParentRoot, g.block.StateRoot, g.block.Body)
}

// Root returns the SSZ root hash of the block.
func (g *gloasBeaconBlock) Root() common.Hash {
	return g.root
}

// ExecutionPayload returns the execution payload attached with
// SetGloasPayloadEnvelope.
func (g *gloasBeaconBlock) ExecutionPayload() (*coretypes.Block, error) {
	if g.payload == nil {
		return nil, errors.New("Gloas execution payload envelope is missing")
	}
	return g.payload, nil
}

// ExecutionRequestsList returns the execution layer requests attached with
// SetGloasPayloadEnvelope.
func (g *gloasBeaconBlock) ExecutionRequestsList() [][]byte {
	return g.requests
}

func decodeGloasBeaconBlock(enc []byte) (*BeaconBlock, error) {
	block := new(gloas.BeaconBlock)
	if err := json.Unmarshal(enc, block); err != nil {
		return nil, err
	}
	if block.Body == nil || block.Body.SignedExecutionPayloadBid == nil {
		return nil, errors.New("Gloas beacon block has no execution payload bid")
	}
	root, err := block.HashTreeRoot()
	if err != nil {
		return nil, fmt.Errorf("failed to compute block root: %v", err)
	}
	return &BeaconBlock{
		data: &gloasBeaconBlock{
			block: block,
			root:  root,
		}}, nil
}

// HasPayloadEnvelope reports whether the block's execution payload is published
// apart from the block, as from the Gloas fork on. Such a block is only complete
// once its payload envelope is attached.
func (b *BeaconBlock) HasPayloadEnvelope() bool {
	_, ok := b.data.(*gloasBeaconBlock)
	return ok
}

// SetGloasPayloadEnvelope attaches the execution payload envelope to a Gloas
// beacon block. The envelope must be bound to the block and its payload hash
// must match the block's signed execution payload bid.
func (b *BeaconBlock) SetGloasPayloadEnvelope(enc []byte) error {
	g, ok := b.data.(*gloasBeaconBlock)
	if !ok {
		return errors.New("not a Gloas beacon block")
	}
	var envelope struct {
		Payload           json.RawMessage `json:"payload"`
		ExecutionRequests json.RawMessage `json:"execution_requests"`
		BeaconBlockRoot   common.Hash     `json:"beacon_block_root"`
	}
	if err := json.Unmarshal(enc, &envelope); err != nil {
		return err
	}
	if envelope.BeaconBlockRoot != g.root {
		return fmt.Errorf("Gloas payload envelope belongs to %x, want %x", envelope.BeaconBlockRoot, g.root)
	}
	payload, requests, err := decodeGloasPayload(envelope.Payload, envelope.ExecutionRequests, common.Hash(g.block.ParentRoot))
	if err != nil {
		return err
	}
	bidHash := common.Hash(g.block.Body.SignedExecutionPayloadBid.Message.BlockHash)
	if payload.Hash() != bidHash {
		return fmt.Errorf("Gloas payload hash is %x, want %x", payload.Hash(), bidHash)
	}
	g.payload = payload
	g.requests = requests
	return nil
}

func decodeGloasPayload(enc, requestsJSON json.RawMessage, parentRoot common.Hash) (*coretypes.Block, [][]byte, error) {
	var payload deneb.ExecutionPayload
	if err := json.Unmarshal(enc, &payload); err != nil {
		return nil, nil, err
	}
	// The block access list is delivered as the hex-encoded RLP blob produced by
	// the execution layer, not as a structured JSON object.
	var extra struct {
		BlockAccessList hexutil.Bytes  `json:"block_access_list"`
		SlotNumber      common.Decimal `json:"slot_number"`
	}
	if err := json.Unmarshal(enc, &extra); err != nil {
		return nil, nil, err
	}
	accessList := new(bal.BlockAccessList)
	if err := rlp.DecodeBytes(extra.BlockAccessList, accessList); err != nil {
		return nil, nil, fmt.Errorf("failed to decode block access list: %w", err)
	}
	// Hash the raw bytes, matching how the execution layer derives the header's
	// block access list hash (keccak256 over the RLP encoding).
	accessHash := crypto.Keccak256Hash(extra.BlockAccessList)
	requests, err := decodeGloasRequests(requestsJSON)
	if err != nil {
		return nil, nil, err
	}
	block, err := convertGloasPayload(&payload, parentRoot, accessList, accessHash, uint64(extra.SlotNumber), requests)
	if err != nil {
		return nil, nil, err
	}
	return block, requests, nil
}

func convertGloasPayload(payload *deneb.ExecutionPayload, parentRoot common.Hash, accessList *bal.BlockAccessList, accessHash common.Hash, slot uint64, requests [][]byte) (*coretypes.Block, error) {
	var header coretypes.Header
	convertDenebHeader(payload, parentRoot, &header)
	header.SlotNumber = &slot
	header.BlockAccessListHash = &accessHash
	transactions, err := convertTransactions(payload.Transactions, &header)
	if err != nil {
		return nil, err
	}
	withdrawals := convertWithdrawals(payload.Withdrawals, &header)
	if requests != nil {
		reqHash := coretypes.CalcRequestsHash(requests)
		header.RequestsHash = &reqHash
	}
	body := coretypes.Body{
		Transactions: transactions,
		Withdrawals:  withdrawals,
	}
	block := coretypes.NewBlockWithHeader(&header).WithBody(body).WithAccessListUnsafe(accessList)
	if hash := block.Hash(); hash != common.Hash(payload.BlockHash) {
		return nil, fmt.Errorf("sanity check failed, payload hash does not match (expected %x, got %x)", payload.BlockHash, hash)
	}
	return block, nil
}

func decodeGloasRequests(enc json.RawMessage) ([][]byte, error) {
	if len(enc) == 0 || bytes.Equal(enc, []byte("null")) {
		return nil, nil
	}
	var requests electra.ExecutionRequests
	if err := json.Unmarshal(enc, &requests); err != nil {
		return nil, err
	}
	result := marshalRequests(&requests)

	var builders struct {
		Deposits []gloasBuilderDeposit `json:"builder_deposits"`
		Exits    []gloasBuilderExit    `json:"builder_exits"`
	}
	if err := json.Unmarshal(enc, &builders); err != nil {
		return nil, err
	}
	for _, request := range builders.Deposits {
		if len(request.Pubkey) != 48 || len(request.WithdrawalCredentials) != 32 || len(request.Signature) != 96 {
			return nil, errors.New("invalid Gloas builder deposit request")
		}
		data := make([]byte, 1, 1+48+32+8+96)
		data[0] = 0x03
		data = append(data, request.Pubkey...)
		data = append(data, request.WithdrawalCredentials...)
		data = binary.LittleEndian.AppendUint64(data, uint64(request.Amount))
		data = append(data, request.Signature...)
		result = append(result, data)
	}
	for _, request := range builders.Exits {
		if len(request.SourceAddress) != 20 || len(request.Pubkey) != 48 {
			return nil, errors.New("invalid Gloas builder exit request")
		}
		data := make([]byte, 1, 1+20+48)
		data[0] = 0x04
		data = append(data, request.SourceAddress...)
		data = append(data, request.Pubkey...)
		result = append(result, data)
	}
	return result, nil
}

type gloasBuilderDeposit struct {
	Pubkey                hexutil.Bytes  `json:"pubkey"`
	WithdrawalCredentials hexutil.Bytes  `json:"withdrawal_credentials"`
	Amount                common.Decimal `json:"amount"`
	Signature             hexutil.Bytes  `json:"signature"`
}

type gloasBuilderExit struct {
	SourceAddress hexutil.Bytes `json:"source_address"`
	Pubkey        hexutil.Bytes `json:"pubkey"`
}
