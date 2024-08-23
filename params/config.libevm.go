package params

import (
	"encoding/json"
	"fmt"
	"math/big"
	"reflect"

	"github.com/ethereum/go-ethereum/libevm/pseudo"
)

// Extras are arbitrary payloads to be added as extra fields in [ChainConfig]
// and [Rules] structs. See [RegisterExtras].
type Extras[C any, R any] struct {
	// NewForRules, if non-nil is called at the end of [ChainConfig.Rules] with
	// the newly created [Rules] and the [ChainConfig] extra payload. Its
	// returned value will be the extra payload of the [Rules]. If NewForRules
	// is nil then so too will the [Rules] extra payload be a nil `*R`.
	//
	// NewForRules MAY modify the [Rules] but MUST NOT modify the [ChainConfig].
	NewForRules func(_ *ChainConfig, _ *Rules, _ *C, blockNum *big.Int, isMerge bool, timestamp uint64) *R
}

// RegisterExtras registers the types `C` and `R` such that they are carried as
// extra payloads in [ChainConfig] and [Rules] structs, respectively. It is
// expected to be called in an `init()` function and MUST NOT be called more
// than once. Both `C` and `R` MUST be structs.
//
// After registration, JSON unmarshalling of a [ChainConfig] will create a new
// `*C` and unmarshal the JSON key "extra" into it. Conversely, JSON marshalling
// will populate the "extra" key with the contents of the `*C`. Calls to
// [ChainConfig.Rules] will call the `NewForRules` function of the registered
// [Extras] to create a new `*R`.
//
// The payloads can be accessed via the [ChainConfig.extraPayload] and
// [Rules.extraPayload] methods, which will always return a `*C` or `*R`
// respectively however these pointers may themselves be nil.
//
// As the `ExtraPayload()` methods are not generic and return `any`, their
// values MUST be type-asserted to the returned type; failure to do so may
// result in a typed-nil bug. This pattern most-closely resembles a fully
// generic implementation and users SHOULD wrap the type assertions in a shared
// package.
func RegisterExtras[C any, R any](e Extras[C, R]) ExtraPayloadGetter[C, R] {
	if registeredExtras != nil {
		panic("re-registration of Extras")
	}
	mustBeStruct[C]()
	mustBeStruct[R]()
	registeredExtras = &e
	return ExtraPayloadGetter[C, R]{}
}

// An ExtraPayloadGettter ...
type ExtraPayloadGetter[C any, R any] struct{}

// FromChainConfig ...
func (ExtraPayloadGetter[C, R]) FromChainConfig(c *ChainConfig) *C {
	return pseudo.MustNewValue[*C](c.extraPayload()).Get()
}

// FromRules ...
func (ExtraPayloadGetter[C, R]) FromRules(r *Rules) *R {
	return pseudo.MustNewValue[*R](r.extraPayload()).Get()
}

func mustBeStruct[T any]() {
	var x T
	if k := reflect.TypeOf(x).Kind(); k != reflect.Struct {
		panic(notStructMessage[T]())
	}
}

func notStructMessage[T any]() string {
	var x T
	return fmt.Sprintf("%T is not a struct", x)
}

var registeredExtras interface {
	nilForChainConfig() *pseudo.Type
	nilForRules() *pseudo.Type
	newForChainConfig() *pseudo.Type
	newForRules(_ *ChainConfig, _ *Rules, blockNum *big.Int, isMerge bool, timestamp uint64) *pseudo.Type
}

var (
	_ json.Unmarshaler = (*ChainConfig)(nil)
	_ json.Marshaler   = (*ChainConfig)(nil)
)

// UnmarshalJSON ... TODO
func (c *ChainConfig) UnmarshalJSON(data []byte) error {
	// We need to bypass this UnmarshalJSON() method when we again call
	// json.Unmarshal(). The `raw` type won't inherit the method.
	type raw ChainConfig
	cc := &struct {
		*raw
		Extra json.RawMessage `json:"extra"`
	}{raw: (*raw)(c)}

	if err := json.Unmarshal(data, cc); err != nil {
		return err
	}
	if registeredExtras == nil || len(cc.Extra) == 0 {
		return nil
	}

	extra := registeredExtras.newForChainConfig()
	if err := json.Unmarshal(cc.Extra, extra); err != nil {
		return err
	}
	c.extra = extra
	return nil
}

// MarshalJSON ... TODO
func (c *ChainConfig) MarshalJSON() ([]byte, error) {
	type raw ChainConfig
	cc := &struct {
		*raw
		Extra any `json:"extra"`
	}{raw: (*raw)(c), Extra: c.extra}
	return json.Marshal(cc)
}

func (c *ChainConfig) addRulesExtra(r *Rules, blockNum *big.Int, isMerge bool, timestamp uint64) {
	r.extra = nil
	if registeredExtras != nil {
		r.extra = registeredExtras.newForRules(c, r, blockNum, isMerge, timestamp)
	}
}

// extraPayload returns the extra payload carried by the ChainConfig and can
// only be called if [RegisterExtras] was called. The returned value is always
// of type `*C` as registered, but may be nil. Callers MUST immediately
// type-assert the returned value to `*C` to avoid typed-nil bugs. See the
// example for the intended usage pattern.
func (c *ChainConfig) extraPayload() *pseudo.Type {
	if registeredExtras == nil {
		panic(fmt.Sprintf("%T.ExtraPayload() called before RegisterExtras()", c))
	}
	if c.extra == nil {
		c.extra = registeredExtras.nilForChainConfig()
	}
	return c.extra
}

// extraPayload returns the extra payload carried by the Rules and can only be
// called if [RegisterExtras] was called. The returned value is always of type
// `*R` as registered, but may be nil. Callers MUST immediately type-assert the
// returned value to `*R` to avoid typed-nil bugs. See the example on
// [ChainConfig.extraPayload] for the intended usage pattern.
func (r *Rules) extraPayload() *pseudo.Type {
	if registeredExtras == nil {
		panic(fmt.Sprintf("%T.ExtraPayload() called before RegisterExtras()", r))
	}
	if r.extra == nil {
		r.extra = registeredExtras.nilForRules()
	}
	return r.extra
}

func (Extras[C, R]) nilForChainConfig() *pseudo.Type { return pseudo.Zero[*C]().Type }
func (Extras[C, R]) nilForRules() *pseudo.Type       { return pseudo.Zero[*R]().Type }

func (*Extras[C, R]) newForChainConfig() *pseudo.Type {
	var x C
	return pseudo.From(&x).Type
}

func (e *Extras[C, R]) newForRules(c *ChainConfig, r *Rules, blockNum *big.Int, isMerge bool, timestamp uint64) *pseudo.Type {
	if e.NewForRules == nil {
		return e.nilForRules()
	}
	return pseudo.From(e.NewForRules(c, r, c.extra.Interface().(*C), blockNum, isMerge, timestamp)).Type
}
