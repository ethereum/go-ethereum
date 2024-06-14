const stripHexPrefix = require('../index.js');
const assert = require('chai').assert;

describe("isHexPrefixed", () => {
  describe("constructor", () => {
    it("should be function export", () => {
      assert.equal(typeof stripHexPrefix, 'function');
    });
  });

  describe("should function normall", () => {
    it('should stripHexPrefix strip prefix of valid strings', () => {
      assert.equal(stripHexPrefix('0xkdsfksfdkj'), 'kdsfksfdkj');
      assert.equal(stripHexPrefix('0xksfdkj'), 'ksfdkj');
      assert.equal(stripHexPrefix('0xkdsfdkj'), 'kdsfdkj');
      assert.equal(stripHexPrefix('0x23442sfdkj'), '23442sfdkj');
      assert.equal(stripHexPrefix('0xkdssdfssfdkj'), 'kdssdfssfdkj');
      assert.equal(stripHexPrefix('0xaaaasfdkj'), 'aaaasfdkj');
      assert.equal(stripHexPrefix('0xkdsdfsfsdfsdfsdfdkj'), 'kdsdfsfsdfsdfsdfdkj');
      assert.equal(stripHexPrefix('0x111dssdddj'), '111dssdddj');
      assert.equal(stripHexPrefix('0x'), '');
      assert.equal(stripHexPrefix(''), '');
      assert.equal(stripHexPrefix('-0xsdfsfd'), '-0xsdfsfd');
      assert.equal(stripHexPrefix('-0x'), '-0x');
    });

    it('should stripHexPrefix strip prefix of mix hexed strings', () => {
      assert.equal(stripHexPrefix('0xkdsfksfdkj'), 'kdsfksfdkj');
      assert.equal(stripHexPrefix('ksfdkj'), 'ksfdkj');
      assert.equal(stripHexPrefix('kdsfdkj'), 'kdsfdkj');
      assert.equal(stripHexPrefix('23442sfdkj'), '23442sfdkj');
      assert.equal(stripHexPrefix('0xkdssdfssfdkj'), 'kdssdfssfdkj');
      assert.equal(stripHexPrefix('aaaasfdkj'), 'aaaasfdkj');
      assert.equal(stripHexPrefix('kdsdfsfsdfsdfsdfdkj'), 'kdsdfsfsdfsdfsdfdkj');
      assert.equal(stripHexPrefix('111dssdddj'), '111dssdddj');
    });

    it('should stripHexPrefix bypass if not string', () => {
      assert.equal(stripHexPrefix(null), null);
      assert.equal(stripHexPrefix(undefined), undefined);
      assert.equal(stripHexPrefix(242423), 242423);
      assert.deepEqual(stripHexPrefix({}), {});
      assert.deepEqual(stripHexPrefix([]), []);
      assert.equal(stripHexPrefix(true), true);
    });
  });
});
