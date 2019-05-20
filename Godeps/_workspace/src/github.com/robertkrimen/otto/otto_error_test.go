package otto

import (
	"testing"
)

func TestOttoError(t *testing.T) {
	tt(t, func() {
		vm := New()

		_, err := vm.Run(`throw "Xyzzy"`)
		is(err, "Xyzzy")

		_, err = vm.Run(`throw new TypeError()`)
		is(err, "TypeError")

		_, err = vm.Run(`throw new TypeError("Nothing happens.")`)
		is(err, "TypeError: Nothing happens.")

		_, err = ToValue([]byte{})
		is(err, "TypeError: invalid value (slice): missing runtime: [] ([]uint8)")

		_, err = vm.Run(`
            (function(){
                return abcdef.length
            })()
        `)
		is(err, "ReferenceError: 'abcdef' is not defined")

		_, err = vm.Run(`
            function start() {
            }

            start()

                xyzzy()
        `)
		is(err, "ReferenceError: 'xyzzy' is not defined")

		_, err = vm.Run(`
            // Just a comment

            xyzzy
        `)
		is(err, "ReferenceError: 'xyzzy' is not defined")

	})
}
