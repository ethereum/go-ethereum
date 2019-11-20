package goja

import (
	"math"
	"reflect"
	"regexp"
	"strconv"
)

var (
	valueFalse    Value = valueBool(false)
	valueTrue     Value = valueBool(true)
	_null         Value = valueNull{}
	_NaN          Value = valueFloat(math.NaN())
	_positiveInf  Value = valueFloat(math.Inf(+1))
	_negativeInf  Value = valueFloat(math.Inf(-1))
	_positiveZero Value
	_negativeZero Value = valueFloat(math.Float64frombits(0 | (1 << 63)))
	_epsilon            = valueFloat(2.2204460492503130808472633361816e-16)
	_undefined    Value = valueUndefined{}
)

var (
	reflectTypeInt    = reflect.TypeOf(int64(0))
	reflectTypeBool   = reflect.TypeOf(false)
	reflectTypeNil    = reflect.TypeOf(nil)
	reflectTypeFloat  = reflect.TypeOf(float64(0))
	reflectTypeMap    = reflect.TypeOf(map[string]interface{}{})
	reflectTypeArray  = reflect.TypeOf([]interface{}{})
	reflectTypeString = reflect.TypeOf("")
)

var intCache [256]Value

type Value interface {
	ToInteger() int64
	ToString() valueString
	String() string
	ToFloat() float64
	ToNumber() Value
	ToBoolean() bool
	ToObject(*Runtime) *Object
	SameAs(Value) bool
	Equals(Value) bool
	StrictEquals(Value) bool
	Export() interface{}
	ExportType() reflect.Type

	assertInt() (int64, bool)
	assertString() (valueString, bool)
	assertFloat() (float64, bool)

	baseObject(r *Runtime) *Object
}

type valueInt int64
type valueFloat float64
type valueBool bool
type valueNull struct{}
type valueUndefined struct {
	valueNull
}

type valueUnresolved struct {
	r   *Runtime
	ref string
}

type memberUnresolved struct {
	valueUnresolved
}

type valueProperty struct {
	value        Value
	writable     bool
	configurable bool
	enumerable   bool
	accessor     bool
	getterFunc   *Object
	setterFunc   *Object
}

func propGetter(o Value, v Value, r *Runtime) *Object {
	if v == _undefined {
		return nil
	}
	if obj, ok := v.(*Object); ok {
		if _, ok := obj.self.assertCallable(); ok {
			return obj
		}
	}
	r.typeErrorResult(true, "Getter must be a function: %s", v.ToString())
	return nil
}

func propSetter(o Value, v Value, r *Runtime) *Object {
	if v == _undefined {
		return nil
	}
	if obj, ok := v.(*Object); ok {
		if _, ok := obj.self.assertCallable(); ok {
			return obj
		}
	}
	r.typeErrorResult(true, "Setter must be a function: %s", v.ToString())
	return nil
}

func (i valueInt) ToInteger() int64 {
	return int64(i)
}

func (i valueInt) ToString() valueString {
	return asciiString(i.String())
}

func (i valueInt) String() string {
	return strconv.FormatInt(int64(i), 10)
}

func (i valueInt) ToFloat() float64 {
	return float64(int64(i))
}

func (i valueInt) ToBoolean() bool {
	return i != 0
}

func (i valueInt) ToObject(r *Runtime) *Object {
	return r.newPrimitiveObject(i, r.global.NumberPrototype, classNumber)
}

func (i valueInt) ToNumber() Value {
	return i
}

func (i valueInt) SameAs(other Value) bool {
	if otherInt, ok := other.assertInt(); ok {
		return int64(i) == otherInt
	}
	return false
}

func (i valueInt) Equals(other Value) bool {
	if o, ok := other.assertInt(); ok {
		return int64(i) == o
	}
	if o, ok := other.assertFloat(); ok {
		return float64(i) == o
	}
	if o, ok := other.assertString(); ok {
		return o.ToNumber().Equals(i)
	}
	if o, ok := other.(valueBool); ok {
		return int64(i) == o.ToInteger()
	}
	if o, ok := other.(*Object); ok {
		return i.Equals(o.self.toPrimitiveNumber())
	}
	return false
}

