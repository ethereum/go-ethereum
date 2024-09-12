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
					NewRules: func(cc *ChainConfig, r *Rules, ex ccExtraA, _ *big.Int, _ bool, _ uint64) rulesExtraA {
						return rulesExtraA{
							A: ex.A,
						}
					},
				})
			},
			ccExtra: pseudo.From(ccExtraA{
				A: "hello",
			}).Type,
			wantRulesExtra: rulesExtraA{
				A: "hello",
			},
		},
		{
			name: "no NewForRules() function results in zero value",
			register: func() {
				RegisterExtras(Extras[ccExtraB, rulesExtraB]{})
			},
			ccExtra: pseudo.From(ccExtraB{
				B: "world",
			}).Type,
			wantRulesExtra: rulesExtraB{},
		},
		{
			name: "no NewForRules() function results in nil pointer",
			register: func() {
				RegisterExtras(Extras[ccExtraB, *rulesExtraB]{})
			},
			ccExtra: pseudo.From(ccExtraB{
				B: "world",
			}).Type,
			wantRulesExtra: (*rulesExtraB)(nil),
		},
		{
			name: "custom JSON handling honoured",
			register: func() {
				RegisterExtras(Extras[rawJSON, struct{ RulesHooks }]{})
			},
			ccExtra: pseudo.From(rawJSON{
				RawMessage: []byte(`"hello, world"`),
			}).Type,
			wantRulesExtra: struct{ RulesHooks }{},
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

func TestModificationOfZeroExtras(t *testing.T) {
	type (
		ccExtra struct {
			X int
			NOOPHooks
		}
		rulesExtra struct {
			X int
			NOOPHooks
		}
	)

	TestOnlyClearRegisteredExtras()
	t.Cleanup(TestOnlyClearRegisteredExtras)
	getter := RegisterExtras(Extras[ccExtra, rulesExtra]{})

	config := new(ChainConfig)
	rules := new(Rules)
	// These assertion helpers are defined before any modifications so that the
	// closure is demonstrably over the original zero values.
	assertChainConfigExtra := func(t *testing.T, want ccExtra, msg string) {
		t.Helper()
		assert.Equalf(t, want, getter.FromChainConfig(config), "%T: "+msg, &config)
	}
	assertRulesExtra := func(t *testing.T, want rulesExtra, msg string) {
		t.Helper()
		assert.Equalf(t, want, getter.FromRules(rules), "%T: "+msg, &rules)
	}

	assertChainConfigExtra(t, ccExtra{}, "zero value")
	assertRulesExtra(t, rulesExtra{}, "zero value")

	const answer = 42
	getter.PointerFromChainConfig(config).X = answer
	assertChainConfigExtra(t, ccExtra{X: answer}, "after setting via pointer field")

	const pi = 314159
	getter.PointerFromRules(rules).X = pi
	assertRulesExtra(t, rulesExtra{X: pi}, "after setting via pointer field")

	ccReplace := ccExtra{X: 142857}
	*getter.PointerFromChainConfig(config) = ccReplace
	assertChainConfigExtra(t, ccReplace, "after replacement of entire extra via `*pointer = x`")

	rulesReplace := rulesExtra{X: 18101986}
	*getter.PointerFromRules(rules) = rulesReplace
	assertRulesExtra(t, rulesReplace, "after replacement of entire extra via `*pointer = x`")

	if t.Failed() {
		// The test of shallow copying is now guaranteed to fail.
		return
	}
	t.Run("shallow copy", func(t *testing.T) {
		ccCopy := *config
		rCopy := *rules

		assert.Equal(t, getter.FromChainConfig(&ccCopy), ccReplace, "ChainConfig extras copied")
		assert.Equal(t, getter.FromRules(&rCopy), rulesReplace, "Rules extras copied")

		const seqUp = 123456789
		getter.PointerFromChainConfig(&ccCopy).X = seqUp
		assertChainConfigExtra(t, ccExtra{X: seqUp}, "original changed because copy only shallow")

		const seqDown = 987654321
		getter.PointerFromRules(&rCopy).X = seqDown
		assertRulesExtra(t, rulesExtra{X: seqDown}, "original changed because copy only shallow")
	})
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
			mustBeStructOrPointerToOne[int]()
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
