package otto

import (
	"encoding/json"
	"math"
	"testing"
)

func TestValue(t *testing.T) {
	tt(t, func() {
		value := UndefinedValue()
		is(value.IsUndefined(), true)
		is(value, UndefinedValue())
		is(value, "undefined")

		is(toValue(false), false)
		is(toValue(1), 1)
		is(toValue(1).float64(), float64(1))
	})
}

func TestObject(t *testing.T) {
	tt(t, func() {
		is(emptyValue.isEmpty(), true)
		//is(newObject().Value(), "[object]")
		//is(newBooleanObject(false).Value(), "false")
		//is(newFunctionObject(nil).Value(), "[function]")
		//is(newNumberObject(1).Value(), "1")
		//is(newStringObject("Hello, World.").Value(), "Hello, World.")
	})
}

type intAlias int

func TestToValue(t *testing.T) {
	tt(t, func() {
		_, tester := test()
		vm := tester.vm

		value, _ := vm.ToValue(nil)
		is(value, "undefined")

		value, _ = vm.ToValue((*byte)(nil))
		is(value, "undefined")

		value, _ = vm.ToValue(intAlias(5))
		is(value, 5)

		{
			tmp := new(int)

			value, _ = vm.ToValue(&tmp)
			is(value, 0)

			*tmp = 1

			value, _ = vm.ToValue(&tmp)
			is(value, 1)

			tmp = nil

			value, _ = vm.ToValue(&tmp)
			is(value, "undefined")
		}

		{
			tmp0 := new(int)
			tmp1 := &tmp0
			tmp2 := &tmp1

			value, _ = vm.ToValue(&tmp2)
			is(value, 0)

			*tmp0 = 1

			value, _ = vm.ToValue(&tmp2)
			is(value, 1)

			tmp0 = nil

			value, _ = vm.ToValue(&tmp2)
			is(value, "undefined")
		}
	})
}

func TestToBoolean(t *testing.T) {
	tt(t, func() {
		is := func(left interface{}, right bool) {
			is(toValue(left).bool(), right)
		}

		is("", false)
		is("xyzzy", true)
		is(1, true)
		is(0, false)
		//is(toValue(newObject()), true)
		is(UndefinedValue(), false)
		is(NullValue(), false)
	})
}

func TestToFloat(t *testing.T) {
	tt(t, func() {
		{
			is := func(left interface{}, right float64) {
				is(toValue(left).float64(), right)
			}
			is("", 0)
			is("xyzzy", math.NaN())
			is("2", 2)
			is(1, 1)
			is(0, 0)
			is(NullValue(), 0)
			//is(newObjectValue(), math.NaN())
		}
		is(math.IsNaN(UndefinedValue().float64()), true)
	})
}

func TestToString(t *testing.T) {
	tt(t, func() {
		is("undefined", UndefinedValue().string())
		is("null", NullValue().string())
		is("true", toValue(true).string())
		is("false", toValue(false).string())

		is(UndefinedValue(), "undefined")
		is(NullValue(), "null")
		is(toValue(true), true)
		is(toValue(false), false)
	})
}

func Test_toInt32(t *testing.T) {
	tt(t, func() {
		test := []interface{}{
			0, int32(0),
			1, int32(1),
			-2147483649.0, int32(2147483647),
			-4294967297.0, int32(-1),
			-4294967296.0, int32(0),
			-4294967295.0, int32(1),
			math.Inf(+1), int32(0),
			math.Inf(-1), int32(0),
		}
		for index := 0; index < len(test)/2; index++ {
			// FIXME terst, Make strict again?
			is(
				toInt32(toValue(test[index*2])),
				test[index*2+1].(int32),
			)
		}
	})
}

func Test_toUint32(t *testing.T) {
	tt(t, func() {
		test := []interface{}{
			0, uint32(0),
			1, uint32(1),
			-2147483649.0, uint32(2147483647),
			-4294967297.0, uint32(4294967295),
			-4294967296.0, uint32(0),
			-4294967295.0, uint32(1),
			math.Inf(+1), uint32(0),
			math.Inf(-1), uint32(0),
		}
		for index := 0; index < len(test)/2; index++ {
			// FIXME terst, Make strict again?
			is(
				toUint32(toValue(test[index*2])),
				test[index*2+1].(uint32),
			)
		}
	})
}

func Test_toUint16(t *testing.T) {
	tt(t, func() {
		test := []interface{}{
			0, uint16(0),
			1, uint16(1),
			-2147483649.0, uint16(65535),
			-4294967297.0, uint16(65535),
			-4294967296.0, uint16(0),
			-4294967295.0, uint16(1),
			math.Inf(+1), uint16(0),
			math.Inf(-1), uint16(0),
		}
		for index := 0; index < len(test)/2; index++ {
			// FIXME terst, Make strict again?
			is(
				toUint16(toValue(test[index*2])),
				test[index*2+1].(uint16),
			)
		}
	})
}

func Test_sameValue(t *testing.T) {
	tt(t, func() {
		is(sameValue(positiveZeroValue(), negativeZeroValue()), false)
		is(sameValue(positiveZeroValue(), toValue(0)), true)
		is(sameValue(NaNValue(), NaNValue()), true)
		is(sameValue(NaNValue(), toValue("Nothing happens.")), false)
	})
}

func TestExport(t *testing.T) {
	tt(t, func() {
		test, vm := test()

		is(test(`null;`).export(), nil)
		is(test(`undefined;`).export(), nil)
		is(test(`true;`).export(), true)
		is(test(`false;`).export(), false)
		is(test(`0;`).export(), 0)
		is(test(`3.1459`).export(), 3.1459)
		is(test(`"Nothing happens";`).export(), "Nothing happens")
		is(test(`String.fromCharCode(97,98,99,100,101,102)`).export(), "abcdef")
		{
			value := test(`({ abc: 1, def: true, ghi: undefined });`).export().(map[string]interface{})
			is(value["abc"], 1)
			is(value["def"], true)
			_, exists := value["ghi"]
			is(exists, false)
		}
		{
			value := test(`[ "abc", 1, "def", true, undefined, null ];`).export().([]interface{})
			is(value[0], "abc")
			is(value[1], 1)
			is(value[2], "def")
			is(value[3], true)
			is(value[4], nil)
			is(value[5], nil)
			is(value[5], interface{}(nil))
		}

		roundtrip := []interface{}{
			true,
			false,
			0,
			3.1459,
			[]interface{}{true, false, 0, 3.1459, "abc"},
			map[string]interface{}{
				"Boolean": true,
				"Number":  3.1459,
				"String":  "abc",
				"Array":   []interface{}{false, 0, "", nil},
				"Object": map[string]interface{}{
					"Boolean": false,
					"Number":  0,
					"String":  "def",
				},
			},
		}

		for _, value := range roundtrip {
			input, err := json.Marshal(value)
			is(err, nil)

			output, err := json.Marshal(test("(" + string(input) + ");").export())
			is(err, nil)

			is(string(input), string(output))
		}

		{
			abc := struct {
				def int
				ghi interface{}
				xyz float32
			}{}
			abc.def = 3
			abc.xyz = 3.1459
			vm.Set("abc", abc)
			is(test(`abc;`).export(), abc)
		}
	})
}
