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

func TestErrorContext(t *testing.T) {
	tt(t, func() {
		vm := New()

		_, err := vm.Run(`
            undefined();
        `)
		{
			err := err.(*Error)
			is(err.message, "'undefined' is not a function")
			is(len(err.trace), 1)
			is(err.trace[0].location(), "<anonymous>:2:13")
		}

		_, err = vm.Run(`
            ({}).abc();
        `)
		{
			err := err.(*Error)
			is(err.message, "'abc' is not a function")
			is(len(err.trace), 1)
			is(err.trace[0].location(), "<anonymous>:2:14")
		}

		_, err = vm.Run(`
            ("abc").abc();
        `)
		{
			err := err.(*Error)
			is(err.message, "'abc' is not a function")
			is(len(err.trace), 1)
			is(err.trace[0].location(), "<anonymous>:2:14")
		}

		_, err = vm.Run(`
            var ghi = "ghi";
            ghi();
        `)
		{
			err := err.(*Error)
			is(err.message, "'ghi' is not a function")
			is(len(err.trace), 1)
			is(err.trace[0].location(), "<anonymous>:3:13")
		}

		_, err = vm.Run(`
            function def() {
                undefined();
            }
            function abc() {
                def();
            }
            abc();
        `)
		{
			err := err.(*Error)
			is(err.message, "'undefined' is not a function")
			is(len(err.trace), 3)
			is(err.trace[0].location(), "def (<anonymous>:3:17)")
			is(err.trace[1].location(), "abc (<anonymous>:6:17)")
			is(err.trace[2].location(), "<anonymous>:8:13")
		}

		_, err = vm.Run(`
            function abc() {
                xyz();
            }
            abc();
        `)
		{
			err := err.(*Error)
			is(err.message, "'xyz' is not defined")
			is(len(err.trace), 2)
			is(err.trace[0].location(), "abc (<anonymous>:3:17)")
			is(err.trace[1].location(), "<anonymous>:5:13")
		}

		_, err = vm.Run(`
            mno + 1;
        `)
		{
			err := err.(*Error)
			is(err.message, "'mno' is not defined")
			is(len(err.trace), 1)
			is(err.trace[0].location(), "<anonymous>:2:13")
		}

		_, err = vm.Run(`
            eval("xyz();");
        `)
		{
			err := err.(*Error)
			is(err.message, "'xyz' is not defined")
			is(len(err.trace), 1)
			is(err.trace[0].location(), "<anonymous>:1:1")
		}

		_, err = vm.Run(`
            xyzzy = "Nothing happens."
            eval("xyzzy();");
        `)
		{
			err := err.(*Error)
			is(err.message, "'xyzzy' is not a function")
			is(len(err.trace), 1)
			is(err.trace[0].location(), "<anonymous>:1:1")
		}

		_, err = vm.Run(`
            throw Error("xyzzy");
        `)
		{
			err := err.(*Error)
			is(err.message, "xyzzy")
			is(len(err.trace), 1)
			is(err.trace[0].location(), "<anonymous>:2:19")
		}

		_, err = vm.Run(`
            throw new Error("xyzzy");
        `)
		{
			err := err.(*Error)
			is(err.message, "xyzzy")
			is(len(err.trace), 1)
			is(err.trace[0].location(), "<anonymous>:2:23")
		}
	})
}
