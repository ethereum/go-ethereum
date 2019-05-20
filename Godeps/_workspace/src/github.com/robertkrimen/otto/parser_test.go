package otto

import (
	"testing"
)

func TestPersistence(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`
        function abc() { return 1; }
        abc.toString();
    `, "function abc() { return 1; }")

		test(`
        function def() { return 3.14159; }
        [ abc.toString(), def.toString() ];
    `, "function abc() { return 1; },function def() { return 3.14159; }")

		test(`
        eval("function ghi() { return 'ghi' }");
        [ abc.toString(), def.toString(), ghi.toString() ];
    `, "function abc() { return 1; },function def() { return 3.14159; },function ghi() { return 'ghi' }")

		test(`
        [ abc.toString(), def.toString(), ghi.toString() ];
    `, "function abc() { return 1; },function def() { return 3.14159; },function ghi() { return 'ghi' }")

		test(`/*









    */`, UndefinedValue())
	})
}
