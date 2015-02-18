package otto

import (
	"testing"
)

func TestError(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`
            [ Error.prototype.name, Error.prototype.message, Error.prototype.hasOwnProperty("message") ];
        `, "Error,,true")
	})
}

func TestError_instanceof(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`(new TypeError()) instanceof Error`, true)
	})
}

func TestPanicValue(t *testing.T) {
	tt(t, func() {
		test, vm := test()

		vm.Set("abc", func(call FunctionCall) Value {
			value, err := call.Otto.Run(`({ def: 3.14159 })`)
			is(err, nil)
			panic(value)
		})

		test(`
            try {
                abc();
            }
            catch (err) {
                error = err;
            }
            [ error instanceof Error, error.message, error.def ];
        `, "false,,3.14159")
	})
}

func Test_catchPanic(t *testing.T) {
	tt(t, func() {
		vm := New()

		_, err := vm.Run(`
            A syntax error that
            does not define
            var;
                abc;
        `)
		is(err, "!=", nil)

		_, err = vm.Call(`abc.def`, nil)
		is(err, "!=", nil)
	})
}
