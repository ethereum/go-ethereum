/**
 * Mnemonist Bitwise Helpers
 * ==========================
 *
 * Miscellaneous helpers helping with bitwise operations.
 */

/**
 * Takes a 32 bits integer and returns its MSB using SWAR strategy.
 *
 * @param  {number} x - Target number.
 * @return {number}
 */
function msb32(x) {
  x |= (x >> 1);
  x |= (x >> 2);
  x |= (x >> 4);
  x |= (x >> 8);
  x |= (x >> 16);

  return (x & ~(x >> 1));
}
exports.msb32 = msb32;

/**
 * Takes a byte and returns its MSB using SWAR strategy.
 *
 * @param  {number} x - Target number.
 * @return {number}
 */
function msb8(x) {
  x |= (x >> 1);
  x |= (x >> 2);
  x |= (x >> 4);

  return (x & ~(x >> 1));
}
exports.msb8 = msb8;

/**
 * Takes a number and return bit at position.
 *
 * @param  {number} x   - Target number.
 * @param  {number} pos - Position.
 * @return {number}
 */
exports.test = function(x, pos) {
  return (x >> pos) & 1;
};

/**
 * Compare two bytes and return their critical bit.
 *
 * @param  {number} a - First byte.
 * @param  {number} b - Second byte.
 * @return {number}
 */
exports.criticalBit8 = function(a, b) {
  return msb8(a ^ b);
};

exports.criticalBit8Mask = function(a, b) {
  return (~msb8(a ^ b) >>> 0) & 0xff;
};

exports.testCriticalBit8 = function(x, mask) {
  return (1 + (x | mask)) >> 8;
};

exports.criticalBit32Mask = function(a, b) {
  return (~msb32(a ^ b) >>> 0) & 0xffffffff;
};

/**
 * Takes a 32 bits integer and returns its population count (number of 1 of
 * the binary representation).
 *
 * @param  {number} x - Target number.
 * @return {number}
 */
exports.popcount = function(x) {
  x -= x >> 1 & 0x55555555;
  x = (x & 0x33333333) + (x >> 2 & 0x33333333);
  x = x + (x >> 4) & 0x0f0f0f0f;
  x += x >> 8;
  x += x >> 16;
  return x & 0x7f;
};

/**
 * Slightly faster popcount function based on a precomputed table of 8bits
 * words.
 *
 * @param  {number} x - Target number.
 * @return {number}
 */
var TABLE8 = new Uint8Array(Math.pow(2, 8));

for (var i = 0, l = TABLE8.length; i < l; i++)
  TABLE8[i] = exports.popcount(i);

exports.table8Popcount = function(x) {
  return (
    TABLE8[x & 0xff] +
    TABLE8[(x >> 8) & 0xff] +
    TABLE8[(x >> 16) & 0xff] +
    TABLE8[(x >> 24) & 0xff]
  );
};
