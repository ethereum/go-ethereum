'use strict'

const {Buffer} = require('buffer')
const encoder = require('./encoder')
const decoder = require('./decoder')
const {MT} = require('./constants')

/**
 * Wrapper around a JavaScript Map object that allows the keys to be
 * any complex type.  The base Map object allows this, but will only
 * compare the keys by identity, not by value.  CborMap translates keys
 * to CBOR first (and base64's them to ensure by-value comparison).
 *
 * This is not a subclass of Object, because it would be tough to get
 * the semantics to be an exact match.
 *
 * @extends Map
 */
class CborMap extends Map {
  /**
   * Creates an instance of CborMap.
   *
   * @param {Iterable<any>} [iterable] An Array or other iterable
   *   object whose elements are key-value pairs (arrays with two elements, e.g.
   *   <code>[[ 1, 'one' ],[ 2, 'two' ]]</code>). Each key-value pair is added
   *   to the new CborMap; null values are treated as undefined.
   */
  constructor(iterable) {
    super(iterable)
  }

  /**
   * @ignore
   */
  static _encode(key) {
    return encoder.encodeCanonical(key).toString('base64')
  }

  /**
   * @ignore
   */
  static _decode(key) {
    return decoder.decodeFirstSync(key, 'base64')
  }

  /**
   * Retrieve a specified element.
   *
   * @param {any} key The key identifying the element to retrieve.
   *   Can be any type, which will be serialized into CBOR and compared by
   *   value.
   * @returns {any} The element if it exists, or <code>undefined</code>.
   */
  get(key) {
    return super.get(CborMap._encode(key))
  }

  /**
   * Adds or updates an element with a specified key and value.
   *
   * @param {any} key The key identifying the element to store.
   *   Can be any type, which will be serialized into CBOR and compared by
   *   value.
   * @param {any} val The element to store.
   * @returns {this} This object.
   */
  set(key, val) {
    return super.set(CborMap._encode(key), val)
  }

  /**
   * Removes the specified element.
   *
   * @param {any} key The key identifying the element to delete. Can be any
   *   type, which will be serialized into CBOR and compared by value.
   * @returns {boolean} True if an element in the Map object existed and has
   *   been removed, or false if the element does not exist.
   */
  delete(key) {
    return super.delete(CborMap._encode(key))
  }

  /**
   * Does an element with the specified key exist?
   *
   * @param {any} key The key identifying the element to check.
   *   Can be any type, which will be serialized into CBOR and compared by
   *   value.
   * @returns {boolean} True if an element with the specified key exists in
   *   the Map object; otherwise false.
   */
  has(key) {
    return super.has(CborMap._encode(key))
  }

  /**
   * Returns a new Iterator object that contains the keys for each element
   * in the Map object in insertion order.  The keys are decoded into their
   * original format.
   *
   * @yields {any} The keys of the map.
   */
  *keys() {
    for (const k of super.keys()) {
      yield CborMap._decode(k)
    }
  }

  /* eslint-disable jsdoc/require-returns-check */
  /**
   * Returns a new Iterator object that contains the [key, value] pairs for
   * each element in the Map object in insertion order.
   *
   * @yields {any[]} Key value pairs.
   * @returns {IterableIterator<any, any>} Key value pairs.
   */
  *entries() {
    for (const kv of super.entries()) {
      yield [CborMap._decode(kv[0]), kv[1]]
    }
  }
  /* eslint-enable jsdoc/require-returns-check */

  /**
   * Returns a new Iterator object that contains the [key, value] pairs for
   * each element in the Map object in insertion order.
   *
   * @returns {IterableIterator} Key value pairs.
   */
  [Symbol.iterator]() {
    return this.entries()
  }

  /**
   * Executes a provided function once per each key/value pair in the Map
   * object, in insertion order.
   *
   * @param {function(any, any, Map): undefined} fun Function to execute for
   *  each element, which takes a value, a key, and the Map being traversed.
   * @param {any} thisArg Value to use as this when executing callback.
   * @throws {TypeError} Invalid function.
   */
  forEach(fun, thisArg) {
    if (typeof fun !== 'function') {
      throw new TypeError('Must be function')
    }
    for (const kv of super.entries()) {
      fun.call(this, kv[1], CborMap._decode(kv[0]), this)
    }
  }

  /**
   * Push the simple value onto the CBOR stream.
   *
   * @param {object} gen The generator to push onto.
   * @returns {boolean} True on success.
   */
  encodeCBOR(gen) {
    if (!gen._pushInt(this.size, MT.MAP)) {
      return false
    }
    if (gen.canonical) {
      const entries = Array.from(super.entries())
        .map(kv => [Buffer.from(kv[0], 'base64'), kv[1]])
      entries.sort((a, b) => a[0].compare(b[0]))
      for (const kv of entries) {
        if (!(gen.push(kv[0]) && gen.pushAny(kv[1]))) {
          return false
        }
      }
    } else {
      for (const kv of super.entries()) {
        if (!(gen.push(Buffer.from(kv[0], 'base64')) && gen.pushAny(kv[1]))) {
          return false
        }
      }
    }
    return true
  }
}

module.exports = CborMap
