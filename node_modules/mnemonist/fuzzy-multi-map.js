/**
 * Mnemonist FuzzyMultiMap
 * ========================
 *
 * Same as the fuzzy map but relying on a MultiMap rather than a Map.
 */
var MultiMap = require('./multi-map.js'),
    forEach = require('obliterator/foreach');

var identity = function(x) {
  return x;
};

/**
 * FuzzyMultiMap.
 *
 * @constructor
 * @param {array|function} descriptor - Hash functions descriptor.
 * @param {function}       Container  - Container to use.
 */
function FuzzyMultiMap(descriptor, Container) {
  this.items = new MultiMap(Container);
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
    throw new Error('mnemonist/FuzzyMultiMap.constructor: invalid hash function given.');

  if (typeof this.readHashFunction !== 'function')
    throw new Error('mnemonist/FuzzyMultiMap.constructor: invalid hash function given.');
}

/**
 * Method used to clear the structure.
 *
 * @return {undefined}
 */
FuzzyMultiMap.prototype.clear = function() {
  this.items.clear();

  // Properties
  this.size = 0;
  this.dimension = 0;
};

/**
 * Method used to add an item to the index.
 *
 * @param  {any} item - Item to add.
 * @return {FuzzyMultiMap}
 */
FuzzyMultiMap.prototype.add = function(item) {
  var key = this.writeHashFunction(item);

  this.items.set(key, item);
  this.size = this.items.size;
  this.dimension = this.items.dimension;

  return this;
};

/**
 * Method used to set an item in the index using the given key.
 *
 * @param  {any} key  - Key to use.
 * @param  {any} item - Item to add.
 * @return {FuzzyMultiMap}
 */
FuzzyMultiMap.prototype.set = function(key, item) {
  key = this.writeHashFunction(key);

  this.items.set(key, item);
  this.size = this.items.size;
  this.dimension = this.items.dimension;

  return this;
};

/**
 * Method used to retrieve an item from the index.
 *
 * @param  {any} key - Key to use.
 * @return {any}
 */
FuzzyMultiMap.prototype.get = function(key) {
  key = this.readHashFunction(key);

  return this.items.get(key);
};

/**
 * Method used to test the existence of an item in the map.
 *
 * @param  {any} key - Key to check.
 * @return {boolean}
 */
FuzzyMultiMap.prototype.has = function(key) {
  key = this.readHashFunction(key);

  return this.items.has(key);
};

/**
 * Method used to iterate over each of the index's values.
 *
 * @param  {function}  callback - Function to call for each item.
 * @param  {object}    scope    - Optional scope.
 * @return {undefined}
 */
FuzzyMultiMap.prototype.forEach = function(callback, scope) {
  scope = arguments.length > 1 ? scope : this;

  this.items.forEach(function(value) {
    callback.call(scope, value, value);
  });
};

/**
 * Method returning an iterator over the index's values.
 *
 * @return {FuzzyMultiMapIterator}
 */
FuzzyMultiMap.prototype.values = function() {
  return this.items.values();
};

/**
 * Attaching the #.values method to Symbol.iterator if possible.
 */
if (typeof Symbol !== 'undefined')
  FuzzyMultiMap.prototype[Symbol.iterator] = FuzzyMultiMap.prototype.values;

/**
 * Convenience known method.
 */
FuzzyMultiMap.prototype.inspect = function() {
  var array = Array.from(this);

  Object.defineProperty(array, 'constructor', {
    value: FuzzyMultiMap,
    enumerable: false
  });

  return array;
};

if (typeof Symbol !== 'undefined')
  FuzzyMultiMap.prototype[Symbol.for('nodejs.util.inspect.custom')] = FuzzyMultiMap.prototype.inspect;

/**
 * Static @.from function taking an arbitrary iterable & converting it into
 * a structure.
 *
 * @param  {Iterable}       iterable   - Target iterable.
 * @param  {array|function} descriptor - Hash functions descriptor.
 * @param  {function}       Container  - Container to use.
 * @param  {boolean}        useSet     - Whether to use #.set or #.add
 * @return {FuzzyMultiMap}
 */
FuzzyMultiMap.from = function(iterable, descriptor, Container, useSet) {
  if (arguments.length === 3) {
    if (typeof Container === 'boolean') {
      useSet = Container;
      Container = Array;
    }
  }

  var map = new FuzzyMultiMap(descriptor, Container);

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
module.exports = FuzzyMultiMap;