func (i valueInt) StrictEquals(other Value) bool {
	if otherInt, ok := other.assertInt(); ok {
		return int64(i) == otherInt
	} else if otherFloat, ok := other.assertFloat(); ok {
		return float64(i) == otherFloat
	}
	return false
}

func (i valueInt) assertInt() (int64, bool) {
	return int64(i), true
}

func (i valueInt) assertFloat() (float64, bool) {
	return 0, false
}

func (i valueInt) assertString() (valueString, bool) {
	return nil, false
}

func (i valueInt) baseObject(r *Runtime) *Object {
	return r.global.NumberPrototype
}

func (i valueInt) Export() interface{} {
	return int64(i)
}

func (i valueInt) ExportType() reflect.Type {
	return reflectTypeInt
}

func (o valueBool) ToInteger() int64 {
	if o {
		return 1
	}
	return 0
}

func (o valueBool) ToString() valueString {
	if o {
		return stringTrue
	}
	return stringFalse
}

func (o valueBool) String() string {
	if o {
		return "true"
	}
	return "false"
}

func (o valueBool) ToFloat() float64 {
	if o {
		return 1.0
	}
	return 0
}

func (o valueBool) ToBoolean() bool {
	return bool(o)
}

func (o valueBool) ToObject(r *Runtime) *Object {
	return r.newPrimitiveObject(o, r.global.BooleanPrototype, "Boolean")
}

func (o valueBool) ToNumber() Value {
	if o {
		return valueInt(1)
	}
	return valueInt(0)
}

func (o valueBool) SameAs(other Value) bool {
	if other, ok := other.(valueBool); ok {
		return o == other
	}
	return false
}

func (b valueBool) Equals(other Value) bool {
	if o, ok := other.(valueBool); ok {
		return b == o
	}

	if b {
		return other.Equals(intToValue(1))
	} else {
		return other.Equals(intToValue(0))
	}

}

func (o valueBool) StrictEquals(other Value) bool {
	if other, ok := other.(valueBool); ok {
		return o == other
	}
	return false
}

func (o valueBool) assertInt() (int64, bool) {
	return 0, false
}

func (o valueBool) assertFloat() (float64, bool) {
	return 0, false
}

func (o valueBool) assertString() (valueString, bool) {
	return nil, false
}

func (o valueBool) baseObject(r *Runtime) *Object {
	return r.global.BooleanPrototype
}

func (o valueBool) Export() interface{} {
	return bool(o)
}

func (o valueBool) ExportType() reflect.Type {
	return reflectTypeBool
}

func (n valueNull) ToInteger() int64 {
	return 0
}

func (n valueNull) ToString() valueString {
	return stringNull
}

func (n valueNull) String() string {
	return "null"
}

func (u valueUndefined) ToString() valueString {
	return stringUndefined
}

func (u valueUndefined) String() string {
	return "undefined"
}

func (u valueUndefined) ToNumber() Value {
	return _NaN
}

func (u valueUndefined) SameAs(other Value) bool {
	_, same := other.(valueUndefined)
	return same
}

func (u valueUndefined) StrictEquals(other Value) bool {
	_, same := other.(valueUndefined)
	return same
}

func (u valueUndefined) ToFloat() float64 {
	return math.NaN()
}

func (n valueNull) ToFloat() float64 {
	return 0
}

func (n valueNull) ToBoolean() bool {
	return false
}

func (n valueNull) ToObject(r *Runtime) *Object {
	r.typeErrorResult(true, "Cannot convert undefined or null to object")
	return nil
	//return r.newObject()
}

func (n valueNull) ToNumber() Value {
	return intToValue(0)
}

func (n valueNull) SameAs(other Value) bool {
	_, same := other.(valueNull)
	return same
}

func (n valueNull) Equals(other Value) bool {
	switch other.(type) {
	case valueUndefined, valueNull:
		return true
	}
	return false
}

