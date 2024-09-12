package params

import (
	"bytes"
	"encoding/json"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/libevm/pseudo"
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
