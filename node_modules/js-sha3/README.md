# js-sha3

[![Build Status](https://travis-ci.org/emn178/js-sha3.svg?branch=master)](https://travis-ci.org/emn178/js-sha3)
[![Coverage Status](https://coveralls.io/repos/emn178/js-sha3/badge.svg?branch=master)](https://coveralls.io/r/emn178/js-sha3?branch=master)  
[![NPM](https://nodei.co/npm/js-sha3.png?stars&downloads)](https://nodei.co/npm/js-sha3/)

A simple SHA-3 / Keccak / Shake hash function for JavaScript supports UTF-8 encoding.

## Notice
* v0.8.0+ will throw an error if try to update hash after finalize.
* Sha3 methods has been renamed to keccak since v0.2.0. It means that sha3 methods of v0.1.x are equal to keccak methods of v0.2.x and later.
* `buffer` method is deprecated. This maybe confuse with Buffer in node.js. Please use `arrayBuffer` instead.

## Demo
[SHA3-512 Online](http://emn178.github.io/online-tools/sha3_512.html)  
[SHA3-384 Online](http://emn178.github.io/online-tools/sha3_384.html)  
[SHA3-256 Online](http://emn178.github.io/online-tools/sha3_256.html)  
[SHA3-224 Online](http://emn178.github.io/online-tools/sha3_224.html)  
[Keccak-512 Online](http://emn178.github.io/online-tools/keccak_512.html)  
[Keccak-384 Online](http://emn178.github.io/online-tools/keccak_384.html)  
[Keccak-256 Online](http://emn178.github.io/online-tools/keccak_256.html)  
[Keccak-224 Online](http://emn178.github.io/online-tools/keccak_224.html)  
[Shake128 Online](http://emn178.github.io/online-tools/shake_128.html)  
[Shake256 Online](http://emn178.github.io/online-tools/shake_256.html)  

## Download
[Compress](https://raw.github.com/emn178/js-sha3/master/build/sha3.min.js)  
[Uncompress](https://raw.github.com/emn178/js-sha3/master/src/sha3.js)

## Installation
You can also install js-sha3 by using Bower.

    bower install js-sha3

For node.js, you can use this command to install:

    npm install js-sha3

## Usage
You could use like this:
```JavaScript
sha3_512('Message to hash');
sha3_384('Message to hash');
sha3_256('Message to hash');
sha3_224('Message to hash');
keccak512('Message to hash');
keccak384('Message to hash');
keccak256('Message to hash');
keccak224('Message to hash');
shake128('Message to hash', 256);
shake256('Message to hash', 512);
cshake128('Message to hash', 256, 'function name', 'customization');
cshake256('Message to hash', 512, 'function name', 'customization');
kmac128('key', 'Message to hash', 256, 'customization');
kmac256('key', 'Message to hash', 512, 'customization');

// Support ArrayBuffer output
var arrayBuffer = keccak224.arrayBuffer('Message to hash');

// Support Array output
var bytes = keccak224.digest('Message to hash');
var bytes = keccak224.array('Message to hash');

// update hash
sha3_512.update('Message ').update('to ').update('hash').hex();
// specify shake output bits at first update
shake128.update('Message ', 256).update('to ').update('hash').hex();

// or to use create
var hash = sha3_512.create();
hash.update('...');
hash.update('...');
hash.hex();

// specify shake output bits when creating
var hash = shake128.create(256);
hash.update('...');
hash.update('...');
hash.hex();

// specify cshake output bits, function name and customization when creating
var hash = cshake128.create(256, 'function name', 'customization');

// specify kmac key, output bits and customization when creating
var hash = kmac128.create('key', 256, 'customization');
```
If you use node.js, you should require the module first:
```JavaScript
sha3_512 = require('js-sha3').sha3_512;
sha3_384 = require('js-sha3').sha3_384;
sha3_256 = require('js-sha3').sha3_256;
sha3_224 = require('js-sha3').sha3_224;
keccak512 = require('js-sha3').keccak512;
keccak384 = require('js-sha3').keccak384;
keccak256 = require('js-sha3').keccak256;
keccak224 = require('js-sha3').keccak224;
shake128 = require('js-sha3').shake128;
shake256 = require('js-sha3').shake256;
cshake128 = require('js-sha3').cshake128;
cshake256 = require('js-sha3').cshake256;
kmac128 = require('js-sha3').kmac128;
kmac256 = require('js-sha3').kmac256;
```
If you use TypeScript, you can import like this:
```TypeScript
import { sha3_512 } from 'js-sha3';
```

## Example
Code
```JavaScript
sha3_512('');
// a69f73cca23a9ac5c8b567dc185a756e97c982164fe25859e0d1dcc1475c80a615b2123af1f5f94c11e3e9402c3ac558f500199d95b6d3e301758586281dcd26

sha3_512('The quick brown fox jumps over the lazy dog');
// 01dedd5de4ef14642445ba5f5b97c15e47b9ad931326e4b0727cd94cefc44fff23f07bf543139939b49128caf436dc1bdee54fcb24023a08d9403f9b4bf0d450

sha3_512('The quick brown fox jumps over the lazy dog.');
// 18f4f4bd419603f95538837003d9d254c26c23765565162247483f65c50303597bc9ce4d289f21d1c2f1f458828e33dc442100331b35e7eb031b5d38ba6460f8

sha3_384('');
// 0c63a75b845e4f7d01107d852e4c2485c51a50aaaa94fc61995e71bbee983a2ac3713831264adb47fb6bd1e058d5f004

sha3_384('The quick brown fox jumps over the lazy dog');
// 7063465e08a93bce31cd89d2e3ca8f602498696e253592ed26f07bf7e703cf328581e1471a7ba7ab119b1a9ebdf8be41

sha3_384('The quick brown fox jumps over the lazy dog.');
// 1a34d81695b622df178bc74df7124fe12fac0f64ba5250b78b99c1273d4b080168e10652894ecad5f1f4d5b965437fb9

sha3_256('');
// a7ffc6f8bf1ed76651c14756a061d662f580ff4de43b49fa82d80a4b80f8434a

sha3_256('The quick brown fox jumps over the lazy dog');
// 69070dda01975c8c120c3aada1b282394e7f032fa9cf32f4cb2259a0897dfc04

sha3_256('The quick brown fox jumps over the lazy dog.');
// a80f839cd4f83f6c3dafc87feae470045e4eb0d366397d5c6ce34ba1739f734d

sha3_224('');
// 6b4e03423667dbb73b6e15454f0eb1abd4597f9a1b078e3f5b5a6bc7

sha3_224('The quick brown fox jumps over the lazy dog');
// d15dadceaa4d5d7bb3b48f446421d542e08ad8887305e28d58335795

sha3_224('The quick brown fox jumps over the lazy dog.');
// 2d0708903833afabdd232a20201176e8b58c5be8a6fe74265ac54db0

keccak512('');
// 0eab42de4c3ceb9235fc91acffe746b29c29a8c366b7c60e4e67c466f36a4304c00fa9caf9d87976ba469bcbe06713b435f091ef2769fb160cdab33d3670680e

keccak512('The quick brown fox jumps over the lazy dog');
// d135bb84d0439dbac432247ee573a23ea7d3c9deb2a968eb31d47c4fb45f1ef4422d6c531b5b9bd6f449ebcc449ea94d0a8f05f62130fda612da53c79659f609

keccak512('The quick brown fox jumps over the lazy dog.');
// ab7192d2b11f51c7dd744e7b3441febf397ca07bf812cceae122ca4ded6387889064f8db9230f173f6d1ab6e24b6e50f065b039f799f5592360a6558eb52d760

keccak384('');
// 2c23146a63a29acf99e73b88f8c24eaa7dc60aa771780ccc006afbfa8fe2479b2dd2b21362337441ac12b515911957ff

keccak384('The quick brown fox jumps over the lazy dog');
// 283990fa9d5fb731d786c5bbee94ea4db4910f18c62c03d173fc0a5e494422e8a0b3da7574dae7fa0baf005e504063b3

keccak384('The quick brown fox jumps over the lazy dog.');
// 9ad8e17325408eddb6edee6147f13856ad819bb7532668b605a24a2d958f88bd5c169e56dc4b2f89ffd325f6006d820b

keccak256('');
// c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470

keccak256('The quick brown fox jumps over the lazy dog');
// 4d741b6f1eb29cb2a9b9911c82f56fa8d73b04959d3d9d222895df6c0b28aa15

keccak256('The quick brown fox jumps over the lazy dog.');
// 578951e24efd62a3d63a86f7cd19aaa53c898fe287d2552133220370240b572d

keccak224('');
// f71837502ba8e10837bdd8d365adb85591895602fc552b48b7390abd

keccak224('The quick brown fox jumps over the lazy dog');
// 310aee6b30c47350576ac2873fa89fd190cdc488442f3ef654cf23fe

keccak224('The quick brown fox jumps over the lazy dog.');
// c59d4eaeac728671c635ff645014e2afa935bebffdb5fbd207ffdeab

shake128('', 256);
// 7f9c2ba4e88f827d616045507605853ed73b8093f6efbc88eb1a6eacfa66ef26

shake256('', 512);
// 46b9dd2b0ba88d13233b3feb743eeb243fcd52ea62b81b82b50c27646ed5762fd75dc4ddd8c0f200cb05019d67b592f6fc821c49479ab48640292eacb3b7c4be
```
It also supports UTF-8 encoding:

Code
```JavaScript
sha3_512('中文');
// 059bbe2efc50cc30e4d8ec5a96be697e2108fcbf9193e1296192eddabc13b143c0120d059399a13d0d42651efe23a6c1ce2d1efb576c5b207fa2516050505af7

sha3_384('中文');
// 9fb5b99e3c546f2738dcd50a14e9aef9c313800c1bf8cf76bc9b2c3a23307841364c5a2d0794702662c5796fb72f5432

sha3_256('中文');
// ac5305da3d18be1aed44aa7c70ea548da243a59a5fd546f489348fd5718fb1a0

sha3_224('中文');
// 106d169e10b61c2a2a05554d3e631ec94467f8316640f29545d163ee

keccak512('中文');
// 2f6a1bd50562230229af34b0ccf46b8754b89d23ae2c5bf7840b4acfcef86f87395edc0a00b2bfef53bafebe3b79de2e3e01cbd8169ddbb08bde888dcc893524

keccak384('中文');
// 743f64bb7544c6ed923be4741b738dde18b7cee384a3a09c4e01acaaac9f19222cdee137702bd3aa05dc198373d87d6c

keccak256('中文');
// 70a2b6579047f0a977fcb5e9120a4e07067bea9abb6916fbc2d13ffb9a4e4eee

keccak224('中文');
// f71837502ba8e10837bdd8d365adb85591895602fc552b48b7390abd
```

It also supports byte `Array`, `Uint8Array`, `ArrayBuffer` input:

Code
```JavaScript
sha3_512([]);
// a69f73cca23a9ac5c8b567dc185a756e97c982164fe25859e0d1dcc1475c80a615b2123af1f5f94c11e3e9402c3ac558f500199d95b6d3e301758586281dcd26

sha3_512(new Uint8Array([]));
// a69f73cca23a9ac5c8b567dc185a756e97c982164fe25859e0d1dcc1475c80a615b2123af1f5f94c11e3e9402c3ac558f500199d95b6d3e301758586281dcd26

// ...
```

## Benchmark
[UTF8](http://jsperf.com/sha3/5)  
[ASCII](http://jsperf.com/sha3/4)

## License
The project is released under the [MIT license](http://www.opensource.org/licenses/MIT).

## Contact
The project's website is located at https://github.com/emn178/js-sha3  
Author: Chen, Yi-Cyuan (emn178@gmail.com)
