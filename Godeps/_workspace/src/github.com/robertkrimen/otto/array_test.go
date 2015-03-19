package otto

import (
	"testing"
)

func TestArray(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`
            var abc = [ undefined, "Nothing happens." ];
            abc.length;
        `, 2)

		test(`
            abc = ""+[0, 1, 2, 3];
            def = [].toString();
            ghi = [null, 4, "null"].toString();
            [ abc, def, ghi ];
        `, "0,1,2,3,,,4,null")

		test(`new Array(0).length`, 0)

		test(`new Array(11).length`, 11)

		test(`new Array(11, 1).length`, 2)

		test(`
            abc = [0, 1, 2, 3];
            abc.xyzzy = "Nothing happens.";
            delete abc[1];
            var xyzzy = delete abc.xyzzy;
            [ abc, xyzzy, abc.xyzzy ];
        `, "0,,2,3,true,")

		test(`
            var abc = [0, 1, 2, 3, 4];
            abc.length = 2;
            abc;
        `, "0,1")

		test(`raise:
            [].length = 3.14159;
        `, "RangeError")

		test(`raise:
            new Array(3.14159);
        `, "RangeError")

		test(`
            Object.defineProperty(Array.prototype, "0", {
                value: 100,
                writable: false,
                configurable: true
            });
            abc = [101];
            abc.hasOwnProperty("0") && abc[0] === 101;
        `, true)

		test(`
            abc = [,,undefined];
            [ abc.hasOwnProperty(0), abc.hasOwnProperty(1), abc.hasOwnProperty(2) ];
        `, "false,false,true")

		test(`
            abc = Object.getOwnPropertyDescriptor(Array, "prototype");
            [   [ typeof Array.prototype ],
                [ abc.writable, abc.enumerable, abc.configurable ] ];
        `, "object,false,false,false")
	})
}

func TestArray_toString(t *testing.T) {
	tt(t, func() {
		{
			test(`
                Array.prototype.toString = function() {
                    return "Nothing happens.";
                }
                abc = Array.prototype.toString();
                def = [].toString();
                ghi = [null, 4, "null"].toString();

                [ abc, def, ghi ].join(",");
            `, "Nothing happens.,Nothing happens.,Nothing happens.")
		}

		{
			test(`
                Array.prototype.join = undefined
                abc = Array.prototype.toString()
                def = [].toString()
                ghi = [null, 4, "null"].toString()

                abc + "," + def + "," + ghi;
            `, "[object Array],[object Array],[object Array]")
		}
	})
}

func TestArray_toLocaleString(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		defer mockUTC()()

		test(`
            [ 3.14159, "abc", undefined, new Date(0) ].toLocaleString();
        `, "3.14159,abc,,1970-01-01 00:00:00")

		test(`raise:
            [ { toLocaleString: undefined } ].toLocaleString();
        `, "TypeError")
	})
}

func TestArray_concat(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`
            abc = [0, 1, 2];
            def = [-1, -2, -3];
            ghi = abc.concat(def);
            jkl = abc.concat(def, 3, 4, 5);
            mno = def.concat(-4, -5, abc);

            [ ghi, jkl, mno ].join(";");
        `, "0,1,2,-1,-2,-3;0,1,2,-1,-2,-3,3,4,5;-1,-2,-3,-4,-5,0,1,2")

		test(`
            var abc = [,1];
            var def = abc.concat([], [,]);

            def.getClass = Object.prototype.toString;

            [ def.getClass(), typeof def[0], def[1], typeof def[2], def.length ];
        `, "[object Array],undefined,1,undefined,3")

		test(`
            Object.defineProperty(Array.prototype, "0", {
                value: 100,
                writable: false,
                configurable: true
            });

            var abc = Array.prototype.concat.call(101);

            var hasProperty = abc.hasOwnProperty("0");
            var instanceOfVerify = typeof abc[0] === "object";
            var verifyValue = false;
            verifyValue = abc[0] == 101;

            var verifyEnumerable = false;
            for (var property in abc) {
                if (property === "0" && abc.hasOwnProperty("0")) {
                    verifyEnumerable = true;
                }
            }

            var verifyWritable = false;
            abc[0] = 12;
            verifyWritable = abc[0] === 12;

            var verifyConfigurable = false;
            delete abc[0];
            verifyConfigurable = abc.hasOwnProperty("0");

            [ hasProperty, instanceOfVerify, verifyValue, !verifyConfigurable, verifyEnumerable, verifyWritable ];
        `, "true,true,true,true,true,true")
	})
}

