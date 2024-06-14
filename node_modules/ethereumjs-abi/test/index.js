var assert = require('assert')
var abi = require('../index.js')
var BN = require('bn.js')

// Official test vectors from https://github.com/ethereum/wiki/wiki/Ethereum-Contract-ABI

describe('official test vector 1 (encoding)', function () {
  it('should equal', function () {
    var a = abi.methodID('baz', [ 'uint32', 'bool' ]).toString('hex') + abi.rawEncode([ 'uint32', 'bool' ], [ 69, 1 ]).toString('hex')
    var b = 'cdcd77c000000000000000000000000000000000000000000000000000000000000000450000000000000000000000000000000000000000000000000000000000000001'
    assert.strict.equal(a, b)
  })
})

/*
describe('official test vector 2 (encoding)', function () {
  it('should equal', function () {
    var a = abi.methodID('bar', [ 'real128x128[2]' ]).toString('hex') + abi.rawEncode([ 'real128x128[2]' ], [ [ 2.125, 8.5 ] ]).toString('hex')
    var b = '3e27986000000000000000000000000000000002400000000000000000000000000000000000000000000000000000000000000880000000000000000000000000000000'
    assert.strict.equal(a, b)
  })
})
*/

describe('official test vector 3 (encoding)', function () {
  it('should equal', function () {
    var a = abi.methodID('sam', [ 'bytes', 'bool', 'uint256[]' ]).toString('hex') + abi.rawEncode([ 'bytes', 'bool', 'uint256[]' ], [ 'dave', true, [ 1, 2, 3 ] ]).toString('hex')
    var b = 'a5643bf20000000000000000000000000000000000000000000000000000000000000060000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000a0000000000000000000000000000000000000000000000000000000000000000464617665000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000003000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000003'
    assert.strict.equal(a, b)
  })
})

describe('official test vector 4 (encoding)', function () {
  it('should equal', function () {
    var a = abi.methodID('f', [ 'uint', 'uint32[]', 'bytes10', 'bytes' ]).toString('hex') + abi.rawEncode([ 'uint', 'uint32[]', 'bytes10', 'bytes' ], [ 0x123, [ 0x456, 0x789 ], '1234567890', 'Hello, world!' ]).toString('hex')
    var b = '8be6524600000000000000000000000000000000000000000000000000000000000001230000000000000000000000000000000000000000000000000000000000000080313233343536373839300000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000e0000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000004560000000000000000000000000000000000000000000000000000000000000789000000000000000000000000000000000000000000000000000000000000000d48656c6c6f2c20776f726c642100000000000000000000000000000000000000'
    assert.strict.equal(a, b)
  })
})

// Homebrew tests

describe('method signature', function () {
  it('should work with test()', function () {
    assert.strict.equal(abi.methodID('test', []).toString('hex'), 'f8a8fd6d')
  })
  it('should work with test(uint)', function () {
    assert.strict.equal(abi.methodID('test', [ 'uint' ]).toString('hex'), '29e99f07')
  })
  it('should work with test(uint256)', function () {
    assert.strict.equal(abi.methodID('test', [ 'uint256' ]).toString('hex'), '29e99f07')
  })
  it('should work with test(uint, uint)', function () {
    assert.strict.equal(abi.methodID('test', [ 'uint', 'uint' ]).toString('hex'), 'eb8ac921')
  })
})

describe('event signature', function () {
  it('should work with test()', function () {
    assert.strict.equal(abi.eventID('test', []).toString('hex'), 'f8a8fd6dd9544ca87214e80c840685bd13ff4682cacb0c90821ed74b1d248926')
  })
  it('should work with test(uint)', function () {
    assert.strict.equal(abi.eventID('test', [ 'uint' ]).toString('hex'), '29e99f07d14aa8d30a12fa0b0789b43183ba1bf6b4a72b95459a3e397cca10d7')
  })
  it('should work with test(uint256)', function () {
    assert.strict.equal(abi.eventID('test', [ 'uint256' ]).toString('hex'), '29e99f07d14aa8d30a12fa0b0789b43183ba1bf6b4a72b95459a3e397cca10d7')
  })
  it('should work with test(uint, uint)', function () {
    assert.strict.equal(abi.eventID('test', [ 'uint', 'uint' ]).toString('hex'), 'eb8ac9210327650aab0044de896b150391af3be06f43d0f74c01f05633b97a70')
  })
})

describe('encoding negative int32', function () {
  it('should equal', function () {
    var a = abi.rawEncode([ 'int32' ], [ -2 ]).toString('hex')
    var b = 'fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe'
    assert.strict.equal(a, b)
  })
})

