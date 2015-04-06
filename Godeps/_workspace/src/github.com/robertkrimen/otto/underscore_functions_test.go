package otto

import (
	"testing"
)

// bind
func Test_underscore_functions_0(t *testing.T) {
	tt(t, func() {
		test, _ := test_()

		test(`
  test("bind", function() {
    var context = {name : 'moe'};
    var func = function(arg) { return "name: " + (this.name || arg); };
    var bound = _.bind(func, context);
    equal(bound(), 'name: moe', 'can bind a function to a context');

    bound = _(func).bind(context);
    equal(bound(), 'name: moe', 'can do OO-style binding');

    bound = _.bind(func, null, 'curly');
    equal(bound(), 'name: curly', 'can bind without specifying a context');

    func = function(salutation, name) { return salutation + ': ' + name; };
    func = _.bind(func, this, 'hello');
    equal(func('moe'), 'hello: moe', 'the function was partially applied in advance');

    func = _.bind(func, this, 'curly');
    equal(func(), 'hello: curly', 'the function was completely applied in advance');

    func = function(salutation, firstname, lastname) { return salutation + ': ' + firstname + ' ' + lastname; };
    func = _.bind(func, this, 'hello', 'moe', 'curly');
    equal(func(), 'hello: moe curly', 'the function was partially applied in advance and can accept multiple arguments');

    func = function(context, message) { equal(this, context, message); };
    _.bind(func, 0, 0, 'can bind a function to <0>')();
    _.bind(func, '', '', 'can bind a function to an empty string')();
    _.bind(func, false, false, 'can bind a function to <false>')();

    // These tests are only meaningful when using a browser without a native bind function
    // To test this with a modern browser, set underscore's nativeBind to undefined
    var F = function () { return this; };
    var Boundf = _.bind(F, {hello: "moe curly"});
    equal(Boundf().hello, "moe curly", "When called without the new operator, it's OK to be bound to the context");
  });
        `)
	})
}

// partial
func Test_underscore_functions_1(t *testing.T) {
	tt(t, func() {
		test, _ := test_()

		test(`
  test("partial", function() {
    var obj = {name: 'moe'};
    var func = function() { return this.name + ' ' + _.toArray(arguments).join(' '); };

    obj.func = _.partial(func, 'a', 'b');
    equal(obj.func('c', 'd'), 'moe a b c d', 'can partially apply');
  });
        `)
	})
}

// bindAll
func Test_underscore_functions_2(t *testing.T) {
	tt(t, func() {
		test, _ := test_()

		test(`
  test("bindAll", function() {
    var curly = {name : 'curly'}, moe = {
      name    : 'moe',
      getName : function() { return 'name: ' + this.name; },
      sayHi   : function() { return 'hi: ' + this.name; }
    };
    curly.getName = moe.getName;
    _.bindAll(moe, 'getName', 'sayHi');
    curly.sayHi = moe.sayHi;
    equal(curly.getName(), 'name: curly', 'unbound function is bound to current object');
    equal(curly.sayHi(), 'hi: moe', 'bound function is still bound to original object');

    curly = {name : 'curly'};
    moe = {
      name    : 'moe',
      getName : function() { return 'name: ' + this.name; },
      sayHi   : function() { return 'hi: ' + this.name; }
    };
    _.bindAll(moe);
    curly.sayHi = moe.sayHi;
    equal(curly.sayHi(), 'hi: moe', 'calling bindAll with no arguments binds all functions to the object');
  });
        `)
	})
}

// memoize
func Test_underscore_functions_3(t *testing.T) {
	tt(t, func() {
		test, _ := test_()

		test(`
  test("memoize", function() {
    var fib = function(n) {
      return n < 2 ? n : fib(n - 1) + fib(n - 2);
    };
    var fastFib = _.memoize(fib);
    equal(fib(10), 55, 'a memoized version of fibonacci produces identical results');
    equal(fastFib(10), 55, 'a memoized version of fibonacci produces identical results');

    var o = function(str) {
      return str;
    };
    var fastO = _.memoize(o);
    equal(o('toString'), 'toString', 'checks hasOwnProperty');
    equal(fastO('toString'), 'toString', 'checks hasOwnProperty');
  });
        `)
	})
}

// once
func Test_underscore_functions_4(t *testing.T) {
	tt(t, func() {
		test, _ := test_()

		test(`
  test("once", function() {
    var num = 0;
    var increment = _.once(function(){ num++; });
    increment();
    increment();
    equal(num, 1);
  });
        `)
	})
}

// wrap
func Test_underscore_functions_5(t *testing.T) {
	tt(t, func() {
		test, _ := test_()

		test(`
  test("wrap", function() {
    var greet = function(name){ return "hi: " + name; };
    var backwards = _.wrap(greet, function(func, name){ return func(name) + ' ' + name.split('').reverse().join(''); });
    equal(backwards('moe'), 'hi: moe eom', 'wrapped the saluation function');

    var inner = function(){ return "Hello "; };
    var obj   = {name : "Moe"};
    obj.hi    = _.wrap(inner, function(fn){ return fn() + this.name; });
    equal(obj.hi(), "Hello Moe");

    var noop    = function(){};
    var wrapped = _.wrap(noop, function(fn){ return Array.prototype.slice.call(arguments, 0); });
    var ret     = wrapped(['whats', 'your'], 'vector', 'victor');
    deepEqual(ret, [noop, ['whats', 'your'], 'vector', 'victor']);
  });
        `)
	})
}

// compose
func Test_underscore_functions_6(t *testing.T) {
	tt(t, func() {
		test, _ := test_()

		test(`
  test("compose", function() {
    var greet = function(name){ return "hi: " + name; };
    var exclaim = function(sentence){ return sentence + '!'; };
    var composed = _.compose(exclaim, greet);
    equal(composed('moe'), 'hi: moe!', 'can compose a function that takes another');

    composed = _.compose(greet, exclaim);
    equal(composed('moe'), 'hi: moe!', 'in this case, the functions are also commutative');
  });
        `)
	})
}

// after
func Test_underscore_functions_7(t *testing.T) {
	tt(t, func() {
		test, _ := test_()

		test(`
  test("after", function() {
    var testAfter = function(afterAmount, timesCalled) {
      var afterCalled = 0;
      var after = _.after(afterAmount, function() {
        afterCalled++;
      });
      while (timesCalled--) after();
      return afterCalled;
    };

    equal(testAfter(5, 5), 1, "after(N) should fire after being called N times");
    equal(testAfter(5, 4), 0, "after(N) should not fire unless called N times");
    equal(testAfter(0, 0), 1, "after(0) should fire immediately");
  });
        `)
	})
}
