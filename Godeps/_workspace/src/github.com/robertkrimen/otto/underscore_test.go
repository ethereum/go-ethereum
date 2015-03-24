package otto

import (
	"./terst"
	"testing"

	"github.com/robertkrimen/otto/underscore"
)

func init() {
	underscore.Disable()
}

// A persistent handle for the underscore tester
// We do not run underscore tests in parallel, so it is okay to stash globally
// (Maybe use sync.Pool in the future...)
var tester_ *_tester

// A tester for underscore: test_ => test(underscore) :)
func test_(arguments ...interface{}) (func(string, ...interface{}) Value, *_tester) {
	tester := tester_
	if tester == nil {
		tester = newTester()
		tester.underscore() // Load underscore and testing shim, etc.
		tester_ = tester
	}

	return tester.test, tester
}

func (self *_tester) underscore() {
	vm := self.vm
	_, err := vm.Run(underscore.Source())
	if err != nil {
		panic(err)
	}

	vm.Set("assert", func(call FunctionCall) Value {
		if !call.Argument(0).bool() {
			message := "Assertion failed"
			if len(call.ArgumentList) > 1 {
				message = call.ArgumentList[1].string()
			}
			t := terst.Caller().T()
			is(message, nil)
			t.Fail()
			return falseValue
		}
		return trueValue
	})

	vm.Run(`
        var templateSettings;

        function _setup() {
            templateSettings = _.clone(_.templateSettings);
        }

        function _teardown() {
            _.templateSettings = templateSettings;
        }

        function module() {
            /* Nothing happens. */
        }
    
        function equals(a, b, emit) {
            assert(a == b, emit + ", <" + a + "> != <" + b + ">");
        }
        var equal = equals;

        function notStrictEqual(a, b, emit) {
            assert(a !== b, emit);
        }

        function strictEqual(a, b, emit) {
            assert(a === b, emit);
        }

        function ok(a, emit) {
            assert(a, emit);
        }

        function raises(fn, want, emit) {
            var have, _ok = false;
            if (typeof want === "string") {
                emit = want;
                want = null;
            }
            
            try {
                fn();
            } catch(tmp) {
                have = tmp;
            }
            
            if (have) {
                if (!want) {
                    _ok = true;
                }
                else if (want instanceof RegExp) {
                    _ok = want.test(have);
                }
                else if (have instanceof want) {
                    _ok = true
                }
                else if (want.call({}, have) === true) {
                    _ok = true;
                }
            }
            
            ok(_ok, emit);
        }

        function test(name){
            _setup()
            try {
                templateSettings = _.clone(_.templateSettings);
                if (arguments.length == 3) {
                    count = 0
                    for (count = 0; count < arguments[1]; count++) {
                        arguments[2]()
                    }
                } else {
                    // For now.
                    arguments[1]()
                }
            }
            finally {
                _teardown()
            }
        }

        function deepEqual(a, b, emit) {
            // Also, for now.
            assert(_.isEqual(a, b), emit)
        }
    `)
}

func Test_underscore(t *testing.T) {
	tt(t, func() {
		test, _ := test_()

		test(`
            _.map([1, 2, 3], function(value){
                return value + 1
            })
        `, "2,3,4")

		test(`
            abc = _.find([1, 2, 3, -1], function(value) { return value == -1 })
        `, -1)

		test(`_.isEqual(1, 1)`, true)
		test(`_.isEqual([], [])`, true)
		test(`_.isEqual(['b', 'd'], ['b', 'd'])`, true)
		test(`_.isEqual(['b', 'd', 'c'], ['b', 'd', 'e'])`, false)
		test(`_.isFunction(function(){})`, true)
		test(`_.template('<p>\u2028<%= "\\u2028\\u2029" %>\u2029</p>')()`, "<p>\u2028\u2028\u2029\u2029</p>")
	})
}

// TODO Test: typeof An argument reference
// TODO Test: abc = {}; abc == Object(abc)
