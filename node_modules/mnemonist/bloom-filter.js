/**
 * Mnemonist Bloom Filter
 * =======================
 *
 * Bloom Filter implementation relying on MurmurHash3.
 */
var murmurhash3 = require('./utils/murmurhash3.js'),
    forEach = require('obliterator/foreach');

/**
 * Constants.
 */
var LN2_SQUARED = Math.LN2 * Math.LN2;

/**
 * Defaults.
 */
var DEFAULTS = {
  errorRate: 0.005
};

/**
 * Function used to convert a string into a Uint16 byte array.
 *
 * @param  {string}      string - Target string.
 * @return {Uint16Array}
 */
function stringToByteArray(string) {
  var array = new Uint16Array(string.length),
      i,
      l;

  for (i = 0, l = string.length; i < l; i++)
    array[i] = string.charCodeAt(i);

  return array;
}

/**
 * Function used to hash the given byte array.
 *
 * @param  {number}      length - Length of the filter's byte array.
 * @param  {number}      seed   - Seed to use for the hash function.
 * @param  {Uint16Array}        - Byte array representing the string.
 * @return {number}             - The hash.
 *
 * @note length * 8 should probably already be computed as well as seeds.
 */
function hashArray(length, seed, array) {
  var hash = murmurhash3((seed * 0xFBA4C795) & 0xFFFFFFFF, array);

  return hash % (length * 8);
}

/**
 * Bloom Filter.
 *
 * @constructor
 * @param {number|object} capacityOrOptions - Capacity or options.
 */
function BloomFilter(capacityOrOptions) {
  var options = {};

  if (!capacityOrOptions)
    throw new Error('mnemonist/BloomFilter.constructor: a BloomFilter must be created with a capacity.');

  if (typeof capacityOrOptions === 'object')
    options = capacityOrOptions;
  else
    options.capacity = capacityOrOptions;

  // Handling capacity
  if (typeof options.capacity !== 'number' || options.capacity <= 0)
    throw new Error('mnemonist/BloomFilter.constructor: `capacity` option should be a positive integer.');

  this.capacity = options.capacity;

  // Handling error rate
  this.errorRate = options.errorRate || DEFAULTS.errorRate;

  if (typeof this.errorRate !== 'number' || options.errorRate <= 0)
    throw new Error('mnemonist/BloomFilter.constructor: `errorRate` option should be a positive float.');

  this.clear();
}

/**
 * Method used to clear the filter.
 *
 * @return {undefined}
 */
BloomFilter.prototype.clear = function() {

  // Optimizing number of bits & number of hash functions
  var bits = -1 / LN2_SQUARED * this.capacity * Math.log(this.errorRate),
      length = (bits / 8) | 0;

  this.hashFunctions = (length * 8 / this.capacity * Math.LN2) | 0;

  // Creating the data array
  this.data = new Uint8Array(length);

  return;
};

/**
 * Method used to add an string to the filter.
 *
 * @param  {string} string - Item to add.
 * @return {BloomFilter}
 *
 * @note Should probably create a hash function working directly on a string.
 */
BloomFilter.prototype.add = function(string) {

  // Converting the string to a byte array
  var array = stringToByteArray(string);

  // Applying the n hash functions
  for (var i = 0, l = this.hashFunctions; i < l; i++) {
    var index = hashArray(this.data.length, i, array),
        position = (1 << (7 & index));

    this.data[index >> 3] |= position;
  }

  return this;
};

/**
 * Method used to test the given string.
 *
 * @param  {string} string - Item to test.
 * @return {boolean}
 */
BloomFilter.prototype.test = function(string) {

  // Converting the string to a byte array
  var array = stringToByteArray(string);

  // Applying the n hash functions
  for (var i = 0, l = this.hashFunctions; i < l; i++) {
    var index = hashArray(this.data.length, i, array);

    if (!(this.data[index >> 3] & (1 << (7 & index))))
      return false;
  }

  return true;
};

/**
 * Convenience known methods.
 */
BloomFilter.prototype.toJSON = function() {
  return this.data;
};

/**
 * Static @.from function taking an arbitrary iterable & converting it into
 * a filter.
 *
 * @param  {Iterable}    iterable - Target iterable.
 * @return {BloomFilter}
 */
BloomFilter.from = function(iterable, options) {
  if (!options) {
    options = iterable.length || iterable.size;

    if (typeof options !== 'number')
      throw new Error('BloomFilter.from: could not infer the filter\'s capacity. Try passing it as second argument.');
  }

  var filter = new BloomFilter(options);

  forEach(iterable, function(value) {
    filter.add(value);
  });

  return filter;
};

/**
 * Exporting.
 */
module.exports = BloomFilter;
