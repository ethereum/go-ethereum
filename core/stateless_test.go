// Copyright 2024 The go-ethereum Authors
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

package core

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/stateless"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"

	_ "embed"
)

//go:embed stateless_testdata/witness.rlp
var witnessData []byte

//go:embed stateless_testdata/block.rlp
var blockData []byte

//go:embed stateless_testdata/chain_config.json
var chainConfigData []byte

func TestExecuteStateless(t *testing.T) {
	var block types.Block
	if err := rlp.DecodeBytes(blockData, &block); err != nil {
		t.Fatalf("failed to unmarshal block: %v", err)
	}
	var witness stateless.Witness
	if err := rlp.DecodeBytes(witnessData, &witness); err != nil {
		t.Fatalf("failed to unmarshal witness: %v", err)
	}
	var config params.ChainConfig
	if err := json.Unmarshal(chainConfigData, &config); err != nil {
		t.Fatalf("failed to unmarshal chain config: %v", err)
	}
	type args struct {
		config   *params.ChainConfig
		vmconfig vm.Config
		block    *types.Block
		witness  *stateless.Witness
	}
	tests := []struct {
		name    string
		args    args
		want    common.Hash
		want1   common.Hash
		wantErr bool
	}{
		{
			"issue:https://github.com/ethereum/go-ethereum/issues/31631",
			args{
				&config,
				vm.Config{},
				&block,
				&witness,
			},
			common.HexToHash("0x9175f245714f8595e099ce523f329f45fde1fe92b57ea10059e4243512ae931d"),
			common.HexToHash("0x19d5540a8f734f731969baa459be6a745da2ca3c0397a41605bc6548dc960bb3"),
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := ExecuteStateless(tt.args.config, tt.args.vmconfig, tt.args.block, tt.args.witness)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExecuteStateless() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ExecuteStateless() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("ExecuteStateless() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
