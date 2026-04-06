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
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package tracetest

import (
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/tests"
	"github.com/holiman/uint256"
)

// prestateTrace is the result of a prestateTrace run.
type prestateTrace = map[common.Address]*account

type account struct {
	Balance string                      `json:"balance"`
	Code    string                      `json:"code"`
	Nonce   uint64                      `json:"nonce"`
	Storage map[common.Hash]common.Hash `json:"storage"`
}

// prestateTracerTest defines a single test to check the stateDiff tracer against.
type prestateTracerTest struct {
	tracerTestEnv
	Result interface{} `json:"result"`
}

func TestPrestateWithDiffMode_7702Deauth(t *testing.T) {
	chainConfig := params.AllDevChainProtocolChanges
	authorityKey, _ := crypto.GenerateKey()
	senderKey, _ := crypto.GenerateKey()
	authorityAddr := crypto.PubkeyToAddress(authorityKey.PublicKey)
	senderAddr := crypto.PubkeyToAddress(senderKey.PublicKey)
	delegateTarget := common.HexToAddress("0x000000000000000000000000000000000000aaaa")
	genesis := &core.Genesis{
		Config:   chainConfig,
		BaseFee:  big.NewInt(params.InitialBaseFee),
		GasLimit: 8000000,
		Alloc: types.GenesisAlloc{
			authorityAddr: {
				Balance: big.NewInt(1e18),
				Code:    types.AddressToDelegation(delegateTarget),
			},
			senderAddr: {
				Balance: big.NewInt(1e18),
			},
		},
	}
	signer := types.LatestSignerForChainID(chainConfig.ChainID)
	auth, err := types.SignSetCode(authorityKey, types.SetCodeAuthorization{
		ChainID: *uint256.MustFromBig(chainConfig.ChainID),
		Address: common.Address{},
		Nonce:   0,
	})
	if err != nil {
		t.Fatalf("failed to sign authorization: %v", err)
	}
	tx := types.MustSignNewTx(senderKey, signer, &types.SetCodeTx{
		ChainID:   uint256.MustFromBig(chainConfig.ChainID),
		Nonce:     0,
		GasFeeCap: uint256.NewInt(params.InitialBaseFee * 2),
		GasTipCap: uint256.NewInt(1),
		Gas:       100000,
		To:        senderAddr,
		AuthList:  []types.SetCodeAuthorization{auth},
	})
	state := tests.MakePreState(rawdb.NewMemoryDatabase(), genesis.Alloc, false, rawdb.HashScheme)
	defer state.Close()
	tracerCfg := json.RawMessage(`{"diffMode":true}`)
	tracer, err := tracers.DefaultDirectory.New("prestateTracer", new(tracers.Context), tracerCfg, chainConfig)
	if err != nil {
		t.Fatalf("failed to create tracer: %v", err)
	}
	blockCtx := vm.BlockContext{
		CanTransfer: core.CanTransfer,
		Transfer:    core.Transfer,
		BlockNumber: big.NewInt(1),
		Time:        0,
		GasLimit:    8000000,
		BaseFee:     big.NewInt(params.InitialBaseFee),
	}
	msg, err := core.TransactionToMessage(tx, signer, blockCtx.BaseFee)
	if err != nil {
		t.Fatalf("failed to convert tx to message: %v", err)
	}
	evm := vm.NewEVM(blockCtx, state.StateDB, chainConfig, vm.Config{Tracer: tracer.Hooks})
	tracer.OnTxStart(evm.GetVMContext(), tx, msg.From)
	vmRet, err := core.ApplyMessage(evm, msg, nil)
	if err != nil {
		t.Fatalf("failed to apply message: %v", err)
	}
	tracer.OnTxEnd(&types.Receipt{GasUsed: vmRet.UsedGas}, nil)

	res, err := tracer.GetResult()
	if err != nil {
		t.Fatalf("failed to get result: %v", err)
	}
	var result struct {
		Post map[common.Address]json.RawMessage `json:"post"`
	}
	if err := json.Unmarshal(res, &result); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	postRaw, ok := result.Post[authorityAddr]
	if !ok {
		t.Fatalf("authority address not found in post state; full result: %s", res)
	}
	var postAccount struct {
		Code     string `json:"code"`
		CodeHash string `json:"codeHash"`
	}
	if err := json.Unmarshal(postRaw, &postAccount); err != nil {
		t.Fatalf("failed to unmarshal post account: %v", err)
	}
	if postAccount.Code != "0x" {
		t.Errorf("post code: got %q, want %q", postAccount.Code, "0x")
	}
	wantCodeHash := types.EmptyCodeHash.Hex()
	if postAccount.CodeHash != wantCodeHash {
		t.Errorf("post codeHash: got %q, want %q", postAccount.CodeHash, wantCodeHash)
	}
}

