package otto

import (
	"testing"
)

func BenchmarkJSON_parse(b *testing.B) {
	vm := New()
	for i := 0; i < b.N; i++ {
		vm.Run(`JSON.parse("1")`)
		vm.Run(`JSON.parse("[1,2,3]")`)
		vm.Run(`JSON.parse('{"a":{"x":100,"y":110},"b":[10,20,30],"c":"zazazaza"}')`)
		vm.Run(`JSON.parse("[1,2,3]", function(k, v) { return undefined })`)
	}
}

func TestJSON_parse(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`
            JSON.parse("1");
        `, 1)

		test(`
            JSON.parse("null");
        `, "null") // TODO Can we make this nil?

		test(`
            var abc = JSON.parse('"a\uFFFFbc"');
            [ abc[0], abc[2], abc[3], abc.length ];
        `, "a,b,c,4")

		test(`
            JSON.parse("[1, 2, 3]");
        `, "1,2,3")

		test(`
            JSON.parse('{ "abc": 1, "def":2 }').abc;
        `, 1)

		test(`
            JSON.parse('{ "abc": { "x": 100, "y": 110 }, "def": [ 10, 20 ,30 ], "ghi": "zazazaza" }').def;
        `, "10,20,30")

		test(`raise:
            JSON.parse("12\t\r\n 34");
        `, "SyntaxError: invalid character '3' after top-level value")

		test(`
            JSON.parse("[1, 2, 3]", function() { return undefined });
        `, "undefined")

		test(`raise:
            JSON.parse("");
        `, "SyntaxError: unexpected end of JSON input")

		test(`raise:
            JSON.parse("[1, 2, 3");
        `, "SyntaxError: unexpected end of JSON input")

		test(`raise:
            JSON.parse("[1, 2, ; abc=10");
        `, "SyntaxError: invalid character ';' looking for beginning of value")

		test(`raise:
            JSON.parse("[1, 2, function(){}]");
        `, "SyntaxError: invalid character 'u' in literal false (expecting 'a')")
	})
}

func TestJSON_stringify(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		defer mockUTC()()

		test(`
            JSON.stringify(function(){});
        `, "undefined")

		test(`
            JSON.stringify(new Boolean(false));
        `, "false")

		test(`
            JSON.stringify({a1: {b1: [1,2,3,4], b2: {c1: 1, c2: 2}}, a2: 'a2'}, null, -5);
        `, `{"a1":{"b1":[1,2,3,4],"b2":{"c1":1,"c2":2}},"a2":"a2"}`)

		test(`
            JSON.stringify(undefined);
        `, "undefined")

		test(`
            JSON.stringify(1);
        `, "1")

		test(`
            JSON.stringify("abc def");
        `, "\"abc def\"")

		test(`
            JSON.stringify(3.14159);
        `, "3.14159")

		test(`
            JSON.stringify([]);
        `, "[]")

		test(`
            JSON.stringify([1, 2, 3]);
        `, "[1,2,3]")

		test(`
            JSON.stringify([true, false, null]);
        `, "[true,false,null]")

		test(`
            JSON.stringify({
                abc: { x: 100, y: 110 },
                def: [ 10, 20, 30 ],
                ghi: "zazazaza"
            });
        `, `{"abc":{"x":100,"y":110},"def":[10,20,30],"ghi":"zazazaza"}`)

		test(`
            JSON.stringify([
                'e',
                {pluribus: 'unum'}
            ], null, '\t');
        `, "[\n\t\"e\",\n\t{\n\t\t\"pluribus\": \"unum\"\n\t}\n]")

		test(`
            JSON.stringify(new Date(0));
        `, `"1970-01-01T00:00:00.000Z"`)

		test(`
            JSON.stringify([ new Date(0) ], function(key, value){
                return this[key] instanceof Date ? 'Date(' + this[key] + ')' : value
            });
        `, `["Date(Thu, 01 Jan 1970 00:00:00 UTC)"]`)

		test(`
            JSON.stringify({
                abc: 1,
                def: 2,
                ghi: 3
            }, ['abc','def']);
        `, `{"abc":1,"def":2}`)

		test(`raise:
            var abc = {
                def: null
            };
            abc.def = abc;
            JSON.stringify(abc)
        `, "TypeError: Converting circular structure to JSON")

		test(`raise:
            var abc= [ null ];
            abc[0] = abc;
            JSON.stringify(abc);
        `, "TypeError: Converting circular structure to JSON")

		test(`raise:
            var abc = {
                def: {}
            };
            abc.def.ghi = abc;
            JSON.stringify(abc)
        `, "TypeError: Converting circular structure to JSON")

		test(`
            var ghi = { "pi": 3.14159 };
            var abc = {
                def: {}
            };
            abc.ghi = ghi;
            abc.def.ghi = ghi;
            JSON.stringify(abc);
        `, `{"def":{"ghi":{"pi":3.14159}},"ghi":{"pi":3.14159}}`)
	})
}
