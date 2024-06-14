## number-to-bn

<div>
  <!-- Dependency Status -->
  <a href="https://david-dm.org/silentcicero/number-to-bn">
    <img src="https://david-dm.org/silentcicero/number-to-bn.svg"
    alt="Dependency Status" />
  </a>

  <!-- devDependency Status -->
  <a href="https://david-dm.org/silentcicero/number-to-bn#info=devDependencies">
    <img src="https://david-dm.org/silentcicero/number-to-bn/dev-status.svg" alt="devDependency Status" />
  </a>

  <!-- Build Status -->
  <a href="https://travis-ci.org/SilentCicero/number-to-bn">
    <img src="https://travis-ci.org/SilentCicero/number-to-bn.svg"
    alt="Build Status" />
  </a>

  <!-- NPM Version -->
  <a href="https://www.npmjs.org/package/number-to-bn">
    <img src="http://img.shields.io/npm/v/number-to-bn.svg"
    alt="NPM version" />
  </a>

  <a href="https://coveralls.io/r/SilentCicero/number-to-bn">
    <img src="https://coveralls.io/repos/github/SilentCicero/number-to-bn/badge.svg" alt="Test Coverage" />
  </a>

  <!-- Javascript Style -->
  <a href="http://airbnb.io/javascript/">
    <img src="https://img.shields.io/badge/code%20style-airbnb-brightgreen.svg" alt="js-airbnb-style" />
  </a>
</div>

<br />

A simple method to convert integer or hex integer numbers to BN.js object instances. Does not supprot decimal numbers.

## Install

```
npm install --save number-to-bn
```

## Usage

```js
const numberToBN = require('number-to-bn');

console.log(numberToBN('-1'));

// result <BN ...> -1

console.log(numberToBN(1));

// result <BN ...> 1

console.log(numberToBN(new BN(100)));

// result <BN ...> 100

console.log(numberToBN(new BigNumber(10000)));

// result <BN ...> 10000

console.log(numberToBN('0x0a'));

// result <BN ...> 10

console.log(numberToBN('-0x0a'));

// result <BN ...> -10

console.log(numberToBN('0.9')); // or {}, [], undefined, 9.9

// throws new Error(...)

console.log(numberToBN(null)); // or {}, [], undefined, 9.9

// throws new Error(...)
```

## Important documents

- [Changelog](CHANGELOG.md)
- [License](https://raw.githubusercontent.com/silentcicero/number-to-bn/master/LICENSE)

## Licence

This project is licensed under the MIT license, Copyright (c) 2016 Nick Dodson. For more information see LICENSE.md.

```
The MIT License

Copyright (c) 2016 Nick Dodson. nickdodson.com

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
```
