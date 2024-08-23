package params

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/libevm/pseudo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testOnlyClearRegisteredExtras SHOULD be called before every call to
// [RegisterExtras] and then defer-called afterwards. This is a workaround for
// the single-call limitation on [RegisterExtras].
func testOnlyClearRegisteredExtras() {
	registeredExtras = nil
}

type rawJSON struct {
	json.RawMessage
}

var _ interface {
	json.Marshaler
	json.Unmarshaler
} = (*rawJSON)(nil)

func TestRegisterExtras(t *testing.T) {
	type (
		ccExtraA struct {
			A string `json:"a"`
		}
		rulesExtraA struct {
			A string
		}
		ccExtraB struct {
			B string `json:"b"`
		}
		rulesExtraB struct {
			B string
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
				RegisterExtras(Extras[rawJSON, struct{}]{})
			},
			ccExtra: pseudo.From(&rawJSON{
				RawMessage: []byte(`"hello, world"`),
			}).Type,
			wantRulesExtra: (*struct{})(nil),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testOnlyClearRegisteredExtras()
			tt.register()
			defer testOnlyClearRegisteredExtras()

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
			// TODO: do we need an explicit test of the JSON output, or is a
			// Marshal-Unmarshal round trip sufficient?

			gotRules := got.Rules(nil, false, 0)
			assert.Equal(t, tt.wantRulesExtra, gotRules.extraPayload().Interface())
		})
	}
}

func TestExtrasPanic(t *testing.T) {
	testOnlyClearRegisteredExtras()
	defer testOnlyClearRegisteredExtras()

	assertPanics(
		t, func() {
			RegisterExtras(Extras[int, struct{}]{})
		},
		notStructMessage[int](),
	)

	assertPanics(
		t, func() {
			RegisterExtras(Extras[struct{}, bool]{})
		},
		notStructMessage[bool](),
	)

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

	RegisterExtras(Extras[struct{}, struct{}]{})

	assertPanics(
		t, func() {
			RegisterExtras(Extras[struct{}, struct{}]{})
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
