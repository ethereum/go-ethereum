package otto

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

var stringToNumberParseInteger = regexp.MustCompile(`^(?:0[xX])`)

func stringToFloat(value string) float64 {
	value = strings.TrimSpace(value)

	if value == "" {
		return 0
	}

	parseFloat := false
	if strings.IndexRune(value, '.') != -1 {
		parseFloat = true
	} else if stringToNumberParseInteger.MatchString(value) {
		parseFloat = false
	} else {
		parseFloat = true
	}

	if parseFloat {
		number, err := strconv.ParseFloat(value, 64)
		if err != nil && err.(*strconv.NumError).Err != strconv.ErrRange {
			return math.NaN()
		}
		return number
	}

	number, err := strconv.ParseInt(value, 0, 64)
	if err != nil {
		return math.NaN()
	}
	return float64(number)
}

func toNumber(value Value) Value {
	if value._valueType == valueNumber {
		return value
	}
	return Value{valueNumber, toFloat(value)}
}

func toFloat(value Value) float64 {
	switch value._valueType {
	case valueUndefined:
		return math.NaN()
	case valueNull:
		return 0
	}
	switch value := value.value.(type) {
	case bool:
		if value {
			return 1
		}
		return 0
	case int:
		return float64(value)
	case int8:
		return float64(value)
	case int16:
		return float64(value)
	case int32:
		return float64(value)
	case int64:
		return float64(value)
	case uint:
		return float64(value)
	case uint8:
		return float64(value)
	case uint16:
		return float64(value)
	case uint32:
		return float64(value)
	case uint64:
		return float64(value)
	case float64:
		return value
	case string:
		return stringToFloat(value)
	case *_object:
		return toFloat(value.DefaultValue(defaultValueHintNumber))
	}
	panic(fmt.Errorf("toFloat(%T)", value.value))
}

const (
	float_2_64   float64 = 18446744073709551616.0
	float_2_63   float64 = 9223372036854775808.0
	float_2_32   float64 = 4294967296.0
	float_2_31   float64 = 2147483648.0
	float_2_16   float64 = 65536.0
	integer_2_32 int64   = 4294967296
	integer_2_31 int64   = 2146483648
	sqrt1_2      float64 = math.Sqrt2 / 2
)

const (
	maxInt8   = math.MaxInt8
	minInt8   = math.MinInt8
	maxInt16  = math.MaxInt16
	minInt16  = math.MinInt16
	maxInt32  = math.MaxInt32
	minInt32  = math.MinInt32
	maxInt64  = math.MaxInt64
	minInt64  = math.MinInt64
	maxUint8  = math.MaxUint8
	maxUint16 = math.MaxUint16
	maxUint32 = math.MaxUint32
	maxUint64 = math.MaxUint64
	maxUint   = ^uint(0)
	minUint   = 0
	maxInt    = int(^uint(0) >> 1)
	minInt    = -maxInt - 1

	// int64
	int64_maxInt    int64 = int64(maxInt)
	int64_minInt    int64 = int64(minInt)
	int64_maxInt8   int64 = math.MaxInt8
	int64_minInt8   int64 = math.MinInt8
	int64_maxInt16  int64 = math.MaxInt16
	int64_minInt16  int64 = math.MinInt16
	int64_maxInt32  int64 = math.MaxInt32
	int64_minInt32  int64 = math.MinInt32
	int64_maxUint8  int64 = math.MaxUint8
	int64_maxUint16 int64 = math.MaxUint16
	int64_maxUint32 int64 = math.MaxUint32

	// float64
	float_maxInt    float64 = float64(int(^uint(0) >> 1))
	float_minInt    float64 = float64(int(-maxInt - 1))
	float_minUint   float64 = float64(0)
	float_maxUint   float64 = float64(uint(^uint(0)))
	float_minUint64 float64 = float64(0)
	float_maxUint64 float64 = math.MaxUint64
	float_maxInt64  float64 = math.MaxInt64
	float_minInt64  float64 = math.MinInt64
)

