// Copyright 2024 the libevm authors.
//
// The libevm additions to go-ethereum are free software: you can redistribute
// them and/or modify them under the terms of the GNU Lesser General Public License
// as published by the Free Software Foundation, either version 3 of the License,
// or (at your option) any later version.
//
// The libevm additions are distributed in the hope that they will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU Lesser
// General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see
// <http://www.gnu.org/licenses/>.
package params

import (
	"bytes"
	"encoding/json"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ava-labs/libevm/libevm/pseudo"
)

type nestedChainConfigExtra struct {
	NestedFoo string `json:"foo"`

	NOOPHooks
}

type rootJSONChainConfigExtra struct {
	TopLevelFoo string `json:"foo"`

	NOOPHooks
}

func TestChainConfigJSONRoundTrip(t *testing.T) {
	tests := []struct {
		name      string
		register  func()
		jsonInput string
		want      *ChainConfig
	}{
		{
			name:     "no registered extras",
			register: func() {},
			jsonInput: `{
				"chainId": 1234
			}`,
			want: &ChainConfig{
				ChainID: big.NewInt(1234),
			},
		},
		{
			name: "reuse top-level JSON with non-pointer",
			register: func() {
				RegisterExtras(Extras[rootJSONChainConfigExtra, NOOPHooks]{
					ReuseJSONRoot: true,
				})
			},
			jsonInput: `{
				"chainId": 5678,
				"foo": "hello"
			}`,
			want: &ChainConfig{
				ChainID: big.NewInt(5678),
				extra:   pseudo.From(rootJSONChainConfigExtra{TopLevelFoo: "hello"}).Type,
			},
		},
		{
			name: "reuse top-level JSON with pointer",
			register: func() {
				RegisterExtras(Extras[*rootJSONChainConfigExtra, NOOPHooks]{
					ReuseJSONRoot: true,
				})
			},
			jsonInput: `{
				"chainId": 5678,
				"foo": "hello"
			}`,
			want: &ChainConfig{
				ChainID: big.NewInt(5678),
				extra:   pseudo.From(&rootJSONChainConfigExtra{TopLevelFoo: "hello"}).Type,
			},
		},
		{
			name: "nested JSON with non-pointer",
			register: func() {
				RegisterExtras(Extras[nestedChainConfigExtra, NOOPHooks]{
					ReuseJSONRoot: false, // explicit zero value only for tests
				})
			},
			jsonInput: `{
				"chainId": 42,
				"extra": {"foo": "world"}
			}`,
			want: &ChainConfig{
				ChainID: big.NewInt(42),
				extra:   pseudo.From(nestedChainConfigExtra{NestedFoo: "world"}).Type,
			},
		},
		{
			name: "nested JSON with pointer",
			register: func() {
				RegisterExtras(Extras[*nestedChainConfigExtra, NOOPHooks]{
					ReuseJSONRoot: false, // explicit zero value only for tests
				})
			},
			jsonInput: `{
				"chainId": 42,
				"extra": {"foo": "world"}
			}`,
			want: &ChainConfig{
				ChainID: big.NewInt(42),
				extra:   pseudo.From(&nestedChainConfigExtra{NestedFoo: "world"}).Type,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			TestOnlyClearRegisteredExtras()
			t.Cleanup(TestOnlyClearRegisteredExtras)
			tt.register()

			t.Run("json.Unmarshal()", func(t *testing.T) {
				got := new(ChainConfig)
				require.NoError(t, json.Unmarshal([]byte(tt.jsonInput), got))
				require.Equal(t, tt.want, got)
			})

			t.Run("json.Marshal()", func(t *testing.T) {
				var want bytes.Buffer
				require.NoError(t, json.Compact(&want, []byte(tt.jsonInput)), "json.Compact()")

				got, err := json.Marshal(tt.want)
				require.NoError(t, err, "json.Marshal()")
				require.Equal(t, want.String(), string(got))
			})
		})
	}
}

