package otto

import (
	"math"
	"reflect"
	"testing"
)

type testStruct struct {
	Abc bool
	Def int
	Ghi string
	Jkl interface{}
}

func (t *testStruct) FuncPointerReciever() string {
	return "abc"
}

func (t testStruct) FuncNoArgsNoRet() {
	return
}

func (t testStruct) FuncNoArgs() string {
	return "abc"
}

func (t testStruct) FuncNoArgsMultRet() (string, error) {
	return "def", nil
}

func (t testStruct) FuncOneArgs(a string) string {
	return a
}

func (t testStruct) FuncMultArgs(a, b string) string {
	return a + b
}

func (t testStruct) FuncVarArgs(as ...string) int {
	return len(as)
}

func TestReflect(t *testing.T) {
	return
	tt(t, func() {
		// Testing dbgf
		// These should panic
		toValue("Xyzzy").toReflectValue(reflect.Ptr)
		stringToReflectValue("Xyzzy", reflect.Ptr)
	})
}

func Test_reflectStruct(t *testing.T) {
	tt(t, func() {
		test, vm := test()

		// testStruct
		{
			abc := &testStruct{}
			vm.Set("abc", abc)

			test(`
                abc.FuncPointerReciever();
            `, "abc")

			test(`
                [ abc.Abc, abc.Ghi ];
            `, "false,")

			abc.Abc = true
			abc.Ghi = "Nothing happens."

			test(`
                [ abc.Abc, abc.Ghi ];
            `, "true,Nothing happens.")

			*abc = testStruct{}

			test(`
                [ abc.Abc, abc.Ghi ];
            `, "false,")

			abc.Abc = true
			abc.Ghi = "Xyzzy"
			vm.Set("abc", abc)

			test(`
                [ abc.Abc, abc.Ghi ];
            `, "true,Xyzzy")

			is(abc.Abc, true)
			test(`
                abc.Abc = false;
                abc.Def = 451;
                abc.Ghi = "Nothing happens.";
                abc.abc = "Something happens.";
                [ abc.Def, abc.abc ];
            `, "451,Something happens.")
			is(abc.Abc, false)
			is(abc.Def, 451)
			is(abc.Ghi, "Nothing happens.")

			test(`
                delete abc.Def;
                delete abc.abc;
                [ abc.Def, abc.abc ];
            `, "451,")
			is(abc.Def, 451)

			test(`
                abc.FuncNoArgsNoRet();
            `, "undefined")
			test(`
                abc.FuncNoArgs();
            `, "abc")
			test(`
                abc.FuncOneArgs("abc");
            `, "abc")
			test(`
                abc.FuncMultArgs("abc", "def");
            `, "abcdef")
			test(`
                abc.FuncVarArgs("abc", "def", "ghi");
            `, 3)

			test(`raise:
                abc.FuncNoArgsMultRet();
            `, "TypeError")
		}
	})
}

func Test_reflectMap(t *testing.T) {
	tt(t, func() {
		test, vm := test()

		// map[string]string
		{
			abc := map[string]string{
				"Xyzzy": "Nothing happens.",
				"def":   "1",
			}
			vm.Set("abc", abc)

			test(`
                abc.xyz = "pqr";
                [ abc.Xyzzy, abc.def, abc.ghi ];
            `, "Nothing happens.,1,")

			is(abc["xyz"], "pqr")
		}

		// map[string]float64
		{
			abc := map[string]float64{
				"Xyzzy": math.Pi,
				"def":   1,
			}
			vm.Set("abc", abc)

			test(`
                abc.xyz = "pqr";
                abc.jkl = 10;
                [ abc.Xyzzy, abc.def, abc.ghi ];
            `, "3.141592653589793,1,")

			is(abc["xyz"], math.NaN())
			is(abc["jkl"], float64(10))
		}

		// map[string]int32
		{
			abc := map[string]int32{
				"Xyzzy": 3,
				"def":   1,
			}
			vm.Set("abc", abc)

			test(`
                abc.xyz = "pqr";
                abc.jkl = 10;
                [ abc.Xyzzy, abc.def, abc.ghi ];
            `, "3,1,")

			is(abc["xyz"], 0)
			is(abc["jkl"], int32(10))

			test(`
                delete abc["Xyzzy"];
            `)

			_, exists := abc["Xyzzy"]
			is(exists, false)
			is(abc["Xyzzy"], 0)
		}

		// map[int32]string
		{
			abc := map[int32]string{
				0: "abc",
				1: "def",
			}
			vm.Set("abc", abc)

			test(`
                abc[2] = "pqr";
                //abc.jkl = 10;
                abc[3] = 10;
                [ abc[0], abc[1], abc[2], abc[3] ]
            `, "abc,def,pqr,10")

			is(abc[2], "pqr")
			is(abc[3], "10")

			test(`
                delete abc[2];
            `)

			_, exists := abc[2]
			is(exists, false)
		}

	})
}

