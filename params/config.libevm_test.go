package params

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/libevm/pseudo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type rawJSON struct {
	json.RawMessage
	NOOPHooks
}

var _ interface {
	json.Marshaler
	json.Unmarshaler
} = (*rawJSON)(nil)

func TestRegisterExtras(t *testing.T) {
	type (
		ccExtraA struct {
			A string `json:"a"`
			ChainConfigHooks
		}
		rulesExtraA struct {
			A string
			RulesHooks
		}
		ccExtraB struct {
			B string `json:"b"`
			ChainConfigHooks
		}
		rulesExtraB struct {
			B string
			RulesHooks
		}
	)

	tests := []struct {
		name           string
		register       func()
		ccExtra        *pseudo.Type
		wantRulesExtra any
	}{
		{
			name: "Rules payload copied from ChainConfig payload",
			register: func() {
				RegisterExtras(Extras[ccExtraA, rulesExtraA]{
					NewRules: func(cc *ChainConfig, r *Rules, ex *ccExtraA, _ *big.Int, _ bool, _ uint64) *rulesExtraA {
						return &rulesExtraA{
							A: ex.A,
						}
					},
				})
			},
			ccExtra: pseudo.From(&ccExtraA{
				A: "hello",
			}).Type,
			wantRulesExtra: &rulesExtraA{
				A: "hello",
			},
		},
		{
			name: "no NewForRules() function results in typed but nil pointer",
			register: func() {
				RegisterExtras(Extras[ccExtraB, rulesExtraB]{})
			},
			ccExtra: pseudo.From(&ccExtraB{
				B: "world",
			}).Type,
			wantRulesExtra: (*rulesExtraB)(nil),
		},
		{
			name: "custom JSON handling honoured",
			register: func() {
				RegisterExtras(Extras[rawJSON, struct{ RulesHooks }]{})
			},
			ccExtra: pseudo.From(&rawJSON{
				RawMessage: []byte(`"hello, world"`),
			}).Type,
			wantRulesExtra: (*struct{ RulesHooks })(nil),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			TestOnlyClearRegisteredExtras()
			tt.register()
			defer TestOnlyClearRegisteredExtras()

			in := &ChainConfig{
				ChainID: big.NewInt(142857),
				extra:   tt.ccExtra,
			}

			buf, err := json.Marshal(in)
			require.NoError(t, err)

			got := new(ChainConfig)
			require.NoError(t, json.Unmarshal(buf, got))
			assert.Equal(t, tt.ccExtra.Interface(), got.extraPayload().Interface())
			assert.Equal(t, in, got)

			gotRules := got.Rules(nil, false, 0)
			assert.Equal(t, tt.wantRulesExtra, gotRules.extraPayload().Interface())
		})
	}
}

func TestExtrasPanic(t *testing.T) {
	TestOnlyClearRegisteredExtras()
	defer TestOnlyClearRegisteredExtras()

	assertPanics(
		t, func() {
			new(ChainConfig).extraPayload()
		},
		"before RegisterExtras",
	)

	assertPanics(
		t, func() {
			new(Rules).extraPayload()
		},
		"before RegisterExtras",
	)

	assertPanics(
		t, func() {
			mustBeStruct[int]()
		},
		notStructMessage[int](),
	)

	RegisterExtras(Extras[struct{ ChainConfigHooks }, struct{ RulesHooks }]{})

	assertPanics(
		t, func() {
			RegisterExtras(Extras[struct{ ChainConfigHooks }, struct{ RulesHooks }]{})
		},
		"re-registration",
	)
}

func assertPanics(t *testing.T, fn func(), wantContains string) {
	t.Helper()
	defer func() {
		switch r := recover().(type) {
		case nil:
			t.Error("function did not panic as expected")
		case string:
			assert.Contains(t, r, wantContains)
		default:
			t.Fatalf("BAD TEST SETUP: recover() got unsupported type %T", r)
		}
	}()
	fn()
}
