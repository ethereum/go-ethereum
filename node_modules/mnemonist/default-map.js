/**
 * Mnemonist DefaultMap
 * =====================
 *
 * JavaScript implementation of a default map that will return a constructed
 * value any time one tries to access an inexisting key. It's quite similar
 * to python's defaultdict.
 */

/**
 * DefaultMap.
 *
 * @constructor
 */
function DefaultMap(factory) {
  if (typeof factory !== 'function')
    throw new Error('mnemonist/DefaultMap.constructor: expecting a function.');

  this.items = new Map();
  this.factory = factory;
  this.size = 0;
}

/**
 * Method used to clear the structure.
 *
 * @return {undefined}
 */
DefaultMap.prototype.clear = function() {

  // Properties
  this.items.clear();
  this.size = 0;
};

/**
 * Method used to get the value set for given key. If the key does not exist,
 * the value will be created using the provided factory.
 *
 * @param  {any} key - Target key.
 * @return {any}
 */
DefaultMap.prototype.get = function(key) {
  var value = this.items.get(key);

  if (typeof value === 'undefined') {
    value = this.factory(key, this.size);
    this.items.set(key, value);
    this.size++;
  }

  return value;
};

/**
 * Method used to get the value set for given key. If the key does not exist,
 * a value won't be created.
 *
 * @param  {any} key - Target key.
 * @return {any}
 */
DefaultMap.prototype.peek = function(key) {
  return this.items.get(key);
};

/**
 * Method used to set a value for given key.
 *
 * @param  {any} key   - Target key.
 * @param  {any} value - Value.
 * @return {DefaultMap}
 */
DefaultMap.prototype.set = function(key, value) {
  this.items.set(key, value);
  this.size = this.items.size;

  return this;
};

/**
 * Method used to test the existence of a key in the map.
 *
 * @param  {any} key   - Target key.
 * @return {boolean}
 */
DefaultMap.prototype.has = function(key) {
  return this.items.has(key);
};

/**
 * Method used to delete target key.
 *
 * @param  {any} key   - Target key.
 * @return {boolean}
 */
DefaultMap.prototype.delete = function(key) {
  var deleted = this.items.delete(key);

  this.size = this.items.size;

  return deleted;
};

/**
 * Method used to iterate over each of the key/value pairs.
 *
 * @param  {function}  callback - Function to call for each item.
 * @param  {object}    scope    - Optional scope.
 * @return {undefined}
 */
DefaultMap.prototype.forEach = function(callback, scope) {
  scope = arguments.length > 1 ? scope : this;

  this.items.forEach(callback, scope);
};

/**
 * Iterators.
 */
DefaultMap.prototype.entries = function() {
  return this.items.entries();
};

DefaultMap.prototype.keys = function() {
  return this.items.keys();
};

DefaultMap.prototype.values = function() {
  return this.items.values();
};

/**
 * Attaching the #.entries method to Symbol.iterator if possible.
 */
if (typeof Symbol !== 'undefined')
  DefaultMap.prototype[Symbol.iterator] = DefaultMap.prototype.entries;

/**
 * Convenience known methods.
 */
DefaultMap.prototype.inspect = function() {
  return this.items;
};

if (typeof Symbol !== 'undefined')
  DefaultMap.prototype[Symbol.for('nodejs.util.inspect.custom')] = DefaultMap.prototype.inspect;

/**
 * Typical factories.
 */
DefaultMap.autoIncrement = function() {
  var i = 0;

  return function() {
    return i++;
  };
};

/**
 * Exporting.
 */
module.exports = DefaultMap;
