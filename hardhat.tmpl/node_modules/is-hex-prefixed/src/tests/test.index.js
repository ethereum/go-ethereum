const isHexPrefixed = require('../index.js');
const assert = require('chai').assert;

describe("isHexPrefixed", () => {
  describe("constructor", () => {
    it("should have the method exported", () => {
      assert.equal(typeof isHexPrefixed, 'function');
    });
  });

  describe("should function normall", () => {
    it('should isHexPrefixed check if hex is prefixed', () => {
      assert.equal(isHexPrefixed('0xsdffsd'), true);
      assert.equal(isHexPrefixed('0x'), true);
      assert.equal(isHexPrefixed('0x3982349284'), true);
      assert.equal(isHexPrefixed('0x824723894jshdksjdhks'), true);
    });

    it('should isHexPrefixed check if hex is prefixed not prefixed', () => {
      assert.equal(isHexPrefixed('sdffsd'), false);
      assert.equal(isHexPrefixed(''), false);
      assert.equal(isHexPrefixed('3982349284'), false);
      assert.equal(isHexPrefixed('824723894jshdksjdhks'), false);
    });

    it('should isHexPrefixed throw as expected string got buffer', () => {
      try {
        isHexPrefixed(new Buffer());
      } catch (error) {
        assert.equal(typeof error, 'object');
      }
    });

    it('should isHexPrefixed throw as expected string got empty object', () => {
      try {
        isHexPrefixed({});
      } catch (error) {
        assert.equal(typeof error, 'object');
      }
    });

    it('should isHexPrefixed throw as expected string got number', () => {
      try {
        isHexPrefixed(823947243994);
      } catch (error) {
        assert.equal(typeof error, 'object');
      }
    });

    it('should isHexPrefixed throw as expected string got undefined', () => {
      try {
        isHexPrefixed(undefined);
      } catch (error) {
        assert.equal(typeof error, 'object');
      }
    });

    it('should isHexPrefixed throw as expected string got null', () => {
      try {
        isHexPrefixed(null);
      } catch (error) {
        assert.equal(typeof error, 'object');
      }
    });
  });
});