func TestArray_splice(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`
            abc = [0, 1, 2];
            def = abc.splice(1, 2, 3, 4, 5);
            ghi = [].concat(abc);
            jkl = ghi.splice(17, 21, 7, 8, 9);
            [ abc, def, ghi, jkl ].join(";");
        `, "0,3,4,5;1,2;0,3,4,5,7,8,9;")
	})
}

func TestArray_shift(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`
            abc = [0, 1, 2];
            def = abc.shift();
            ghi = [].concat(abc);
            jkl = abc.shift();
            mno = [].concat(abc);
            pqr = abc.shift();
            stu = [].concat(abc);
            vwx = abc.shift();

            [ abc, def, ghi, jkl, mno, pqr, stu, vwx ].join(";");
        `, ";0;1,2;1;2;2;;")
	})
}

func TestArray_push(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`
            abc = [0];
            def = abc.push(1);
            ghi = [].concat(abc);
            jkl = abc.push(2,3,4);

            [ abc, def, ghi, jkl ].join(";");
        `, "0,1,2,3,4;2;0,1;5")
	})
}

func TestArray_pop(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`
            abc = [0,1];
            def = abc.pop();
            ghi = [].concat(abc);
            jkl = abc.pop();
            mno = [].concat(abc);
            pqr = abc.pop();

            [ abc, def, ghi, jkl, mno, pqr ].join(";");
        `, ";1;0;0;;")
	})
}

func TestArray_slice(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`
            abc = [0,1,2,3];
            def = abc.slice();
            ghi = abc.slice(1);
            jkl = abc.slice(3,-1);
            mno = abc.slice(2,-1);
            pqr = abc.slice(-1, -10);

            [ abc, def, ghi, jkl, mno, pqr ].join(";");
        `, "0,1,2,3;0,1,2,3;1,2,3;;2;")

		// Array.protoype.slice is generic
		test(`
            abc = { 0: 0, 1: 1, 2: 2, 3: 3 };
            abc.length = 4;
            def = Array.prototype.slice.call(abc);
            ghi = Array.prototype.slice.call(abc,1);
            jkl = Array.prototype.slice.call(abc,3,-1);
            mno = Array.prototype.slice.call(abc,2,-1);
            pqr = Array.prototype.slice.call(abc,-1,-10);

            [ abc, def, ghi, jkl, pqr ].join(";");
        `, "[object Object];0,1,2,3;1,2,3;;")
	})
}

func TestArray_sliceArguments(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`
            (function(){
                return Array.prototype.slice.call(arguments, 1)
            })({}, 1, 2, 3);
        `, "1,2,3")
	})
}

func TestArray_unshift(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`
            abc = [];
            def = abc.unshift(0);
            ghi = [].concat(abc);
            jkl = abc.unshift(1,2,3,4);

            [ abc, def, ghi, jkl ].join(";");
        `, "1,2,3,4,0;1;0;5")
	})
}

func TestArray_reverse(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`
            abc = [0,1,2,3].reverse();
            def = [0,1,2].reverse();

            [ abc, def ];
        `, "3,2,1,0,2,1,0")
	})
}

func TestArray_sort(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`
            abc = [0,1,2,3].sort();
            def = [3,2,1,0].sort();
            ghi = [].sort();
            jkl = [0].sort();
            mno = [1,0].sort();
            pqr = [1,5,-10, 100, 8, 72, 401, 0.05].sort();
            stu = [1,5,-10, 100, 8, 72, 401, 0.05].sort(function(x, y){
                return x == y ? 0 : x < y ? -1 : 1
            });

            [ abc, def, ghi, jkl, mno, pqr, stu ].join(";");
        `, "0,1,2,3;0,1,2,3;;0;0,1;-10,0.05,1,100,401,5,72,8;-10,0.05,1,5,8,72,100,401")

		test(`Array.prototype.sort.length`, 1)
	})
}

func TestArray_isArray(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`
        [ Array.isArray.length, Array.isArray(), Array.isArray([]), Array.isArray({}) ];
        `, "1,false,true,false")

		test(`Array.isArray(Math)`, false)
	})
}

