# keccak

This module provides native bindings to [Keccak sponge function family][1] from [Keccak Code Package][2]. In browser pure JavaScript implementation will be used.

## Usage

You can use this package as [node Hash][3].

```js
const createKeccakHash = require('keccak')

console.log(createKeccakHash('keccak256').digest().toString('hex'))
// => c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470

console.log(createKeccakHash('keccak256').update('Hello world!').digest('hex'))
// => ecd0e108a98e192af1d2c25055f4e3bed784b5c877204e73219a5203251feaab
```

Also object has two useful methods: `_resetState` and `_clone`

```js
const createKeccakHash = require('keccak')

console.log(createKeccakHash('keccak256').update('Hello World!')._resetState().digest('hex'))
// => c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470

const hash1 = createKeccakHash('keccak256').update('Hello')
const hash2 = hash1._clone()
console.log(hash1.digest('hex'))
// => 06b3dfaec148fb1bb2b066f10ec285e7c9bf402ab32aa78a5d38e34566810cd2
console.log(hash1.update(' world!').digest('hex'))
// => throw Error: Digest already called
console.log(hash2.update(' world!').digest('hex'))
// => ecd0e108a98e192af1d2c25055f4e3bed784b5c877204e73219a5203251feaab
```

### Why I should use this package?

I thought it will be popular question, so I decide write explanation in readme.

I know a few popular packages on [npm][4] related with [Keccak][1]:

  - [sha3][5] ([phusion/node-sha3][6] on github) — not actual because support _only keccak_.
  - [js-sha3][7] ([emn178/js-sha3][8] on github) — brilliant package which support keccak, sha3, shake. But not implement [node Hash][3] interface unfortunately!
  - [browserify-sha3][9] ([wanderer/browserify-sha3][10] on github) — based on [js-sha3][7] (but not support shake!). Support [node Hash][3] interface, but without [streams][11].
  - [keccakjs][12] ([axic/keccakjs][13] on github) — uses [sha3][5] and [browserify-sha3][9] as fallback. As result _keccak only_ with [node Hash][3] interface without [streams][11].

## LICENSE

This library is free and open-source software released under the MIT license.

[1]: http://keccak.noekeon.org/
[2]: https://github.com/gvanas/KeccakCodePackage
[3]: https://nodejs.org/api/crypto.html#crypto_class_hash
[4]: http://npmjs.com/
[5]: https://www.npmjs.com/package/sha3
[6]: https://github.com/phusion/node-sha3
[7]: https://www.npmjs.com/package/js-sha3
[8]: https://github.com/emn178/js-sha3
[9]: https://www.npmjs.com/package/browserify-sha3
[10]: https://github.com/wanderer/browserify-sha3
[11]: http://nodejs.org/api/stream.html
[12]: https://www.npmjs.com/package/keccakjs
[13]: https://github.com/axic/keccakjs
