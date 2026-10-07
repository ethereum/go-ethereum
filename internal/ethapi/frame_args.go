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

package ethapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"slices"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

// FrameArgs preserves omitted gas limits until RPC preparation.
type FrameArgs struct {
	Mode         *hexutil.Uint64 `json:"mode"`
	Flags        hexutil.Uint64  `json:"flags"`
	Target       *common.Address `json:"target,omitempty"`
	ExecutionGas *hexutil.Uint64 `json:"executionGas"`
	StateGas     *hexutil.Uint64 `json:"stateGas"`
	Value        *hexutil.Big    `json:"value"`
	Data         hexutil.Bytes   `json:"data"`
}

// UnmarshalJSON rejects null gas limits, which are different from omitted limits.
func (f *FrameArgs) UnmarshalJSON(input []byte) error {
	type frameArgs FrameArgs
	var decoded frameArgs
	if err := json.Unmarshal(input, &decoded); err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(input, &fields); err != nil {
		return err
	}
	for _, name := range []string{"executionGas", "stateGas"} {
		if bytes.Equal(bytes.TrimSpace(fields[name]), []byte("null")) {
			return fmt.Errorf("%s cannot be null", name)
		}
	}
	*f = FrameArgs(decoded)
	return nil
}

// FrameSignatureArgs distinguishes placeholders from complete empty witnesses.
type FrameSignatureArgs struct {
	Scheme    *hexutil.Uint64 `json:"scheme"`
	Signer    hexutil.Bytes   `json:"signer,omitempty"`
	Msg       hexutil.Bytes   `json:"msg,omitempty"`
	Signature *hexutil.Bytes  `json:"signature,omitempty"`
}

func (args *TransactionArgs) isFrame() bool {
	return args.Frames != nil || args.Signatures != nil || args.Type != nil && uint64(*args.Type) == types.FrameTxType
}

func (args *TransactionArgs) frameDefaults(cap uint64) error {
	if args.Type != nil && uint64(*args.Type) != types.FrameTxType {
		return errors.New("frames require transaction type 0x6")
	}
	if len(args.Frames) == 0 || len(args.Frames) > params.FrameTxMaxFrames {
		return errors.New("invalid frame count")
	}
	if args.GasPrice != nil || args.To != nil || args.Value != nil && args.Value.ToInt().Sign() != 0 || len(args.data()) != 0 || args.AuthorizationList != nil || args.Blobs != nil || args.Commitments != nil || args.Proofs != nil {
		return errors.New("frame transactions cannot contain legacy transaction fields")
	}
	if cap == 0 {
		cap = params.MaxTxGas
	}
	// Copy before filling defaults because callers may reuse their request.
	args.Frames = slices.Clone(args.Frames)
	for i := range args.Frames {
		f := &args.Frames[i]
		if f.Mode == nil || *f.Mode > 2 || f.Flags > 4 {
			return fmt.Errorf("invalid frame %d mode or flags", i)
		}
		if f.Value == nil {
			f.Value = new(hexutil.Big)
		}
		if f.Value.ToInt().Sign() < 0 || f.Value.ToInt().BitLen() > 256 {
			return fmt.Errorf("invalid frame %d value", i)
		}
		if f.ExecutionGas == nil {
			v := hexutil.Uint64(cap)
			f.ExecutionGas = &v
		}
		if f.StateGas == nil {
			v := hexutil.Uint64(cap)
			f.StateGas = &v
		}
	}
	for i, s := range args.Signatures {
		if s.Scheme == nil || *s.Scheme > 2 {
			return fmt.Errorf("invalid signature %d scheme", i)
		}
		if len(s.Signer) != 0 && (len(s.Signer) != common.AddressLength || *s.Scheme == 0) {
			return fmt.Errorf("invalid signature %d signer", i)
		}
		if len(s.Msg) != 0 && (len(s.Msg) != common.HashLength || common.BytesToHash(s.Msg) == (common.Hash{})) {
			return fmt.Errorf("invalid signature %d message", i)
		}
		if s.Signature != nil && len(*s.Signature) != 0 {
			b := *s.Signature
			if *s.Scheme == 1 && (len(b) != 65 || b[0] > 1) || *s.Scheme == 2 && len(b) != 128 {
				return fmt.Errorf("invalid signature %d encoding", i)
			}
		}
	}
	return nil
}

func (args *TransactionArgs) frameTransaction() *types.Transaction {
	frames := make([]types.Frame, len(args.Frames))
	for i, f := range args.Frames {
		frames[i] = types.Frame{Mode: uint64(*f.Mode), Flags: uint64(f.Flags), Target: f.Target, GasLimits: types.Limits{Execution: uint64(*f.ExecutionGas), State: uint64(*f.StateGas)}, Value: uint256.MustFromBig(f.Value.ToInt()), Data: f.Data}
	}
	signatures := make(types.SignatureList, len(args.Signatures))
	for i, s := range args.Signatures {
		var signature []byte
		if s.Signature != nil && len(*s.Signature) != 0 {
			signature = *s.Signature
		} else {
			// Reserve the full nonzero-byte cost of protocol signatures during preparation.
			switch uint64(*s.Scheme) {
			case types.FrameTxSchemeSecp256k1:
				signature = bytes.Repeat([]byte{1}, 65)
			case types.FrameTxSchemeP256:
				signature = bytes.Repeat([]byte{1}, 128)
			}
		}
		signatures[i] = types.SignatureEntry{Scheme: uint64(*s.Scheme), Signer: s.Signer, Msg: s.Msg, Signature: signature}
	}
	return types.NewTx(&types.FrameTx{ChainID: uint256.MustFromBig(args.ChainID.ToInt()), Nonce: uint64(*args.Nonce), Sender: args.from(), Frames: frames, Signatures: signatures, Fees: types.Fees{MaxFeePerGas: uint256.MustFromBig(args.MaxFeePerGas.ToInt()), MaxPriorityFeePerGas: uint256.MustFromBig(args.MaxPriorityFeePerGas.ToInt()), MaxFeePerBlobGas: uint256.MustFromBig(args.BlobFeeCap.ToInt())}, BlobVersionedHashes: args.BlobHashes})
}