func TestArray_indexOf(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`['a', 'b', 'c', 'b'].indexOf('b')`, 1)

		test(`['a', 'b', 'c', 'b'].indexOf('b', 2)`, 3)

		test(`['a', 'b', 'c', 'b'].indexOf('b', -2)`, 3)

		test(`
            Object.prototype.indexOf = Array.prototype.indexOf;
            var abc = {0: 'a', 1: 'b', 2: 'c', length: 3};
            abc.indexOf('c');
        `, 2)

		test(`[true].indexOf(true, "-Infinity")`, 0)

		test(`
            var target = {};
            Math[3] = target;
            Math.length = 5;
            Array.prototype.indexOf.call(Math, target) === 3;
        `, true)

		test(`
            var _NaN = NaN;
            var abc = new Array("NaN", undefined, 0, false, null, {toString:function(){return NaN}}, "false", _NaN, NaN);
            abc.indexOf(NaN);
        `, -1)

		test(`
            var abc = {toString:function (){return 0}};
            var def = 1;
            var ghi = -(4/3);
            var jkl = new Array(false, undefined, null, "0", abc, -1.3333333333333, "string", -0, true, +0, def, 1, 0, false, ghi, -(4/3));
            [ jkl.indexOf(-(4/3)), jkl.indexOf(0), jkl.indexOf(-0), jkl.indexOf(1) ];
        `, "14,7,7,10")
	})
}

func TestArray_lastIndexOf(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`['a', 'b', 'c', 'b'].lastIndexOf('b')`, 3)

		test(`['a', 'b', 'c', 'b'].lastIndexOf('b', 2)`, 1)

		test(`['a', 'b', 'c', 'b'].lastIndexOf('b', -2)`, 1)

		test(`
            Object.prototype.lastIndexOf = Array.prototype.lastIndexOf;
            var abc = {0: 'a', 1: 'b', 2: 'c', 3: 'b', length: 4};
            abc.lastIndexOf('b');
        `, 3)

		test(`
            var target = {};
            Math[3] = target;
            Math.length = 5;
            [ Array.prototype.lastIndexOf.call(Math, target) === 3 ];
        `, "true")

		test(`
            var _NaN = NaN;
            var abc = new Array("NaN", undefined, 0, false, null, {toString:function(){return NaN}}, "false", _NaN, NaN);
            abc.lastIndexOf(NaN);
        `, -1)

		test(`
            var abc = {toString:function (){return 0}};
            var def = 1;
            var ghi = -(4/3);
            var jkl = new Array(false, undefined, null, "0", abc, -1.3333333333333, "string", -0, true, +0, def, 1, 0, false, ghi, -(4/3));
            [ jkl.lastIndexOf(-(4/3)), jkl.indexOf(0), jkl.indexOf(-0), jkl.indexOf(1) ];
        `, "15,7,7,10")
	})
}

func TestArray_every(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`raise: [].every()`, "TypeError")

		test(`raise: [].every("abc")`, "TypeError")

		test(`[].every(function() { return false })`, true)

		test(`[1,2,3].every(function() { return false })`, false)

		test(`[1,2,3].every(function() { return true })`, true)

		test(`[1,2,3].every(function(_, index) { if (index === 1) return true })`, false)

		test(`
            var abc = function(value, index, object) {
                return ('[object Math]' !== Object.prototype.toString.call(object));
            };

            Math.length = 1;
            Math[0] = 1;
            !Array.prototype.every.call(Math, abc);
        `, true)

		test(`
            var def = false;

            var abc = function(value, index, object) {
                def = true;
                return this === Math;
            };

            [11].every(abc, Math) && def;
        `, true)

		test(`
            var def = false;

            var abc = function(value, index, object) {
                def = true;
                return Math;
            };

            [11].every(abc) && def;
        `, true)
	})
}

func TestArray_some(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`raise: [].some("abc")`, "TypeError")

		test(`[].some(function() { return true })`, false)

		test(`[1,2,3].some(function() { return false })`, false)

		test(`[1,2,3].some(function() { return true })`, true)

		test(`[1,2,3].some(function(_, index) { if (index === 1) return true })`, true)

		test(`
            var abc = function(value, index, object) {
                return ('[object Math]' !== Object.prototype.toString.call(object));
            };

            Math.length = 1;
            Math[0] = 1;
            !Array.prototype.some.call(Math, abc);
        `, true)

		test(`
            var abc = function(value, index, object) {
                return this === Math;
            };

            [11].some(abc, Math);
        `, true)

		test(`
            var abc = function(value, index, object) {
                return Math;
            };

            [11].some(abc);
        `, true)
	})
}

