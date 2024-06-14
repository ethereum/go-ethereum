## strip-hex-prefix

<div>
  <!-- Dependency Status -->
  <a href="https://david-dm.org/silentcicero/strip-hex-prefix">
    <img src="https://david-dm.org/silentcicero/strip-hex-prefix.svg"
    alt="Dependency Status" />
  </a>

  <!-- devDependency Status -->
  <a href="https://david-dm.org/silentcicero/strip-hex-prefix#info=devDependencies">
    <img src="https://david-dm.org/silentcicero/strip-hex-prefix/dev-status.svg" alt="devDependency Status" />
  </a>

  <!-- Build Status -->
  <a href="https://travis-ci.org/SilentCicero/strip-hex-prefix">
    <img src="https://travis-ci.org/SilentCicero/strip-hex-prefix.svg"
    alt="Build Status" />
  </a>

  <!-- NPM Version -->
  <a href="https://www.npmjs.org/package/strip-hex-prefix">
    <img src="http://img.shields.io/npm/v/strip-hex-prefix.svg"
    alt="NPM version" />
  </a>

  <a href="https://coveralls.io/r/SilentCicero/strip-hex-prefix">
    <img src="https://coveralls.io/repos/github/SilentCicero/strip-hex-prefix/badge.svg" alt="Test Coverage" />
  </a>

  <!-- Javascript Style -->
  <a href="http://airbnb.io/javascript/">
    <img src="https://img.shields.io/badge/code%20style-airbnb-brightgreen.svg" alt="js-airbnb-style" />
  </a>
</div>

<br />

A simple method to strip the hex prefix of a string, if present.

Will bypass if not a string.

## Install

```
npm install --save strip-hex-prefix
```

## Usage

```js
const stripHexPrefix = require('strip-hex-prefix');

console.log(stripHexPrefix('0x'));

// result ''

console.log(stripHexPrefix('0xhjsfdj'));

// result 'hjsfdj'

console.log(stripHexPrefix('0x87sf7373ds8sfsdhgs73y87ssgsdf89'));

// result '87sf7373ds8sfsdhgs73y87ssgsdf89'

console.log(stripHexPrefix({}));

// result {}

console.log(stripHexPrefix('-0x'));

// result '-0x'
```

## Important documents

- [Changelog](CHANGELOG.md)
- [License](https://raw.githubusercontent.com/silentcicero/strip-hex-prefix/master/LICENSE)

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
