package otto

import (
	"testing"
)

func Test_panic(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		// Test that property.value is set to something if writable is set
		// to something
		test(`
            var abc = [];
            Object.defineProperty(abc, "0", { writable: false });
            Object.defineProperty(abc, "0", { writable: false });
            "0" in abc;
        `, true)

		test(`raise:
            var abc = [];
            Object.defineProperty(abc, "0", { writable: false });
            Object.defineProperty(abc, "0", { value: false, writable: false });
        `, "TypeError")

		// Test that a regular expression can contain \c0410 (CYRILLIC CAPITAL LETTER A)
		// without panicking
		test(`
            var abc = 0x0410;
            var def = String.fromCharCode(abc);
            new RegExp("\\c" + def).exec(def);
        `, "null")

		// Test transforming a transformable regular expression without a panic
		test(`
		    new RegExp("\\u0000");
            new RegExp("\\undefined").test("undefined");
        `, true)
	})
}
