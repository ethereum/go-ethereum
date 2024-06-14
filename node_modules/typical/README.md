[![view on npm](http://img.shields.io/npm/v/typical.svg)](https://www.npmjs.org/package/typical)
[![npm module downloads](http://img.shields.io/npm/dt/typical.svg)](https://www.npmjs.org/package/typical)
[![Build Status](https://travis-ci.org/75lb/typical.svg?branch=master)](https://travis-ci.org/75lb/typical)
[![Coverage Status](https://coveralls.io/repos/github/75lb/typical/badge.svg?branch=master)](https://coveralls.io/github/75lb/typical?branch=master)
[![Dependency Status](https://badgen.net/david/dep/75lb/typical)](https://david-dm.org/75lb/typical)
[![js-standard-style](https://img.shields.io/badge/code%20style-standard-brightgreen.svg)](https://github.com/feross/standard)

<a name="module_typical"></a>

## typical
For type-checking Javascript values.

**Example**  
```js
const t = require('typical')
```

* [typical](#module_typical)
    * [.isNumber(n)](#module_typical.isNumber) ⇒ <code>boolean</code>
    * [.isPlainObject(input)](#module_typical.isPlainObject) ⇒ <code>boolean</code>
    * [.isArrayLike(input)](#module_typical.isArrayLike) ⇒ <code>boolean</code>
    * [.isObject(input)](#module_typical.isObject) ⇒ <code>boolean</code>
    * [.isDefined(input)](#module_typical.isDefined) ⇒ <code>boolean</code>
    * [.isString(input)](#module_typical.isString) ⇒ <code>boolean</code>
    * [.isBoolean(input)](#module_typical.isBoolean) ⇒ <code>boolean</code>
    * [.isFunction(input)](#module_typical.isFunction) ⇒ <code>boolean</code>
    * [.isClass(input)](#module_typical.isClass) ⇒ <code>boolean</code>
    * [.isPrimitive(input)](#module_typical.isPrimitive) ⇒ <code>boolean</code>
    * [.isPromise(input)](#module_typical.isPromise) ⇒ <code>boolean</code>
    * [.isIterable(input)](#module_typical.isIterable) ⇒ <code>boolean</code>

<a name="module_typical.isNumber"></a>

### t.isNumber(n) ⇒ <code>boolean</code>
Returns true if input is a number

**Kind**: static method of [<code>typical</code>](#module_typical)  

| Param | Type | Description |
| --- | --- | --- |
| n | <code>\*</code> | the input to test |

**Example**  
```js
> t.isNumber(0)
true
> t.isNumber(1)
true
> t.isNumber(1.1)
true
> t.isNumber(0xff)
true
> t.isNumber(0644)
true
> t.isNumber(6.2e5)
true
> t.isNumber(NaN)
false
> t.isNumber(Infinity)
false
```
<a name="module_typical.isPlainObject"></a>

### t.isPlainObject(input) ⇒ <code>boolean</code>
A plain object is a simple object literal, it is not an instance of a class. Returns true if the input `typeof` is `object` and directly decends from `Object`.

**Kind**: static method of [<code>typical</code>](#module_typical)  

| Param | Type | Description |
| --- | --- | --- |
| input | <code>\*</code> | the input to test |

**Example**  
```js
> t.isPlainObject({ something: 'one' })
true
> t.isPlainObject(new Date())
false
> t.isPlainObject([ 0, 1 ])
false
> t.isPlainObject(/test/)
false
> t.isPlainObject(1)
false
> t.isPlainObject('one')
false
> t.isPlainObject(null)
false
> t.isPlainObject((function * () {})())
false
> t.isPlainObject(function * () {})
false
```
<a name="module_typical.isArrayLike"></a>

### t.isArrayLike(input) ⇒ <code>boolean</code>
An array-like value has all the properties of an array, but is not an array instance. Examples in the `arguments` object. Returns true if the input value is an object, not null and has a `length` property with a numeric value.

**Kind**: static method of [<code>typical</code>](#module_typical)  

| Param | Type | Description |
| --- | --- | --- |
| input | <code>\*</code> | the input to test |

**Example**  
```js
function sum(x, y){
    console.log(t.isArrayLike(arguments))
    // prints `true`
}
```
<a name="module_typical.isObject"></a>

### t.isObject(input) ⇒ <code>boolean</code>
returns true if the typeof input is `'object'`, but not null!

**Kind**: static method of [<code>typical</code>](#module_typical)  

| Param | Type | Description |
| --- | --- | --- |
| input | <code>\*</code> | the input to test |

<a name="module_typical.isDefined"></a>

### t.isDefined(input) ⇒ <code>boolean</code>
Returns true if the input value is defined

**Kind**: static method of [<code>typical</code>](#module_typical)  

| Param | Type | Description |
| --- | --- | --- |
| input | <code>\*</code> | the input to test |

<a name="module_typical.isString"></a>

### t.isString(input) ⇒ <code>boolean</code>
Returns true if the input value is a string

**Kind**: static method of [<code>typical</code>](#module_typical)  

| Param | Type | Description |
| --- | --- | --- |
| input | <code>\*</code> | the input to test |

<a name="module_typical.isBoolean"></a>

### t.isBoolean(input) ⇒ <code>boolean</code>
Returns true if the input value is a boolean

**Kind**: static method of [<code>typical</code>](#module_typical)  

| Param | Type | Description |
| --- | --- | --- |
| input | <code>\*</code> | the input to test |

<a name="module_typical.isFunction"></a>

### t.isFunction(input) ⇒ <code>boolean</code>
Returns true if the input value is a function

**Kind**: static method of [<code>typical</code>](#module_typical)  

| Param | Type | Description |
| --- | --- | --- |
| input | <code>\*</code> | the input to test |

<a name="module_typical.isClass"></a>

### t.isClass(input) ⇒ <code>boolean</code>
Returns true if the input value is an es2015 `class`.

**Kind**: static method of [<code>typical</code>](#module_typical)  

| Param | Type | Description |
| --- | --- | --- |
| input | <code>\*</code> | the input to test |

<a name="module_typical.isPrimitive"></a>

### t.isPrimitive(input) ⇒ <code>boolean</code>
Returns true if the input is a string, number, symbol, boolean, null or undefined value.

**Kind**: static method of [<code>typical</code>](#module_typical)  

| Param | Type | Description |
| --- | --- | --- |
| input | <code>\*</code> | the input to test |

<a name="module_typical.isPromise"></a>

### t.isPromise(input) ⇒ <code>boolean</code>
Returns true if the input is a Promise.

**Kind**: static method of [<code>typical</code>](#module_typical)  

| Param | Type | Description |
| --- | --- | --- |
| input | <code>\*</code> | the input to test |

<a name="module_typical.isIterable"></a>

### t.isIterable(input) ⇒ <code>boolean</code>
Returns true if the input is an iterable (`Map`, `Set`, `Array`, Generator etc.).

**Kind**: static method of [<code>typical</code>](#module_typical)  

| Param | Type | Description |
| --- | --- | --- |
| input | <code>\*</code> | the input to test |

**Example**  
```js
> t.isIterable('string')
true
> t.isIterable(new Map())
true
> t.isIterable([])
true
> t.isIterable((function * () {})())
true
> t.isIterable(Promise.resolve())
false
> t.isIterable(Promise)
false
> t.isIterable(true)
false
> t.isIterable({})
false
> t.isIterable(0)
false
> t.isIterable(1.1)
false
> t.isIterable(NaN)
false
> t.isIterable(Infinity)
false
> t.isIterable(function () {})
false
> t.isIterable(Date)
false
> t.isIterable()
false
> t.isIterable({ then: function () {} })
false
```

## Load anywhere

This library is compatible with Node.js, the Web and any style of module loader. It can be loaded anywhere, natively without transpilation.

Node.js:

```js
const typical = require('typical')
```

Within Node.js with ECMAScript Module support enabled:

```js
import typical from 'typical'
```

Within a modern browser ECMAScript Module:

```js
import typical from './node_modules/typical/index.mjs'
```

Old browser (adds `window.typical`):

```html
<script nomodule src="./node_modules/typical/dist/index.js"></script>
```

* * *

&copy; 2014-19 Lloyd Brookes \<75pound@gmail.com\>. Documented by [jsdoc-to-markdown](https://github.com/jsdoc2md/jsdoc-to-markdown).