func (n valueNull) StrictEquals(other Value) bool {
	_, same := other.(valueNull)
	return same
}

func (n valueNull) assertInt() (int64, bool) {
	return 0, false
}

func (n valueNull) assertFloat() (float64, bool) {
	return 0, false
}

func (n valueNull) assertString() (valueString, bool) {
	return nil, false
}

func (n valueNull) baseObject(r *Runtime) *Object {
	return nil
}

func (n valueNull) Export() interface{} {
	return nil
}

func (n valueNull) ExportType() reflect.Type {
	return reflectTypeNil
}

func (p *valueProperty) ToInteger() int64 {
	return 0
}

func (p *valueProperty) ToString() valueString {
	return stringEmpty
}

func (p *valueProperty) String() string {
	return ""
}

func (p *valueProperty) ToFloat() float64 {
	return math.NaN()
}

func (p *valueProperty) ToBoolean() bool {
	return false
}

func (p *valueProperty) ToObject(r *Runtime) *Object {
	return nil
}

func (p *valueProperty) ToNumber() Value {
	return nil
}

func (p *valueProperty) assertInt() (int64, bool) {
	return 0, false
}

func (p *valueProperty) assertFloat() (float64, bool) {
	return 0, false
}

func (p *valueProperty) assertString() (valueString, bool) {
	return nil, false
}

func (p *valueProperty) isWritable() bool {
	return p.writable || p.setterFunc != nil
}

func (p *valueProperty) get(this Value) Value {
	if p.getterFunc == nil {
		if p.value != nil {
			return p.value
		}
		return _undefined
	}
	call, _ := p.getterFunc.self.assertCallable()
	return call(FunctionCall{
		This: this,
	})
}

func (p *valueProperty) set(this, v Value) {
	if p.setterFunc == nil {
		p.value = v
		return
	}
	call, _ := p.setterFunc.self.assertCallable()
	call(FunctionCall{
		This:      this,
		Arguments: []Value{v},
	})
}

func (p *valueProperty) SameAs(other Value) bool {
	if otherProp, ok := other.(*valueProperty); ok {
		return p == otherProp
	}
	return false
}

func (p *valueProperty) Equals(other Value) bool {
	return false
}

func (p *valueProperty) StrictEquals(other Value) bool {
	return false
}

func (n *valueProperty) baseObject(r *Runtime) *Object {
	r.typeErrorResult(true, "BUG: baseObject() is called on valueProperty") // TODO error message
	return nil
}

func (n *valueProperty) Export() interface{} {
	panic("Cannot export valueProperty")
}

func (n *valueProperty) ExportType() reflect.Type {
	panic("Cannot export valueProperty")
}

func (f valueFloat) ToInteger() int64 {
	switch {
	case math.IsNaN(float64(f)):
		return 0
	case math.IsInf(float64(f), 1):
		return int64(math.MaxInt64)
	case math.IsInf(float64(f), -1):
		return int64(math.MinInt64)
	}
	return int64(f)
}

func (f valueFloat) ToString() valueString {
	return asciiString(f.String())
}

var matchLeading0Exponent = regexp.MustCompile(`([eE][\+\-])0+([1-9])`) // 1e-07 => 1e-7

