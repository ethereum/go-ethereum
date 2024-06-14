# bs58check

[![NPM Package](https://img.shields.io/npm/v/bs58check.svg?style=flat-square)](https://www.npmjs.org/package/bs58check)
[![Build Status](https://img.shields.io/travis/bitcoinjs/bs58check.svg?branch=master&style=flat-square)](https://travis-ci.org/bitcoinjs/bs58check)
[![Dependency status](https://img.shields.io/david/bitcoinjs/bs58check.svg?style=flat-square)](https://david-dm.org/bitcoinjs/bs58check#info=dependencies)

[![js-standard-style](https://cdn.rawgit.com/feross/standard/master/badge.svg)](https://github.com/feross/standard)

A straight forward implementation of base58check extending upon bs58.


## Example

```javascript
var bs58check = require('bs58check')

var decoded = bs58check.decode('5Kd3NBUAdUnhyzenEwVLy9pBKxSwXvE9FMPyR4UKZvpe6E3AgLr')

console.log(decoded)
// => <Buffer 80 ed db dc 11 68 f1 da ea db d3 e4 4c 1e 3f 8f 5a 28 4c 20 29 f7 8a d2 6a f9 85 83 a4 99 de 5b 19>

console.log(bs58check.encode(decoded))
// => 5Kd3NBUAdUnhyzenEwVLy9pBKxSwXvE9FMPyR4UKZvpe6E3AgLr
```


## LICENSE [MIT](LICENSE)
