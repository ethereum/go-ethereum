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
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package params_test

import (
	"math/big"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/params/forks"
	"github.com/ethereum/go-ethereum/params/presets"
)

func TestConfigValidateErrors(t *testing.T) {
	files, err := filepath.Glob("testdata/invalid-*.json")
	if err != nil {
		t.Fatal(err)
	}

	type test struct {
		Config params.Config2 `json:"config"`
		Error  string         `json:"error"`
	}

	for _, f := range files {
		name := filepath.Base(f)
		t.Run(name, func(t *testing.T) {
			var test test
			if err := common.LoadJSON(f, &test); err != nil {
				t.Fatal(err)
			}
			err := test.Config.Validate()
			if err == nil {
				t.Fatal("expected validation error, got none")
			}
			if err.Error() != test.Error {
				t.Fatal("wrong error:\n got:", err.Error(), "want:", test.Error)
			}
		})
	}
}

func TestCheckCompatible2(t *testing.T) {
	type test struct {
		stored, new   *params.Config2
		headBlock     uint64
		headTimestamp uint64
		wantErr       *params.ConfigCompatError
	}
	tests := []test{
		{stored: presets.AllEthashProtocolChanges, new: presets.AllEthashProtocolChanges, headBlock: 0, headTimestamp: 0, wantErr: nil},
		{stored: presets.AllEthashProtocolChanges, new: presets.AllEthashProtocolChanges, headBlock: 0, headTimestamp: uint64(time.Now().Unix()), wantErr: nil},
		{stored: presets.AllEthashProtocolChanges, new: presets.AllEthashProtocolChanges, headBlock: 100, wantErr: nil},
		{
			// Here we check that it's OK to reschedule a time-based fork that's still in the future.
			stored:    params.NewConfig2(params.Activations{forks.SpuriousDragon: 10}),
			new:       params.NewConfig2(params.Activations{forks.SpuriousDragon: 20}),
			headBlock: 9,
			wantErr:   nil,
		},
		{
			stored:    presets.AllEthashProtocolChanges,
			new:       params.NewConfig2(params.Activations{}),
			headBlock: 3,
			wantErr: &params.ConfigCompatError{
				What:          "arrowGlacierBlock",
				StoredBlock:   big.NewInt(0),
				NewBlock:      nil,
				RewindToBlock: 0,
			},
		},
		{
			stored:    presets.AllEthashProtocolChanges,
			new:       params.NewConfig2(params.Activations{forks.ArrowGlacier: 1}),
			headBlock: 3,
			wantErr: &params.ConfigCompatError{
				What:          "arrowGlacierBlock",
				StoredBlock:   big.NewInt(0),
				NewBlock:      big.NewInt(1),
				RewindToBlock: 0,
			},
		},
		{
			stored: params.NewConfig2(params.Activations{
				forks.Homestead:        30,
				forks.TangerineWhistle: 10,
			}),
			new: params.NewConfig2(params.Activations{
				forks.Homestead:        25,
				forks.TangerineWhistle: 20,
			}),
			headBlock: 25,
			wantErr: &params.ConfigCompatError{
				What:          "eip150Block",
				StoredBlock:   big.NewInt(10),
				NewBlock:      big.NewInt(20),
				RewindToBlock: 9,
			},
		},
		{
			// Special case for Petersburg, which activates with Constantinople if undefined.
			stored:    params.NewConfig2(params.Activations{forks.Constantinople: 30}),
			new:       params.NewConfig2(params.Activations{forks.Constantinople: 30, forks.Petersburg: 30}),
			headBlock: 40,
			wantErr:   nil,
		},
		{
			// If Petersburg and Constantinople are scheduled to different blocks, the compatibility check is stricter.
			stored:    params.NewConfig2(params.Activations{forks.Constantinople: 30}),
			new:       params.NewConfig2(params.Activations{forks.Constantinople: 30, forks.Petersburg: 31}),
			headBlock: 40,
			wantErr: &params.ConfigCompatError{
				What:          "petersburgBlock",
				StoredBlock:   nil,
				NewBlock:      big.NewInt(31),
				RewindToBlock: 30,
			},
		},
		{
			// This one checks that it's OK to reschedule a time-based fork that's still in the future.
			stored:        params.NewConfig2(params.Activations{forks.Shanghai: 10}),
			new:           params.NewConfig2(params.Activations{forks.Shanghai: 20}),
			headTimestamp: 9,
			wantErr:       nil,
		},
		{
			// Here's an error for the config from the previous test, the chain has passed the
			// fork in the stored configuration, so it cannot be rescheduled.
			stored:        params.NewConfig2(params.Activations{forks.Shanghai: 10}),
			new:           params.NewConfig2(params.Activations{forks.Shanghai: 20}),
			headTimestamp: 25,
			wantErr: &params.ConfigCompatError{
				What:         "shanghaiTime",
				StoredTime:   newUint64(10),
				NewTime:      newUint64(20),
				RewindToTime: 9,
			},
		},
	}

	for _, test := range tests {
		err := test.stored.CheckCompatible(test.new, test.headBlock, test.headTimestamp)
		if !reflect.DeepEqual(err, test.wantErr) {
			t.Errorf("error mismatch:\nstored: %v\nnew: %v\nheadBlock: %v\nheadTimestamp: %v\nerr: %v\nwant: %v", test.stored, test.new, test.headBlock, test.headTimestamp, err, test.wantErr)
		}
	}
}

func newUint64(i uint64) *uint64 {
	return &i
}
