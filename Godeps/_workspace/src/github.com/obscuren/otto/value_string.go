package otto

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"unicode/utf16"
)

var matchLeading0Exponent = regexp.MustCompile(`([eE][\+\-])0+([1-9])`) // 1e-07 => 1e-7

func floatToString(value float64, bitsize int) string {
	// TODO Fit to ECMA-262 9.8.1 specification
	if math.IsNaN(value) {
		return "NaN"
	} else if math.IsInf(value, 0) {
		if math.Signbit(value) {
			return "-Infinity"
		}
		return "Infinity"
	}
	exponent := math.Log10(math.Abs(value))
	if exponent >= 21 || exponent < -6 {
		return matchLeading0Exponent.ReplaceAllString(strconv.FormatFloat(value, 'g', -1, bitsize), "$1$2")
	}
	return strconv.FormatFloat(value, 'f', -1, bitsize)
}

func numberToStringRadix(value Value, radix int) string {
	float := toFloat(value)
	if math.IsNaN(float) {
		return "NaN"
	} else if math.IsInf(float, 1) {
		return "Infinity"
	} else if math.IsInf(float, -1) {
		return "-Infinity"
	}
	// FIXME This is very broken
	// Need to do proper radix conversion for floats, ...
	// This truncates large floats (so bad).
	return strconv.FormatInt(int64(float), radix)
}

func toString(value Value) string {
	if value._valueType == valueString {
		switch value := value.value.(type) {
		case string:
			return value
		case []uint16:
			return string(utf16.Decode(value))
		}
	}
	if value.IsUndefined() {
		return "undefined"
	}
	if value.IsNull() {
		return "null"
	}
	switch value := value.value.(type) {
	case bool:
		return strconv.FormatBool(value)
	case int:
		return strconv.FormatInt(int64(value), 10)
	case int8:
		return strconv.FormatInt(int64(value), 10)
	case int16:
		return strconv.FormatInt(int64(value), 10)
	case int32:
		return strconv.FormatInt(int64(value), 10)
	case int64:
		return strconv.FormatInt(value, 10)
	case uint:
		return strconv.FormatUint(uint64(value), 10)
	case uint8:
		return strconv.FormatUint(uint64(value), 10)
	case uint16:
		return strconv.FormatUint(uint64(value), 10)
	case uint32:
		return strconv.FormatUint(uint64(value), 10)
	case uint64:
		return strconv.FormatUint(value, 10)
	case float32:
		if value == 0 {
			return "0" // Take care not to return -0
		}
		return floatToString(float64(value), 32)
	case float64:
		if value == 0 {
			return "0" // Take care not to return -0
		}
		return floatToString(value, 64)
	case []uint16:
		return string(utf16.Decode(value))
	case string:
		return value
	case *_object:
		return toString(value.DefaultValue(defaultValueHintString))
	}
	panic(fmt.Errorf("toString(%v %T)", value.value, value.value))
}