func Test_reflectSlice(t *testing.T) {
	tt(t, func() {
		test, vm := test()

		// []bool
		{
			abc := []bool{
				false,
				true,
				true,
				false,
			}
			vm.Set("abc", abc)

			test(`
                abc;
            `, "false,true,true,false")

			test(`
                abc[0] = true;
                abc[abc.length-1] = true;
                delete abc[2];
                abc;
            `, "true,true,false,true")

			is(abc, []bool{true, true, false, true})
			is(abc[len(abc)-1], true)
		}

		// []int32
		{
			abc := make([]int32, 4)
			vm.Set("abc", abc)

			test(`
                abc;
            `, "0,0,0,0")

			test(`
                abc[0] = 4.2;
                abc[1] = "42";
                abc[2] = 3.14;
                abc;
            `, "4,42,3,0")

			is(abc, []int32{4, 42, 3, 0})

			test(`
                delete abc[1];
                delete abc[2];
            `)
			is(abc[1], 0)
			is(abc[2], 0)
		}
	})
}

func Test_reflectArray(t *testing.T) {
	tt(t, func() {
		test, vm := test()

		// []bool
		{
			abc := [4]bool{
				false,
				true,
				true,
				false,
			}
			vm.Set("abc", abc)

			test(`
                abc;
            `, "false,true,true,false")
			// Unaddressable array

			test(`
                abc[0] = true;
                abc[abc.length-1] = true;
                abc;
            `, "false,true,true,false")
			// Again, unaddressable array

			is(abc, [4]bool{false, true, true, false})
			is(abc[len(abc)-1], false)
			// ...
		}

		// []int32
		{
			abc := make([]int32, 4)
			vm.Set("abc", abc)

			test(`
                abc;
            `, "0,0,0,0")

			test(`
                abc[0] = 4.2;
                abc[1] = "42";
                abc[2] = 3.14;
                abc;
            `, "4,42,3,0")

			is(abc, []int32{4, 42, 3, 0})
		}

		// []bool
		{
			abc := [4]bool{
				false,
				true,
				true,
				false,
			}
			vm.Set("abc", &abc)

			test(`
                abc;
            `, "false,true,true,false")

			test(`
                abc[0] = true;
                abc[abc.length-1] = true;
                delete abc[2];
                abc;
            `, "true,true,false,true")

			is(abc, [4]bool{true, true, false, true})
			is(abc[len(abc)-1], true)
		}

	})
}

func Test_reflectArray_concat(t *testing.T) {
	tt(t, func() {
		test, vm := test()

		vm.Set("ghi", []string{"jkl", "mno"})
		vm.Set("pqr", []interface{}{"jkl", 42, 3.14159, true})
		test(`
            var def = {
                "abc": ["abc"],
                "xyz": ["xyz"]
            };
            xyz = pqr.concat(ghi, def.abc, def, def.xyz);
            [ xyz, xyz.length ];
        `, "jkl,42,3.14159,true,jkl,mno,abc,[object Object],xyz,9")
	})
}

func Test_reflectMapInterface(t *testing.T) {
	tt(t, func() {
		test, vm := test()

		{
			abc := map[string]interface{}{
				"Xyzzy": "Nothing happens.",
				"def":   "1",
				"jkl":   "jkl",
			}
			vm.Set("abc", abc)
			vm.Set("mno", &testStruct{})

			test(`
                abc.xyz = "pqr";
                abc.ghi = {};
                abc.jkl = 3.14159;
                abc.mno = mno;
                mno.Abc = true;
                mno.Ghi = "Something happens.";
                [ abc.Xyzzy, abc.def, abc.ghi, abc.mno ];
            `, "Nothing happens.,1,[object Object],[object Object]")

			is(abc["xyz"], "pqr")
			is(abc["ghi"], "[object Object]")
			is(abc["jkl"], float64(3.14159))
			mno, valid := abc["mno"].(*testStruct)
			is(valid, true)
			is(mno.Abc, true)
			is(mno.Ghi, "Something happens.")
		}
	})
}
