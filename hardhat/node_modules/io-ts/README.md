[![build status](https://img.shields.io/travis/gcanti/io-ts/master.svg?style=flat-square)](https://travis-ci.org/gcanti/io-ts)
[![dependency status](https://img.shields.io/david/gcanti/io-ts.svg?style=flat-square)](https://david-dm.org/gcanti/io-ts)
![npm downloads](https://img.shields.io/npm/dm/io-ts.svg)
[![Minified Size](https://badgen.net/bundlephobia/minzip/io-ts)](https://bundlephobia.com/result?p=io-ts)

Table of contents

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->

- [Installation](#installation)
- [The idea](#the-idea)
- [TypeScript compatibility](#typescript-compatibility)
- [Error reporters](#error-reporters)
- [Custom error messages](#custom-error-messages)
- [Community](#community)
- [TypeScript integration](#typescript-integration)
- [Implemented types / combinators](#implemented-types--combinators)
- [Recursive types](#recursive-types)
  - [Mutually recursive types](#mutually-recursive-types)
- [Branded types / Refinements](#branded-types--refinements)
- [Exact types](#exact-types)
- [Mixing required and optional props](#mixing-required-and-optional-props)
- [Custom types](#custom-types)
- [Generic Types](#generic-types)
- [Piping](#piping)
- [Tips and Tricks](#tips-and-tricks)
  - [Is there a way to turn the checks off in production code?](#is-there-a-way-to-turn-the-checks-off-in-production-code)
  - [Union of string literals](#union-of-string-literals)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# Installation

To install the stable version:

```sh
npm i io-ts
```

# The idea

Blog post: ["Typescript and validations at runtime boundaries"](https://lorefnon.tech/2018/03/25/typescript-and-validations-at-runtime-boundaries/) by [@lorefnon](https://github.com/lorefnon)

A value of type `Type<A, O, I>` (called "codec") is the runtime representation of the static type `A`.

Also a codec can

- decode inputs of type `I` (through `decode`)
- encode outputs of type `O` (through `encode`)
- be used as a custom type guard (through `is`)

```ts
class Type<A, O, I> {
  readonly _A: A
  readonly _O: O
  readonly _I: I
  constructor(
    /** a unique name for this codec */
    readonly name: string,
    /** a custom type guard */
    readonly is: (u: unknown) => u is A,
    /** succeeds if a value of type I can be decoded to a value of type A */
    readonly validate: (input: I, context: Context) => Either<Errors, A>,
    /** converts a value of type A to a value of type O */
    readonly encode: (a: A) => O
  ) {}
  /** a version of `validate` with a default context */
  decode(i: I): Either<Errors, A>
}
```

Note. The `Either` type is defined in [fp-ts](https://github.com/gcanti/fp-ts), a library containing implementations of
common algebraic types in TypeScript.

**Example**

A codec representing `string` can be defined as

```ts
import * as t from 'io-ts'

const isString = (u: unknown): u is string => typeof u === 'string'

const string = new t.Type<string, string, unknown>(
  'string',
  isString,
  (u, c) => (isString(u) ? t.success(u) : t.failure(u, c)),
  t.identity
)
```

A codec can be used to validate an object in memory (for example an API payload)

```ts
import * as t from 'io-ts'

const User = t.type({
  userId: t.number,
  name: t.string
})

// validation succeeded
User.decode(JSON.parse('{"userId":1,"name":"Giulio"}')) // => Right({ userId: 1, name: "Giulio" })

// validation failed
User.decode(JSON.parse('{"name":"Giulio"}')) // => Left([...])
```

# TypeScript compatibility

The stable version is tested against TypeScript 3.5.2

| io-ts version | required TypeScript version |
| ------------- | --------------------------- |
| 1.6.x+        | 3.2.2+                      |
| 1.5.3         | 3.0.1+                      |
| 1.5.2-        | 2.7.2+                      |

**Note**. This library is conceived, tested and is supposed to be consumed by TypeScript with the `strict` flag turned on.

**Note**. If you are running `< typescript@3.0.1` you have to polyfill `unknown`.

You can use [unknown-ts](https://github.com/gcanti/unknown-ts) as a polyfill.

# Error reporters

A reporter implements the following interface

```ts
interface Reporter<A> {
  report: (validation: Validation<any>) => A
}
```

This package exports a default `PathReporter` reporter

Example

```ts
import { PathReporter } from 'io-ts/lib/PathReporter'

const result = User.decode({ name: 'Giulio' })

console.log(PathReporter.report(result))
// => [ 'Invalid value undefined supplied to : { userId: number, name: string }/userId: number' ]
```

You can define your own reporter. `Errors` has the following type

```ts
interface ContextEntry {
  readonly key: string
  readonly type: Decoder<any, any>
}

interface Context extends ReadonlyArray<ContextEntry> {}

interface ValidationError {
  readonly value: unknown
  readonly context: Context
}

interface Errors extends Array<ValidationError> {}
```

Example

```ts
const getPaths = <A>(v: t.Validation<A>): Array<string> => {
  return v.fold(errors => errors.map(error => error.context.map(({ key }) => key).join('.')), () => ['no errors'])
}

console.log(getPaths(User.decode({}))) // => [ '.userId', '.name' ]
```

# Custom error messages

You can set your own error message by providing a `message` argument to `failure`

Example

```ts
const NumberFromString = new t.Type<number, string, unknown>(
  'NumberFromString',
  t.number.is,
  (u, c) =>
    t.string.validate(u, c).chain(s => {
      const n = +s
      return isNaN(n) ? t.failure(u, c, 'cannot parse to a number') : t.success(n)
    }),
  String
)

console.log(PathReporter.report(NumberFromString.decode('a')))
// => ['cannot parse to a number']
```

You can also use the [`withMessage`](https://gcanti.github.io/io-ts-types/modules/withMessage.ts.html) helper from [io-ts-types](https://github.com/gcanti/io-ts-types)

# Community

- [io-ts-types](https://github.com/gcanti/io-ts-types) - A collection of codecs and combinators for use with
  io-ts
- [io-ts-reporters](https://github.com/OliverJAsh/io-ts-reporters) - Error reporters for io-ts
- [geojson-iots](https://github.com/pierremarc/geojson-iots) - codecs for GeoJSON as defined in rfc7946 made with
  io-ts
- [graphql-to-io-ts](https://github.com/micimize/graphql-to-io-ts) - Generate typescript and cooresponding io-ts types from a graphql
  schema
- [io-ts-promise](https://github.com/aeirola/io-ts-promise) - Convenience library for using io-ts with promise-based APIs

# TypeScript integration

codecs can be inspected

![instrospection](images/introspection.png)

This library uses TypeScript extensively. Its API is defined in a way which automatically infers types for produced
values

![inference](images/inference.png)

Note that the type annotation isn't needed, TypeScript infers the type automatically based on a schema (and comments are preserved).

Static types can be extracted from codecs using the `TypeOf` operator

```ts
type User = t.TypeOf<typeof User>

// same as
type User = {
  userId: number
  name: string
}
```

# Implemented types / combinators

| Type                        | TypeScript                  | codec / combinator                                                   |
| --------------------------- | --------------------------- | -------------------------------------------------------------------- |
| null                        | `null`                      | `t.null` or `t.nullType`                                             |
| undefined                   | `undefined`                 | `t.undefined`                                                        |
| void                        | `void`                      | `t.void` or `t.voidType`                                             |
| string                      | `string`                    | `t.string`                                                           |
| number                      | `number`                    | `t.number`                                                           |
| boolean                     | `boolean`                   | `t.boolean`                                                          |
| unknown                     | `unknown`                   | `t.unknown`                                                          |
| never                       | `never`                     | `t.never`                                                            |
| object                      | `object`                    | `t.object`                                                           |
| array of unknown            | `Array<unknown>`            | `t.UnknownArray`                                                     |
| array of type               | `Array<A>`                  | `t.array(A)`                                                         |
| record of unknown           | `Record<string, unknown>`   | `t.UnknownRecord`                                                    |
| record of type              | `Record<K, A>`              | `t.record(K, A)`                                                     |
| function                    | `Function`                  | `t.Function`                                                         |
| literal                     | `'s'`                       | `t.literal('s')`                                                     |
| partial                     | `Partial<{ name: string }>` | `t.partial({ name: t.string })`                                      |
| readonly                    | `Readonly<A>`               | `t.readonly(A)`                                                      |
| readonly array              | `ReadonlyArray<A>`          | `t.readonlyArray(A)`                                                 |
| type alias                  | `type T = { name: A }`      | `t.type({ name: A })`                                                |
| tuple                       | `[ A, B ]`                  | `t.tuple([ A, B ])`                                                  |
| union                       | `A \| B`                    | `t.union([ A, B ])`                                                  |
| intersection                | `A & B`                     | `t.intersection([ A, B ])`                                           |
| keyof                       | `keyof M`                   | `t.keyof(M)` (**only supports string keys**)                         |
| recursive types             | ✘                           | `t.recursion(name, definition)`                                      |
| branded types / refinements | ✘                           | `t.brand(A, predicate, brand)`                                       |
| integer                     | ✘                           | `t.Int` (built-in branded codec)                                     |
| exact types                 | ✘                           | `t.exact(type)`                                                      |
| strict                      | ✘                           | `t.strict({ name: A })` (an alias of `t.exact(t.type({ name: A })))` |

# Recursive types

Recursive types can't be inferred by TypeScript so you must provide the static type as a hint

```ts
interface Category {
  name: string
  categories: Array<Category>
}

const Category: t.Type<Category> = t.recursion('Category', () =>
  t.type({
    name: t.string,
    categories: t.array(Category)
  })
)
```

## Mutually recursive types

```ts
interface Foo {
  type: 'Foo'
  b: Bar | undefined
}

interface Bar {
  type: 'Bar'
  a: Foo | undefined
}

const Foo: t.Type<Foo> = t.recursion('Foo', () =>
  t.interface({
    type: t.literal('Foo'),
    b: t.union([Bar, t.undefined])
  })
)

const Bar: t.Type<Bar> = t.recursion('Bar', () =>
  t.interface({
    type: t.literal('Bar'),
    a: t.union([Foo, t.undefined])
  })
)
```

# Branded types / Refinements

You can brand / refine a codec (_any_ codec) using the `brand` combinator

```ts
// a unique brand for positive numbers
interface PositiveBrand {
  readonly Positive: unique symbol // use `unique symbol` here to ensure uniqueness across modules / packages
}

const Positive = t.brand(
  t.number, // a codec representing the type to be refined
  (n): n is t.Branded<number, PositiveBrand> => n >= 0, // a custom type guard using the build-in helper `Branded`
  'Positive' // the name must match the readonly field in the brand
)

type Positive = t.TypeOf<typeof Positive>
/*
same as
type Positive = number & t.Brand<PositiveBrand>
*/
```

Branded codecs can be merged with `t.intersection`

```ts
// t.Int is a built-in branded codec
const PositiveInt = t.intersection([t.Int, Positive])

type PositiveInt = t.TypeOf<typeof PositiveInt>
/*
same as
type PositiveInt = number & t.Brand<t.IntBrand> & t.Brand<PositiveBrand>
*/
```

# Exact types

You can make a codec exact (which means that additional properties are stripped) using the `exact` combinator

```ts
const ExactUser = t.exact(User)

User.decode({ userId: 1, name: 'Giulio', age: 45 }) // ok, result is right({ userId: 1, name: 'Giulio', age: 45 })
ExactUser.decode({ userId: 1, name: 'Giulio', age: 43 }) // ok but result is right({ userId: 1, name: 'Giulio' })
```

# Mixing required and optional props

You can mix required and optional props using an intersection

```ts
const A = t.type({
  foo: t.string
})

const B = t.partial({
  bar: t.number
})

const C = t.intersection([A, B])

type C = t.TypeOf<typeof C>

// same as
type C = {
  foo: string
} & {
  bar?: number | undefined
}
```

You can apply `partial` to an already defined codec via its `props` field

```ts
const PartialUser = t.partial(User.props)

type PartialUser = t.TypeOf<typeof PartialUser>

// same as
type PartialUser = {
  name?: string
  age?: number
}
```

# Custom types

You can define your own types. Let's see an example

```ts
// represents a Date from an ISO string
const DateFromString = new t.Type<Date, string, unknown>(
  'DateFromString',
  (u): u is Date => u instanceof Date,
  (u, c) =>
    t.string.validate(u, c).chain(s => {
      const d = new Date(s)
      return isNaN(d.getTime()) ? t.failure(u, c) : t.success(d)
    }),
  a => a.toISOString()
)

const s = new Date(1973, 10, 30).toISOString()

DateFromString.decode(s)
// right(new Date('1973-11-29T23:00:00.000Z'))

DateFromString.decode('foo')
// left(errors...)
```

Note that you can **deserialize** while validating.

# Generic Types

Polymorphic codecs are represented using functions.
For example, the following typescript:

```ts
interface ResponseBody<T> {
  result: T
  _links: Links
}
interface Links {
  previous: string
  next: string
}
```

Would be:

```ts
// t.Mixed = t.Type<any, any, unknown>
const ResponseBody = <C extends t.Mixed>(codec: C) =>
  t.interface({
    result: codec,
    _links: Links
  })

const Links = t.interface({
  previous: t.string,
  next: t.string
})
```

And used like:

```ts
const UserModel = t.type({
  name: t.string
})

functionThatRequiresRuntimeType(ResponseBody(t.array(UserModel)), ...params)
```

# Piping

You can pipe two codecs if their type parameters do align

```ts
const NumberCodec = new t.Type<number, string, string>(
  'NumberCodec',
  t.number.is,
  (s, c) => {
    const n = parseFloat(s)
    return isNaN(n) ? t.failure(s, c) : t.success(n)
  },
  String
)

const NumberFromString = t.string.pipe(
  NumberCodec,
  'NumberFromString'
)
```

# Tips and Tricks

## Is there a way to turn the checks off in production code?

No, however you can define your own logic for that (if you _really_ trust the input)

```ts
import * as t from 'io-ts'
import { Either, right } from 'fp-ts/lib/Either'

const { NODE_ENV } = process.env

export function unsafeDecode<A, O, I>(value: I, codec: t.Type<A, O, I>): Either<t.Errors, A> {
  if (NODE_ENV !== 'production' || codec.encode !== t.identity) {
    return codec.decode(value)
  } else {
    // unsafe cast
    return right(value as any)
  }
}

// or...

import { failure } from 'io-ts/lib/PathReporter'

export function unsafeGet<A, O, I>(value: I, codec: t.Type<A, O, I>): A {
  if (NODE_ENV !== 'production' || type.encode !== t.identity) {
    return codec.decode(value).getOrElseL(errors => {
      throw new Error(failure(errors).join('\n'))
    })
  } else {
    // unsafe cast
    return value as any
  }
}
```

## Union of string literals

Use `keyof` instead of `union` when defining a union of string literals

```ts
const Bad = t.union([
  t.literal('foo'),
  t.literal('bar'),
  t.literal('baz')
  // etc...
])

const Good = t.keyof({
  foo: null,
  bar: null,
  baz: null
  // etc...
})
```

Benefits

- unique check for free
- better performance, `O(log(n))` vs `O(n)`

Beware that `keyof` is designed to work with objects containing string keys. If you intend to define a numbers enumeration, you have to use an `union` of number literals :

```ts
const HttpCode = t.union([
  t.literal(200),
  t.literal(201),
  t.literal(202)
  // etc...
])
```
