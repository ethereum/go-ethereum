/**
 * Mnemonist BitSet
 * =================
 *
 * JavaScript implementation of a fixed-size BitSet based upon a Uint32Array.
 *
 * Notes:
 *   - (i >> 5) is the same as ((i / 32) | 0)
 *   - (i & 0x0000001f) is the same as (i % 32)
 *   - I could use a Float64Array to store more in less blocks but I would lose
 *     the benefits of byte comparison to keep track of size without popcounts.
 */
var Iterator = require('obliterator/iterator'),
    bitwise = require('./utils/bitwise.js');

/**
 * BitSet.
 *
 * @constructor
 */
function BitSet(length) {

  // Properties
  this.length = length;
  this.clear();

  // Methods

  // Statics
}

/**
 * Method used to clear the bit set.
 *
 * @return {undefined}
 */
BitSet.prototype.clear = function() {

  // Properties
  this.size = 0;
  this.array = new Uint32Array(Math.ceil(this.length / 32));
};

/**
 * Method used to set the given bit's value.
 *
 * @param  {number} index - Target bit index.
 * @param  {number} value - Value to set.
 * @return {BitSet}
 */
BitSet.prototype.set = function(index, value) {
  var byteIndex = index >> 5,
      pos = index & 0x0000001f,
      oldBytes = this.array[byteIndex],
      newBytes;

  if (value === 0 || value === false)
    newBytes = this.array[byteIndex] &= ~(1 << pos);
  else
    newBytes = this.array[byteIndex] |= (1 << pos);

  // The operands of all bitwise operators are converted to *signed* 32-bit integers.
  // Source: https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Operators/Bitwise_Operators#Signed_32-bit_integers
  // Shifting by 31 changes the sign (i.e. 1 << 31 = -2147483648).
  // Therefore, get unsigned representation by applying '>>> 0'.
  newBytes = newBytes >>> 0;

  // Updating size
  if (newBytes > oldBytes)
    this.size++;
  else if (newBytes < oldBytes)
    this.size--;

  return this;
};

/**
* Method used to reset the given bit's value.
*
* @param  {number} index - Target bit index.
* @return {BitSet}
*/
BitSet.prototype.reset = function(index) {
  var byteIndex = index >> 5,
      pos = index & 0x0000001f,
      oldBytes = this.array[byteIndex],
      newBytes;

  newBytes = this.array[byteIndex] &= ~(1 << pos);

  // Updating size
  if (newBytes < oldBytes)
    this.size--;

  return this;
};

/**
 * Method used to flip the value of the given bit.
 *
 * @param  {number} index - Target bit index.
 * @return {BitSet}
 */
BitSet.prototype.flip = function(index) {
  var byteIndex = index >> 5,
      pos = index & 0x0000001f,
      oldBytes = this.array[byteIndex];

  var newBytes = this.array[byteIndex] ^= (1 << pos);

  // Get unsigned representation.
  newBytes = newBytes >>> 0;

  // Updating size
  if (newBytes > oldBytes)
    this.size++;
  else if (newBytes < oldBytes)
    this.size--;

  return this;
};

/**
 * Method used to get the given bit's value.
 *
 * @param  {number} index - Target bit index.
 * @return {number}
 */
BitSet.prototype.get = function(index) {
  var byteIndex = index >> 5,
      pos = index & 0x0000001f;

  return (this.array[byteIndex] >> pos) & 1;
};

/**
 * Method used to test the given bit's value.
 *
 * @param  {number} index - Target bit index.
 * @return {BitSet}
 */
BitSet.prototype.test = function(index) {
  return Boolean(this.get(index));
};

/**
 * Method used to return the number of 1 from the beginning of the set up to
 * the ith index.
 *
 * @param  {number} i - Ith index (cannot be > length).
 * @return {number}
 */
BitSet.prototype.rank = function(i) {
  if (this.size === 0)
    return 0;

  var byteIndex = i >> 5,
      pos = i & 0x0000001f,
      r = 0;

  // Accessing the bytes before the last one
  for (var j = 0; j < byteIndex; j++)
    r += bitwise.table8Popcount(this.array[j]);

  // Handling masked last byte
  var maskedByte = this.array[byteIndex] & ((1 << pos) - 1);

  r += bitwise.table8Popcount(maskedByte);

  return r;
};

