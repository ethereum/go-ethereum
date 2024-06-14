/**
 * Mnemonist Fuzzy Map
 * ====================
 *
 * The fuzzy map is a map whose keys are processed by a function before
 * read/write operations. This can often result in multiple keys accessing
 * the same resource (example: a map with lowercased keys).
 */
var forEach = require('obliterator/foreach');

var identity = function(x) {
  return x;
};

/**
 * FuzzyMap.
 *
 * @constructor
 * @param {array|function} descriptor - Hash functions descriptor.
 */
function FuzzyMap(descriptor) {
  this.items = new Map();
  this.clear();

  if (Array.isArray(descriptor)) {
    this.writeHashFunction = descriptor[0];
    this.readHashFunction = descriptor[1];
  }
  else {
    this.writeHashFunction = descriptor;
    this.readHashFunction = descriptor;
  }

  if (!this.writeHashFunction)
    this.writeHashFunction = identity;
  if (!this.readHashFunction)
    this.readHashFunction = identity;

  if (typeof this.writeHashFunction !== 'function')
    throw new Error('mnemonist/FuzzyMap.constructor: invalid hash function given.');

  if (typeof this.readHashFunction !== 'function')
    throw new Error('mnemonist/FuzzyMap.constructor: invalid hash function given.');
}

/**
 * Method used to clear the structure.
 *
 * @return {undefined}
 */
FuzzyMap.prototype.clear = function() {
  this.items.clear();

  // Properties
  this.size = 0;
};

/**
 * Method used to add an item to the FuzzyMap.
 *
 * @param  {any} item - Item to add.
 * @return {FuzzyMap}
 */
FuzzyMap.prototype.add = function(item) {
  var key = this.writeHashFunction(item);

  this.items.set(key, item);
  this.size = this.items.size;

  return this;
};

/**
 * Method used to set an item in the FuzzyMap using the given key.
 *
 * @param  {any} key  - Key to use.
 * @param  {any} item - Item to add.
 * @return {FuzzyMap}
 */
FuzzyMap.prototype.set = function(key, item) {
  key = this.writeHashFunction(key);

  this.items.set(key, item);
  this.size = this.items.size;

  return this;
};

/**
 * Method used to retrieve an item from the FuzzyMap.
 *
 * @param  {any} key - Key to use.
 * @return {any}
 */
FuzzyMap.prototype.get = function(key) {
  key = this.readHashFunction(key);

  return this.items.get(key);
};

/**
 * Method used to test the existence of an item in the map.
 *
 * @param  {any} key - Key to check.
 * @return {boolean}
 */
FuzzyMap.prototype.has = function(key) {
  key = this.readHashFunction(key);

  return this.items.has(key);
};

/**
 * Method used to iterate over each of the FuzzyMap's values.
 *
 * @param  {function}  callback - Function to call for each item.
 * @param  {object}    scope    - Optional scope.
 * @return {undefined}
 */
FuzzyMap.prototype.forEach = function(callback, scope) {
  scope = arguments.length > 1 ? scope : this;

  this.items.forEach(function(value) {
    callback.call(scope, value, value);
  });
};

/**
 * Method returning an iterator over the FuzzyMap's values.
 *
 * @return {FuzzyMapIterator}
 */
FuzzyMap.prototype.values = function() {
  return this.items.values();
};

/**
 * Attaching the #.values method to Symbol.iterator if possible.
 */
if (typeof Symbol !== 'undefined')
  FuzzyMap.prototype[Symbol.iterator] = FuzzyMap.prototype.values;

/**
 * Convenience known method.
 */
FuzzyMap.prototype.inspect = function() {
  var array = Array.from(this.items.values());

  Object.defineProperty(array, 'constructor', {
    value: FuzzyMap,
    enumerable: false
  });

  return array;
};

if (typeof Symbol !== 'undefined')
  FuzzyMap.prototype[Symbol.for('nodejs.util.inspect.custom')] = FuzzyMap.prototype.inspect;

/**
 * Static @.from function taking an arbitrary iterable & converting it into
 * a structure.
 *
 * @param  {Iterable}       iterable   - Target iterable.
 * @param  {array|function} descriptor - Hash functions descriptor.
 * @param  {boolean}        useSet     - Whether to use #.set or #.add
 * @return {FuzzyMap}
 */
FuzzyMap.from = function(iterable, descriptor, useSet) {
  var map = new FuzzyMap(descriptor);

  forEach(iterable, function(value, key) {
    if (useSet)
      map.set(key, value);
    else
      map.add(value);
  });

  return map;
};

/**
 * Exporting.
 */
module.exports = FuzzyMap;
