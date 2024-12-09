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
	"encoding/json"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ava-labs/libevm/libevm/pseudo"
	"github.com/ava-labs/libevm/libevm/register"
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

			input := &ChainConfig{
				ChainID: big.NewInt(142857),
				extra:   tt.ccExtra,
			}

			buf, err := json.Marshal(input)
			require.NoError(t, err)

			got := new(ChainConfig)
			require.NoError(t, json.Unmarshal(buf, got))
			assert.Equal(t, tt.ccExtra.Interface(), got.extraPayload().Interface())
			assert.Equal(t, input, got)

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
	extras := RegisterExtras(Extras[ccExtra, rulesExtra]{})

	config := new(ChainConfig)
	rules := new(Rules)
	// These assertion helpers are defined before any modifications so that the
	// closure is demonstrably over the original zero values.
	assertChainConfigExtra := func(t *testing.T, want ccExtra, msg string) {
		t.Helper()
		assert.Equalf(t, want, extras.ChainConfig.Get(config), "%T: "+msg, &config)
	}
	assertRulesExtra := func(t *testing.T, want rulesExtra, msg string) {
		t.Helper()
		assert.Equalf(t, want, extras.Rules.Get(rules), "%T: "+msg, &rules)
	}

	assertChainConfigExtra(t, ccExtra{}, "zero value")
	assertRulesExtra(t, rulesExtra{}, "zero value")

	const answer = 42
	extras.ChainConfig.GetPointer(config).X = answer
	assertChainConfigExtra(t, ccExtra{X: answer}, "after setting via pointer field")

	const pi = 314159
	extras.Rules.GetPointer(rules).X = pi
	assertRulesExtra(t, rulesExtra{X: pi}, "after setting via pointer field")

	ccReplace := ccExtra{X: 142857}
	extras.ChainConfig.Set(config, ccReplace)
	assertChainConfigExtra(t, ccReplace, "after replacement of entire extra via `*pointer = x`")

	rulesReplace := rulesExtra{X: 18101986}
	extras.Rules.Set(rules, rulesReplace)
	assertRulesExtra(t, rulesReplace, "after replacement of entire extra via `*pointer = x`")

	if t.Failed() {
		// The test of shallow copying is now guaranteed to fail.
		return
	}
	t.Run("copy", func(t *testing.T) {
		const (
			// Arbitrary test values.
			seqUp   = 123456789
			seqDown = 987654321
		)

		ccCopy := *config
		t.Run("ChainConfig", func(t *testing.T) {
			assert.Equal(t, extras.ChainConfig.Get(&ccCopy), ccReplace, "extras copied")

			extras.ChainConfig.GetPointer(&ccCopy).X = seqUp
			assertChainConfigExtra(t, ccExtra{X: seqUp}, "original changed via copied.PointerFromChainConfig because copy only shallow")

			ccReplace = ccExtra{X: seqDown}
			extras.ChainConfig.Set(&ccCopy, ccReplace)
			assert.Equal(t, extras.ChainConfig.Get(&ccCopy), ccReplace, "SetOnChainConfig effect")
			assertChainConfigExtra(t, ccExtra{X: seqUp}, "original unchanged after copied.SetOnChainConfig")
		})

		rCopy := *rules
		t.Run("Rules", func(t *testing.T) {
			assert.Equal(t, extras.Rules.Get(&rCopy), rulesReplace, "extras copied")

			extras.Rules.GetPointer(&rCopy).X = seqUp
			assertRulesExtra(t, rulesExtra{X: seqUp}, "original changed via copied.PointerFromRuels because copy only shallow")

			rulesReplace = rulesExtra{X: seqDown}
			extras.Rules.Set(&rCopy, rulesReplace)
			assert.Equal(t, extras.Rules.Get(&rCopy), rulesReplace, "SetOnRules effect")
			assertRulesExtra(t, rulesExtra{X: seqUp}, "original unchanged after copied.SetOnRules")
		})
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
		register.ErrReRegistration.Error(),
	)
}

func assertPanics(t *testing.T, fn func(), wantContains string) {
	t.Helper()
	defer func() {
		t.Helper()
		switch r := recover().(type) {
		case nil:
			t.Error("function did not panic when panic expected")
		case string:
			assert.Contains(t, r, wantContains)
		case error:
			assert.Contains(t, r.Error(), wantContains)
		default:
			t.Fatalf("BAD TEST SETUP: recover() got unsupported type %T", r)
		}
	}()
	fn()
}