describe('encoding negative int256', function () {
  it('should equal', function () {
    var a = abi.rawEncode([ 'int256' ], [ new BN('-19999999999999999999999999999999999999999999999999999999999999', 10) ]).toString('hex')
    var b = 'fffffffffffff38dd0f10627f5529bdb2c52d4846810af0ac000000000000001'
    assert.strict.equal(a, b)
  })
})

describe('encoding string >32bytes', function () {
  it('should equal', function () {
    var a = abi.rawEncode([ 'string' ], [ ' hello world hello world hello world hello world  hello world hello world hello world hello world  hello world hello world hello world hello world hello world hello world hello world hello world' ]).toString('hex')
    var b = '000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000000c22068656c6c6f20776f726c642068656c6c6f20776f726c642068656c6c6f20776f726c642068656c6c6f20776f726c64202068656c6c6f20776f726c642068656c6c6f20776f726c642068656c6c6f20776f726c642068656c6c6f20776f726c64202068656c6c6f20776f726c642068656c6c6f20776f726c642068656c6c6f20776f726c642068656c6c6f20776f726c642068656c6c6f20776f726c642068656c6c6f20776f726c642068656c6c6f20776f726c642068656c6c6f20776f726c64000000000000000000000000000000000000000000000000000000000000'
    assert.strict.equal(a, b)
  })
})

describe('encoding uint32 response', function () {
  it('should equal', function () {
    var a = abi.rawEncode([ 'uint32' ], [ 42 ]).toString('hex')
    var b = '000000000000000000000000000000000000000000000000000000000000002a'
    assert.strict.equal(a, b)
  })
})

describe('encoding string response (unsupported)', function () {
  it('should equal', function () {
    var a = abi.rawEncode([ 'string' ], [ 'a response string (unsupported)' ]).toString('hex')
    var b = '0000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000001f6120726573706f6e736520737472696e672028756e737570706f727465642900'
    assert.strict.equal(a, b)
  })
})

describe('encoding', function () {
  it('should work for uint256', function () {
    var a = abi.rawEncode([ 'uint256' ], [ 1 ]).toString('hex')
    var b = '0000000000000000000000000000000000000000000000000000000000000001'
    assert.strict.equal(a, b)
  })
  it('should work for uint', function () {
    var a = abi.rawEncode([ 'uint' ], [ 1 ]).toString('hex')
    var b = '0000000000000000000000000000000000000000000000000000000000000001'
    assert.strict.equal(a, b)
  })
  it('should work for int256', function () {
    var a = abi.rawEncode([ 'int256' ], [ -1 ]).toString('hex')
    var b = 'ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff'
    assert.strict.equal(a, b)
  })
  it('should work for string and uint256[2]', function () {
    var a = abi.rawEncode([ 'string', 'uint256[2]' ], [ 'foo', [5, 6] ]).toString('hex')
    var b = '0000000000000000000000000000000000000000000000000000000000000060000000000000000000000000000000000000000000000000000000000000000500000000000000000000000000000000000000000000000000000000000000060000000000000000000000000000000000000000000000000000000000000003666f6f0000000000000000000000000000000000000000000000000000000000'
    assert.strict.equal(a, b)
  })
})

describe('encoding bytes33', function () {
  it('should fail', function () {
    assert.throws(function () {
      abi.rawEncode('fail', [ 'bytes33' ], [ '' ])
    }, Error)
  })
})

describe('encoding uint0', function () {
  it('should fail', function () {
    assert.throws(function () {
      abi.rawEncode('fail', [ 'uint0' ], [ 1 ])
    }, Error)
  })
})

describe('encoding uint257', function () {
  it('should fail', function () {
    assert.throws(function () {
      abi.rawEncode('fail', [ 'uint257' ], [ 1 ])
    }, Error)
  })
})

describe('encoding int0', function () {
  it('should fail', function () {
    assert.throws(function () {
      abi.rawEncode([ 'int0' ], [ 1 ])
    }, Error)
  })
})

describe('encoding int257', function () {
  it('should fail', function () {
    assert.throws(function () {
      abi.rawEncode([ 'int257' ], [ 1 ])
    }, Error)
  })
})

describe('encoding uint[2] with [1,2,3]', function () {
  it('should fail', function () {
    assert.throws(function () {
      abi.rawEncode([ 'uint[2]' ], [ [ 1, 2, 3 ] ])
    }, Error)
  })
})

describe('encoding uint8 with 9bit data', function () {
  it('should fail', function () {
    assert.throws(function () {
      abi.rawEncode([ 'uint8' ], [ new BN(1).iushln(9) ])
    }, Error)
  })
})

it('decoding address with leading 0', function () {
  var decoded = abi.rawDecode([ 'address' ], Buffer.from('0000000000000000000000000005b7d915458ef540ade6068dfe2f44e8fa733c', 'hex'))
  assert.strict.deepEqual(abi.stringify([ 'address' ], decoded), [ '0x0005b7d915458ef540ade6068dfe2f44e8fa733c' ])
})

