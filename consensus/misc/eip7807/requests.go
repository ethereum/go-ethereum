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

package eip7807

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/ssz"
)

// requestsContainer names a consensus-layer definition of ExecutionRequests,
// the container whose hash tree root is the requests_hash of the payload.
type requestsContainer int

const (
	// electraContainer is the definition EIP-7807 links: a Container of
	// three bounded lists, which cannot hold the EIP-8282 builder requests.
	electraContainer requestsContainer = iota
	// gloasContainer is the current consensus definition: a
	// ProgressiveContainer of five progressive lists, the builder deposit
	// and exit requests included.
	gloasContainer
)

// requestsFork is the container RequestsRoot computes. The EIP says
// requests_hash matches "the ExecutionRequests root used by the Consensus
// Layer" and links the Electra container; the consensus layer has since moved
// to the Gloas one, and the two give different roots for the same requests.
// Electra follows the EIP's text and the roots other implementations pin;
// the switch is here for when the EIP settles it.
const requestsFork = electraContainer

// requestFields holds, per EIP-7685 request type, the widths of the record's
// fields. Each record is a fixed-size SSZ container, so its serialization is
// its fields concatenated and the data of a request type is its records
// concatenated. Types 0 to 2 are EIP-7685, 3 and 4 EIP-8282.
var requestFields = [...][]int{
	0x00: {48, 32, 8, 96, 8}, // DepositRequest: pubkey, withdrawal_credentials, amount, signature, index
	0x01: {20, 48, 8},        // WithdrawalRequest: source_address, validator_pubkey, amount
	0x02: {20, 48, 48},       // ConsolidationRequest: source_address, source_pubkey, target_pubkey
	0x03: {48, 32, 8, 96},    // BuilderDepositRequest: pubkey, withdrawal_credentials, amount, signature
	0x04: {20, 48},           // BuilderExitRequest: source_address, pubkey
}

// electraLimits are the list limits of the Electra container: the presets
// MAX_DEPOSIT_REQUESTS_PER_PAYLOAD, MAX_WITHDRAWAL_REQUESTS_PER_PAYLOAD and
// MAX_CONSOLIDATION_REQUESTS_PER_PAYLOAD. A limit sets the depth of its
// list's tree, so it enters the root: a chain whose preset set them
// differently would need its own values here. The mainnet and minimal
// presets agree on all three. The Gloas progressive lists have none, which
// is one reason the consensus layer moved to them.
var electraLimits = [...]uint64{0x00: 8192, 0x01: 16, 0x02: 2}

// gloasRequestsFields is the active_fields configuration of the Gloas
// container: ProgressiveContainer(active_fields=[1] * 5).
var gloasRequestsFields = ssz.MustActiveFields(1, 1, 1, 1, 1)

// RequestsRoot computes ExecutionRequests.hash_tree_root() over a block's
// EIP-7685 requests, each a type byte followed by that type's records, as
// the engine API and the block processor carry them. Requests must come in
// strictly increasing type order; a request with no records is skipped, as
// the EIP-7685 requests hash skips it.
func RequestsRoot(requests [][]byte) (common.Hash, error) {
	lists, err := splitRequests(requests)
	if err != nil {
		return common.Hash{}, err
	}
	if requestsFork == gloasContainer {
		return gloasRequestsRoot(lists), nil
	}
	return electraRequestsRoot(lists)
}

// splitRequests separates the requests into one record list per type.
func splitRequests(requests [][]byte) (lists [len(requestFields)][][]byte, err error) {
	prev := -1
	for _, req := range requests {
		if len(req) <= 1 {
			continue
		}
		typ, data := int(req[0]), req[1:]
		if typ >= len(requestFields) {
			return lists, fmt.Errorf("unknown request type %d", typ)
		}
		if typ <= prev {
			return lists, fmt.Errorf("request type %d out of order after %d", typ, prev)
		}
		prev = typ
		size := 0
		for _, n := range requestFields[typ] {
			size += n
		}
		if len(data)%size != 0 {
			return lists, fmt.Errorf("request type %d data of %d bytes is not a multiple of the %d-byte record", typ, len(data), size)
		}
		for off := 0; off < len(data); off += size {
			lists[typ] = append(lists[typ], data[off:off+size])
		}
	}
	return lists, nil
}

// electraRequestsRoot computes the root of the Electra container: three
// List[Request, limit] fields in a classic container.
func electraRequestsRoot(lists [len(requestFields)][][]byte) (common.Hash, error) {
	if len(lists[0x03]) != 0 || len(lists[0x04]) != 0 {
		return common.Hash{}, errors.New("builder requests have no field in the Electra ExecutionRequests")
	}
	roots := make([][32]byte, len(electraLimits))
	for typ, limit := range electraLimits {
		if uint64(len(lists[typ])) > limit {
			return common.Hash{}, fmt.Errorf("%d requests of type %d exceed the limit %d", len(lists[typ]), typ, limit)
		}
		roots[typ] = ssz.MixInLength(ssz.Merkleize(recordRoots(lists[typ], requestFields[typ]), limit), uint64(len(lists[typ])))
	}
	return common.Hash(ssz.Merkleize(roots, uint64(len(roots)))), nil
}

// gloasRequestsRoot computes the root of the Gloas container: five
// ProgressiveList[Request] fields in a progressive container.
func gloasRequestsRoot(lists [len(requestFields)][][]byte) common.Hash {
	roots := make([][32]byte, len(requestFields))
	for typ := range roots {
		roots[typ] = ssz.HashTreeRootProgressiveRoots(recordRoots(lists[typ], requestFields[typ]))
	}
	return common.Hash(ssz.HashTreeRootProgressiveContainer(gloasRequestsFields, roots))
}

// recordRoots computes the root of each record, a container of the given
// fixed-size fields: every field packs its serialization into chunks and
// merkleizes them, and the container merkleizes the field roots.
func recordRoots(records [][]byte, fields []int) [][32]byte {
	roots := make([][32]byte, len(records))
	for i, rec := range records {
		fieldRoots := make([][32]byte, len(fields))
		off := 0
		for j, n := range fields {
			chunks := ssz.Pack(rec[off : off+n])
			fieldRoots[j] = ssz.Merkleize(chunks, uint64(len(chunks)))
			off += n
		}
		roots[i] = ssz.Merkleize(fieldRoots, uint64(len(fieldRoots)))
	}
	return roots
}
