'use strict'

const constants = require('./constants')
const utils = require('./utils')
const INTERNAL_JSON = Symbol('INTERNAL_JSON')

function setBuffersToJSON(obj, fn) {
  // The data item tagged can be a byte string or any other data item.  In the
  // latter case, the tag applies to all of the byte string data items
  // contained in the data item, except for those contained in a nested data
  // item tagged with an expected conversion.
  if (utils.isBufferish(obj)) {
    obj.toJSON = fn
  } else if (Array.isArray(obj)) {
    for (const v of obj) {
      setBuffersToJSON(v, fn)
    }
  } else if (obj && (typeof obj === 'object')) {
    // FFS, complexity in the protocol.

    // There's some circular dependency in here.
    // eslint-disable-next-line no-use-before-define
    if (!(obj instanceof Tagged) || (obj.tag < 21) || (obj.tag > 23)) {
      for (const v of Object.values(obj)) {
        setBuffersToJSON(v, fn)
      }
    }
  }
}

function b64this() {
  // eslint-disable-next-line no-invalid-this
  return utils.base64(this)
}

function b64urlThis() {
  // eslint-disable-next-line no-invalid-this
  return utils.base64url(this)
}

function hexThis() {
  // eslint-disable-next-line no-invalid-this
  return this.toString('hex')
}

function swapEndian(ab, size, byteOffset, byteLength) {
  const dv = new DataView(ab)
  const [getter, setter] = {
    2: [dv.getUint16, dv.setUint16],
    4: [dv.getUint32, dv.setUint32],
    8: [dv.getBigUint64, dv.setBigUint64],
  }[size]

  const end = byteOffset + byteLength
  for (let offset = byteOffset; offset < end; offset += size) {
    setter.call(dv, offset, getter.call(dv, offset, true))
  }
}

/**
 * Convert a tagged value to a more interesting JavaScript type.  Errors
 * thrown in this function will be captured into the "err" property of the
 * original Tagged instance.
 *
 * @callback TagFunction
 * @param {any} value The value inside the tag.
 * @param {Tagged} tag The enclosing Tagged instance; useful if you want to
 *   modify it and return it.  Also available as "this".
 * @returns {any} The transformed value.
 */

/* eslint-disable jsdoc/check-types */
/**
 * A mapping from tag number to a tag decoding function.
 *
 * @typedef {Object.<string, TagFunction>} TagMap
 */
/* eslint-enable jsdoc/check-types */

/**
 * @type {TagMap}
 * @private
 */