// Homebrew decoding tests

describe('decoding uint32', function () {
  it('should equal', function () {
    var a = abi.rawDecode([ 'uint32' ], Buffer.from('000000000000000000000000000000000000000000000000000000000000002a', 'hex'))
    var b = new BN(42)
    assert.strict.equal(a.length, 1)
    assert.strict.equal(a[0].toString(), b.toString())
  })
})

describe('decoding uint256[]', function () {
  it('should equal', function () {
    var a = abi.rawDecode([ 'uint256[]' ], Buffer.from('00000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000003000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000003', 'hex'))
    var b = new BN(1)
    var c = new BN(2)
    var d = new BN(3)

    assert.strict.equal(a.length, 1)
    assert.strict.equal(a[0].length, 3)
    assert.strict.equal(a[0][0].toString(), b.toString())
    assert.strict.equal(a[0][1].toString(), c.toString())
    assert.strict.equal(a[0][2].toString(), d.toString())
  })
})

describe('decoding bytes', function () {
  it('should equal', function () {
    var a = abi.rawDecode([ 'bytes' ], Buffer.from('0000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000b68656c6c6f20776f726c64000000000000000000000000000000000000000000', 'hex'))
    var b = Buffer.from('68656c6c6f20776f726c64', 'hex')

    assert.strict.equal(a.length, 1)
    assert.strict.equal(a[0].toString(), b.toString())
  })
})

describe('decoding string', function () {
  it('should equal', function () {
    var a = abi.rawDecode([ 'string' ], Buffer.from('0000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000b68656c6c6f20776f726c64000000000000000000000000000000000000000000', 'hex'))
    var b = 'hello world'
    assert.strict.equal(a.length, 1)
    assert.strict.equal(a[0], b)
  })
})

describe('decoding int32', function () {
  it('should equal', function () {
    var a = abi.rawDecode([ 'int32' ], Buffer.from('fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe', 'hex'))
    var b = new BN(-2)
    assert.strict.equal(a.length, 1)
    assert.strict.equal(a[0].toString(), b.toString())

    a = abi.rawDecode([ 'int64' ], Buffer.from('ffffffffffffffffffffffffffffffffffffffffffffffffffffb29c26f344fe', 'hex'))
    b = new BN(-85091238591234)
    assert.strict.equal(a.length, 1)
    assert.strict.equal(a[0].toString(), b.toString())
  })
  it('should fail', function () {
    assert.throws(function () {
      abi.rawDecode([ 'int32' ], Buffer.from('ffffffffffffffffffffffffffffffffffffffffffffffffffffb29c26f344fe', 'hex'))
    }, Error)
  })
})

describe('decoding bool, uint32', function () {
  it('should equal', function () {
    var a = abi.rawDecode([ 'bool', 'uint32' ], Buffer.from('0000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000002a', 'hex'))
    assert.strict.equal(a.length, 2)
    assert.strict.equal(a[0], true)
    assert.strict.equal(a[1].toString(), new BN(42).toString())
  })
})

describe('decoding bool, uint256[]', function () {
  it('should equal', function () {
    var a = abi.rawDecode([ 'bool', 'uint256[]' ], Buffer.from('000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000002a', 'hex'))
    assert.strict.equal(a.length, 2)
    assert.strict.equal(a[0], true)
    assert.strict.equal(a[1].length, 1)
    assert.strict.equal(a[1][0].toString(), new BN(42).toString())
  })
})

describe('decoding uint256[], bool', function () {
  it('should equal', function () {
    var a = abi.rawDecode([ 'uint256[]', 'bool' ], Buffer.from('000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000002a', 'hex'))
    assert.strict.equal(a.length, 2)
    assert.strict.equal(a[1], true)
    assert.strict.equal(a[0].length, 1)
    assert.strict.equal(a[0][0].toString(), new BN(42).toString())
  })
})

describe('decoding fixed-array', function () {
  it('uint[3]', function () {
    var a = abi.rawDecode([ 'uint[3]' ], Buffer.from('000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000003', 'hex'))
    assert.strict.equal(a.length, 1)
    assert.strict.equal(a[0].length, 3)
    assert.strict.equal(a[0][0].toString(10), '1')
    assert.strict.equal(a[0][1].toString(10), '2')
    assert.strict.equal(a[0][2].toString(10), '3')
  })
})

describe('decoding (uint[2], uint)', function () {
  it('should work', function () {
    var a = abi.rawDecode([ 'uint[2]', 'uint' ], Buffer.from('0000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000005c0000000000000000000000000000000000000000000000000000000000000003', 'hex'))
    assert.strict.equal(a.length, 2)
    assert.strict.equal(a[0].length, 2)
    assert.strict.equal(a[0][0].toString(10), '1')
    assert.strict.equal(a[0][1].toString(10), '92')
    assert.strict.equal(a[1].toString(10), '3')
  })
})

