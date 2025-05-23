<h3 align="center">
  <a href="https://gcanti.github.io/fp-ts/">
    <img src="./docs/fp-ts-logo.png">
  </a>
</h3>

<p align="center">
Functional programming in TypeScript
</p>

<p align="center">
  <a href="https://travis-ci.org/gcanti/fp-ts">
    <img src="https://img.shields.io/travis/gcanti/fp-ts/master.svg?style=flat-square" alt="build status" height="20">
  </a>
  <a href="https://david-dm.org/gcanti/fp-ts">
    <img src="https://img.shields.io/david/gcanti/fp-ts.svg?style=flat-square" alt="dependency status" height="20">
  </a>
  <a href="https://www.npmjs.com/package/fp-ts">
    <img src="https://img.shields.io/npm/dm/fp-ts.svg" alt="npm downloads" height="20">
  </a>
</p>

# Typed functional programming in TypeScript

`fp-ts` is a library for _typed functional programming_ in TypeScript.

`fp-ts` aims to allow developers to use _popular patterns and abstractions_ that are available in most functional languages. For this, it includes the most popular data types, type classes and abstractions such as [Option](https://gcanti.github.io/fp-ts/modules/Option.ts), [Either](https://gcanti.github.io/fp-ts/modules/Either.ts), [IO](https://gcanti.github.io/fp-ts/modules/IO.ts), [Task](https://gcanti.github.io/fp-ts/modules/Task.ts), [Functor](https://gcanti.github.io/fp-ts/modules/Functor.ts), [Applicative](https://gcanti.github.io/fp-ts/modules/Applicative.ts), [Monad](https://gcanti.github.io/fp-ts/modules/Monad.ts) to empower users to write pure FP apps and libraries built atop higher order abstractions.

A distinctive feature of `fp-ts` with respect to other functional libraries is its implementation of [Higher Kinded Types](<https://en.wikipedia.org/wiki/Kind_(type_theory)>), which TypeScript doesn't support natively.

**Inspired by**

- [Haskell](https://haskell-lang.org)
- [PureScript](http://www.purescript.org)
- [Scala](https://www.scala-lang.org/)

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->

**Table of contents**

- [Installation](#installation)
  - [TypeScript compatibility](#typescript-compatibility)
- [Documentation](#documentation)
- [Development](#development)
- [License](#license)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# Installation

To install the stable version:

```
npm install fp-ts
```

Make sure to always have a single version of `fp-ts` installed in your project. Multiple versions are known to cause `tsc` to hang during compilation. You can check the versions currently installed using `npm ls fp-ts` (make sure there's a single version and all the others are marked as `deduped`).

## Consuming fp-ts

Most examples will use the following import syntax:

```ts
import { Option, some, none } from 'fp-ts/lib/Option'
```

This will give you the widest support across tools, as you will be importing CommonJS modules.

If you use a bundler such as webpack or Rollup that supports tree-shaking, you can take advantage of this by opting to import ECMAScript modules instead:

```ts
import { Option, some, none } from 'fp-ts/es6/Option'
```

Note that there are caveats such as some tools (e.g. Jest) not supporting ES6 module syntax natively yet.

## TypeScript compatibility

**Strictness** – This library is conceived, tested and is supposed to be consumed by TypeScript with the `strict` flag turned on.

| `fp-ts` version | required `typescript` version |
| --------------- | ----------------------------- |
| 1.15.x+         | 3.1+                          |
| <= 1.14.4       | 2.8+ (\*)                     |

(\*) If you are running `< typescript@3.0.1` you have to polyfill the `unknown` type. You can use [unknown-ts](https://github.com/gcanti/unknown-ts) as a polyfill.

# Documentation

- [Docs](https://gcanti.github.io/fp-ts)
- [Tutorial](https://gcanti.github.io/fp-ts/basics/)
- [Ecosystem](https://gcanti.github.io/fp-ts/introduction/ecosystem)
- [API Reference](https://gcanti.github.io/fp-ts/modules/)

# Development

- [Code conventions](https://gcanti.github.io/fp-ts/introduction/code-conventions)

# License

The MIT License (MIT)
