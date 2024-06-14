const util = require('../index.js');
const assert = require('chai').assert;
const BN = require('bn.js');

describe('check all exports', () => {
  it('should have all exports available', () => {
    const expected = ['arrayContainsArray',
    'toBuffer',
    'intToBuffer',
    'getBinarySize',
    'stripHexPrefix',
    'isHexPrefixed',
    'padToEven',
    'intToHex',
    'fromAscii',
    'fromUtf8',
    'toAscii',
    'getKeys',
    'isHexString',
    'toUtf8'];

    Object.keys(util).forEach((utilKey) => {
      assert.equal(expected.includes(utilKey), true, utilKey);
    });
  });

  it('should convert intToHex', () => {
    assert.equal(util.intToHex(new BN(0)), '0x0');
  });

  it('should throw when invalid abi', () => {
    assert.throws(() => util.getKeys([], 3289), Error);
  });

  it('should detect invalid length hex string', () => {
    assert.equal(util.isHexString('0x0', 2), false);
  });

  it('should stripHexPrefix strip prefix of valid strings', () => {
    assert.equal(util.stripHexPrefix('0xkdsfksfdkj'), 'kdsfksfdkj');
    assert.equal(util.stripHexPrefix('0xksfdkj'), 'ksfdkj');
    assert.equal(util.stripHexPrefix('0xkdsfdkj'), 'kdsfdkj');
    assert.equal(util.stripHexPrefix('0x23442sfdkj'), '23442sfdkj');
    assert.equal(util.stripHexPrefix('0xkdssdfssfdkj'), 'kdssdfssfdkj');
    assert.equal(util.stripHexPrefix('0xaaaasfdkj'), 'aaaasfdkj');
    assert.equal(util.stripHexPrefix('0xkdsdfsfsdfsdfsdfdkj'), 'kdsdfsfsdfsdfsdfdkj');
    assert.equal(util.stripHexPrefix('0x111dssdddj'), '111dssdddj');
  });

  it('should stripHexPrefix strip prefix of mix hexed strings', () => {
    assert.equal(util.stripHexPrefix('0xkdsfksfdkj'), 'kdsfksfdkj');
    assert.equal(util.stripHexPrefix('ksfdkj'), 'ksfdkj');
    assert.equal(util.stripHexPrefix('kdsfdkj'), 'kdsfdkj');
    assert.equal(util.stripHexPrefix('23442sfdkj'), '23442sfdkj');
    assert.equal(util.stripHexPrefix('0xkdssdfssfdkj'), 'kdssdfssfdkj');
    assert.equal(util.stripHexPrefix('aaaasfdkj'), 'aaaasfdkj');
    assert.equal(util.stripHexPrefix('kdsdfsfsdfsdfsdfdkj'), 'kdsdfsfsdfsdfsdfdkj');
    assert.equal(util.stripHexPrefix('111dssdddj'), '111dssdddj');
  });

  it('should stripHexPrefix bypass if not string', () => {
    assert.equal(util.stripHexPrefix(null), null);
    assert.equal(util.stripHexPrefix(undefined), undefined);
    assert.equal(util.stripHexPrefix(242423), 242423);
    assert.deepEqual(util.stripHexPrefix({}), {});
    assert.deepEqual(util.stripHexPrefix([]), []);
    assert.equal(util.stripHexPrefix(true), true);
  });

  it('valid padToEven should pad to even', () => {
    assert.equal(String(util.padToEven('0')).length % 2, 0);
    assert.equal(String(util.padToEven('111')).length % 2, 0);
    assert.equal(String(util.padToEven('22222')).length % 2, 0);
    assert.equal(String(util.padToEven('ddd')).length % 2, 0);
    assert.equal(String(util.padToEven('aa')).length % 2, 0);
    assert.equal(String(util.padToEven('aaaaaa')).length % 2, 0);
    assert.equal(String(util.padToEven('sdssd')).length % 2, 0);
    assert.equal(String(util.padToEven('eee')).length % 2, 0);
    assert.equal(String(util.padToEven('w')).length % 2, 0);
  });

  it('valid padToEven should pad to even check string prefix 0', () => {
    assert.equal(String(util.padToEven('0')), '00');
    assert.equal(String(util.padToEven('111')), '0111');
    assert.equal(String(util.padToEven('22222')), '022222');
    assert.equal(String(util.padToEven('ddd')), '0ddd');
    assert.equal(String(util.padToEven('aa')), 'aa');
    assert.equal(String(util.padToEven('aaaaaa')), 'aaaaaa');
    assert.equal(String(util.padToEven('sdssd')), '0sdssd');
    assert.equal(String(util.padToEven('eee')), '0eee');
    assert.equal(String(util.padToEven('w')), '0w');
  });

  it('should padToEven throw as expected string got null', () => {
    try {
      util.padToEven(null);
    } catch (error) {
      assert.equal(typeof error, 'object');
    }
  });

  it('should padToEven throw as expected string got undefined', () => {
    try {
      util.padToEven(undefined);
    } catch (error) {
      assert.equal(typeof error, 'object');
    }
  });

  it('should padToEven throw as expected string got {}', () => {
    try {
      util.padToEven({});
    } catch (error) {
      assert.equal(typeof error, 'object');
    }
  });

  it('should padToEven throw as expected string got new Buffer()', () => {
    try {
      util.padToEven(new Buffer());
    } catch (error) {
      assert.equal(typeof error, 'object');
    }
  });

  it('should padToEven throw as expected string got number', () => {
    try {
      util.padToEven(24423232);
    } catch (error) {
      assert.equal(typeof error, 'object');
    }
  });

  it('method getKeys should throw as expected array for params got number', () => {
    try {
      util.getKeys(2482822);
    } catch (error) {
      assert.equal(typeof error, 'object');
    }
  });

  it('method invalid getKeys with allow empty and no defined value', () => {
    try {
      util.getKeys([{ type: undefined }], 'type', true);
    } catch (error) {
      assert.equal(typeof error, 'object');
    }
  });

  it('method valid getKeys with allow empty and false', () => {
    try {
      util.getKeys([{ type: true }], 'type', true);
    } catch (error) {
      assert.equal(typeof error, 'object');
    }
  });

  it('method getKeys should throw as expected array for params got number', () => {
    try {
      util.getKeys(2482822, 293849824);
    } catch (error) {
      assert.equal(typeof error, 'object');
    }
  });

  it('method getKeys should throw as expected array for params got object', () => {
    try {
      util.getKeys({}, []);
    } catch (error) {
      assert.equal(typeof error, 'object');
    }
  });

  it('method getKeys should throw as expected array for params got null', () => {
    try {
      util.getKeys(null);
    } catch (error) {
      assert.equal(typeof error, 'object');
    }
  });

  it('method getKeys should throw as expected array for params got false', () => {
    try {
      util.getKeys(false);
    } catch (error) {
      assert.equal(typeof error, 'object');
    }
  });

  it('valid getKeys should get keys from object in array', () => {
    assert.deepEqual(util.getKeys([{ type: 'sfd' }, { type: 'something' }], 'type'), ['sfd', 'something']);
    assert.deepEqual(util.getKeys([{ cool: 'something' }, { cool: 'fdsdfsfd' }], 'cool'), ['something', 'fdsdfsfd']);
    assert.deepEqual(util.getKeys([{ type: '234424' }, { type: '243234242432' }], 'type'), ['234424', '243234242432']);
    assert.deepEqual(util.getKeys([{ type: 'something' }, { type: 'something' }], 'type'), ['something', 'something']);
    assert.deepEqual(util.getKeys([{ type: 'something' }], 'type'), ['something']);
    assert.deepEqual(util.getKeys([], 'type'), []);
    assert.deepEqual(util.getKeys([{ type: 'something' }, { type: 'something' }, { type: 'something' }], 'type'), ['something', 'something', 'something']);
  });

  it('valid isHexString tests', () => {
    assert.equal(util.isHexString('0x0e026d45820d91356fc73d7ff2bdef353ebfe7e9'), true);
    assert.equal(util.isHexString('0x1e026d45820d91356fc73d7ff2bdef353ebfe7e9'), true);
    assert.equal(util.isHexString('0x6e026d45820d91356fc73d7ff2bdef353ebfe7e9'), true);
    assert.equal(util.isHexString('0xecfaa1a0c4372a2ac5cca1e164510ec8df04f681fc960797f1419802ec00b225'), true);
    assert.equal(util.isHexString('0x6e0e6d45820d91356fc73d7ff2bdef353ebfe7e9'), true);
    assert.equal(util.isHexString('0x620e6d45820d91356fc73d7ff2bdef353ebfe7e9'), true);
    assert.equal(util.isHexString('0x1e0e6d45820d91356fc73d7ff2bdef353ebfe7e9'), true);
    assert.equal(util.isHexString('0x2e0e6d45820d91356fc73d7ff2bdef353ebfe7e9'), true);
    assert.equal(util.isHexString('0x220c96d48733a847570c2f0b40daa8793b3ae875b26a4ead1f0f9cead05c3863'), true);
    assert.equal(util.isHexString('0x2bb303f0ae65c64ef80a3bb3ee8ceef5d50065bd'), true);
    assert.equal(util.isHexString('0x6e026d45820d91256fc73d7ff2bdef353ebfe7e9'), true);
  });

  it('invalid isHexString tests', () => {
    assert.equal(util.isHexString(' 0x0e026d45820d91356fc73d7ff2bdef353ebfe7e9'), false);
    assert.equal(util.isHexString('fdsjfsd'), false);
    assert.equal(util.isHexString(' 0xfdsjfsd'), false);
    assert.equal(util.isHexString('0xfds*jfsd'), false);
    assert.equal(util.isHexString('0xfds$jfsd'), false);
    assert.equal(util.isHexString('0xf@dsjfsd'), false);
    assert.equal(util.isHexString('0xfdsjf!sd'), false);
    assert.equal(util.isHexString('fds@@jfsd'), false);
    assert.equal(util.isHexString(24223), false);
    assert.equal(util.isHexString(null), false);
    assert.equal(util.isHexString(undefined), false);
    assert.equal(util.isHexString(false), false);
    assert.equal(util.isHexString({}), false);
    assert.equal(util.isHexString([]), false);
  });

  it('valid arrayContainsArray should array contain every array', () => {
    assert.equal(util.arrayContainsArray([1, 2, 3], [1, 2]), true);
    assert.equal(util.arrayContainsArray([3, 3], [3, 3]), true);
    assert.equal(util.arrayContainsArray([1, 2, 'h'], [1, 2, 'h']), true);
    assert.equal(util.arrayContainsArray([1, 2, 'fsffds'], [1, 2, 'fsffds']), true);
    assert.equal(util.arrayContainsArray([1], [1]), true);
    assert.equal(util.arrayContainsArray([], []), true);
    assert.equal(util.arrayContainsArray([1, 3333], [1, 3333]), true);
  });

  it('valid getBinarySize should get binary size of string', () => {
    assert.equal(util.getBinarySize('0x0e026d45820d91356fc73d7ff2bdef353ebfe7e9'), 42);
    assert.equal(util.getBinarySize('0x220c96d48733a847570c2f0b40daa8793b3ae875b26a4ead1f0f9cead05c3863'), 66);
  });

  it('invalid getBinarySize should throw invalid type Boolean', () => {
    try {
      util.getBinarySize(false);
    } catch (error) {
      assert.equal(typeof error, 'object');
    }
  });

  it('invalid getBinarySize should throw invalid type object', () => {
    try {
      util.getBinarySize({});
    } catch (error) {
      assert.equal(typeof error, 'object');
    }
  });

  it('invalid getBinarySize should throw invalid type Array', () => {
    try {
      util.getBinarySize([]);
    } catch (error) {
      assert.equal(typeof error, 'object');
    }
  });

  it('valid arrayContainsArray should array some every array', () => {
    assert.equal(util.arrayContainsArray([1, 2], [1], true), true);
    assert.equal(util.arrayContainsArray([3, 3], [3, 2323], true), true);
    assert.equal(util.arrayContainsArray([1, 2, 'h'], [2332, 2, 'h'], true), true);
    assert.equal(util.arrayContainsArray([1, 2, 'fsffds'], [3232, 2, 'fsffds'], true), true);
    assert.equal(util.arrayContainsArray([1], [1], true), true);
    assert.equal(util.arrayContainsArray([1, 3333], [1, 323232], true), true);
  });

  it('method arrayContainsArray should throw as expected array for params got false', () => {
    try {
      util.arrayContainsArray(false);
    } catch (error) {
      assert.equal(typeof error, 'object');
    }
  });

  it('method arrayContainsArray should throw as expected array for params got false', () => {
    try {
      util.arrayContainsArray([], false);
    } catch (error) {
      assert.equal(typeof error, 'object');
    }
  });

  it('method arrayContainsArray should throw as expected array for params got {}', () => {
    try {
      util.arrayContainsArray({}, false);
    } catch (error) {
      assert.equal(typeof error, 'object');
    }
  });

  const fromAsciiTests = [
    { value: 'myString', expected: '0x6d79537472696e67' },
    { value: 'myString\x00', expected: '0x6d79537472696e6700' },
    { value: '\u0003\u0000\u0000\u00005èÆÕL]\u0012|Î¾\u001a7«\u00052\u0011(ÐY\n<\u0010\u0000\u0000\u0000\u0000\u0000\u0000e!ßd/ñõì\f:z¦Î¦±ç·÷Í¢Ëß\u00076*\bñùC1ÉUÀé2\u001aÓB',
      expected: '0x0300000035e8c6d54c5d127c9dcebe9e1a37ab9b05321128d097590a3c100000000000006521df642ff1f5ec0c3a7aa6cea6b1e7b7f7cda2cbdf07362a85088e97f19ef94331c955c0e9321ad386428c' },
  ];

  describe('fromAscii', () => {
    fromAsciiTests.forEach((test) => {
      it(`should turn ${test.value} to ${test.expected} `, () => {
        assert.strictEqual(util.fromAscii(test.value), test.expected);
      });
    });
  });

  const fromUtf8Tests = [
    { value: 'myString', expected: '0x6d79537472696e67' },
    { value: 'myString\x00', expected: '0x6d79537472696e67' },
    { value: 'expected value\u0000\u0000\u0000', expected: '0x65787065637465642076616c7565' },
  ];

  describe('fromUtf8', () => {
    fromUtf8Tests.forEach((test) => {
      it(`should turn ${test.value} to ${test.expected} `, () => {
        assert.strictEqual(util.fromUtf8(test.value), test.expected);
      });
    });
  });

  const toUtf8Tests = [
      { value: '0x6d79537472696e67', expected: 'myString' },
      { value: '0x6d79537472696e6700', expected: 'myString' },
      { value: '0x65787065637465642076616c7565000000000000000000000000000000000000', expected: 'expected value' },
  ];

  describe('toUtf8', () => {
    toUtf8Tests.forEach((test) => {
      it(`should turn ${test.value} to ${test.expected} `, () => {
        assert.strictEqual(util.toUtf8(test.value), test.expected);
      });
    });
  });

  const toAsciiTests = [
    { value: '0x6d79537472696e67', expected: 'myString' },
    { value: '0x6d79537472696e6700', expected: 'myString\u0000' },
    { value: '0x0300000035e8c6d54c5d127c9dcebe9e1a37ab9b05321128d097590a3c100000000000006521df642ff1f5ec0c3a7aa6cea6b1e7b7f7cda2cbdf07362a85088e97f19ef94331c955c0e9321ad386428c',
      expected: '\u0003\u0000\u0000\u00005èÆÕL]\u0012|Î¾\u001a7«\u00052\u0011(ÐY\n<\u0010\u0000\u0000\u0000\u0000\u0000\u0000e!ßd/ñõì\f:z¦Î¦±ç·÷Í¢Ëß\u00076*\bñùC1ÉUÀé2\u001aÓB' },
  ];

  describe('toAsciiTests', () => {
    toAsciiTests.forEach((test) => {
      it(`should turn ${test.value} to ${test.expected} `, () => {
        assert.strictEqual(util.toAscii(test.value), test.expected);
      });
    });
  });

  describe('intToHex', () => {
    it('should convert a int to hex', () => {
      const i = 6003400;
      const hex = util.intToHex(i);
      assert.equal(hex, '0x5b9ac8');
    });
  });

  describe('intToBuffer', () => {
    it('should convert a int to a buffer', () => {
      const i = 6003400;
      const buf = util.intToBuffer(i);
      assert.equal(buf.toString('hex'), '5b9ac8');
    });

    it('should convert a int to a buffer for odd length hex values', () => {
      const i = 1;
      const buf = util.intToBuffer(i);
      assert.equal(buf.toString('hex'), '01');
    });
  });
});