/* FIXME: should check that the whole input buffer was consumed
describe('decoding uint[2] with [1,2,3]', function () {
  it('should fail', function () {
    assert.throws(function () {
      abi.rawDecode([ 'uint[2]' ], Buffer.from('00000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000003000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000003', 'hex'))
    }, Error)
  })
})
*/

describe('stringify', function () {
  it('should be hex prefixed for address', function () {
    assert.strict.deepEqual(abi.stringify([ 'address' ], [ new BN('1234', 16) ]), [ '0x1234' ])
  })

  it('should be hex prefixed for bytes', function () {
    assert.strict.deepEqual(abi.stringify([ 'bytes' ], [ Buffer.from('1234', 'hex') ]), [ '0x1234' ])
  })

  it('should be hex prefixed for bytesN', function () {
    assert.strict.deepEqual(abi.stringify([ 'bytes32' ], [ Buffer.from('1234', 'hex') ]), [ '0x1234' ])
  })

  it('should be a number for uint', function () {
    assert.strict.deepEqual(abi.stringify([ 'uint' ], [ 42 ]), [ '42' ])
  })

  it('should be a number for uintN', function () {
    assert.strict.deepEqual(abi.stringify([ 'uint8' ], [ 42 ]), [ '42' ])
  })

  it('should be a number for int', function () {
    assert.strict.deepEqual(abi.stringify([ 'int' ], [ -42 ]), [ '-42' ])
  })

  it('should be a number for intN', function () {
    assert.strict.deepEqual(abi.stringify([ 'int8' ], [ -42 ]), [ '-42' ])
  })

  it('should work for bool (true)', function () {
    assert.strict.deepEqual(abi.stringify([ 'bool' ], [ true ]), [ 'true' ])
  })

  it('should work for bool (false)', function () {
    assert.strict.deepEqual(abi.stringify([ 'bool' ], [ false ]), [ 'false' ])
  })

  it('should work for address[]', function () {
    assert.strict.deepEqual(abi.stringify([ 'address[]' ], [ [ new BN('1234', 16), new BN('5678', 16) ] ]), [ '0x1234, 0x5678' ])
  })

  it('should work for address[2]', function () {
    assert.strict.deepEqual(abi.stringify([ 'address[2]' ], [ [ new BN('1234', 16), new BN('5678', 16) ] ]), [ '0x1234, 0x5678' ])
  })

  it('should work for bytes[]', function () {
    assert.strict.deepEqual(abi.stringify([ 'bytes[]' ], [ [ Buffer.from('1234', 'hex'), Buffer.from('5678', 'hex') ] ]), [ '0x1234, 0x5678' ])
  })

  it('should work for bytes[2]', function () {
    assert.strict.deepEqual(abi.stringify([ 'bytes[2]' ], [ [ Buffer.from('1234', 'hex'), Buffer.from('5678', 'hex') ] ]), [ '0x1234, 0x5678' ])
  })

  it('should work for uint[]', function () {
    assert.strict.deepEqual(abi.stringify([ 'uint[]' ], [ [ 1, 2, 3 ] ]), [ '1, 2, 3' ])
  })

  it('should work for uint[3]', function () {
    assert.strict.deepEqual(abi.stringify([ 'uint[3]' ], [ [ 1, 2, 3 ] ]), [ '1, 2, 3' ])
  })

  it('should work for int[]', function () {
    assert.strict.deepEqual(abi.stringify([ 'int[]' ], [ [ -1, -2, -3 ] ]), [ '-1, -2, -3' ])
  })

  it('should work for int[3]', function () {
    assert.strict.deepEqual(abi.stringify([ 'int[3]' ], [ [ -1, -2, -3 ] ]), [ '-1, -2, -3' ])
  })

  it('should work for multiple entries', function () {
    assert.strict.deepEqual(abi.stringify([ 'bool', 'bool' ], [ true, true ]), [ 'true', 'true' ])
  })
})

// Tests for Solidity's tight packing
describe('solidity tight packing bool', function () {
  it('should equal', function () {
    var a = abi.solidityPack(
      [ 'bool' ],
      [ true ]
    )
    var b = '01'
    assert.strict.equal(a.toString('hex'), b.toString('hex'))

    a = abi.solidityPack(
      [ 'bool' ],
      [ false ]
    )
    b = '00'
    assert.strict.equal(a.toString('hex'), b.toString('hex'))
  })
})

