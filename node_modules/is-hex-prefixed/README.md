## is-hex-prefixed

<div>
  <!-- Dependency Status -->
  <a href="https://david-dm.org/silentcicero/is-hex-prefixed">
    <img src="https://david-dm.org/silentcicero/is-hex-prefixed.svg"
    alt="Dependency Status" />
  </a>

  <!-- devDependency Status -->
  <a href="https://david-dm.org/silentcicero/is-hex-prefixed#info=devDependencies">
    <img src="https://david-dm.org/silentcicero/is-hex-prefixed/dev-status.svg" alt="devDependency Status" />
  </a>

  <!-- Build Status -->
  <a href="https://travis-ci.org/SilentCicero/is-hex-prefixed">
    <img src="https://travis-ci.org/SilentCicero/is-hex-prefixed.svg"
    alt="Build Status" />
  </a>

  <!-- NPM Version -->
  <a href="https://www.npmjs.org/package/is-hex-prefixed">
    <img src="http://img.shields.io/npm/v/is-hex-prefixed.svg"
    alt="NPM version" />
  </a>

  <!-- Test Coverage -->
  <a href="https://coveralls.io/r/SilentCicero/is-hex-prefixed">
    <img src="https://coveralls.io/repos/github/SilentCicero/is-hex-prefixed/badge.svg" alt="Test Coverage" />
  </a>

  <!-- Javascript Style -->
  <a href="http://airbnb.io/javascript/">
    <img src="https://img.shields.io/badge/code%20style-airbnb-brightgreen.svg" alt="js-airbnb-style" />
  </a>
</div>

<br />

A simple method to check if a string is hex prefixed.

## Install

```
npm install --save is-hex-prefixed
```

## Usage

```js
const isHexPrefixed = require('is-hex-prefixed');

console.log(isHexPrefixed('0x..'));

// result true

console.log(isHexPrefixed('dfsk'));

// result false

console.log(isHexPrefixed({}));

// result throw new Error

console.log(isHexPrefixed('-0x'));

// result false
```

## Important documents

- [Changelog](CHANGELOG.md)
- [License](https://raw.githubusercontent.com/silentcicero/is-hex-prefixed/master/LICENSE)

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
