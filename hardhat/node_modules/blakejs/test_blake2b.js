const test = require('tape')
const blake2b = require('./blake2b')
const util = require('./util')
const fs = require('fs')

const blake2bHex = blake2b.blake2bHex

test('BLAKE2b basic', function (t) {
  // From the example computation in the RFC
  t.equal(blake2bHex('abc'),
    'ba80a53f981c4d0d6a2797b69f12f6e94c212f14685ac4b74b12bb6fdbffa2d1' +
    '7d87c5392aab792dc252d5de4533cc9518d38aa8dbf1925ab92386edd4009923')

  t.equal(blake2bHex(''),
    '786a02f742015903c6c6fd852552d272912f4740e15847618a86e217f71f5419' +
    'd25e1031afee585313896444934eb04b903a685b1448b755d56f701afe9be2ce')

  t.equal(blake2bHex('The quick brown fox jumps over the lazy dog'),
    'a8add4bdddfd93e4877d2746e62817b116364a1fa7bc148d95090bc7333b3673' +
    'f82401cf7aa2e4cb1ecd90296e3f14cb5413f8ed77be73045b13914cdcd6a918')

  // with personal
  t.equal(blake2bHex('abc', null, null, 'salt123456789012', 'personal12345678'),
    'f6860a4cb1baadf6e3cc36c81b974d1167fbcd36dfbe55f3c2a561ab1863fb3d' +
    '2afc40ea9a95c3ad46f100e2a65c3a3b841d616fda1ea45b47074baccfe3b644')

  t.end()
})

test('Input types', function (t) {
  // Supports string, Uint8Array, and Buffer inputs
  // We already verify that blake2bHex('abc') produces the correct hash above
  t.equal(blake2bHex(new Uint8Array([97, 98, 99])), blake2bHex('abc'))
  t.equal(blake2bHex(Buffer.from([97, 98, 99])), blake2bHex('abc'))
  t.end()
})

test('BLAKE2b generated test vectors', function (t) {
  const contents = fs.readFileSync('generated_test_vectors.txt', 'utf8')
  contents.split('\n').forEach(function (line) {
    if (line.length === 0) {
      return
    }
    const parts = line.split('\t')
    const inputHex = parts[0]
    const keyHex = parts[1]
    const outLen = parts[2]
    const outHex = parts[3]

    t.equal(blake2bHex(hexToBytes(inputHex), hexToBytes(keyHex), outLen), outHex)
  })
  t.end()
})

test('BLAKE2b performance', function (t) {
  const N = 1 << 22 // number of bytes to hash
  const RUNS = 3 // how often to repeat, to allow JIT to finish

  console.log('Benchmarking BLAKE2b(' + (N >> 20) + ' MB input)')
  util.testSpeed(blake2bHex, N, RUNS)
  t.end()
})

test('Byte counter should support values up to 2**53', function (t) {
  const testCases = [
    { t: 1, a0: 1, a1: 0 },
    { t: 0xffffffff, a0: 0xffffffff, a1: 0 },
    { t: 0x100000000, a0: 0, a1: 1 },
    { t: 0x123456789abcd, a0: 0x6789abcd, a1: 0x12345 },
    // test 2**53 - 1
    { t: 0x1fffffffffffff, a0: 0xffffffff, a1: 0x1fffff }]

  testCases.forEach(function (testCase) {
    const arr = new Uint32Array([0, 0])

    // test the code that's inlined in both blake2s.js and blake2b.js
    // to make sure it splits byte counters up to 2**53 into uint32s correctly
    arr[0] ^= testCase.t
    arr[1] ^= (testCase.t / 0x100000000)

    t.equal(arr[0], testCase.a0)
    t.equal(arr[1], testCase.a1)
  })
  t.end()
})

function hexToBytes (hex) {
  const ret = new Uint8Array(hex.length / 2)
  for (let i = 0; i < ret.length; i++) {
    ret[i] = parseInt(hex.substring(i * 2, i * 2 + 2), 16)
  }
  return ret
}