describe('solidity tight packing address', function () {
  it('should equal', function () {
    var a = abi.solidityPack(
      [ 'address' ],
      [ new BN('43989fb883ba8111221e89123897538475893837', 16) ]
    )
    var b = '43989fb883ba8111221e89123897538475893837'
    assert.strict.equal(a.toString('hex'), b.toString('hex'))
  })
})

describe('solidity tight packing string', function () {
  it('should equal', function () {
    var a = abi.solidityPack(
      [ 'string' ],
      [ 'test' ]
    )
    var b = '74657374'
    assert.strict.equal(a.toString('hex'), b.toString('hex'))
  })
})

describe('solidity tight packing bytes', function () {
  it('should equal', function () {
    var a = abi.solidityPack(
      [ 'bytes' ],
      [ Buffer.from('123456', 'hex') ]
    )
    var b = '123456'
    assert.strict.equal(a.toString('hex'), b.toString('hex'))
  })
})

describe('solidity tight packing bytes8', function () {
  it('should equal', function () {
    var a = abi.solidityPack(
      [ 'bytes8' ],
      [ Buffer.from('123456', 'hex') ]
    )
    var b = '1234560000000000'
    assert.strict.equal(a.toString('hex'), b.toString('hex'))
  })
})

describe('solidity tight packing uint', function () {
  it('should equal', function () {
    var a = abi.solidityPack(
      [ 'uint' ],
      [ 42 ]
    )
    var b = '000000000000000000000000000000000000000000000000000000000000002a'
    assert.strict.equal(a.toString('hex'), b.toString('hex'))
  })
})

describe('solidity tight packing uint16', function () {
  it('should equal', function () {
    var a = abi.solidityPack(
      [ 'uint16' ],
      [ 42 ]
    )
    var b = '002a'
    assert.strict.equal(a.toString('hex'), b.toString('hex'))
  })
})

describe('solidity tight packing int', function () {
  it('should equal', function () {
    var a = abi.solidityPack(
      [ 'int' ],
      [ -42 ]
    )
    var b = 'ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffd6'
    assert.strict.equal(a.toString('hex'), b.toString('hex'))
  })
})

describe('solidity tight packing int16', function () {
  it('should equal', function () {
    var a = abi.solidityPack(
      [ 'int16' ],
      [ -42 ]
    )
    var b = 'ffd6'
    assert.strict.equal(a.toString('hex'), b.toString('hex'))
  })
})

describe('solidity tight packing multiple arguments', function () {
  it('should equal', function () {
    var a = abi.solidityPack(
      [ 'bytes32', 'uint32', 'uint32', 'uint32', 'uint32' ],
      [ Buffer.from('123456', 'hex'), 6, 7, 8, 9 ]
    )
    var b = '123456000000000000000000000000000000000000000000000000000000000000000006000000070000000800000009'
    assert.strict.equal(a.toString('hex'), b.toString('hex'))
  })
})

describe('solidity tight packing uint32[]', function () {
  it('should equal', function () {
    var a = abi.solidityPack(
      ['uint32[]'],
      [[8, 9]]
    )
    var b = '00000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000009'
    assert.strict.equal(a.toString('hex'), b.toString('hex'))
  })
})

describe('solidity tight packing bool[][]', function () {
  it('should equal', function () {
    let a = abi.solidityPack(
      ['bool[][]'],
      [[[true, false], [false, true]]]
    )
    let b = '0000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001'
    assert.strict.equal(a.toString('hex'), b.toString('hex'))
  })
})

describe('solidity tight packing address[]', function () {
  it('should equal', function () {
    let a = abi.solidityPack(
      ['address[]'],
      [[new BN('43989fb883ba8111221e89123897538475893837', 16)]]
    )
    let b = '00000000000000000000000043989fb883ba8111221e89123897538475893837'
    assert.strict.equal(a.toString('hex'), b.toString('hex'))
  })
})

describe('solidity tight packing uint32[2]', function () {
  it('should equal', function () {
    let a = abi.solidityPack(
      ['uint32[2]'],
      [[11, 12]]
    )
    let b = '000000000000000000000000000000000000000000000000000000000000000b000000000000000000000000000000000000000000000000000000000000000c'
    assert.strict.equal(a.toString('hex'), b.toString('hex'))
  })
})

describe('solidity tight packing uint32[2] with wrong array length', function () {
  it('should throw', function () {
    assert.throws(function () {
      abi.solidityPack(
        ['uint32[2]'],
        [[11, 12, 13]]
      )
    })
  })
})

describe('solidity tight packing sha3', function () {
  it('should equal', function () {
    var a = abi.soliditySHA3(
      [ 'address', 'address', 'uint', 'uint' ],
      [ new BN('43989fb883ba8111221e89123897538475893837', 16), 0, 10000, 1448075779 ]
    )
    var b = 'c3ab5ca31a013757f26a88561f0ff5057a97dfcc33f43d6b479abc3ac2d1d595'
    assert.strict.equal(a.toString('hex'), b.toString('hex'))
  })
})