const TAGS = {
  // Standard date/time string; see Section 3.4.1
  0: v => new Date(v),
  // Epoch-based date/time; see Section 3.4.2
  1: v => new Date(v * 1000),
  // Positive bignum; see Section 3.4.3
  2: v => utils.bufferToBigInt(v),
  // Negative bignum; see Section 3.4.3
  3: v => constants.BI.MINUS_ONE - utils.bufferToBigInt(v),
  // Expected conversion to base64url encoding; see Section 3.4.5.2
  21: (v, tag) => {
    if (utils.isBufferish(v)) {
      tag[INTERNAL_JSON] = b64urlThis
    } else {
      setBuffersToJSON(v, b64urlThis)
    }
    return tag
  },
  // Expected conversion to base64 encoding; see Section 3.4.5.2
  22: (v, tag) => {
    if (utils.isBufferish(v)) {
      tag[INTERNAL_JSON] = b64this
    } else {
      setBuffersToJSON(v, b64this)
    }
    return tag
  },
  // Expected conversion to base16 encoding; see Section Section 3.4.5.2
  23: (v, tag) => {
    if (utils.isBufferish(v)) {
      tag[INTERNAL_JSON] = hexThis
    } else {
      setBuffersToJSON(v, hexThis)
    }
    return tag
  },
  // URI; see Section 3.4.5.3
  32: v => new URL(v),
  // Base64url; see Section 3.4.5.3
  33: (v, tag) => {
    // If any of the following apply:
    // -  the encoded text string contains non-alphabet characters or
    //    only 1 alphabet character in the last block of 4 (where
    //    alphabet is defined by Section 5 of [RFC4648] for tag number 33
    //    and Section 4 of [RFC4648] for tag number 34), or
    if (!v.match(/^[a-zA-Z0-9_-]+$/)) {
      throw new Error('Invalid base64url characters')
    }
    const last = v.length % 4
    if (last === 1) {
      throw new Error('Invalid base64url length')
    }
    // -  the padding bits in a 2- or 3-character block are not 0, or
    if (last === 2) {
      // The last 4 bits of the last character need to be zero.
      if ('AQgw'.indexOf(v[v.length - 1]) === -1) {
        throw new Error('Invalid base64 padding')
      }
    } else if (last === 3) {
      // The last 2 bits of the last character need to be zero.
      if ('AEIMQUYcgkosw048'.indexOf(v[v.length - 1]) === -1) {
        throw new Error('Invalid base64 padding')
      }
    }

    //    Or
    // -  the base64url encoding has padding characters,
    // (caught above)

    // the string is invalid.
    return tag
  },
  // Base64; see Section 3.4.5.3
  34: (v, tag) => {
    // If any of the following apply:
    // -  the encoded text string contains non-alphabet characters or
    //    only 1 alphabet character in the last block of 4 (where
    //    alphabet is defined by Section 5 of [RFC4648] for tag number 33
    //    and Section 4 of [RFC4648] for tag number 34), or
    const m = v.match(/^[a-zA-Z0-9+/]+(?<padding>={0,2})$/)
    if (!m) {
      throw new Error('Invalid base64 characters')
    }
    if ((v.length % 4) !== 0) {
      throw new Error('Invalid base64 length')
    }
    // -  the padding bits in a 2- or 3-character block are not 0, or
    if (m.groups.padding === '=') {
      // The last 4 bits of the last character need to be zero.
      if ('AQgw'.indexOf(v[v.length - 2]) === -1) {
        throw new Error('Invalid base64 padding')
      }
    } else if (m.groups.padding === '==') {
      // The last 2 bits of the last character need to be zero.
      if ('AEIMQUYcgkosw048'.indexOf(v[v.length - 3]) === -1) {
        throw new Error('Invalid base64 padding')
      }
    }

    // -  the base64 encoding has the wrong number of padding characters,
    // (caught above)
    // the string is invalid.
    return tag
  },
  // Regular expression; see Section 2.4.4.3
  35: v => new RegExp(v),
  // https://github.com/input-output-hk/cbor-sets-spec/blob/master/CBOR_SETS.md
  258: v => new Set(v),
}

const TYPED_ARRAY_TAGS = {
  64: Uint8Array,
  65: Uint16Array,
  66: Uint32Array,
  // 67: BigUint64Array,  Safari doesn't implement
  68: Uint8ClampedArray,
  69: Uint16Array,
  70: Uint32Array,
  // 71: BigUint64Array,  Safari doesn't implement
  72: Int8Array,
  73: Int16Array,
  74: Int32Array,
  // 75: BigInt64Array,  Safari doesn't implement
  // 76: reserved
  77: Int16Array,
  78: Int32Array,
  // 79: BigInt64Array,  Safari doesn't implement
  // 80: not implemented, float16 array
  81: Float32Array,
  82: Float64Array,
  // 83: not implemented, float128 array
  // 84: not implemented, float16 array
  85: Float32Array,
  86: Float64Array,
  // 87: not implemented, float128 array
}

// Safari
if (typeof BigUint64Array !== 'undefined') {
  TYPED_ARRAY_TAGS[67] = BigUint64Array
  TYPED_ARRAY_TAGS[71] = BigUint64Array
}
if (typeof BigInt64Array !== 'undefined') {
  TYPED_ARRAY_TAGS[75] = BigInt64Array
  TYPED_ARRAY_TAGS[79] = BigInt64Array
}