/**
 * Method used to return the position of the rth 1 in the set or -1 if the
 * set is empty.
 *
 * Note: usually select is implemented using binary search over rank but I
 * tend to think the following linear implementation is faster since here
 * rank is O(n) anyway.
 *
 * @param  {number} r - Rth 1 to select (should be < length).
 * @return {number}
 */
BitSet.prototype.select = function(r) {
  if (this.size === 0)
    return -1;

  // TODO: throw?
  if (r >= this.length)
    return -1;

  var byte,
      b = 32,
      p = 0,
      c = 0;

  for (var i = 0, l = this.array.length; i < l; i++) {
    byte = this.array[i];

    // The byte is empty, let's continue
    if (byte === 0)
      continue;

    // TODO: This branching might not be useful here
    if (i === l - 1)
      b = this.length % 32 || 32;

    // TODO: popcount should speed things up here

    for (var j = 0; j < b; j++, p++) {
      c += (byte >> j) & 1;

      if (c === r)
        return p;
    }
  }
};

/**
 * Method used to iterate over the bit set's values.
 *
 * @param  {function}  callback - Function to call for each item.
 * @param  {object}    scope    - Optional scope.
 * @return {undefined}
 */
BitSet.prototype.forEach = function(callback, scope) {
  scope = arguments.length > 1 ? scope : this;

  var length = this.length,
      byte,
      bit,
      b = 32;

  for (var i = 0, l = this.array.length; i < l; i++) {
    byte = this.array[i];

    if (i === l - 1)
      b = length % 32 || 32;

    for (var j = 0; j < b; j++) {
      bit = (byte >> j) & 1;

      callback.call(scope, bit, i * 32 + j);
    }
  }
};

/**
 * Method used to create an iterator over a set's values.
 *
 * @return {Iterator}
 */
BitSet.prototype.values = function() {
  var length = this.length,
      inner = false,
      byte,
      bit,
      array = this.array,
      l = array.length,
      i = 0,
      j = -1,
      b = 32;

  return new Iterator(function next() {
    if (!inner) {

      if (i >= l)
        return {
          done: true
        };

      if (i === l - 1)
        b = length % 32 || 32;

      byte = array[i++];
      inner = true;
      j = -1;
    }

    j++;

    if (j >= b) {
      inner = false;
      return next();
    }

    bit = (byte >> j) & 1;

    return {
      value: bit
    };
  });
};

/**
 * Method used to create an iterator over a set's entries.
 *
 * @return {Iterator}
 */
BitSet.prototype.entries = function() {
  var length = this.length,
      inner = false,
      byte,
      bit,
      array = this.array,
      index,
      l = array.length,
      i = 0,
      j = -1,
      b = 32;

  return new Iterator(function next() {
    if (!inner) {

      if (i >= l)
        return {
          done: true
        };

      if (i === l - 1)
        b = length % 32 || 32;

      byte = array[i++];
      inner = true;
      j = -1;
    }

    j++;
    index = (~-i) * 32 + j;

    if (j >= b) {
      inner = false;
      return next();
    }

    bit = (byte >> j) & 1;

    return {
      value: [index, bit]
    };
  });
};

/**
 * Attaching the #.values method to Symbol.iterator if possible.
 */
if (typeof Symbol !== 'undefined')
  BitSet.prototype[Symbol.iterator] = BitSet.prototype.values;

/**
 * Convenience known methods.
 */
BitSet.prototype.inspect = function() {
  var proxy = new Uint8Array(this.length);

  this.forEach(function(bit, i) {
    proxy[i] = bit;
  });

  // Trick so that node displays the name of the constructor
  Object.defineProperty(proxy, 'constructor', {
    value: BitSet,
    enumerable: false
  });

  return proxy;
};

if (typeof Symbol !== 'undefined')
  BitSet.prototype[Symbol.for('nodejs.util.inspect.custom')] = BitSet.prototype.inspect;

BitSet.prototype.toJSON = function() {
  return Array.from(this.array);
};

/**
 * Exporting.
 */
module.exports = BitSet;
