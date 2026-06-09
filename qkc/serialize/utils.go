// Ported verbatim from github.com/QuarkChain/goquarkchain/serialize (byte-compatible).

package serialize

import (
	"math/big"
	"reflect"
)

var (
	serializableInterface = reflect.TypeOf(new(Serializable)).Elem()
	bigInt                = reflect.TypeOf(big.Int{})
	typUint128            = reflect.TypeOf(Uint128{})
	typUint256            = reflect.TypeOf(Uint256{})
	big0                  = big.NewInt(0)
)

type BigUint struct {
	Value *big.Int
}

type Uint128 BigUint
type Uint256 BigUint

func (ui *Uint128) Serialize(w *[]byte) error {
	return serializeFixSizeBigUint(ui.Value, 16, w)
}

func (ui *Uint128) Deserialize(bb *ByteBuffer) error {
	if ui.Value == nil {
		ui.Value = new(big.Int)
	}
	return deserializeFixSizeBigUint(bb, ui.Value, 16)
}

func (ui *Uint256) Serialize(w *[]byte) error {
	return serializeFixSizeBigUint(ui.Value, 32, w)
}

func (ui *Uint256) Deserialize(bb *ByteBuffer) error {
	if ui.Value == nil {
		ui.Value = new(big.Int)
	}
	return deserializeFixSizeBigUint(bb, ui.Value, 32)
}

type Serializable interface {
	Serialize(w *[]byte) error
	Deserialize(bb *ByteBuffer) error
}

func isUint(k reflect.Kind) bool {
	return k >= reflect.Uint && k <= reflect.Uintptr
}

func isByte(typ reflect.Type) bool {
	return typ.Kind() == reflect.Uint8 && !typ.Implements(serializableInterface)
}
