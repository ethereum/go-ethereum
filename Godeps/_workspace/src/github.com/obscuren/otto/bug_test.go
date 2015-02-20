package otto

import (
	"testing"
	"time"
)

func Test_262(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		// 11.13.1-1-1
		test(`raise:
            eval("42 = 42;");
        `, "ReferenceError: Invalid left-hand side in assignment")
	})
}

func Test_issue5(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`'abc' === 'def'`, false)
		test(`'\t' === '\r'`, false)
	})
}

func Test_issue13(t *testing.T) {
	tt(t, func() {
		test, tester := test()
		vm := tester.vm

		value, err := vm.ToValue(map[string]interface{}{
			"string": "Xyzzy",
			"number": 42,
			"array":  []string{"def", "ghi"},
		})
		if err != nil {
			t.Error(err)
			t.FailNow()
		}

		fn, err := vm.Object(`
            (function(value){
                return ""+[value.string, value.number, value.array]
            })
        `)
		if err != nil {
			t.Error(err)
			t.FailNow()
		}

		result, err := fn.Value().Call(fn.Value(), value)
		if err != nil {
			t.Error(err)
			t.FailNow()
		}
		is(result.toString(), "Xyzzy,42,def,ghi")

		anything := struct {
			Abc interface{}
		}{
			Abc: map[string]interface{}{
				"def": []interface{}{
					[]interface{}{
						"a", "b", "c", "", "d", "e",
					},
					map[string]interface{}{
						"jkl": "Nothing happens.",
					},
				},
				"ghi": -1,
			},
		}

		vm.Set("anything", anything)
		test(`
            [
                anything,
                "~",
                anything.Abc,
                "~",
                anything.Abc.def,
                "~",
                anything.Abc.def[1].jkl,
                "~",
                anything.Abc.ghi,
            ];
        `, "[object Object],~,[object Object],~,a,b,c,,d,e,[object Object],~,Nothing happens.,~,-1")
	})
}

func Test_issue16(t *testing.T) {
	tt(t, func() {
		test, vm := test()

		test(`
            var def = {
                "abc": ["abc"],
                "xyz": ["xyz"]
            };
            def.abc.concat(def.xyz);
        `, "abc,xyz")

		vm.Set("ghi", []string{"jkl", "mno"})

		test(`
            def.abc.concat(def.xyz).concat(ghi);
        `, "abc,xyz,jkl,mno")

		test(`
            ghi.concat(def.abc.concat(def.xyz));
        `, "jkl,mno,abc,xyz")

		vm.Set("pqr", []interface{}{"jkl", 42, 3.14159, true})

		test(`
            pqr.concat(ghi, def.abc, def, def.xyz);
        `, "jkl,42,3.14159,true,jkl,mno,abc,[object Object],xyz")

		test(`
            pqr.concat(ghi, def.abc, def, def.xyz).length;
        `, 9)
	})
}

func Test_issue21(t *testing.T) {
	tt(t, func() {
		vm1 := New()
		vm1.Run(`
            abc = {}
            abc.ghi = "Nothing happens.";
            var jkl = 0;
            abc.def = function() {
                jkl += 1;
                return 1;
            }
        `)
		abc, err := vm1.Get("abc")
		is(err, nil)

		vm2 := New()
		vm2.Set("cba", abc)
		_, err = vm2.Run(`
            var pqr = 0;
            cba.mno = function() {
                pqr -= 1;
                return 1;
            }
            cba.def();
            cba.def();
            cba.def();
        `)
		is(err, nil)

		jkl, err := vm1.Get("jkl")
		is(err, nil)
		is(jkl, 3)

		_, err = vm1.Run(`
            abc.mno();
            abc.mno();
            abc.mno();
        `)
		is(err, nil)

		pqr, err := vm2.Get("pqr")
		is(err, nil)
		is(pqr, -3)
	})
}

func Test_issue24(t *testing.T) {
	tt(t, func() {
		_, vm := test()

		{
			vm.Set("abc", []string{"abc", "def", "ghi"})
			value, err := vm.Get("abc")
			is(err, nil)
			export, _ := value.Export()
			{
				value, valid := export.([]string)
				is(valid, true)

				is(value[0], "abc")
				is(value[2], "ghi")
			}
		}

		{
			vm.Set("abc", [...]string{"abc", "def", "ghi"})
			value, err := vm.Get("abc")
			is(err, nil)
			export, _ := value.Export()
			{
				value, valid := export.([3]string)
				is(valid, true)

				is(value[0], "abc")
				is(value[2], "ghi")
			}
		}

		{
			vm.Set("abc", &[...]string{"abc", "def", "ghi"})
			value, err := vm.Get("abc")
			is(err, nil)
			export, _ := value.Export()
			{
				value, valid := export.(*[3]string)
				is(valid, true)

				is(value[0], "abc")
				is(value[2], "ghi")
			}
		}

		{
			vm.Set("abc", map[int]string{0: "abc", 1: "def", 2: "ghi"})
			value, err := vm.Get("abc")
			is(err, nil)
			export, _ := value.Export()
			{
				value, valid := export.(map[int]string)
				is(valid, true)

				is(value[0], "abc")
				is(value[2], "ghi")
			}
		}

		{
			vm.Set("abc", testStruct{Abc: true, Ghi: "Nothing happens."})
			value, err := vm.Get("abc")
			is(err, nil)
			export, _ := value.Export()
			{
				value, valid := export.(testStruct)
				is(valid, true)

				is(value.Abc, true)
				is(value.Ghi, "Nothing happens.")
			}
		}

		{
			vm.Set("abc", &testStruct{Abc: true, Ghi: "Nothing happens."})
			value, err := vm.Get("abc")
			is(err, nil)
			export, _ := value.Export()
			{
				value, valid := export.(*testStruct)
				is(valid, true)

				is(value.Abc, true)
				is(value.Ghi, "Nothing happens.")
			}
		}
	})
}