func TestPrestateTracerLegacy(t *testing.T) {
	testPrestateTracer("prestateTracerLegacy", "prestate_tracer_legacy", t)
}

func TestPrestateTracer(t *testing.T) {
	testPrestateTracer("prestateTracer", "prestate_tracer", t)
}

func TestPrestateWithDiffModeTracer(t *testing.T) {
	testPrestateTracer("prestateTracer", "prestate_tracer_with_diff_mode", t)
}

func testPrestateTracer(tracerName string, dirPath string, t *testing.T) {
	files, err := os.ReadDir(filepath.Join("testdata", dirPath))
	if err != nil {
		t.Fatalf("failed to retrieve tracer test suite: %v", err)
	}
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".json") {
			continue
		}
		t.Run(camel(strings.TrimSuffix(file.Name(), ".json")), func(t *testing.T) {
			t.Parallel()

			var (
				test = new(prestateTracerTest)
				tx   = new(types.Transaction)
			)
			// Call tracer test found, read if from disk
			if blob, err := os.ReadFile(filepath.Join("testdata", dirPath, file.Name())); err != nil {
				t.Fatalf("failed to read testcase: %v", err)
			} else if err := json.Unmarshal(blob, test); err != nil {
				t.Fatalf("failed to parse testcase: %v", err)
			}
			if err := tx.UnmarshalBinary(common.FromHex(test.Input)); err != nil {
				t.Fatalf("failed to parse testcase input: %v", err)
			}
			// Configure a blockchain with the given prestate
			var (
				signer  = types.MakeSigner(test.Genesis.Config, new(big.Int).SetUint64(uint64(test.Context.Number)), uint64(test.Context.Time))
				context = test.Context.toBlockContext(test.Genesis)
				state   = tests.MakePreState(rawdb.NewMemoryDatabase(), test.Genesis.Alloc, false, rawdb.HashScheme)
			)
			defer state.Close()

			tracer, err := tracers.DefaultDirectory.New(tracerName, new(tracers.Context), test.TracerConfig, test.Genesis.Config)
			if err != nil {
				t.Fatalf("failed to create call tracer: %v", err)
			}

			msg, err := core.TransactionToMessage(tx, signer, context.BaseFee)
			if err != nil {
				t.Fatalf("failed to prepare transaction for tracing: %v", err)
			}
			evm := vm.NewEVM(context, state.StateDB, test.Genesis.Config, vm.Config{Tracer: tracer.Hooks})
			tracer.OnTxStart(evm.GetVMContext(), tx, msg.From)
			vmRet, err := core.ApplyMessage(evm, msg, nil)
			if err != nil {
				t.Fatalf("failed to execute transaction: %v", err)
			}
			if vmRet.Failed() {
				t.Logf("(warn) transaction failed: %v", vmRet.Err)
			}
			tracer.OnTxEnd(&types.Receipt{GasUsed: vmRet.UsedGas}, nil)
			// Retrieve the trace result and compare against the expected
			res, err := tracer.GetResult()
			if err != nil {
				t.Fatalf("failed to retrieve trace result: %v", err)
			}
			// The legacy javascript calltracer marshals json in js, which
			// is not deterministic (as opposed to the golang json encoder).
			if strings.HasSuffix(dirPath, "_legacy") {
				// This is a tweak to make it deterministic. Can be removed when
				// we remove the legacy tracer.
				var x prestateTrace
				json.Unmarshal(res, &x)
				res, _ = json.Marshal(x)
			}
			want, err := json.Marshal(test.Result)
			if err != nil {
				t.Fatalf("failed to marshal test: %v", err)
			}
			if string(want) != string(res) {
				t.Fatalf("trace mismatch\n have: %v\n want: %v\n", string(res), string(want))
			}
		})
	}
}
