'use strict'

const {MT, SIMPLE, SYMS} = require('./constants')

/**
 * A CBOR Simple Value that does not map onto a known constant.
 */
class Simple {
  /**
   * Creates an instance of Simple.
   *
   * @param {number} value The simple value's integer value.
   */
  constructor(value) {
    if (typeof value !== 'number') {
      throw new Error(`Invalid Simple type: ${typeof value}`)
    }
    if ((value < 0) || (value > 255) || ((value | 0) !== value)) {
      throw new Error(`value must be a small positive integer: ${value}`)
    }
    this.value = value
  }

  /**
   * Debug string for simple value.
   *
   * @returns {string} Formated string of `simple(value)`.
   */
  toString() {
    return `simple(${this.value})`
  }

  /**
   * Debug string for simple value.
   *
   * @param {number} depth How deep are we?
   * @param {object} opts Options.
   * @returns {string} Formatted string of `simple(value)`.
   */
  [Symbol.for('nodejs.util.inspect.custom')](depth, opts) {
    return `simple(${this.value})`
  }

  /**
   * Push the simple value onto the CBOR stream.
   *
   * @param {object} gen The generator to push onto.
   * @returns {boolean} True on success.
   */
  encodeCBOR(gen) {
    return gen._pushInt(this.value, MT.SIMPLE_FLOAT)
  }

  /**
   * Is the given object a Simple?
   *
   * @param {any} obj Object to test.
   * @returns {boolean} Is it Simple?
   */
  static isSimple(obj) {
    return obj instanceof Simple
  }

  /**
   * Decode from the CBOR additional information into a JavaScript value.
   * If the CBOR item has no parent, return a "safe" symbol instead of
   * `null` or `undefined`, so that the value can be passed through a
   * stream in object mode.
   *
   * @param {number} val The CBOR additional info to convert.
   * @param {boolean} [has_parent=true] Does the CBOR item have a parent?
   * @param {boolean} [parent_indefinite=false] Is the parent element
   *   indefinitely encoded?
   * @returns {(null|undefined|boolean|symbol|Simple)} The decoded value.
   * @throws {Error} Invalid BREAK.
   */
  static decode(val, has_parent = true, parent_indefinite = false) {
    switch (val) {
      case SIMPLE.FALSE:
        return false
      case SIMPLE.TRUE:
        return true
      case SIMPLE.NULL:
        if (has_parent) {
          return null
        }
        return SYMS.NULL
      case SIMPLE.UNDEFINED:
        if (has_parent) {
          return undefined
        }
        return SYMS.UNDEFINED
      case -1:
        if (!has_parent || !parent_indefinite) {
          throw new Error('Invalid BREAK')
        }
        return SYMS.BREAK
      default:
        return new Simple(val)
    }
  }
}

module.exports = Simple
