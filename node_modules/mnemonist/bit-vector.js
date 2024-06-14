/**
 * Mnemonist BitVector
 * ====================
 *
 * JavaScript implementation of a dynamic BitSet based upon a Uint32Array.
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
 * Constants.
 */
var DEFAULT_GROWING_POLICY = function(capacity) {
  return Math.max(1, Math.ceil(capacity * 1.5));
};

/**
 * Helpers.
 */
function createByteArray(capacity) {
  return new Uint32Array(Math.ceil(capacity / 32));
}

/**
 * BitVector.
 *
 * @constructor
 */
function BitVector(initialLengthOrOptions) {
  var initialLength = initialLengthOrOptions || 0,
      policy = DEFAULT_GROWING_POLICY;

  if (typeof initialLengthOrOptions === 'object') {
    initialLength = (
      initialLengthOrOptions.initialLength ||
      initialLengthOrOptions.initialCapacity ||
      0
    );
    policy = initialLengthOrOptions.policy || policy;
  }

  this.size = 0;
  this.length = initialLength;
  this.capacity = Math.ceil(this.length / 32) * 32;
  this.policy = policy;
  this.array = createByteArray(this.capacity);
}

/**
 * Method used to set the given bit's value.
 *
 * @param  {number} index - Target bit index.
 * @param  {number|boolean} value - Value to set.
 * @return {BitVector}
 */
BitVector.prototype.set = function(index, value) {

  // Out of bounds?
  if (this.length < index)
    throw new Error('BitVector.set: index out of bounds.');

  var byteIndex = index >> 5,
      pos = index & 0x0000001f,
      oldBytes = this.array[byteIndex],
      newBytes;

  if (value === 0 || value === false)
    newBytes = this.array[byteIndex] &= ~(1 << pos);
  else
    newBytes = this.array[byteIndex] |= (1 << pos);

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
* Method used to reset the given bit's value.
*
* @param  {number} index - Target bit index.
* @return {BitVector}
*/
BitVector.prototype.reset = function(index) {
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
 * @return {BitVector}
 */
BitVector.prototype.flip = function(index) {
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
 * Method used to apply the growing policy.
 *
 * @param  {number} [override] - Override capacity.
 * @return {number}
 */
BitVector.prototype.applyPolicy = function(override) {
  var newCapacity = this.policy(override || this.capacity);

  if (typeof newCapacity !== 'number' || newCapacity < 0)
    throw new Error('mnemonist/bit-vector.applyPolicy: policy returned an invalid value (expecting a positive integer).');

  if (newCapacity <= this.capacity)
    throw new Error('mnemonist/bit-vector.applyPolicy: policy returned a less or equal capacity to allocate.');

  // TODO: we should probably check that the returned number is an integer

  // Ceil to nearest 32
  return Math.ceil(newCapacity / 32) * 32;
};

/**
 * Method used to reallocate the underlying array.
 *
 * @param  {number}       capacity - Target capacity.
 * @return {BitVector}
 */
BitVector.prototype.reallocate = function(capacity) {
  var virtualCapacity = capacity;

  capacity = Math.ceil(capacity / 32) * 32;

  if (virtualCapacity < this.length)
    this.length = virtualCapacity;

  if (capacity === this.capacity)
    return this;

  var oldArray = this.array;

  var storageLength = capacity / 32;

  if (storageLength === this.array.length)
    return this;

  if (storageLength > this.array.length) {
    this.array = new Uint32Array(storageLength);
    this.array.set(oldArray, 0);
  }
  else {
    this.array = oldArray.slice(0, storageLength);
  }

  this.capacity = capacity;

  return this;
};

/**
 * Method used to grow the array.
 *
 * @param  {number}       [capacity] - Optional capacity to match.
 * @return {BitVector}
 */
BitVector.prototype.grow = function(capacity) {
  var newCapacity;

  if (typeof capacity === 'number') {

    if (this.capacity >= capacity)
      return this;

    // We need to match the given capacity
    newCapacity = this.capacity;

    while (newCapacity < capacity)
      newCapacity = this.applyPolicy(newCapacity);

    this.reallocate(newCapacity);

    return this;
  }

  // We need to run the policy once
  newCapacity = this.applyPolicy();
  this.reallocate(newCapacity);

  return this;
};

/**
 * Method used to resize the array. Won't deallocate.
 *
 * @param  {number}       length - Target length.
 * @return {BitVector}
 */
BitVector.prototype.resize = function(length) {
  if (length === this.length)
    return this;

  if (length < this.length) {
    this.length = length;
    return this;
  }

  this.length = length;
  this.reallocate(length);

  return this;
};

/**
 * Method used to push a value in the set.
 *
 * @param  {number|boolean} value
 * @return {BitVector}
 */
BitVector.prototype.push = function(value) {
  if (this.capacity === this.length)
    this.grow();

  if (value === 0 || value === false)
    return ++this.length;

  this.size++;

  var index = this.length++,
      byteIndex = index >> 5,
      pos = index & 0x0000001f;

  this.array[byteIndex] |= (1 << pos);

  return this.length;
};

/**
 * Method used to pop the last value of the set.
 *
 * @return {number} - The popped value.
 */
BitVector.prototype.pop = function() {
  if (this.length === 0)
    return;

  var index = --this.length;

  var byteIndex = index >> 5,
      pos = index & 0x0000001f;

  return (this.array[byteIndex] >> pos) & 1;
};

/**
 * Method used to get the given bit's value.
 *
 * @param  {number} index - Target bit index.
 * @return {number}
 */
BitVector.prototype.get = function(index) {
  if (this.length < index)
    return undefined;

  var byteIndex = index >> 5,
      pos = index & 0x0000001f;

  return (this.array[byteIndex] >> pos) & 1;
};

/**
 * Method used to test the given bit's value.
 *
 * @param  {number} index - Target bit index.
 * @return {BitVector}
 */
BitVector.prototype.test = function(index) {
  if (this.length < index)
    return false;

  return Boolean(this.get(index));
};

/**
 * Method used to return the number of 1 from the beginning of the set up to
 * the ith index.
 *
 * @param  {number} i - Ith index (cannot be > length).
 * @return {number}
 */
BitVector.prototype.rank = function(i) {
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
BitVector.prototype.select = function(r) {
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
BitVector.prototype.forEach = function(callback, scope) {
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
BitVector.prototype.values = function() {
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
BitVector.prototype.entries = function() {
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
  BitVector.prototype[Symbol.iterator] = BitVector.prototype.values;

/**
 * Convenience known methods.
 */
BitVector.prototype.inspect = function() {
  var proxy = new Uint8Array(this.length);

  this.forEach(function(bit, i) {
    proxy[i] = bit;
  });

  // Trick so that node displays the name of the constructor
  Object.defineProperty(proxy, 'constructor', {
    value: BitVector,
    enumerable: false
  });

  return proxy;
};

if (typeof Symbol !== 'undefined')
  BitVector.prototype[Symbol.for('nodejs.util.inspect.custom')] = BitVector.prototype.inspect;

BitVector.prototype.toJSON = function() {
  return Array.from(this.array.slice(0, (this.length >> 5) + 1));
};

/**
 * Exporting.
 */
module.exports = BitVector;
