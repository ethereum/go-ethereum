# bech32
[![build status](https://secure.travis-ci.org/bitcoinjs/bech32.png)](http://travis-ci.org/bitcoinjs/bech32)
[![Version](http://img.shields.io/npm/v/bech32.svg)](https://www.npmjs.org/package/bech32)

A [BIP173](https://github.com/bitcoin/bips/blob/master/bip-0173.mediawiki) compatible Bech32 encoding/decoding library.


## Example
``` javascript
let bech32 = require('bech32')

bech32.decode('abcdef1qpzry9x8gf2tvdw0s3jn54khce6mua7lmqqqxw')
// => {
// 	 prefix: 'abcdef',
// 	 words: [0,1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21,22,23,24,25,26,27,28,29,30,31]
// }

let words = bech32.toWords(Buffer.from('foobar', 'utf8'))
bech32.encode('foo', words)
// => 'foo1vehk7cnpwgry9h96'
```


### Advanced
BIP173 enforces a limitation of 90 characters,  if extend the `LIMIT` parameter beyond this,  be aware that the [effectiveness of checksum decreases as the length increases](https://github.com/bitcoin/bips/blob/master/bip-0173.mediawiki#checksum-design).

It is highly recommended **NOT** exceed 1023 characters, as the module could only guarantee detecting 1 error.


## Credits
- [Peter Wuille](https://github.com/sipa/bech32) for the reference JavaScript implementation, and for authoring the Bech32 [BIP173](https://github.com/bitcoin/bips/blob/master/bip-0173.mediawiki).


## License [MIT](LICENSE)
