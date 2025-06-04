[![Build Status](https://github.com/Yomguithereal/obliterator/workflows/Tests/badge.svg)](https://github.com/Yomguithereal/obliterator/actions)

# Obliterator

Obliterator is a dead simple JavaScript/TypeScript library providing miscellaneous higher-order iterator/iterable functions such as combining two or more iterators into a single one.

Note that when possible, `obliterator` also consider sequences such as arrays, strings etc. as valid iterables (although they are not proper ES6 iterables values), for convenience.

# Installation

```
npm install --save obliterator
```

Note that `obliterator` comes along with its TypeScript declarations.

# Usage

## Summary

_Classes_

- [Iterator](#iterator)

_Functions_

- [chain](#chain)
- [combinations](#combinations)
- [consume](#consume)
- [every](#every)
- [filter](#filter)
- [find](#find)
- [forEach](#foreach)
- [forEachWithNullKeys](#foreachwithnullkeys)
- [includes](#includes)
- [iter](#iter)
- [map](#map)
- [match](#match)
- [permutations](#permutations)
- [powerSet](#powerSet)
- [some](#some)
- [split](#split)
- [take](#take)

## Iterator

A handy Iterator class easily usable with ES2015's `for ... of` loop constructs & spread operator.

```js
import Iterator from 'obliterator/iterator';
// Or
import {Iterator} from 'obliterator';

const iterator = new Iterator(function () {
  // Define what the `next` function does
  return {done: false, value: 34};
});

// Checking that the given value is an iterator (native or else)
Iterator.is(value);

// Creating an empty iterator
const emptyIterator = Iterator.empty();

// Creating a simple iterator from a single value
const simpleIterator = Iterator.of(34);

// Creating a simple iterator from multiple values
const multipleIterator = Iterator.of(1, 2, 3);
```

## chain

Variadic function chaining all the given iterable-like values.

```js
import chain from 'obliterator/chain';
// Or
import {chain} from 'obliterator';

const set1 = new Set('a');
const set2 = new Set('bc');

const chained = chain(set1.values(), set2);

chained.next();
>>> {done: false, value: 'a'}
chained.next();
>>> {done: false, value: 'b'}
```

## combinations

Returns an iterator of combinations of the given array and of the given size.

Note that for performance reasons, the yielded combination is always the same object.

```js
import combinations from 'obliterator/combinations';
// Or
import {combinations} from 'obliterator';

const iterator = combinations(['A', 'B', 'C', 'D'], 2);

iterator.next().value;
>>> ['A', 'B']
iterator.next().value;
>>> ['A', 'C']
```

## consume

Function consuming the given iterator fully or for n steps.

```js
import consume from 'obliterator/consume';
// Or
import {consume} from 'obliterator';

const set = new Set([1, 2, 3]);

// Consuming the whole iterator
let iterator = set.values();
consume(iterator);
iterator.next().done >>> true;

// Consuming n steps
let iterator = set.values();
consume(iterator, 2);
iterator.next().value >>> 3;
```

## every

Function returning whether all items of an iterable-like match the given predicate function.

```js
import every from 'obliterator/every';
// Or
import {every} from 'obliterator';

every([2, 4, 6], n => n % 2 === 0);
>>> true

every([1, 2, 3], n => n % 2 === 0);
>>> false
```

## filter

Function returning an iterator filtering another one's values using the given predicate function.

```js
import filter from 'obliterator/filter';
// Or
import {filter} from 'obliterator';

const set = new Set([1, 2, 3, 4, 5]);

const even = x => x % 2 === 0;

const iterator = filter(set.values(), even);

iterator.next().value >>> 2;
iterator.next().value >>> 4;
```

## find

Function returning the next item matching given predicate function in an iterable-like.

```js
import find from 'obliterator/find';
// Or
import {find} from 'obliterator';

const set = new Set([1, 2, 3, 4, 5]);

const even = x => x % 2 === 0;

const values = set.values();

find(values, even);
>>> 2

find(values, even);
>>> 4

find(values, even);
>>> undefined
```

## forEach

Function able to iterate over almost any JavaScript iterable value using a callback.

Supported values range from arrays, typed arrays, sets, maps, objects, strings, arguments, iterators, arbitrary iterables etc.

```js
import forEach from 'obliterator/foreach';
// Or
import {forEach} from 'obliterator';

const set = new Set(['apple', 'banana']);

forEach(set.values(), (value, i) => {
  console.log(i, value);
});

// Iterating over a string
forEach('abc', (char, i) => ...);

// Iterating over a map
forEach(map, (value, key) => ...);
```

## forEachWithNullKeys

Variant of [forEach](#foreach) one can use to iterate over mixed values but with the twist that iterables without proper keys (lists, sets etc.), will yield `null` instead of an index key.

Supported values range from arrays, typed arrays, sets, maps, objects, strings, arguments, iterators, arbitrary iterables etc.

```js
import {forEachWithNullKeys} from 'obliterator/foreach';

const set = new Set(['apple', 'banana']);

forEach(set, (value, key) => {
  console.log(key, value);
});
>>> null, 'apple'
>>> null, 'banana'
```

## includes

Function returning whether the given value can be found in given iterable-like.

```js
import {includes} from 'obliterator';
// Or
import includes from 'obliterator/includes';

includes([1, 2, 3], 3);
>>> true;

includes('test', 'a');
>>> false;
```

## iter

Function casting any iterable-like value to a proper iterator. Will throw an error if the given value cannot be cast as an iterator.

```js
import {iter} from 'obliterator';
// Or
import iter from 'obliterator/iter';

iter('test');
iter(new Set([1, 2, 3]));

// This will throw:
iter(null);
```

## map

Function returning an iterator mapping another one's values using the given function.

```js
import map from 'obliterator/map';
// Or
import {map} from 'obliterator';

const set = new Set([1, 2, 3, 4, 5]);

const triple = x => x * 3;

const iterator = map(set.values(), triple);

iterator.next().value >>> 3;
iterator.next().value >>> 6;
```

## match

Function returning an iterator over the matches of a given regex applied to the target string.

```js
import match from 'obliterator/match';
// Or
import {match} from 'obliterator';

const iterator = match(/t/, 'test');

iterator.next().value.index >>> 0;
iterator.next().value.index >>> 3;
```

## permutations

Returns an iterator of permutations of the given array and of the given size.

Note that for performance reasons, the yielded permutation is always the same object.

```js
import permutations from 'obliterator/permutations';
// Or
import {permutations} from 'obliterator';

let iterator = permutations([1, 2, 3]);

iterator.next().value
>>> [1, 2, 3]
iterator.next().value
>>> [1, 3, 2]

iterator = permutations(['A', 'B', 'C', 'D'], 2);

iterator.next().value;
>>> ['A', 'B']
iterator.next().value;
>>> ['A', 'C']
```

## powerSet

Returns an iterator of sets composing the power set of the given array.

```js
import powerSet from 'obliterator/power-set';
// Or
import {powerSet} from 'obliterator';

const iterator = powerSet(['A', 'B', 'C']);

iterator.next().value;
>>> []
iterator.next().value;
>>> ['A']
```

## some

Returns whether the given iterable-like has some item matching the given predicate function.

```js
import some from 'obliterator/some';
// Or
import {some} from 'obliterator';

some(new Set([1, 2, 3]), n => n % 2 === 0);
>>> true

some('test', c => c === 'a');
>>> false
```

## split

Returns an iterator over the splits of the target string, according to the given RegExp pattern.

```js
import split from 'obliterator/split';
// Or
import {split} from 'obliterator';

const iterator = split(/;/g, 'hello;world;super');

iterator.next().value;
>>> 'hello'
iterator.next().value;
>>> 'world'
```

## take

Function taking values from given iterator and returning them in an array.

```js
import take from 'obliterator/take';
// Or
import {take} from 'obliterator';

const set = new Set([1, 2, 3]);

// To take n values from the iterator
take(set.values(), 2);
>>> [1, 2]

// To convert the full iterator into an array
take(set.values());
>>> [1, 2, 3]
```

# Contribution

Contributions are obviously welcome. Please be sure to lint the code & add the relevant unit tests before submitting any PR.

```
git clone git@github.com:Yomguithereal/obliterator.git
cd obliterator
npm install

# To lint the code
npm run lint

# To run the unit tests
npm test
```

# License

[MIT](LICENSE.txt)