func TestArray_forEach(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`raise: [].forEach("abc")`, "TypeError")

		test(`
            var abc = 0;
            [].forEach(function(value) {
                abc += value;
            });
            abc;
        `, 0)

		test(`
            abc = 0;
            var def = [];
            [1,2,3].forEach(function(value, index) {
                abc += value;
                def.push(index);
            });
            [ abc, def ];
        `, "6,0,1,2")

		test(`
            var def = false;
            var abc = function(value, index, object) {
                def = ('[object Math]' === Object.prototype.toString.call(object));
            };

            Math.length = 1;
            Math[0] = 1;
            Array.prototype.forEach.call(Math, abc);
            def;
        `, true)

		test(`
            var def = false;
            var abc = function(value, index, object) {
                def = this === Math;
            };

            [11].forEach(abc, Math);
            def;
        `, true)
	})
}

func TestArray_indexing(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`
            var abc = new Array(0, 1);
            var def = abc.length;
            abc[4294967296] = 10; // 2^32 => 0
            abc[4294967297] = 11; // 2^32+1 => 1
            [ def, abc.length, abc[0], abc[1], abc[4294967296] ];
        `, "2,2,0,1,10")

		test(`
            abc = new Array(0, 1);
            def = abc.length;
            abc[4294967295] = 10;
            var ghi = abc.length;
            abc[4294967299] = 12;
            var jkl = abc.length;
            abc[4294967294] = 11;
            [ def, ghi, jkl, abc.length, abc[4294967295], abc[4294967299] ];
        `, "2,2,2,4294967295,10,12")
	})
}

func TestArray_map(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`raise: [].map("abc")`, "TypeError")

		test(`[].map(function() { return 1 }).length`, 0)

		test(`[1,2,3].map(function(value) { return value * value })`, "1,4,9")

		test(`[1,2,3].map(function(value) { return 1 })`, "1,1,1")

		test(`
            var abc = function(value, index, object) {
                return ('[object Math]' === Object.prototype.toString.call(object));
            };

            Math.length = 1;
            Math[0] = 1;
            Array.prototype.map.call(Math, abc)[0];
        `, true)

		test(`
            var abc = function(value, index, object) {
                return this === Math;
            };

            [11].map(abc, Math)[0];
        `, true)
	})
}

func TestArray_filter(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`raise: [].filter("abc")`, "TypeError")

		test(`[].filter(function() { return 1 }).length`, 0)

		test(`[1,2,3].filter(function() { return false }).length`, 0)

		test(`[1,2,3].filter(function() { return true })`, "1,2,3")
	})
}

func TestArray_reduce(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`raise: [].reduce("abc")`, "TypeError")

		test(`raise: [].reduce(function() {})`, "TypeError")

		test(`[].reduce(function() {}, 0)`, 0)

		test(`[].reduce(function() {}, undefined)`, "undefined")

		test(`['a','b','c'].reduce(function(result, value) { return result+', '+value })`, "a, b, c")

		test(`[1,2,3].reduce(function(result, value) { return result + value }, 4)`, 10)

		test(`[1,2,3].reduce(function(result, value) { return result + value })`, 6)
	})
}

func TestArray_reduceRight(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`raise: [].reduceRight("abc")`, "TypeError")

		test(`raise: [].reduceRight(function() {})`, "TypeError")

		test(`[].reduceRight(function() {}, 0)`, 0)

		test(`[].reduceRight(function() {}, undefined)`, "undefined")

		test(`['a','b','c'].reduceRight(function(result, value) { return result+', '+value })`, "c, b, a")

		test(`[1,2,3].reduceRight(function(result, value) { return result + value }, 4)`, 10)

		test(`[1,2,3].reduceRight(function(result, value) { return result + value })`, 6)
	})
}

func TestArray_defineOwnProperty(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`
            var abc = [];
            Object.defineProperty(abc, "length", {
                writable: false
            });
            abc.length;
        `, 0)

		test(`raise:
            var abc = [];
            var exception;
            Object.defineProperty(abc, "length", {
                writable: false
            });
            Object.defineProperty(abc, "length", {
                writable: true
            });
        `, "TypeError")
	})
}

func TestArray_new(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`
            var abc = new Array(null);
            var def = new Array(undefined);
            [ abc.length, abc[0] === null, def.length, def[0] === undefined ]
        `, "1,true,1,true")

		test(`
            var abc = new Array(new Number(0));
            var def = new Array(new Number(4294967295));
            [ abc.length, typeof abc[0], abc[0] == 0, def.length, typeof def[0], def[0] == 4294967295 ]
        `, "1,object,true,1,object,true")
	})
}
