package otto

import (
	"testing"
)

func TestNumber(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`
            var abc = Object.getOwnPropertyDescriptor(Number, "prototype");
            [   [ typeof Number.prototype ],
                [ abc.writable, abc.enumerable, abc.configurable ] ];
        `, "object,false,false,false")
	})
}

func TestNumber_toString(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`
            new Number(451).toString();
        `, "451")

		test(`
            new Number(451).toString(10);
        `, "451")

		test(`
            new Number(451).toString(8);
        `, "703")

		test(`raise:
            new Number(451).toString(1);
        `, "RangeError: RangeError: toString() radix must be between 2 and 36")

		test(`raise:
            new Number(451).toString(Infinity);
        `, "RangeError: RangeError: toString() radix must be between 2 and 36")

		test(`
            new Number(NaN).toString()
        `, "NaN")

		test(`
            new Number(Infinity).toString()
        `, "Infinity")

		test(`
            new Number(Infinity).toString(16)
        `, "Infinity")

		test(`
            [
                Number.prototype.toString(undefined),
                new Number().toString(undefined),
                new Number(0).toString(undefined),
                new Number(-1).toString(undefined),
                new Number(1).toString(undefined),
                new Number(Number.NaN).toString(undefined),
                new Number(Number.POSITIVE_INFINITY).toString(undefined),
                new Number(Number.NEGATIVE_INFINITY).toString(undefined)
            ]
        `, "0,0,0,-1,1,NaN,Infinity,-Infinity")
	})
}

func TestNumber_toFixed(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`new Number(451).toFixed(2)`, "451.00")
		test(`12345.6789.toFixed()`, "12346")
		test(`12345.6789.toFixed(1)`, "12345.7")
		test(`12345.6789.toFixed(6)`, "12345.678900")
		test(`(1.23e-20).toFixed(2)`, "0.00")
		test(`2.34.toFixed(1)`, "2.3") // FIXME Wtf? "2.3"
		test(`-2.34.toFixed(1)`, -2.3) // FIXME Wtf? -2.3
		test(`(-2.34).toFixed(1)`, "-2.3")

		test(`raise:
            new Number("a").toFixed(Number.POSITIVE_INFINITY);
        `, "RangeError: toFixed() precision must be between 0 and 20")

		test(`
            [
                new Number(1e21).toFixed(),
                new Number(1e21).toFixed(0),
                new Number(1e21).toFixed(1),
                new Number(1e21).toFixed(1.1),
                new Number(1e21).toFixed(0.9),
                new Number(1e21).toFixed("1"),
                new Number(1e21).toFixed("1.1"),
                new Number(1e21).toFixed("0.9"),
                new Number(1e21).toFixed(Number.NaN),
                new Number(1e21).toFixed("some string")
            ];
    `, "1e+21,1e+21,1e+21,1e+21,1e+21,1e+21,1e+21,1e+21,1e+21,1e+21")

		test(`raise:
            new Number(1e21).toFixed(Number.POSITIVE_INFINITY);
        `, "RangeError: toFixed() precision must be between 0 and 20")
	})
}

func TestNumber_toExponential(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`new Number(451).toExponential(2)`, "4.51e+02")
		test(`77.1234.toExponential()`, "7.71234e+01")
		test(`77.1234.toExponential(4)`, "7.7123e+01")
		test(`77.1234.toExponential(2)`, "7.71e+01")
		test(`77 .toExponential()`, "7.7e+01")
	})
}

func TestNumber_toPrecision(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`new Number(451).toPrecision()`, "451")
		test(`new Number(451).toPrecision(1)`, "5e+02")
		test(`5.123456.toPrecision()`, "5.123456")
		test(`5.123456.toPrecision(5)`, "5.1235")
		test(`5.123456.toPrecision(2)`, "5.1")
		test(`5.123456.toPrecision(1)`, "5")
	})
}

func TestNumber_toLocaleString(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`
            [
                new Number(451).toLocaleString(),
                new Number(451).toLocaleString(10),
                new Number(451).toLocaleString(8)
            ];
        `, "451,451,703")
	})
}

func TestValue_number(t *testing.T) {
	tt(t, func() {
		nm := toValue(0.0).number()
		is(nm.kind, numberInteger)

		nm = toValue(3.14159).number()
		is(nm.kind, numberFloat)
	})
}

func Test_NaN(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`
            [ NaN === NaN, NaN == NaN ];
        `, "false,false")
	})
}
