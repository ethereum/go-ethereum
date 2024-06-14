/**
 * Mnemonist DefaultWeakMap
 * =========================
 *
 * JavaScript implementation of a default weak map that will return a constructed
 * value any time one tries to access an non-existing key. It is similar to
 * DefaultMap but uses ES6 WeakMap that only holds weak reference to keys.
 */

/**
 * DefaultWeakMap.
 *
 * @constructor
 */
function DefaultWeakMap(factory) {
  if (typeof factory !== 'function')
    throw new Error('mnemonist/DefaultWeakMap.constructor: expecting a function.');

  this.items = new WeakMap();
  this.factory = factory;
}

/**
 * Method used to clear the structure.
 *
 * @return {undefined}
 */
DefaultWeakMap.prototype.clear = function() {

  // Properties
  this.items = new WeakMap();
};

/**
 * Method used to get the value set for given key. If the key does not exist,
 * the value will be created using the provided factory.
 *
 * @param  {any} key - Target key.
 * @return {any}
 */
DefaultWeakMap.prototype.get = function(key) {
  var value = this.items.get(key);

  if (typeof value === 'undefined') {
    value = this.factory(key);
    this.items.set(key, value);
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
DefaultWeakMap.prototype.peek = function(key) {
  return this.items.get(key);
};

/**
 * Method used to set a value for given key.
 *
 * @param  {any} key   - Target key.
 * @param  {any} value - Value.
 * @return {DefaultMap}
 */
DefaultWeakMap.prototype.set = function(key, value) {
  this.items.set(key, value);
  return this;
};

/**
 * Method used to test the existence of a key in the map.
 *
 * @param  {any} key   - Target key.
 * @return {boolean}
 */
DefaultWeakMap.prototype.has = function(key) {
  return this.items.has(key);
};

/**
 * Method used to delete target key.
 *
 * @param  {any} key   - Target key.
 * @return {boolean}
 */
DefaultWeakMap.prototype.delete = function(key) {
  return this.items.delete(key);
};

/**
 * Convenience known methods.
 */
DefaultWeakMap.prototype.inspect = function() {
  return this.items;
};

if (typeof Symbol !== 'undefined')
  DefaultWeakMap.prototype[Symbol.for('nodejs.util.inspect.custom')] = DefaultWeakMap.prototype.inspect;

/**
 * Exporting.
 */
module.exports = DefaultWeakMap;
