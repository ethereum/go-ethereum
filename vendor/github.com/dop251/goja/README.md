goja
====

ECMAScript 5.1(+) implementation in Go.

[![GoDoc](https://godoc.org/github.com/dop251/goja?status.svg)](https://godoc.org/github.com/dop251/goja)

Goja is an implementation of ECMAScript 5.1 in pure Go with emphasis on standard compliance and
performance.

This project was largely inspired by [otto](https://github.com/robertkrimen/otto).

Features
--------

 * Full ECMAScript 5.1 support (yes, including regex and strict mode).
 * Passes nearly all [tc39 tests](https://github.com/tc39/test262) tagged with es5id. The goal is to pass all of them. Note, the last working commit is https://github.com/tc39/test262/commit/1ba3a7c4a93fc93b3d0d7e4146f59934a896837d. The next commit made use of template strings which goja does not support.
 * Capable of running Babel, Typescript compiler and pretty much anything written in ES5.
 * Sourcemaps.
 
FAQ
---

### How fast is it?

Although it's faster than many scripting language implementations in Go I have seen 
(for example it's 6-7 times faster than otto on average) it is not a
replacement for V8 or SpiderMonkey or any other general-purpose JavaScript engine.
You can find some benchmarks [here](https://github.com/dop251/goja/issues/2).

### Why would I want to use it over a V8 wrapper?

It greatly depends on your usage scenario. If most of the work is done in javascript
(for example crypto or any other heavy calculations) you are definitely better off with V8.

If you need a scripting language that drives an engine written in Go so
you need to make frequent calls between Go and javascript passing complex data structures
then the cgo overhead may outweigh the benefits of having a faster javascript engine.

Because it's written in pure Go there are no external dependencies, it's very easy to build and it
should run on any platform supported by Go.

It gives you a much better control over execution environment so can be useful for research.

### Is it goroutine-safe?

No. An instance of goja.Runtime can only be used by a single goroutine
at a time. You can create as many instances of Runtime as you like but 
it's not possible to pass object values between runtimes.

### Where is setTimeout()?

setTimeout() assumes concurrent execution of code which requires an execution
environment, for example an event loop similar to nodejs or a browser.
There is a [separate project](https://github.com/dop251/goja_nodejs) aimed at providing some of the NodeJS functionality
and it includes an event loop.

### Can you implement (feature X from ES6 or higher)?

It's very unlikely that I will be adding new functionality any time soon. It don't have enough time 
for adding full ES6 support and I don't want to end up with something that is stuck in between ES5 and ES6.
Most of the new features are available through shims and transpilers. Goja can run Babel and any
other transpiler as long as it's written in ES5. You can even add a wrapper that will do the translation
on the fly. Sourcemaps are supported.

### How do I contribute?

Before submitting a pull request please make sure that:

- You followed ECMA standard as close as possible. If adding a new feature make sure you've read the specification,
do not just base it on a couple of examples that work fine.
- Your change does not have a significant negative impact on performance (unless it's a bugfix and it's unavoidable)
- It passes all relevant tc39 tests.

Current Status
--------------

 * API is still work in progress and is subject to change.
 * Some of the AnnexB functionality is missing.
 * No typed arrays yet.

Basic Example
-------------

```go
vm := goja.New()
v, err := vm.RunString("2 + 2")
if err != nil {
    panic(err)
}
if num := v.Export().(int64); num != 4 {
    panic(num)
}
```

Passing Values to JS
--------------------

Any Go value can be passed to JS using Runtime.ToValue() method. Primitive types (ints and uints, floats, string, bool)
are converted to the corresponding JavaScript primitives.

*func(FunctionCall) Value* is treated as a native JavaScript function.

*func(ConstructorCall) \*Object* is treated as a JavaScript constructor (see Native Constructors).

*map[string]interface{}* is converted into a host object that largely behaves like a JavaScript Object.

*[]interface{}* is converted into a host object that behaves largely like a JavaScript Array, however it's not extensible
because extending it can change the pointer so it becomes detached from the original.

**[]interface{}* is same as above, but the array becomes extensible.

A function is wrapped within a native JavaScript function. When called the arguments are automatically converted to
the appropriate Go types. If conversion is not possible, a TypeError is thrown.

A slice type is converted into a generic reflect based host object that behaves similar to an unexpandable Array.

A map type with numeric or string keys and no methods is converted into a host object where properties are map keys.

A map type with methods is converted into a host object where properties are method names,
the map values are not accessible. This is to avoid ambiguity between m\["Property"\] and m.Property.

Any other type is converted to a generic reflect based host object. Depending on the underlying type it behaves similar
to a Number, String, Boolean or Object.

Note that these conversions wrap the original value which means any changes made inside JS
are reflected on the value and calling Export() returns the original value. This applies to all
reflect based types.

Exporting Values from JS
------------------------

A JS value can be exported into its default Go representation using Value.Export() method.

Alternatively it can be exported into a specific Go variable using Runtime.ExportTo() method.

Native Constructors
-------------------

In order to implement a constructor function in Go:
```go
func MyObject(call goja.ConstructorCall) *Object {
    // call.This contains the newly created object as per http://www.ecma-international.org/ecma-262/5.1/index.html#sec-13.2.2
    // call.Arguments contain arguments passed to the function

    call.This.Set("method", method)

    //...

    // If return value is a non-nil *Object, it will be used instead of call.This
    // This way it is possible to return a Go struct or a map converted
    // into goja.Value using runtime.ToValue(), however in this case
    // instanceof will not work as expected.
    return nil
}

runtime.Set("MyObject", MyObject)

```

Then it can be used in JS as follows:

```js
var o = new MyObject(arg);
var o1 = MyObject(arg); // same thing
o instanceof MyObject && o1 instanceof MyObject; // true
```

Regular Expressions
-------------------

Goja uses the embedded Go regexp library where possible, otherwise it falls back to [regexp2](https://github.com/dlclark/regexp2).

Exceptions
----------

Any exception thrown in JavaScript is returned as an error of type *Exception. It is possible to extract the value thrown
by using the Value() method:

```go
vm := New()
_, err := vm.RunString(`

throw("Test");

`)

if jserr, ok := err.(*Exception); ok {
    if jserr.Value().Export() != "Test" {
        panic("wrong value")
    }
} else {
    panic("wrong type")
}
```

If a native Go function panics with a Value, it is thrown as a Javascript exception (and therefore can be caught):

```go
var vm *Runtime

func Test() {
    panic(vm.ToValue("Error"))
}

vm = New()
vm.Set("Test", Test)
_, err := vm.RunString(`

try {
    Test();
} catch(e) {
    if (e !== "Error") {
        throw e;
    }
}

`)

if err != nil {
    panic(err)
}
```

Interrupting
------------

```go
func TestInterrupt(t *testing.T) {
    const SCRIPT = `
    var i = 0;
    for (;;) {
        i++;
    }
    `

    vm := New()
    time.AfterFunc(200 * time.Millisecond, func() {
        vm.Interrupt("halt")
    })

    _, err := vm.RunString(SCRIPT)
    if err == nil {
        t.Fatal("Err is nil")
    }
    // err is of type *InterruptError and its Value() method returns whatever has been passed to vm.Interrupt()
}
```

NodeJS Compatibility
--------------------

There is a [separate project](https://github.com/dop251/goja_nodejs) aimed at providing some of the NodeJS functionality.