describe('solidity tight packing sha3 #2', function () {
  it('should equal', function () {
    var a = abi.soliditySHA3(
      [ 'bytes32', 'uint32', 'uint32', 'uint32', 'uint32' ],
      [ Buffer.from('123456', 'hex'), 6, 7, 8, 9 ]
    )
    var b = '1f2eedb6c2ac3e4b4e4c9f7598e626baf1e15a4e848d295479f46ec85d967cba'
    assert.strict.equal(a.toString('hex'), b.toString('hex'))
  })
})

describe('solidity tight packing sha256', function () {
  it('should equal', function () {
    var a = abi.soliditySHA256(
      [ 'address', 'address', 'uint', 'uint' ],
      [ new BN('43989fb883ba8111221e89123897538475893837', 16), 0, 10000, 1448075779 ]
    )
    var b = '344d8cb0711672efbdfe991f35943847c1058e1ecf515ff63ad936b91fd16231'
    assert.strict.equal(a.toString('hex'), b.toString('hex'))
  })
})

describe('solidity tight packing ripemd160', function () {
  it('should equal', function () {
    var a = abi.solidityRIPEMD160(
      [ 'address', 'address', 'uint', 'uint' ],
      [ new BN('43989fb883ba8111221e89123897538475893837', 16), 0, 10000, 1448075779 ]
    )
    var b = '000000000000000000000000a398cc72490f72048efa52c4e92067e8499672e7'
    assert.strict.equal(a.toString('hex'), b.toString('hex'))
  })
})

describe('solidity tight packing with small ints', function () {
  it('should equal', function () {
    var a = abi.soliditySHA3(
      [ 'address', 'address', 'int64', 'uint192' ],
      [ new BN('43989fb883ba8111221e89123897538475893837', 16), 0, 10000, 1448075779 ]
    )
    var b = '1c34bbd3d419c05d028a9f13a81a1212e33cb21f4b96ce1310442911c62c6986'
    assert.strict.equal(a.toString('hex'), b.toString('hex'))
  })
})

describe('converting from serpent types', function () {
  it('should equal', function () {
    assert.strict.deepEqual(abi.fromSerpent('s'), [ 'bytes' ])
    assert.strict.deepEqual(abi.fromSerpent('i'), [ 'int256' ])
    assert.strict.deepEqual(abi.fromSerpent('a'), [ 'int256[]' ])
    assert.strict.deepEqual(abi.fromSerpent('b8'), [ 'bytes8' ])
    assert.strict.deepEqual(abi.fromSerpent('b8i'), [ 'bytes8', 'int256' ])
    assert.strict.deepEqual(abi.fromSerpent('b32'), [ 'bytes32' ])
    assert.strict.deepEqual(abi.fromSerpent('b32i'), [ 'bytes32', 'int256' ])
    assert.strict.deepEqual(abi.fromSerpent('sb8ib8a'), [ 'bytes', 'bytes8', 'int256', 'bytes8', 'int256[]' ])
    assert.throws(function () {
      abi.fromSerpent('i8')
    })
    assert.throws(function () {
      abi.fromSerpent('x')
    })
  })
})

describe('converting to serpent types', function () {
  it('should equal', function () {
    assert.strict.equal(abi.toSerpent([ 'bytes' ]), 's')
    assert.strict.equal(abi.toSerpent([ 'int256' ]), 'i')
    assert.strict.equal(abi.toSerpent([ 'int256[]' ]), 'a')
    assert.strict.equal(abi.toSerpent([ 'bytes8' ]), 'b8')
    assert.strict.equal(abi.toSerpent([ 'bytes32' ]), 'b32')
    assert.strict.equal(abi.toSerpent([ 'bytes', 'bytes8', 'int256', 'bytes8', 'int256[]' ]), 'sb8ib8a')
    assert.throws(function () {
      abi.toSerpent('int8')
    })
    assert.throws(function () {
      abi.toSerpent('bool')
    })
  })
})

describe('utf8 handling', function () {
  it('should encode latin and extensions', function () {
    var a = abi.rawEncode([ 'string' ], [ 'ethereum számítógép' ]).toString('hex')
    var b = '00000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000017657468657265756d20737ac3a16dc3ad74c3b367c3a970000000000000000000'
    assert.strict.equal(a, b)
  })
  it('should encode non-latin characters', function () {
    var a = abi.rawEncode([ 'string' ], [ '为什么那么认真？' ]).toString('hex')
    var b = '00000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000018e4b8bae4bb80e4b988e982a3e4b988e8aea4e79c9fefbc9f0000000000000000'
    assert.strict.equal(a, b)
  })
  it('should decode latin and extensions', function () {
    var a = 'ethereum számítógép'
    var b = abi.rawDecode([ 'string' ], Buffer.from('00000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000017657468657265756d20737ac3a16dc3ad74c3b367c3a970000000000000000000', 'hex'))
    assert.strict.equal(a, b[0])
  })
  it('should decode non-latin characters', function () {
    var a = '为什么那么认真？'
    var b = abi.rawDecode([ 'string' ], Buffer.from('00000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000018e4b8bae4bb80e4b988e982a3e4b988e8aea4e79c9fefbc9f0000000000000000', 'hex'))
    assert.strict.equal(a, b[0])
  })
})