function _toTypedArray(val, tagged) {
  if (!utils.isBufferish(val)) {
    throw new TypeError('val not a buffer')
  }
  const {tag} = tagged
  // See https://tools.ietf.org/html/rfc8746
  const TypedClass = TYPED_ARRAY_TAGS[tag]
  if (!TypedClass) {
    throw new Error(`Invalid typed array tag: ${tag}`)
  }
  const little = tag & 0b00000100
  const float = (tag & 0b00010000) >> 4
  const sz = 2 ** (float + (tag & 0b00000011))

  if ((!little !== utils.isBigEndian()) && (sz > 1)) {
    swapEndian(val.buffer, sz, val.byteOffset, val.byteLength)
  }

  const ab = val.buffer.slice(val.byteOffset, val.byteOffset + val.byteLength)
  return new TypedClass(ab)
}

for (const n of Object.keys(TYPED_ARRAY_TAGS)) {
  TAGS[n] = _toTypedArray
}

/**
 * @type {TagMap}
 * @private
 */
let current_TAGS = {}

/**
 * A CBOR tagged item, where the tag does not have semantics specified at the
 * moment, or those semantics threw an error during parsing. Typically this will
 * be an extension point you're not yet expecting.
 */
class Tagged {
  /**
   * Creates an instance of Tagged.
   *
   * @param {number} tag The number of the tag.
   * @param {any} value The value inside the tag.
   * @param {Error} [err] The error that was thrown parsing the tag, or null.
   */
  constructor(tag, value, err) {
    this.tag = tag
    this.value = value
    this.err = err
    if (typeof this.tag !== 'number') {
      throw new Error(`Invalid tag type (${typeof this.tag})`)
    }
    if ((this.tag < 0) || ((this.tag | 0) !== this.tag)) {
      throw new Error(`Tag must be a positive integer: ${this.tag}`)
    }
  }

  toJSON() {
    if (this[INTERNAL_JSON]) {
      return this[INTERNAL_JSON].call(this.value)
    }
    const ret = {
      tag: this.tag,
      value: this.value,
    }
    if (this.err) {
      ret.err = this.err
    }
    return ret
  }

  /**
   * Convert to a String.
   *
   * @returns {string} String of the form '1(2)'.
   */
  toString() {
    return `${this.tag}(${JSON.stringify(this.value)})`
  }

  /**
   * Push the simple value onto the CBOR stream.
   *
   * @param {object} gen The generator to push onto.
   * @returns {boolean} True on success.
   */
  encodeCBOR(gen) {
    gen._pushTag(this.tag)
    return gen.pushAny(this.value)
  }

  /**
   * If we have a converter for this type, do the conversion.  Some converters
   * are built-in.  Additional ones can be passed in.  If you want to remove
   * a built-in converter, pass a converter in whose value is 'null' instead
   * of a function.
   *
   * @param {object} converters Keys in the object are a tag number, the value
   *   is a function that takes the decoded CBOR and returns a JavaScript value
   *   of the appropriate type.  Throw an exception in the function on errors.
   * @returns {any} The converted item.
   */
  convert(converters) {
    let f = (converters == null) ? undefined : converters[this.tag]
    if (typeof f !== 'function') {
      f = Tagged.TAGS[this.tag]
      if (typeof f !== 'function') {
        return this
      }
    }
    try {
      return f.call(this, this.value, this)
    } catch (error) {
      if (error && error.message && (error.message.length > 0)) {
        this.err = error.message
      } else {
        this.err = error
      }
      return this
    }
  }

  /**
   * The current set of supported tags.  May be modified by plugins.
   *
   * @type {TagMap}
   * @static
   */
  static get TAGS() {
    return current_TAGS
  }

  static set TAGS(val) {
    current_TAGS = val
  }

  /**
   * Reset the supported tags to the original set, before any plugins modified
   * the list.
   */
  static reset() {
    Tagged.TAGS = {...TAGS}
  }
}
Tagged.INTERNAL_JSON = INTERNAL_JSON
Tagged.reset()
module.exports = Tagged