// prepareFrames estimates only omitted dimensions, preserving explicit limits.
func (args *TransactionArgs) prepareFrames(ctx context.Context, b Backend, db *state.StateDB, header *types.Header, blockContext *vm.BlockContext, precompiles vm.PrecompiledContracts, gasCap uint64) error {
	if !b.ChainConfig().IsBogota(header.Number, header.Time) {
		return errors.New("frame transactions are not active")
	}
	cap := header.GasLimit
	if gasCap != 0 {
		cap = min(cap, gasCap)
	}
	missing := make([][2]bool, len(args.Frames))
	for i, f := range args.Frames {
		missing[i] = [2]bool{f.ExecutionGas == nil, f.StateGas == nil}
	}
	if err := args.CallDefaults(cap, header.BaseFee, b.ChainConfig().ChainID); err != nil {
		return err
	}
	for _, frame := range args.Frames {
		if uint64(*frame.ExecutionGas) > cap || uint64(*frame.StateGas) > cap {
			return fmt.Errorf("frame gas exceeds allowance (%d)", cap)
		}
	}
	shape := args.frameTransaction()
	for i := range shape.Frames() {
		if missing[i][0] {
			shape.Frames()[i].GasLimits.Execution = 0
		}
		if missing[i][1] {
			shape.Frames()[i].GasLimits.State = 0
		}
	}
	if err := shape.FrameTxValidateStatic(); err != nil {
		return err
	}
	if shape.Gas() > cap {
		return fmt.Errorf("frame gas required exceeds allowance (%d)", cap)
	}
	run := func() (*core.ExecutionResult, error) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		bc := *blockContext
		probeCtx, cancel := context.WithCancel(ctx)
		defer cancel()
		return applyMessage(probeCtx, b, *args, db.Copy(), header, b.RPCEVMTimeout(), core.NewGasPool(math.MaxUint64), &bc, &vm.Config{NoBaseFee: true}, precompiles)
	}
	hasMissing := false
	for _, dimensions := range missing {
		hasMissing = hasMissing || dimensions[0] || dimensions[1]
	}
	if hasMissing {
		for i, dimensions := range missing {
			if dimensions[0] {
				v := hexutil.Uint64(1)
				args.Frames[i].ExecutionGas = &v
			}
			if dimensions[1] {
				v := hexutil.Uint64(1)
				args.Frames[i].StateGas = &v
			}
		}
		var baseline *core.ExecutionResult
		for {
			result, err := run()
			failed := -1
			if err != nil {
				var frameErr *core.FrameExecutionError
				if !errors.As(err, &frameErr) || !errors.Is(err, vm.ErrOutOfGas) {
					return err
				}
				failed = frameErr.Index
			} else {
				for i, output := range result.FrameResults {
					if errors.Is(output.Err, vm.ErrOutOfGas) && (missing[i][0] || missing[i][1]) {
						failed = i
						break
					}
				}
			}
			if failed == -1 {
				baseline = result
				break
			}
			grew := false
			for dim, omitted := range missing[failed] {
				if !omitted {
					continue
				}
				field := &args.Frames[failed].ExecutionGas
				if dim == 1 {
					field = &args.Frames[failed].StateGas
				}
				previous := uint64(**field)
				total := args.frameTransaction().Gas()
				next := previous
				if total < cap {
					next += min(previous, cap-total)
				}
				if next > previous {
					v := hexutil.Uint64(next)
					*field = &v
					grew = true
				}
			}
			if !grew {
				if err != nil {
					return err
				}
				return vm.ErrOutOfGas
			}
		}

		for i, dimensions := range missing {
			for dim, omitted := range dimensions {
				if !omitted {
					continue
				}
				field := &args.Frames[i].ExecutionGas
				if dim == 1 {
					field = &args.Frames[i].StateGas
				}
				lo, hi := uint64(0), uint64(**field)
				for lo < hi {
					mid := lo + (hi-lo)/2
					value := hexutil.Uint64(mid)
					*field = &value
					result, err := run()
					if ctx.Err() != nil {
						return ctx.Err()
					}
					if err == nil && sameFrameOutcomes(baseline, result) {
						hi = mid
					} else {
						lo = mid + 1
					}
				}
				value := hexutil.Uint64(hi)
				*field = &value
			}
		}
	}
	tx := args.frameTransaction()
	if err := tx.FrameTxValidateStatic(); err != nil {
		return err
	}
	if tx.Gas() > cap {
		return fmt.Errorf("frame gas required exceeds allowance (%d)", cap)
	}
	gas := hexutil.Uint64(tx.Gas())
	args.Gas = &gas
	return nil
}

func sameFrameOutcomes(a, b *core.ExecutionResult) bool {
	if len(a.FrameReceipts) != len(b.FrameReceipts) {
		return false
	}
	for i, receipt := range a.FrameReceipts {
		if receipt.Status != b.FrameReceipts[i].Status || !bytes.Equal(a.FrameResults[i].ReturnData, b.FrameResults[i].ReturnData) || !errors.Is(b.FrameResults[i].Err, a.FrameResults[i].Err) {
			return false
		}
	}
	return true
}