func toIntegerFloat(value Value) float64 {
	float := value.toFloat()
	if math.IsInf(float, 0) {
	} else if math.IsNaN(float) {
		float = 0
	} else if float > 0 {
		float = math.Floor(float)
	} else {
		float = math.Ceil(float)
	}
	return float
}

type _integerKind int

const (
	integerValid    _integerKind = iota // 3.0 => 3.0
	integerFloat                        // 3.14159 => 3.0
	integerInfinite                     // Infinity => 2**63-1
	integerInvalid                      // NaN => 0
)

type _integer struct {
	_integerKind
	value int64
}

func (self _integer) valid() bool {
	return self._integerKind == integerValid || self._integerKind == integerFloat
}

func (self _integer) exact() bool {
	return self._integerKind == integerValid
}

func (self _integer) infinite() bool {
	return self._integerKind == integerInfinite
}

func toInteger(value Value) (integer _integer) {
	switch value := value.value.(type) {
	case int8:
		integer.value = int64(value)
		return
	case int16:
		integer.value = int64(value)
		return
	case uint8:
		integer.value = int64(value)
		return
	case uint16:
		integer.value = int64(value)
		return
	case uint32:
		integer.value = int64(value)
		return
	case int:
		integer.value = int64(value)
		return
	case int64:
		integer.value = value
		return
	}
	{
		value := value.toFloat()
		if value == 0 {
			return
		}
		if math.IsNaN(value) {
			integer._integerKind = integerInvalid
			return
		}
		if math.IsInf(value, 0) {
			integer._integerKind = integerInfinite
		}
		if value >= float_maxInt64 {
			integer.value = math.MaxInt64
			return
		}
		if value <= float_minInt64 {
			integer.value = math.MinInt64
			return
		}
		{
			value0 := value
			value1 := float64(0)
			if value0 > 0 {
				value1 = math.Floor(value0)
			} else {
				value1 = math.Ceil(value0)
			}

			if value0 != value1 {
				integer._integerKind = integerFloat
			}
			integer.value = int64(value1)
			return
		}
	}
}

// ECMA 262: 9.5
func toInt32(value Value) int32 {
	{
		switch value := value.value.(type) {
		case int8:
			return int32(value)
		case int16:
			return int32(value)
		case int32:
			return value
		}
	}
	floatValue := value.toFloat()
	if math.IsNaN(floatValue) || math.IsInf(floatValue, 0) {
		return 0
	}
	if floatValue == 0 { // This will work for +0 & -0
		return 0
	}
	remainder := math.Mod(floatValue, float_2_32)
	if remainder > 0 {
		remainder = math.Floor(remainder)
	} else {
		remainder = math.Ceil(remainder) + float_2_32
	}
	if remainder > float_2_31 {
		return int32(remainder - float_2_32)
	}
	return int32(remainder)
}

func toUint32(value Value) uint32 {
	{
		switch value := value.value.(type) {
		case int8:
			return uint32(value)
		case int16:
			return uint32(value)
		case uint8:
			return uint32(value)
		case uint16:
			return uint32(value)
		case uint32:
			return value
		}
	}
	floatValue := value.toFloat()
	if math.IsNaN(floatValue) || math.IsInf(floatValue, 0) {
		return 0
	}
	if floatValue == 0 {
		return 0
	}
	remainder := math.Mod(floatValue, float_2_32)
	if remainder > 0 {
		remainder = math.Floor(remainder)
	} else {
		remainder = math.Ceil(remainder) + float_2_32
	}
	return uint32(remainder)
}

func toUint16(value Value) uint16 {
	{
		switch value := value.value.(type) {
		case int8:
			return uint16(value)
		case uint8:
			return uint16(value)
		case uint16:
			return value
		}
	}
	floatValue := value.toFloat()
	if math.IsNaN(floatValue) || math.IsInf(floatValue, 0) {
		return 0
	}
	if floatValue == 0 {
		return 0
	}
	remainder := math.Mod(floatValue, float_2_16)
	if remainder > 0 {
		remainder = math.Floor(remainder)
	} else {
		remainder = math.Ceil(remainder) + float_2_16
	}
	return uint16(remainder)
}
