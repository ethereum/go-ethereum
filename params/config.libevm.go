package params

import (
	"fmt"
	"math/big"
	"reflect"
	"runtime"
	"strings"

	"github.com/ethereum/go-ethereum/libevm/pseudo"
)

// Extras are arbitrary payloads to be added as extra fields in [ChainConfig]
// and [Rules] structs. See [RegisterExtras].
type Extras[C ChainConfigHooks, R RulesHooks] struct {
	// ReuseJSONRoot, if true, signals that JSON unmarshalling of a
	// [ChainConfig] MUST reuse the root JSON input when unmarshalling the extra
	// payload. If false, it is assumed that the extra JSON payload is nested in
	// the "extra" key.
	//
	// *NOTE* this requires multiple passes for both marshalling and
	// unmarshalling of JSON so is inefficient and should be used as a last
	// resort.
	ReuseJSONRoot bool
	// NewRules, if non-nil is called at the end of [ChainConfig.Rules] with the
	// newly created [Rules] and other context from the method call. Its
	// returned value will be the extra payload of the [Rules]. If NewRules is
	// nil then so too will the [Rules] extra payload be a zero-value `R`.
	//
	// NewRules MAY modify the [Rules] but MUST NOT modify the [ChainConfig].
	// TODO(arr4n): add the [Rules] to the return signature to make it clearer
	// that the caller can modify the generated Rules.
	NewRules func(_ *ChainConfig, _ *Rules, _ C, blockNum *big.Int, isMerge bool, timestamp uint64) R
}

// RegisterExtras registers the types `C` and `R` such that they are carried as
// extra payloads in [ChainConfig] and [Rules] structs, respectively. It is
// expected to be called in an `init()` function and MUST NOT be called more
// than once. Both `C` and `R` MUST be structs or pointers to structs.
//
// After registration, JSON unmarshalling of a [ChainConfig] will create a new
// `C` and unmarshal the JSON key "extra" into it. Conversely, JSON marshalling
// will populate the "extra" key with the contents of the `C`. Both the
// [json.Marshaler] and [json.Unmarshaler] interfaces are honoured if
// implemented by `C` and/or `R.`
//
// Calls to [ChainConfig.Rules] will call the `NewRules` function of the
// registered [Extras] to create a new `R`.
//
// The payloads can be accessed via the [ExtraPayloadGetter.FromChainConfig] and
// [ExtraPayloadGetter.FromRules] methods of the getter returned by
// RegisterExtras. Where stated in the interface definitions, they will also be
// used as hooks to alter Ethereum behaviour; if this isn't desired then they
// can embed [NOOPHooks] to satisfy either interface.
func RegisterExtras[C ChainConfigHooks, R RulesHooks](e Extras[C, R]) ExtraPayloadGetter[C, R] {
	if registeredExtras != nil {
		panic("re-registration of Extras")
	}
	mustBeStructOrPointerToOne[C]()
	mustBeStructOrPointerToOne[R]()

	getter := e.getter()
	registeredExtras = &extraConstructors{
		newChainConfig: pseudo.NewConstructor[C]().Zero,
		newRules:       pseudo.NewConstructor[R]().Zero,
		reuseJSONRoot:  e.ReuseJSONRoot,
		newForRules:    e.newForRules,
		getter:         getter,
	}
	return getter
}

// TestOnlyClearRegisteredExtras clears the [Extras] previously passed to
// [RegisterExtras]. It panics if called from a non-testing call stack.
//
// In tests it SHOULD be called before every call to [RegisterExtras] and then
// defer-called afterwards, either directly or via testing.TB.Cleanup(). This is
// a workaround for the single-call limitation on [RegisterExtras].
func TestOnlyClearRegisteredExtras() {
	pc := make([]uintptr, 10)
	runtime.Callers(0, pc)
	frames := runtime.CallersFrames(pc)
	for {
		f, more := frames.Next()
		if strings.Contains(f.File, "/testing/") || strings.HasSuffix(f.File, "_test.go") {
			registeredExtras = nil
			return
		}
		if !more {
			panic("no _test.go file in call stack")
		}
	}
}

// registeredExtras holds non-generic constructors for the [Extras] types
// registered via [RegisterExtras].
var registeredExtras *extraConstructors

type extraConstructors struct {
	newChainConfig, newRules func() *pseudo.Type
	reuseJSONRoot            bool
	newForRules              func(_ *ChainConfig, _ *Rules, blockNum *big.Int, isMerge bool, timestamp uint64) *pseudo.Type
	// use top-level hooksFrom<X>() functions instead of these as they handle
	// instances where no [Extras] were registered.
	getter interface {
		hooksFromChainConfig(*ChainConfig) ChainConfigHooks
		hooksFromRules(*Rules) RulesHooks
	}
}