describe('encoding ufixed128x128', function () {
  it('should equal', function () {
    var a = abi.rawEncode([ 'ufixed128x128' ], [ 1 ]).toString('hex')
    var b = '0000000000000000000000000000000100000000000000000000000000000000'
    assert.strict.equal(a, b)
  })
})

describe('encoding fixed128x128', function () {
  it('should equal', function () {
    var a = abi.rawEncode([ 'fixed128x128' ], [ -1 ]).toString('hex')
    var b = 'ffffffffffffffffffffffffffffffff00000000000000000000000000000000'
    assert.strict.equal(a, b)
  })
})

describe('decoding ufixed128x128', function () {
  it('should equal', function () {
    var a = Buffer.from('0000000000000000000000000000000100000000000000000000000000000000', 'hex')
    var b = abi.rawDecode([ 'ufixed128x128' ], a)
    assert.strict.equal(b[0].toNumber(), 1)
  })
  it('decimals should fail', function () {
    var a = Buffer.from('0000000000000000000000000000000100000000000000000000000000000001', 'hex')
    assert.throws(function () {
      abi.rawDecode([ 'ufixed128x128' ], a)
    }, /^Error: Decimals not supported yet/)
  })
})

describe('decoding fixed128x128', function () {
  it('should equal', function () {
    var a = Buffer.from('ffffffffffffffffffffffffffffffff00000000000000000000000000000000', 'hex')
    var b = abi.rawDecode([ 'fixed128x128' ], a)
    assert.strict.equal(b[0].toNumber(), -1)
  })
  it('decimals should fail', function () {
    var a = Buffer.from('ffffffffffffffffffffffffffffffff00000000000000000000000000000001', 'hex')
    assert.throws(function () {
      abi.rawDecode([ 'fixed128x128' ], a)
    }, /^Error: Decimals not supported yet/)
  })
})

describe('encoding -1 as uint', function () {
  it('should throw', function () {
    assert.throws(function () {
      abi.rawEncode([ 'uint' ], [ -1 ])
    }, /^Error: Supplied uint is negative/)
  })
})

describe('encoding 256 bits as bytes', function () {
  it('should not leave trailing zeroes', function () {
    var a = abi.rawEncode([ 'bytes' ], [ Buffer.from('ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff', 'hex') ])
    assert.strict.equal(a.toString('hex'), '00000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000020ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff')
  })
})

describe('decoding (uint128[2][3], uint)', function () {
  it('should work', function () {
    var data = `0000000000000000000000000000000000000000000000000000000000000001
0000000000000000000000000000000000000000000000000000000000000002
0000000000000000000000000000000000000000000000000000000000000003
0000000000000000000000000000000000000000000000000000000000000004
0000000000000000000000000000000000000000000000000000000000000005
0000000000000000000000000000000000000000000000000000000000000006
000000000000000000000000000000000000000000000000000000000000000a`
    var a = abi.rawDecode([ 'uint128[2][3]', 'uint' ], Buffer.from(data.replace(/\n/g, ''), 'hex'))
    assert.strict.equal(a.length, 2)
    assert.strict.equal(a[0][0][0].toString(10), '1')
    assert.strict.equal(a[0][0][1].toString(10), '2')
    assert.strict.equal(a[0][1][0].toString(10), '3')
    assert.strict.equal(a[0][1][1].toString(10), '4')
    assert.strict.equal(a[0][2][0].toString(10), '5')
    assert.strict.equal(a[0][2][1].toString(10), '6')
    assert.strict.equal(a[1].toString(10), '10')
  })
})

