package params

import (
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/libevm/pseudo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testOnlyClearRegisteredExtras() {
	registeredExtras = nil
}

func ExampleRegisterExtras() {
	type (
		chainConfigExtra struct {
			Foo string `json:"foo"`
		}
		rulesExtra struct {
			FooCopy string
		}
	)

	// In practice, this would be called inside an init() func.
	RegisterExtras(Extras[chainConfigExtra, rulesExtra]{
		NewForRules: func(cc *ChainConfig, r *Rules, cEx *chainConfigExtra, blockNum *big.Int, isMerge bool, timestamp uint64) *rulesExtra {
			// This function is called at the end of ChainConfig.Rules(),
			// receiving a pointer to the Rules that will be returned. It MAY
			// modify the Rules but MUST NOT modify the ChainConfig. The value
			// that it returns will be available via Rules.ExtraPayload().
			return &rulesExtra{
				FooCopy: fmt.Sprintf("copy of: %q", cEx.Foo),
			}
		},
	})
	defer testOnlyClearRegisteredExtras()

	// ChainConfig now unmarshals any JSON field named "extra" into a pointer to
	// the registered type, which is available via the ExtraPayload() method.
	buf := []byte(`{
		"chainId": 1234,
		"extra": {
			"foo": "hello, world"
		}
	}`)

	var config ChainConfig
	if err := json.Unmarshal(buf, &config); err != nil {
		log.Fatal(err)
	}

	fmt.Println(config.ChainID)
	// The values returned by ExtraPayload() are guaranteed to be pointers to
	// the registered types. They MAY, however, be nil pointers. In practice,
	// callers SHOULD abstract the type assertion in a reusable function to
	// provide a seamless devex.
	ccExtra := config.ExtraPayload().Interface().(*chainConfigExtra)
	rules := config.Rules(nil, false, 0)
	rExtra := rules.ExtraPayload().Interface().(*rulesExtra)

	if ccExtra != nil {
		fmt.Println(ccExtra.Foo)
	}
	if rExtra != nil {
		fmt.Println(rExtra.FooCopy)
	}

	// Output:
	// 1234
	// hello, world
	// copy of: "hello, world"
}

func ExampleChainConfig_ExtraPayload() {
	type (
		chainConfigExtra struct{}
		rulesExtra       struct{}
	)
	// Typically called in an `init()` function.
	RegisterExtras(Extras[chainConfigExtra, rulesExtra]{ /*...*/ })
	defer testOnlyClearRegisteredExtras()

	var c ChainConfig // Sourced from elsewhere, typically unmarshalled from JSON.

	// Both ChainConfig.ExtraPayload() and Rules.ExtraPayload() return `any`
	// that are guaranteed to be pointers to the registered types.
	extra := c.ExtraPayload().Interface().(*chainConfigExtra)

	// Act on the extra payload...
	if extra != nil {
		// ...
	}
}

type rawJSON struct {
	json.RawMessage
}

var (
	_ json.Unmarshaler = (*rawJSON)(nil)
	_ json.Marshaler   = (*rawJSON)(nil)
)

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
					NewForRules: func(cc *ChainConfig, r *Rules, ex *ccExtraA, _ *big.Int, _ bool, _ uint64) *rulesExtraA {
						return &rulesExtraA{
							A: ex.A,
						}
					},
				})
			},
			ccExtra: pseudo.OnlyType(pseudo.From(&ccExtraA{
				A: "hello",
			})),
			wantRulesExtra: &rulesExtraA{
				A: "hello",
			},
		},
		{
			name: "no NewForRules() function results in typed but nil pointer",
			register: func() {
				RegisterExtras(Extras[ccExtraB, rulesExtraB]{})
			},
			ccExtra: pseudo.OnlyType(pseudo.From(&ccExtraB{
				B: "world",
			})),
			wantRulesExtra: (*rulesExtraB)(nil),
		},
		{
			name: "custom JSON handling honoured",
			register: func() {
				RegisterExtras(Extras[rawJSON, struct{}]{})
			},
			ccExtra: pseudo.OnlyType(pseudo.From(&rawJSON{
				RawMessage: []byte(`"hello, world"`),
			})),
			wantRulesExtra: (*struct{})(nil),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
			assert.Equal(t, tt.ccExtra.Interface(), got.ExtraPayload().Interface())
			assert.Equal(t, in, got)
			// TODO: do we need an explicit test of the JSON output, or is a
			// Marshal-Unmarshal round trip sufficient?

			gotRules := got.Rules(nil, false, 0)
			assert.Equal(t, tt.wantRulesExtra, gotRules.ExtraPayload().Interface())
		})
	}
}

func TestExtrasPanic(t *testing.T) {
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
			new(ChainConfig).ExtraPayload()
		},
		"before RegisterExtras",
	)

	assertPanics(
		t, func() {
			new(Rules).ExtraPayload()
		},
		"before RegisterExtras",
	)

	RegisterExtras(Extras[struct{}, struct{}]{})
	defer testOnlyClearRegisteredExtras()

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