func (e *Extras[C, R]) newForRules(c *ChainConfig, r *Rules, blockNum *big.Int, isMerge bool, timestamp uint64) *pseudo.Type {
	if e.NewRules == nil {
		return registeredExtras.newRules()
	}
	rExtra := e.NewRules(c, r, e.getter().FromChainConfig(c), blockNum, isMerge, timestamp)
	return pseudo.From(rExtra).Type
}

func (*Extras[C, R]) getter() (g ExtraPayloadGetter[C, R]) { return }

// mustBeStructOrPointerToOne panics if `T` isn't a struct or a *struct.
func mustBeStructOrPointerToOne[T any]() {
	var x T
	switch t := reflect.TypeOf(x); t.Kind() {
	case reflect.Struct:
		return
	case reflect.Pointer:
		if t.Elem().Kind() == reflect.Struct {
			return
		}
	}
	panic(notStructMessage[T]())
}

// notStructMessage returns the message with which [mustBeStructOrPointerToOne]
// might panic. It exists to avoid change-detector tests should the message
// contents change.
func notStructMessage[T any]() string {
	var x T
	return fmt.Sprintf("%T is not a struct nor a pointer to a struct", x)
}

// An ExtraPayloadGettter provides strongly typed access to the extra payloads
// carried by [ChainConfig] and [Rules] structs. The only valid way to construct
// a getter is by a call to [RegisterExtras].
type ExtraPayloadGetter[C ChainConfigHooks, R RulesHooks] struct {
	_ struct{} // make godoc show unexported fields so nobody tries to make their own getter ;)
}

// FromChainConfig returns the ChainConfig's extra payload.
func (ExtraPayloadGetter[C, R]) FromChainConfig(c *ChainConfig) C {
	return pseudo.MustNewValue[C](c.extraPayload()).Get()
}

// PointerFromChainConfig returns a pointer to the ChainConfig's extra payload.
// This is guaranteed to be non-nil.
func (ExtraPayloadGetter[C, R]) PointerFromChainConfig(c *ChainConfig) *C {
	return pseudo.MustPointerTo[C](c.extraPayload()).Value.Get()
}

// hooksFromChainConfig is equivalent to FromChainConfig(), but returns an
// interface instead of the concrete type implementing it; this allows it to be
// used in non-generic code.
func (e ExtraPayloadGetter[C, R]) hooksFromChainConfig(c *ChainConfig) ChainConfigHooks {
	return e.FromChainConfig(c)
}

// FromRules returns the Rules' extra payload.
func (ExtraPayloadGetter[C, R]) FromRules(r *Rules) R {
	return pseudo.MustNewValue[R](r.extraPayload()).Get()
}

// PointerFromRules returns a pointer to the Rules's extra payload. This is
// guaranteed to be non-nil.
func (ExtraPayloadGetter[C, R]) PointerFromRules(r *Rules) *R {
	return pseudo.MustPointerTo[R](r.extraPayload()).Value.Get()
}

// hooksFromRules is the [RulesHooks] equivalent of hooksFromChainConfig().
func (e ExtraPayloadGetter[C, R]) hooksFromRules(r *Rules) RulesHooks {
	return e.FromRules(r)
}

// addRulesExtra is called at the end of [ChainConfig.Rules]; it exists to
// abstract the libevm-specific behaviour outside of original geth code.
func (c *ChainConfig) addRulesExtra(r *Rules, blockNum *big.Int, isMerge bool, timestamp uint64) {
	r.extra = nil
	if registeredExtras != nil {
		r.extra = registeredExtras.newForRules(c, r, blockNum, isMerge, timestamp)
	}
}

// extraPayload returns the ChainConfig's extra payload iff [RegisterExtras] has
// already been called. If the payload hasn't been populated (typically via
// unmarshalling of JSON), a nil value is constructed and returned.
func (c *ChainConfig) extraPayload() *pseudo.Type {
	if registeredExtras == nil {
		// This will only happen if someone constructs an [ExtraPayloadGetter]
		// directly, without a call to [RegisterExtras].
		//
		// See https://google.github.io/styleguide/go/best-practices#when-to-panic
		panic(fmt.Sprintf("%T.ExtraPayload() called before RegisterExtras()", c))
	}
	if c.extra == nil {
		c.extra = registeredExtras.newChainConfig()
	}
	return c.extra
}

// extraPayload is equivalent to [ChainConfig.extraPayload].
func (r *Rules) extraPayload() *pseudo.Type {
	if registeredExtras == nil {
		// See ChainConfig.extraPayload() equivalent.
		panic(fmt.Sprintf("%T.ExtraPayload() called before RegisterExtras()", r))
	}
	if r.extra == nil {
		r.extra = registeredExtras.newRules()
	}
	return r.extra
}