func (f valueFloat) String() string {
	value := float64(f)
	if math.IsNaN(value) {
		return "NaN"
	} else if math.IsInf(value, 0) {
		if math.Signbit(value) {
			return "-Infinity"
		}
		return "Infinity"
	} else if f == _negativeZero {
		return "0"
	}
	exponent := math.Log10(math.Abs(value))
	if exponent >= 21 || exponent < -6 {
		return matchLeading0Exponent.ReplaceAllString(strconv.FormatFloat(value, 'g', -1, 64), "$1$2")
	}
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func (f valueFloat) ToFloat() float64 {
	return float64(f)
}

func (f valueFloat) ToBoolean() bool {
	return float64(f) != 0.0 && !math.IsNaN(float64(f))
}

func (f valueFloat) ToObject(r *Runtime) *Object {
	return r.newPrimitiveObject(f, r.global.NumberPrototype, "Number")
}

func (f valueFloat) ToNumber() Value {
	return f
}

func (f valueFloat) SameAs(other Value) bool {
	if o, ok := other.assertFloat(); ok {
		this := float64(f)
		if math.IsNaN(this) && math.IsNaN(o) {
			return true
		} else {
			ret := this == o
			if ret && this == 0 {
				ret = math.Signbit(this) == math.Signbit(o)
			}
			return ret
		}
	} else if o, ok := other.assertInt(); ok {
		this := float64(f)
		ret := this == float64(o)
		if ret && this == 0 {
			ret = !math.Signbit(this)
		}
		return ret
	}
	return false
}

func (f valueFloat) Equals(other Value) bool {
	if o, ok := other.assertFloat(); ok {
		return float64(f) == o
	}

	if o, ok := other.assertInt(); ok {
		return float64(f) == float64(o)
	}

	if _, ok := other.assertString(); ok {
		return float64(f) == other.ToFloat()
	}

	if o, ok := other.(valueBool); ok {
		return float64(f) == o.ToFloat()
	}

	if o, ok := other.(*Object); ok {
		return f.Equals(o.self.toPrimitiveNumber())
	}

	return false
}

func (f valueFloat) StrictEquals(other Value) bool {
	if o, ok := other.assertFloat(); ok {
		return float64(f) == o
	} else if o, ok := other.assertInt(); ok {
		return float64(f) == float64(o)
	}
	return false
}

func (f valueFloat) assertInt() (int64, bool) {
	return 0, false
}

func (f valueFloat) assertFloat() (float64, bool) {
	return float64(f), true
}

func (f valueFloat) assertString() (valueString, bool) {
	return nil, false
}

func (f valueFloat) baseObject(r *Runtime) *Object {
	return r.global.NumberPrototype
}

func (f valueFloat) Export() interface{} {
	return float64(f)
}

func (f valueFloat) ExportType() reflect.Type {
	return reflectTypeFloat
}

func (o *Object) ToInteger() int64 {
	return o.self.toPrimitiveNumber().ToNumber().ToInteger()
}

func (o *Object) ToString() valueString {
	return o.self.toPrimitiveString().ToString()
}

func (o *Object) String() string {
	return o.self.toPrimitiveString().String()
}

func (o *Object) ToFloat() float64 {
	return o.self.toPrimitiveNumber().ToFloat()
}

func (o *Object) ToBoolean() bool {
	return true
}

func (o *Object) ToObject(r *Runtime) *Object {
	return o
}

func (o *Object) ToNumber() Value {
	return o.self.toPrimitiveNumber().ToNumber()
}

func (o *Object) SameAs(other Value) bool {
	if other, ok := other.(*Object); ok {
		return o == other
	}
	return false
}

func (o *Object) Equals(other Value) bool {
	if other, ok := other.(*Object); ok {
		return o == other || o.self.equal(other.self)
	}

	if _, ok := other.assertInt(); ok {
		return o.self.toPrimitive().Equals(other)
	}

	if _, ok := other.assertFloat(); ok {
		return o.self.toPrimitive().Equals(other)
	}

	if other, ok := other.(valueBool); ok {
		return o.Equals(other.ToNumber())
	}

	if _, ok := other.assertString(); ok {
		return o.self.toPrimitive().Equals(other)
	}
	return false
}

func (o *Object) StrictEquals(other Value) bool {
	if other, ok := other.(*Object); ok {
		return o == other || o.self.equal(other.self)
	}
	return false
}

func (o *Object) assertInt() (int64, bool) {
	return 0, false
}

func (o *Object) assertFloat() (float64, bool) {
	return 0, false
}

func (o *Object) assertString() (valueString, bool) {
	return nil, false
}

func (o *Object) baseObject(r *Runtime) *Object {
	return o
}

func (o *Object) Export() interface{} {
	return o.self.export()
}

func (o *Object) ExportType() reflect.Type {
	return o.self.exportType()
}

func (o *Object) Get(name string) Value {
	return o.self.getStr(name)
}

func (o *Object) Keys() (keys []string) {
	for item, f := o.self.enumerate(false, false)(); f != nil; item, f = f() {
		keys = append(keys, item.name)
	}

	return
}

// DefineDataProperty is a Go equivalent of Object.defineProperty(o, name, {value: value, writable: writable,
// configurable: configurable, enumerable: enumerable})
func (o *Object) DefineDataProperty(name string, value Value, writable, configurable, enumerable Flag) error {
	return tryFunc(func() {
		o.self.defineOwnProperty(newStringValue(name), propertyDescr{
			Value:        value,
			Writable:     writable,
			Configurable: configurable,
			Enumerable:   enumerable,
		}, true)
	})
}

// DefineAccessorProperty is a Go equivalent of Object.defineProperty(o, name, {get: getter, set: setter,
// configurable: configurable, enumerable: enumerable})
func (o *Object) DefineAccessorProperty(name string, getter, setter Value, configurable, enumerable Flag) error {
	return tryFunc(func() {
		o.self.defineOwnProperty(newStringValue(name), propertyDescr{
			Getter:       getter,
			Setter:       setter,
			Configurable: configurable,
			Enumerable:   enumerable,
		}, true)
	})
}

func (o *Object) Set(name string, value interface{}) error {
	return tryFunc(func() {
		o.self.putStr(name, o.runtime.ToValue(value), true)
	})
}

// MarshalJSON returns JSON representation of the Object. It is equivalent to JSON.stringify(o).
// Note, this implements json.Marshaler so that json.Marshal() can be used without the need to Export().
func (o *Object) MarshalJSON() ([]byte, error) {
	ctx := _builtinJSON_stringifyContext{
		r: o.runtime,
	}
	ex := o.runtime.vm.try(func() {
		if !ctx.do(o) {
			ctx.buf.WriteString("null")
		}
	})
	if ex != nil {
		return nil, ex
	}
	return ctx.buf.Bytes(), nil
}

// ClassName returns the class name
func (o *Object) ClassName() string {
	return o.self.className()
}

func (o valueUnresolved) throw() {
	o.r.throwReferenceError(o.ref)
}

func (o valueUnresolved) ToInteger() int64 {
	o.throw()
	return 0
}

func (o valueUnresolved) ToString() valueString {
	o.throw()
	return nil
}

func (o valueUnresolved) String() string {
	o.throw()
	return ""
}

func (o valueUnresolved) ToFloat() float64 {
	o.throw()
	return 0
}

func (o valueUnresolved) ToBoolean() bool {
	o.throw()
	return false
}

func (o valueUnresolved) ToObject(r *Runtime) *Object {
	o.throw()
	return nil
}

func (o valueUnresolved) ToNumber() Value {
	o.throw()
	return nil
}

func (o valueUnresolved) SameAs(other Value) bool {
	o.throw()
	return false
}

func (o valueUnresolved) Equals(other Value) bool {
	o.throw()
	return false
}

func (o valueUnresolved) StrictEquals(other Value) bool {
	o.throw()
	return false
}

func (o valueUnresolved) assertInt() (int64, bool) {
	o.throw()
	return 0, false
}

func (o valueUnresolved) assertFloat() (float64, bool) {
	o.throw()
	return 0, false
}

func (o valueUnresolved) assertString() (valueString, bool) {
	o.throw()
	return nil, false
}

func (o valueUnresolved) baseObject(r *Runtime) *Object {
	o.throw()
	return nil
}

func (o valueUnresolved) Export() interface{} {
	o.throw()
	return nil
}

func (o valueUnresolved) ExportType() reflect.Type {
	o.throw()
	return nil
}

func init() {
	for i := 0; i < 256; i++ {
		intCache[i] = valueInt(i - 128)
	}
	_positiveZero = intToValue(0)
}