describe('decoding (uint128[2][3][2], uint)', function () {
  it('should work', function () {
    var data = `0000000000000000000000000000000000000000000000000000000000000001
0000000000000000000000000000000000000000000000000000000000000002
0000000000000000000000000000000000000000000000000000000000000003
0000000000000000000000000000000000000000000000000000000000000004
0000000000000000000000000000000000000000000000000000000000000005
0000000000000000000000000000000000000000000000000000000000000006
0000000000000000000000000000000000000000000000000000000000000001
0000000000000000000000000000000000000000000000000000000000000002
0000000000000000000000000000000000000000000000000000000000000003
0000000000000000000000000000000000000000000000000000000000000004
0000000000000000000000000000000000000000000000000000000000000005
0000000000000000000000000000000000000000000000000000000000000006
000000000000000000000000000000000000000000000000000000000000000a`
    var a = abi.rawDecode([ 'uint128[2][3][2]', 'uint' ], Buffer.from(data.replace(/\n/g, ''), 'hex'))
    assert.strict.equal(a.length, 2)
    assert.strict.equal(a[0][0][0][0].toString(10), '1')
    assert.strict.equal(a[0][0][0][1].toString(10), '2')
    assert.strict.equal(a[0][0][1][0].toString(10), '3')
    assert.strict.equal(a[0][0][1][1].toString(10), '4')
    assert.strict.equal(a[0][0][2][0].toString(10), '5')
    assert.strict.equal(a[0][0][2][1].toString(10), '6')
    assert.strict.equal(a[0][1][0][0].toString(10), '1')
    assert.strict.equal(a[0][1][0][1].toString(10), '2')
    assert.strict.equal(a[0][1][1][0].toString(10), '3')
    assert.strict.equal(a[0][1][1][1].toString(10), '4')
    assert.strict.equal(a[0][1][2][0].toString(10), '5')
    assert.strict.equal(a[0][1][2][1].toString(10), '6')
    assert.strict.equal(a[1].toString(10), '10')
  })
})

describe('decoding (uint[3][], uint)', function () {
  it('should work', function () {
    var data = `0000000000000000000000000000000000000000000000000000000000000040
000000000000000000000000000000000000000000000000000000000000000a
0000000000000000000000000000000000000000000000000000000000000002
0000000000000000000000000000000000000000000000000000000000000001
0000000000000000000000000000000000000000000000000000000000000002
0000000000000000000000000000000000000000000000000000000000000003
0000000000000000000000000000000000000000000000000000000000000004
0000000000000000000000000000000000000000000000000000000000000005
0000000000000000000000000000000000000000000000000000000000000006`
    var a = abi.rawDecode([ 'uint[3][]', 'uint' ], Buffer.from(data.replace(/\n/g, ''), 'hex'))
    assert.strict.equal(a.length, 2)
    assert.strict.equal(a[0][0][0].toString(10), '1')
    assert.strict.equal(a[0][0][1].toString(10), '2')
    assert.strict.equal(a[0][0][2].toString(10), '3')
    assert.strict.equal(a[0][1][0].toString(10), '4')
    assert.strict.equal(a[0][1][1].toString(10), '5')
    assert.strict.equal(a[0][1][2].toString(10), '6')
    assert.strict.equal(a[1].toString(10), '10')
  })
})

describe('decoding (uint[][3], uint)', function () {
  it('should work', function () {
    var data = `0000000000000000000000000000000000000000000000000000000000000080
00000000000000000000000000000000000000000000000000000000000000e0
0000000000000000000000000000000000000000000000000000000000000140
000000000000000000000000000000000000000000000000000000000000000a
0000000000000000000000000000000000000000000000000000000000000002
0000000000000000000000000000000000000000000000000000000000000001
0000000000000000000000000000000000000000000000000000000000000002
0000000000000000000000000000000000000000000000000000000000000002
0000000000000000000000000000000000000000000000000000000000000003
0000000000000000000000000000000000000000000000000000000000000004
0000000000000000000000000000000000000000000000000000000000000002
0000000000000000000000000000000000000000000000000000000000000005
0000000000000000000000000000000000000000000000000000000000000006`
    var a = abi.rawDecode([ 'uint[][3]', 'uint' ], Buffer.from(data.replace(/\n/g, ''), 'hex'))
    assert.strict.equal(a.length, 2)
    assert.strict.equal(a[1].toString(10), '10')
    assert.strict.equal(a[0][0][0].toString(10), '1')
    assert.strict.equal(a[0][0][1].toString(10), '2')
    assert.strict.equal(a[0][1][0].toString(10), '3')
    assert.strict.equal(a[0][1][1].toString(10), '4')
    assert.strict.equal(a[0][2][0].toString(10), '5')
    assert.strict.equal(a[0][2][1].toString(10), '6')
  })
})

// Homebrew tests for simpleEncode() family
describe('encoding contract function calls with simpleEncode', function () {
  it('should encode functions with arguments', function () {
    var a = abi.simpleEncode('inc(uint)', '0x13').toString('hex')
    var b = '812600df0000000000000000000000000000000000000000000000000000000000000013'
    assert.strict.equal(a, b)
  })
  it('should encode functions without arguments', function () {
    var a = abi.simpleEncode('meaningOfLife()').toString('hex')
    var b = '5353455a'
    assert.strict.equal(a, b)
  })
})