func Test_issue39(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`
            var abc = 0, def = [], ghi = function() {
                if (abc < 10) return ++abc;
                return undefined;
            }
            for (var jkl; (jkl = ghi());) def.push(jkl);
            def;
        `, "1,2,3,4,5,6,7,8,9,10")

		test(`
            var abc = ["1", "2", "3", "4"];
            var def = [];
            for (var ghi; (ghi = abc.shift());) {
                def.push(ghi);
            }
            def;
        `, "1,2,3,4")
	})
}

func Test_issue64(t *testing.T) {
	tt(t, func() {
		test, vm := test()

		defer mockTimeLocal(time.UTC)()

		abc := map[string]interface{}{
			"time": time.Unix(0, 0),
		}
		vm.Set("abc", abc)

		def := struct {
			Public  string
			private string
		}{
			"Public", "private",
		}
		vm.Set("def", def)

		test(`"sec" in abc.time`, false)

		test(`
            [ "Public" in def, "private" in def, def.Public, def.private ];
        `, "true,false,Public,")

		test(`JSON.stringify(abc)`, `{"time":"1970-01-01T00:00:00Z"}`)
	})
}

func Test_7_3_1(t *testing.T) {
	tt(t, func() {
		test(`

            eval("var test7_3_1\u2028abc = 66;");
            [ abc, typeof test7_3_1 ];
        `, "66,undefined")
	})
}

func Test_7_3_3(t *testing.T) {
	tt(t, func() {
		test(`raise:
            eval("//\u2028 =;");
        `, "SyntaxError: Unexpected token =")
	})
}

func Test_S7_3_A2_1_T1(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`raise:
            eval("'\u000Astr\u000Aing\u000A'")
        `, "SyntaxError: Unexpected token ILLEGAL")
	})
}

func Test_S7_8_3_A2_1_T1(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`
            [ .0 === 0.0, .0, .1 === 0.1, .1 ]
        `, "true,0,true,0.1")
	})
}

func Test_S7_8_4_A4_2_T3(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`
            "\a"
        `, "a")
	})
}

func Test_S7_9_A1(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`
            var def;
            abc: for (var i = 0; i <= 0; i++) {
                for (var j = 0; j <= 1; j++) {
                    if (j === 0) {
                        continue abc;
                    } else {
                        def = true;
                    }
                }
            }
            [ def, i, j ];
        `, ",1,0")
	})
}

func Test_S7_9_A3(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`
            (function(){
                return
                1;
            })()
        `, "undefined")
	})
}

func Test_7_3_10(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`
            eval("var \u0061\u0062\u0063 = 3.14159;");
            abc;
        `, 3.14159)

		test(`
            abc = undefined;
            eval("var \\u0061\\u0062\\u0063 = 3.14159;");
            abc;
        `, 3.14159)
	})
}

func Test_bug(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		// 10.4.2-1-5
		test(`
            "abc\
def"
        `, "abcdef")

		test(`
            eval("'abc';\
            'def'")
        `, "def")

		// S12.6.1_A10
		test(`
            var abc = 0;
            do {
            if(typeof(def) === "function"){
                abc = -1;
                break;
            } else {
                abc = 1;
                break;
            }
            } while(function def(){});
            abc;
        `, 1)

		// S12.7_A7
		test(`raise:
            abc:
            while (true) {
                eval("continue abc");
            }
        `, "SyntaxError: Undefined label 'abc'")

		// S15.1.2.1_A3.3_T3
		test(`raise:
            eval("return");
        `, "SyntaxError: Illegal return statement")

		// 15.2.3.3-2-33
		test(`
            var abc = { "AB\n\\cd": 1 };
            Object.getOwnPropertyDescriptor(abc, "AB\n\\cd").value;
        `, 1)

		// S15.3_A2_T1
		test(`raise:
            Function.call(this, "var x / = 1;");
        `, "SyntaxError: Unexpected token /")

		// ?
		test(`
            (function(){
                var abc = [];
                (function(){
                    abc.push(0);
                    abc.push(1);
                })(undefined);
                if ((function(){ return true; })()) {
                    (function(){
                        abc.push(2);
                    })();
                }
                return abc;
            })();
        `, "0,1,2")

		if false {
			// 15.9.5.43-0-10
			// Should be an invalid date
			test(`
                date = new Date(1970, 0, -99999999, 0, 0, 0, 1);
            `, "")
		}

		// S7.8.3_A1.2_T1
		test(`
            [ 0e1, 1e1, 2e1, 3e1, 4e1, 5e1, 6e1, 7e1, 8e1, 9e1 ];
        `, "0,10,20,30,40,50,60,70,80,90")

		// S15.10.2.7_A3_T2
		test(`
            var abc = /\s+abc\s+/.exec("\t abc def");
            [ abc.length, abc.index, abc.input, abc ];
        `, "1,0,\t abc def,\t abc ")
	})
}
