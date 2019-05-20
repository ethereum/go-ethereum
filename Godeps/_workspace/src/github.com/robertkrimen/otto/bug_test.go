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
		is(result.string(), "Xyzzy,42,def,ghi")

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
			vm.Set("abc", _abcStruct{Abc: true, Ghi: "Nothing happens."})
			value, err := vm.Get("abc")
			is(err, nil)
			export, _ := value.Export()
			{
				value, valid := export.(_abcStruct)
				is(valid, true)

				is(value.Abc, true)
				is(value.Ghi, "Nothing happens.")
			}
		}

		{
			vm.Set("abc", &_abcStruct{Abc: true, Ghi: "Nothing happens."})
			value, err := vm.Get("abc")
			is(err, nil)
			export, _ := value.Export()
			{
				value, valid := export.(*_abcStruct)
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

func Test_issue73(t *testing.T) {
	tt(t, func() {
		test, vm := test()

		vm.Set("abc", [4]int{3, 2, 1, 0})

		test(`
            var def = [ 0, 1, 2, 3 ];
            JSON.stringify(def) + JSON.stringify(abc);
        `, "[0,1,2,3][3,2,1,0]")
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

func Test_issue79(t *testing.T) {
	tt(t, func() {
		test, vm := test()

		vm.Set("abc", []_abcStruct{
			{
				Ghi: "一",
				Def: 1,
			},
			{
				Def: 3,
				Ghi: "三",
			},
			{
				Def: 2,
				Ghi: "二",
			},
			{
				Def: 4,
				Ghi: "四",
			},
		})

		test(`
            abc.sort(function(a,b){ return b.Def-a.Def });
            def = [];
            for (i = 0; i < abc.length; i++) {
                def.push(abc[i].String())
            }
            def;
        `, "四,三,二,一")
	})
}

func Test_issue80(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		test(`
            JSON.stringify([
                1401868959,
                14018689591,
                140186895901,
                1401868959001,
                14018689590001,
                140186895900001,
                1401868959000001,
                1401868959000001.5,
                14018689590000001,
                140186895900000001,
                1401868959000000001,
                14018689590000000001,
                140186895900000000001,
                140186895900000000001.5
            ]);
        `, "[1401868959,14018689591,140186895901,1401868959001,14018689590001,140186895900001,1401868959000001,1.4018689590000015e+15,14018689590000001,140186895900000001,1401868959000000001,1.401868959e+19,1.401868959e+20,1.401868959e+20]")
	})
}

func Test_issue87(t *testing.T) {
	tt(t, func() {
		test, vm := test()

		test(`
            var def = 0;
            abc: {
                for (;;) {
                    def = !1;
                    break abc;
                }
                def = !0;
            }
            def;
        `, false)

		_, err := vm.Run(`
/*
CryptoJS v3.1.2
code.google.com/p/crypto-js
(c) 2009-2013 by Jeff Mott. All rights reserved.
code.google.com/p/crypto-js/wiki/License
*/
var CryptoJS=CryptoJS||function(h,s){var f={},g=f.lib={},q=function(){},m=g.Base={extend:function(a){q.prototype=this;var c=new q;a&&c.mixIn(a);c.hasOwnProperty("init")||(c.init=function(){c.$super.init.apply(this,arguments)});c.init.prototype=c;c.$super=this;return c},create:function(){var a=this.extend();a.init.apply(a,arguments);return a},init:function(){},mixIn:function(a){for(var c in a)a.hasOwnProperty(c)&&(this[c]=a[c]);a.hasOwnProperty("toString")&&(this.toString=a.toString)},clone:function(){return this.init.prototype.extend(this)}},
r=g.WordArray=m.extend({init:function(a,c){a=this.words=a||[];this.sigBytes=c!=s?c:4*a.length},toString:function(a){return(a||k).stringify(this)},concat:function(a){var c=this.words,d=a.words,b=this.sigBytes;a=a.sigBytes;this.clamp();if(b%4)for(var e=0;e<a;e++)c[b+e>>>2]|=(d[e>>>2]>>>24-8*(e%4)&255)<<24-8*((b+e)%4);else if(65535<d.length)for(e=0;e<a;e+=4)c[b+e>>>2]=d[e>>>2];else c.push.apply(c,d);this.sigBytes+=a;return this},clamp:function(){var a=this.words,c=this.sigBytes;a[c>>>2]&=4294967295<<
32-8*(c%4);a.length=h.ceil(c/4)},clone:function(){var a=m.clone.call(this);a.words=this.words.slice(0);return a},random:function(a){for(var c=[],d=0;d<a;d+=4)c.push(4294967296*h.random()|0);return new r.init(c,a)}}),l=f.enc={},k=l.Hex={stringify:function(a){var c=a.words;a=a.sigBytes;for(var d=[],b=0;b<a;b++){var e=c[b>>>2]>>>24-8*(b%4)&255;d.push((e>>>4).toString(16));d.push((e&15).toString(16))}return d.join("")},parse:function(a){for(var c=a.length,d=[],b=0;b<c;b+=2)d[b>>>3]|=parseInt(a.substr(b,
2),16)<<24-4*(b%8);return new r.init(d,c/2)}},n=l.Latin1={stringify:function(a){var c=a.words;a=a.sigBytes;for(var d=[],b=0;b<a;b++)d.push(String.fromCharCode(c[b>>>2]>>>24-8*(b%4)&255));return d.join("")},parse:function(a){for(var c=a.length,d=[],b=0;b<c;b++)d[b>>>2]|=(a.charCodeAt(b)&255)<<24-8*(b%4);return new r.init(d,c)}},j=l.Utf8={stringify:function(a){try{return decodeURIComponent(escape(n.stringify(a)))}catch(c){throw Error("Malformed UTF-8 data");}},parse:function(a){return n.parse(unescape(encodeURIComponent(a)))}},
u=g.BufferedBlockAlgorithm=m.extend({reset:function(){this._data=new r.init;this._nDataBytes=0},_append:function(a){"string"==typeof a&&(a=j.parse(a));this._data.concat(a);this._nDataBytes+=a.sigBytes},_process:function(a){var c=this._data,d=c.words,b=c.sigBytes,e=this.blockSize,f=b/(4*e),f=a?h.ceil(f):h.max((f|0)-this._minBufferSize,0);a=f*e;b=h.min(4*a,b);if(a){for(var g=0;g<a;g+=e)this._doProcessBlock(d,g);g=d.splice(0,a);c.sigBytes-=b}return new r.init(g,b)},clone:function(){var a=m.clone.call(this);
a._data=this._data.clone();return a},_minBufferSize:0});g.Hasher=u.extend({cfg:m.extend(),init:function(a){this.cfg=this.cfg.extend(a);this.reset()},reset:function(){u.reset.call(this);this._doReset()},update:function(a){this._append(a);this._process();return this},finalize:function(a){a&&this._append(a);return this._doFinalize()},blockSize:16,_createHelper:function(a){return function(c,d){return(new a.init(d)).finalize(c)}},_createHmacHelper:function(a){return function(c,d){return(new t.HMAC.init(a,
d)).finalize(c)}}});var t=f.algo={};return f}(Math);
(function(h){for(var s=CryptoJS,f=s.lib,g=f.WordArray,q=f.Hasher,f=s.algo,m=[],r=[],l=function(a){return 4294967296*(a-(a|0))|0},k=2,n=0;64>n;){var j;a:{j=k;for(var u=h.sqrt(j),t=2;t<=u;t++)if(!(j%t)){j=!1;break a}j=!0}j&&(8>n&&(m[n]=l(h.pow(k,0.5))),r[n]=l(h.pow(k,1/3)),n++);k++}var a=[],f=f.SHA256=q.extend({_doReset:function(){this._hash=new g.init(m.slice(0))},_doProcessBlock:function(c,d){for(var b=this._hash.words,e=b[0],f=b[1],g=b[2],j=b[3],h=b[4],m=b[5],n=b[6],q=b[7],p=0;64>p;p++){if(16>p)a[p]=
c[d+p]|0;else{var k=a[p-15],l=a[p-2];a[p]=((k<<25|k>>>7)^(k<<14|k>>>18)^k>>>3)+a[p-7]+((l<<15|l>>>17)^(l<<13|l>>>19)^l>>>10)+a[p-16]}k=q+((h<<26|h>>>6)^(h<<21|h>>>11)^(h<<7|h>>>25))+(h&m^~h&n)+r[p]+a[p];l=((e<<30|e>>>2)^(e<<19|e>>>13)^(e<<10|e>>>22))+(e&f^e&g^f&g);q=n;n=m;m=h;h=j+k|0;j=g;g=f;f=e;e=k+l|0}b[0]=b[0]+e|0;b[1]=b[1]+f|0;b[2]=b[2]+g|0;b[3]=b[3]+j|0;b[4]=b[4]+h|0;b[5]=b[5]+m|0;b[6]=b[6]+n|0;b[7]=b[7]+q|0},_doFinalize:function(){var a=this._data,d=a.words,b=8*this._nDataBytes,e=8*a.sigBytes;
d[e>>>5]|=128<<24-e%32;d[(e+64>>>9<<4)+14]=h.floor(b/4294967296);d[(e+64>>>9<<4)+15]=b;a.sigBytes=4*d.length;this._process();return this._hash},clone:function(){var a=q.clone.call(this);a._hash=this._hash.clone();return a}});s.SHA256=q._createHelper(f);s.HmacSHA256=q._createHmacHelper(f)})(Math);
(function(){var h=CryptoJS,s=h.enc.Utf8;h.algo.HMAC=h.lib.Base.extend({init:function(f,g){f=this._hasher=new f.init;"string"==typeof g&&(g=s.parse(g));var h=f.blockSize,m=4*h;g.sigBytes>m&&(g=f.finalize(g));g.clamp();for(var r=this._oKey=g.clone(),l=this._iKey=g.clone(),k=r.words,n=l.words,j=0;j<h;j++)k[j]^=1549556828,n[j]^=909522486;r.sigBytes=l.sigBytes=m;this.reset()},reset:function(){var f=this._hasher;f.reset();f.update(this._iKey)},update:function(f){this._hasher.update(f);return this},finalize:function(f){var g=
this._hasher;f=g.finalize(f);g.reset();return g.finalize(this._oKey.clone().concat(f))}})})();
        `)
		is(err, nil)

		test(`CryptoJS.HmacSHA256("Message", "secret");`, "aa747c502a898200f9e4fa21bac68136f886a0e27aec70ba06daf2e2a5cb5597")
	})
}
