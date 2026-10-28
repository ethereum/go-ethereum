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

package traceapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/tracers/parity"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/rpc"
)

// Outputs selects the independent output families of the trace namespace.
type Outputs []string

func (t *Outputs) UnmarshalJSON(input []byte) error {
	if bytes.Equal(bytes.TrimSpace(input), []byte("null")) {
		return fmt.Errorf("trace types must be an array")
	}
	type plain Outputs
	if err := json.Unmarshal(input, (*plain)(t)); err != nil {
		return err
	}
	return t.validate()
}

func (t Outputs) validate() error {
	seen := make(map[string]bool)
	for _, name := range t {
		if name != "trace" && name != "stateDiff" && name != "vmTrace" {
			return invalidParams("unknown trace type %q", name)
		}
		if seen[name] {
			return invalidParams("duplicate trace type %q", name)
		}
		seen[name] = true
	}
	return nil
}

func (t Outputs) has(name string) bool {
	for _, s := range t {
		if s == name {
			return true
		}
	}
	return false
}

// CallArgs validates known unsigned-call fields and ignores extensions.
type CallArgs struct {
	ethapi.TransactionArgs
	Type *hexutil.Uint64 `json:"type,omitempty"`
}

func (a *CallArgs) UnmarshalJSON(input []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(input, &fields); err != nil {
		return err
	}
	if fields == nil {
		return fmt.Errorf("call must be an object")
	}
	*a = CallArgs{}
	for key, value := range fields {
		switch key {
		case "from", "to", "gas", "gasPrice", "maxFeePerGas", "maxPriorityFeePerGas", "value", "data", "input", "nonce", "type", "accessList", "chainId", "maxFeePerBlobGas", "blobVersionedHashes", "authorizationList":
		default:
			delete(fields, key)
			continue
		}
		// An explicit null is an omitted member; only a null to has a meaning
		// of its own, contract creation.
		if key != "to" && bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			delete(fields, key)
		}
	}
	if raw, ok := fields["type"]; ok {
		var encoded string
		if err := json.Unmarshal(raw, &encoded); err != nil {
			return err
		}
		// The type is a hex byte, allowing both 0x2 and 0x02.
		if len(encoded) == 3 && encoded[:2] == "0x" {
			encoded = "0x0" + encoded[2:]
		}
		decoded, err := hexutil.Decode(encoded)
		if err != nil || len(decoded) != 1 {
			return fmt.Errorf("type must be a hex-encoded byte")
		}
		kind := hexutil.Uint64(decoded[0])
		if kind > types.SetCodeTxType {
			return fmt.Errorf("unsigned trace calls support transaction types 0 through 4")
		}
		a.Type = &kind
	}
	// Filter before decoding so fields outside the call object cannot accidentally
	// acquire meaning from TransactionArgs (for example, blob sidecars).
	filtered, err := json.Marshal(fields)
	if err != nil {
		return err
	}
	return json.Unmarshal(filtered, &a.TransactionArgs)
}

// CallManyEntry is exactly one [call, traceTypes] pair.
type CallManyEntry struct {
	Call  CallArgs
	Types Outputs
}

func (e *CallManyEntry) UnmarshalJSON(input []byte) error {
	var pair []json.RawMessage
	if err := json.Unmarshal(input, &pair); err != nil {
		return err
	}
	if len(pair) != 2 {
		return fmt.Errorf("expected [call, traceTypes]")
	}
	if err := json.Unmarshal(pair[0], &e.Call); err != nil {
		return err
	}
	return json.Unmarshal(pair[1], &e.Types)
}

// CallManyArgs is a non-null list of sequential call pairs.
type CallManyArgs []CallManyEntry

func (c *CallManyArgs) UnmarshalJSON(input []byte) error {
	if bytes.Equal(bytes.TrimSpace(input), []byte("null")) {
		return fmt.Errorf("calls must be an array")
	}
	type plain CallManyArgs
	return json.Unmarshal(input, (*plain)(c))
}

// Path identifies a path in a call tree, not an index in a flat list.
type Path []hexutil.Uint64

func (p *Path) UnmarshalJSON(input []byte) error {
	if bytes.Equal(bytes.TrimSpace(input), []byte("null")) {
		return fmt.Errorf("position must be an array")
	}
	type plain Path
	return json.Unmarshal(input, (*plain)(p))
}

// FilterArgs describes an inclusive block range or an exact block hash, and pagination after matching.
type FilterArgs struct {
	BlockHash   *common.Hash     `json:"blockHash"`
	FromBlock   *rpc.BlockNumber `json:"fromBlock"`
	ToBlock     *rpc.BlockNumber `json:"toBlock"`
	FromAddress []common.Address `json:"fromAddress"`
	ToAddress   []common.Address `json:"toAddress"`
	Mode        FilterMode       `json:"mode"`
	After       *uint64          `json:"after"`
	Count       *uint64          `json:"count"`
}

// FilterMode selects how populated address lists combine. The zero value is intersection.
type FilterMode string

const (
	FilterIntersection FilterMode = "intersection"
	FilterUnion        FilterMode = "union"
)

func (m *FilterMode) UnmarshalJSON(input []byte) error {
	var value string
	if err := json.Unmarshal(input, &value); err != nil {
		return fmt.Errorf("mode must be a string")
	}
	switch mode := FilterMode(value); mode {
	case FilterIntersection, FilterUnion:
		*m = mode
		return nil
	}
	return fmt.Errorf("unknown filter mode %q", value)
}

// filterMembers are the FilterArgs member names, which encoding/json
// matches case-insensitively.
var filterMembers = []string{"blockHash", "fromBlock", "toBlock", "fromAddress", "toAddress", "mode", "after", "count"}

func (f *FilterArgs) UnmarshalJSON(input []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(input, &fields); err != nil {
		return err
	}
	if fields == nil {
		return fmt.Errorf("filter must be an object")
	}
	// An explicit null is an omitted member: a null mode is intersection and
	// null address lists are unrestricted, like omitted or empty lists.
	for key, value := range fields {
		member := slices.ContainsFunc(filterMembers, func(name string) bool { return strings.EqualFold(name, key) })
		if member && bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			delete(fields, key)
		}
	}
	filtered, err := json.Marshal(fields)
	if err != nil {
		return err
	}
	type plain FilterArgs
	*f = FilterArgs{}
	dec := json.NewDecoder(bytes.NewReader(filtered))
	dec.DisallowUnknownFields()
	return dec.Decode((*plain)(f))
}

// Execution is the output of one transaction, identified by its hash when
// it was replayed.
type Execution struct {
	*parity.Result
	TransactionHash *common.Hash `json:"transactionHash,omitempty"`
}

type rewardAction struct {
	Author     common.Address `json:"author"`
	Value      *hexutil.Big   `json:"value"`
	RewardType string         `json:"rewardType"`
}