func TestUnmarshalChainConfigJSON_Errors(t *testing.T) {
	t.Parallel()

	type testExtra struct {
		Field string `json:"field"`
	}

	testCases := map[string]struct {
		jsonData      string // string for convenience
		extra         *testExtra
		reuseJSONRoot bool
		wantConfig    ChainConfig
		wantExtra     any
		wantErrRegex  string
	}{
		"invalid_json": {
			extra:        &testExtra{},
			wantExtra:    &testExtra{},
			wantErrRegex: `^decoding JSON into combination of \*.+\.ChainConfig and \*.+\.testExtra \(as "extra" key\): .+$`,
		},
		"nil_extra_at_root_depth": {
			jsonData:      `{"chainId": 1}`,
			extra:         nil,
			reuseJSONRoot: true,
			wantExtra:     (*testExtra)(nil),
			wantErrRegex:  `^\*.+.testExtra argument is nil; use \*.+\.ChainConfig\.UnmarshalJSON\(\) directly$`,
		},
		"nil_extra_at_extra_key": {
			jsonData:     `{"chainId": 1}`,
			extra:        nil,
			wantExtra:    (*testExtra)(nil),
			wantErrRegex: `^\*.+\.testExtra argument is nil; use \*.+\.ChainConfig.UnmarshalJSON\(\) directly$`,
		},
		"wrong_extra_type_at_extra_key": {
			jsonData:     `{"chainId": 1, "extra": 1}`,
			extra:        &testExtra{},
			wantConfig:   ChainConfig{ChainID: big.NewInt(1)},
			wantExtra:    &testExtra{},
			wantErrRegex: `^decoding JSON into combination of \*.+\.ChainConfig and \*.+\.testExtra \(as "extra" key\): .+$`,
		},
		"wrong_extra_type_at_root_depth": {
			jsonData:      `{"chainId": 1, "field": 1}`,
			extra:         &testExtra{},
			reuseJSONRoot: true,
			wantConfig:    ChainConfig{ChainID: big.NewInt(1)},
			wantExtra:     &testExtra{},
			wantErrRegex:  `^decoding JSON into \*.+\.testExtra: .+`,
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			data := []byte(testCase.jsonData)
			config := ChainConfig{}
			err := UnmarshalChainConfigJSON(data, &config, testCase.extra, testCase.reuseJSONRoot)
			if testCase.wantErrRegex == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.Regexp(t, testCase.wantErrRegex, err.Error())
			}
			assert.Equal(t, testCase.wantConfig, config)
			assert.Equal(t, testCase.wantExtra, testCase.extra)
		})
	}
}

func TestMarshalChainConfigJSON_Errors(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		config        ChainConfig
		extra         any
		reuseJSONRoot bool
		wantJSONData  string // string for convenience
		wantErrRegex  string
	}{
		"invalid_extra_at_extra_key": {
			extra: struct {
				Field chan struct{} `json:"field"`
			}{},
			wantErrRegex: `^encoding combination of .+\.ChainConfig and .+ to JSON: .+$`,
		},
		"nil_extra_at_extra_key": {
			wantJSONData: `{"chainId":null}`,
		},
		"invalid_extra_at_root_depth": {
			extra: struct {
				Field chan struct{} `json:"field"`
			}{},
			reuseJSONRoot: true,
			wantErrRegex:  "^converting extra config to JSON raw messages: .+$",
		},
		"duplicate_key": {
			extra: struct {
				Field string `json:"chainId"`
			}{},
			reuseJSONRoot: true,
			wantErrRegex:  `^duplicate JSON key "chainId" in ChainConfig and extra struct .+$`,
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			config := ChainConfig{}
			data, err := MarshalChainConfigJSON(config, testCase.extra, testCase.reuseJSONRoot)
			if testCase.wantErrRegex == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Regexp(t, testCase.wantErrRegex, err.Error())
			}
			assert.Equal(t, testCase.wantJSONData, string(data))
		})
	}
}
