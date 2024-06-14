'use strict'

const stream = require('stream')
const NoFilter = require('nofilter')
const utils = require('./utils')
const constants = require('./constants')
const {
  MT, NUMBYTES, SHIFT32, SIMPLE, SYMS, TAG, BI,
} = constants
const {Buffer} = require('buffer')

const HALF = (MT.SIMPLE_FLOAT << 5) | NUMBYTES.TWO
const FLOAT = (MT.SIMPLE_FLOAT << 5) | NUMBYTES.FOUR
const DOUBLE = (MT.SIMPLE_FLOAT << 5) | NUMBYTES.EIGHT
const TRUE = (MT.SIMPLE_FLOAT << 5) | SIMPLE.TRUE
const FALSE = (MT.SIMPLE_FLOAT << 5) | SIMPLE.FALSE
const UNDEFINED = (MT.SIMPLE_FLOAT << 5) | SIMPLE.UNDEFINED
const NULL = (MT.SIMPLE_FLOAT << 5) | SIMPLE.NULL

const BREAK = Buffer.from([0xff])
const BUF_NAN = Buffer.from('f97e00', 'hex')
const BUF_INF_NEG = Buffer.from('f9fc00', 'hex')
const BUF_INF_POS = Buffer.from('f97c00', 'hex')
const BUF_NEG_ZERO = Buffer.from('f98000', 'hex')

/**
 * Generate the CBOR for a value.  If you are using this, you'll either need
 * to call {@link Encoder.write} with a Buffer, or look into the internals of
 * Encoder to reuse existing non-documented behavior.
 *
 * @callback EncodeFunction
 * @param {Encoder} enc The encoder to use.
 * @param {any} val The value to encode.
 * @returns {boolean} True on success.
 */

/* eslint-disable jsdoc/check-types */
/**
 * A mapping from tag number to a tag decoding function.
 *
 * @typedef {Object.<string, EncodeFunction>} SemanticMap
 */
/* eslint-enable jsdoc/check-types */

/**
 * @type {SemanticMap}
 * @private
 */
const SEMANTIC_TYPES = {}

/**
 * @type {SemanticMap}
 * @private
 */
let current_SEMANTIC_TYPES = {}

/**
 * @param {string} str String to normalize.
 * @returns {"number"|"float"|"int"|"string"} Normalized.
 * @throws {TypeError} Invalid input.
 * @private
 */
function parseDateType(str) {
  if (!str) {
    return 'number'
  }
  switch (str.toLowerCase()) {
    case 'number':
      return 'number'
    case 'float':
      return 'float'
    case 'int':
    case 'integer':
      return 'int'
    case 'string':
      return 'string'
  }
  throw new TypeError(`dateType invalid, got "${str}"`)
}

/**
 * @typedef ObjectOptions
 * @property {boolean} [indefinite = false] Force indefinite encoding for this
 *   object.
 * @property {boolean} [skipTypes = false] Do not use available type mappings
 *   for this object, but encode it as a "normal" JS object would be.
 */

/**
 * @typedef EncodingOptions
 * @property {any[]|object} [genTypes=[]] Array of pairs of
 *   `type`, `function(Encoder)` for semantic types to be encoded.  Not
 *   needed for Array, Date, Buffer, Map, RegExp, Set, or URL.
 *   If an object, the keys are the constructor names for the types.
 * @property {boolean} [canonical=false] Should the output be
 *   canonicalized.
 * @property {boolean|WeakSet} [detectLoops=false] Should object loops
 *   be detected?  This will currently add memory to track every part of the
 *   object being encoded in a WeakSet.  Do not encode
 *   the same object twice on the same encoder, without calling
 *   `removeLoopDetectors` in between, which will clear the WeakSet.
 *   You may pass in your own WeakSet to be used; this is useful in some
 *   recursive scenarios.
 * @property {("number"|"float"|"int"|"string")} [dateType="number"] -
 *   how should dates be encoded?  "number" means float or int, if no
 *   fractional seconds.
 * @property {any} [encodeUndefined=undefined] How should an
 *   "undefined" in the input be encoded.  By default, just encode a CBOR
 *   undefined.  If this is a buffer, use those bytes without re-encoding
 *   them.  If this is a function, the function will be called (which is a
 *   good time to throw an exception, if that's what you want), and the
 *   return value will be used according to these rules.  Anything else will
 *   be encoded as CBOR.
 * @property {boolean} [disallowUndefinedKeys=false] Should
 *   "undefined" be disallowed as a key in a Map that is serialized?  If
 *   this is true, encode(new Map([[undefined, 1]])) will throw an
 *   exception.  Note that it is impossible to get a key of undefined in a
 *   normal JS object.
 * @property {boolean} [collapseBigIntegers=false] Should integers
 *   that come in as ECMAscript bigint's be encoded
 *   as normal CBOR integers if they fit, discarding type information?
 * @property {number} [chunkSize=4096] Number of characters or bytes
 *   for each chunk, if obj is a string or Buffer, when indefinite encoding.
 * @property {boolean} [omitUndefinedProperties=false] When encoding
 *   objects or Maps, do not include a key if its corresponding value is
 *   `undefined`.
 */

/**
 * Transform JavaScript values into CBOR bytes.  The `Writable` side of
 * the stream is in object mode.
 *
 * @extends stream.Transform
 */
class Encoder extends stream.Transform {
  /**
   * Creates an instance of Encoder.
   *
   * @param {EncodingOptions} [options={}] Options for the encoder.
   */
  constructor(options = {}) {
    const {
      canonical = false,
      encodeUndefined,
      disallowUndefinedKeys = false,
      dateType = 'number',
      collapseBigIntegers = false,
      detectLoops = false,
      omitUndefinedProperties = false,
      genTypes = [],
      ...superOpts
    } = options

    super({
      ...superOpts,
      readableObjectMode: false,
      writableObjectMode: true,
    })

    this.canonical = canonical
    this.encodeUndefined = encodeUndefined
    this.disallowUndefinedKeys = disallowUndefinedKeys
    this.dateType = parseDateType(dateType)
    this.collapseBigIntegers = this.canonical ? true : collapseBigIntegers

    /** @type {WeakSet?} */
    this.detectLoops = undefined
    if (typeof detectLoops === 'boolean') {
      if (detectLoops) {
        this.detectLoops = new WeakSet()
      }
    } else if (detectLoops instanceof WeakSet) {
      this.detectLoops = detectLoops
    } else {
      throw new TypeError('detectLoops must be boolean or WeakSet')
    }
    this.omitUndefinedProperties = omitUndefinedProperties

    this.semanticTypes = {...Encoder.SEMANTIC_TYPES}

    if (Array.isArray(genTypes)) {
      for (let i = 0, len = genTypes.length; i < len; i += 2) {
        this.addSemanticType(genTypes[i], genTypes[i + 1])
      }
    } else {
      for (const [k, v] of Object.entries(genTypes)) {
        this.addSemanticType(k, v)
      }
    }
  }

  /**
   * Transforming.
   *
   * @param {any} fresh Buffer to transcode.
   * @param {BufferEncoding} encoding Name of encoding.
   * @param {stream.TransformCallback} cb Callback when done.
   * @ignore
   */
  _transform(fresh, encoding, cb) {
    const ret = this.pushAny(fresh)
    // Old transformers might not return bool.  undefined !== false
    cb((ret === false) ? new Error('Push Error') : undefined)
  }

  /**
   * Flushing.
   *
   * @param {stream.TransformCallback} cb Callback when done.
   * @ignore
   */
  // eslint-disable-next-line class-methods-use-this
  _flush(cb) {
    cb()
  }

  /**
   * @param {number} val Number(0-255) to encode.
   * @returns {boolean} True on success.
   * @ignore
   */
  _pushUInt8(val) {
    const b = Buffer.allocUnsafe(1)
    b.writeUInt8(val, 0)
    return this.push(b)
  }

  /**
   * @param {number} val Number(0-65535) to encode.
   * @returns {boolean} True on success.
   * @ignore
   */
  _pushUInt16BE(val) {
    const b = Buffer.allocUnsafe(2)
    b.writeUInt16BE(val, 0)
    return this.push(b)
  }

  /**
   * @param {number} val Number(0..2**32-1) to encode.
   * @returns {boolean} True on success.
   * @ignore
   */
  _pushUInt32BE(val) {
    const b = Buffer.allocUnsafe(4)
    b.writeUInt32BE(val, 0)
    return this.push(b)
  }

  /**
   * @param {number} val Number to encode as 4-byte float.
   * @returns {boolean} True on success.
   * @ignore
   */
  _pushFloatBE(val) {
    const b = Buffer.allocUnsafe(4)
    b.writeFloatBE(val, 0)
    return this.push(b)
  }

  /**
   * @param {number} val Number to encode as 8-byte double.
   * @returns {boolean} True on success.
   * @ignore
   */
  _pushDoubleBE(val) {
    const b = Buffer.allocUnsafe(8)
    b.writeDoubleBE(val, 0)
    return this.push(b)
  }

  /**
   * @returns {boolean} True on success.
   * @ignore
   */
  _pushNaN() {
    return this.push(BUF_NAN)
  }

  /**
   * @param {number} obj Positive or negative infinity.
   * @returns {boolean} True on success.
   * @ignore
   */
  _pushInfinity(obj) {
    const half = (obj < 0) ? BUF_INF_NEG : BUF_INF_POS
    return this.push(half)
  }

  /**
   * Choose the best float representation for a number and encode it.
   *
   * @param {number} obj A number that is known to be not-integer, but not
   *   how many bytes of precision it needs.
   * @returns {boolean} True on success.
   * @ignore
   */
  _pushFloat(obj) {
    if (this.canonical) {
      // TODO: is this enough slower to hide behind canonical?
      // It's certainly enough of a hack (see utils.parseHalf)

      // From section 3.9:
      // If a protocol allows for IEEE floats, then additional canonicalization
      // rules might need to be added.  One example rule might be to have all
      // floats start as a 64-bit float, then do a test conversion to a 32-bit
      // float; if the result is the same numeric value, use the shorter value
      // and repeat the process with a test conversion to a 16-bit float.  (This
      // rule selects 16-bit float for positive and negative Infinity as well.)

      // which seems pretty much backwards to me.
      const b2 = Buffer.allocUnsafe(2)
      if (utils.writeHalf(b2, obj)) {
        // I have convinced myself that there are no cases where writeHalf
        // will return true but `utils.parseHalf(b2) !== obj)`
        return this._pushUInt8(HALF) && this.push(b2)
      }
    }
    if (Math.fround(obj) === obj) {
      return this._pushUInt8(FLOAT) && this._pushFloatBE(obj)
    }

    return this._pushUInt8(DOUBLE) && this._pushDoubleBE(obj)
  }

  /**
   * Choose the best integer representation for a postive number and encode
   * it.  If the number is over MAX_SAFE_INTEGER, fall back on float (but I
   * don't remember why).
   *
   * @param {number} obj A positive number that is known to be an integer,
   *   but not how many bytes of precision it needs.
   * @param {number} mt The Major Type number to combine with the integer.
   *   Not yet shifted.
   * @param {number} [orig] The number before it was transformed to positive.
   *   If the mt is NEG_INT, and the positive number is over MAX_SAFE_INT,
   *   then we'll encode this as a float rather than making the number
   *   negative again and losing precision.
   * @returns {boolean} True on success.
   * @ignore
   */
  _pushInt(obj, mt, orig) {
    const m = mt << 5

    if (obj < 24) {
      return this._pushUInt8(m | obj)
    }
    if (obj <= 0xff) {
      return this._pushUInt8(m | NUMBYTES.ONE) && this._pushUInt8(obj)
    }
    if (obj <= 0xffff) {
      return this._pushUInt8(m | NUMBYTES.TWO) && this._pushUInt16BE(obj)
    }
    if (obj <= 0xffffffff) {
      return this._pushUInt8(m | NUMBYTES.FOUR) && this._pushUInt32BE(obj)
    }
    let max = Number.MAX_SAFE_INTEGER
    if (mt === MT.NEG_INT) {
      // Special case for Number.MIN_SAFE_INTEGER - 1
      max--
    }
    if (obj <= max) {
      return this._pushUInt8(m | NUMBYTES.EIGHT) &&
        this._pushUInt32BE(Math.floor(obj / SHIFT32)) &&
        this._pushUInt32BE(obj % SHIFT32)
    }
    if (mt === MT.NEG_INT) {
      return this._pushFloat(orig)
    }
    return this._pushFloat(obj)
  }

  /**
   * Choose the best integer representation for a number and encode it.
   *
   * @param {number} obj A number that is known to be an integer,
   *   but not how many bytes of precision it needs.
   * @returns {boolean} True on success.
   * @ignore
   */
  _pushIntNum(obj) {
    if (Object.is(obj, -0)) {
      return this.push(BUF_NEG_ZERO)
    }

    if (obj < 0) {
      return this._pushInt(-obj - 1, MT.NEG_INT, obj)
    }
    return this._pushInt(obj, MT.POS_INT)
  }

  /**
   * @param {number} obj Plain JS number to encode.
   * @returns {boolean} True on success.
   * @ignore
   */
  _pushNumber(obj) {
    if (isNaN(obj)) {
      return this._pushNaN()
    }
    if (!isFinite(obj)) {
      return this._pushInfinity(obj)
    }
    if (Math.round(obj) === obj) {
      return this._pushIntNum(obj)
    }
    return this._pushFloat(obj)
  }

  /**
   * @param {string} obj String to encode.
   * @returns {boolean} True on success.
   * @ignore
   */
  _pushString(obj) {
    const len = Buffer.byteLength(obj, 'utf8')
    return this._pushInt(len, MT.UTF8_STRING) && this.push(obj, 'utf8')
  }

  /**
   * @param {boolean} obj Bool to encode.
   * @returns {boolean} True on success.
   * @ignore
   */
  _pushBoolean(obj) {
    return this._pushUInt8(obj ? TRUE : FALSE)
  }

  /**
   * @param {undefined} obj Ignored.
   * @returns {boolean} True on success.
   * @ignore
   */
  _pushUndefined(obj) {
    switch (typeof this.encodeUndefined) {
      case 'undefined':
        return this._pushUInt8(UNDEFINED)
      case 'function':
        return this.pushAny(this.encodeUndefined(obj))
      case 'object': {
        const buf = utils.bufferishToBuffer(this.encodeUndefined)
        if (buf) {
          return this.push(buf)
        }
      }
    }
    return this.pushAny(this.encodeUndefined)
  }

  /**
   * @param {null} obj Ignored.
   * @returns {boolean} True on success.
   * @ignore
   */
  _pushNull(obj) {
    return this._pushUInt8(NULL)
  }

  /**
   * @param {number} tag Tag number to encode.
   * @returns {boolean} True on success.
   * @ignore
   */
  _pushTag(tag) {
    return this._pushInt(tag, MT.TAG)
  }

  /**
   * @param {bigint} obj BigInt to encode.
   * @returns {boolean} True on success.
   * @ignore
   */
  _pushJSBigint(obj) {
    let m = MT.POS_INT
    let tag = TAG.POS_BIGINT
    // BigInt doesn't have -0
    if (obj < 0) {
      obj = -obj + BI.MINUS_ONE
      m = MT.NEG_INT
      tag = TAG.NEG_BIGINT
    }

    if (this.collapseBigIntegers &&
        (obj <= BI.MAXINT64)) {
      // Special handiling for 64bits
      if (obj <= 0xffffffff) {
        return this._pushInt(Number(obj), m)
      }
      return this._pushUInt8((m << 5) | NUMBYTES.EIGHT) &&
        this._pushUInt32BE(Number(obj / BI.SHIFT32)) &&
        this._pushUInt32BE(Number(obj % BI.SHIFT32))
    }

    let str = obj.toString(16)
    if (str.length % 2) {
      str = `0${str}`
    }
    const buf = Buffer.from(str, 'hex')
    return this._pushTag(tag) && Encoder._pushBuffer(this, buf)
  }

  /**
   * @param {object} obj Object to encode.
   * @param {ObjectOptions} [opts] Options for encoding this object.
   * @returns {boolean} True on success.
   * @throws {Error} Loop detected.
   * @ignore
   */
  _pushObject(obj, opts) {
    if (!obj) {
      return this._pushNull(obj)
    }
    opts = {
      indefinite: false,
      skipTypes: false,
      ...opts,
    }
    if (!opts.indefinite) {
      // This will only happen the first time through for indefinite encoding
      if (this.detectLoops) {
        if (this.detectLoops.has(obj)) {
          throw new Error(`\
Loop detected while CBOR encoding.
Call removeLoopDetectors before resuming.`)
        } else {
          this.detectLoops.add(obj)
        }
      }
    }
    if (!opts.skipTypes) {
      const f = obj.encodeCBOR
      if (typeof f === 'function') {
        return f.call(obj, this)
      }
      const converter = this.semanticTypes[obj.constructor.name]
      if (converter) {
        return converter.call(obj, this, obj)
      }
    }
    const keys = Object.keys(obj).filter(k => {
      const tv = typeof obj[k]
      return (tv !== 'function') &&
        (!this.omitUndefinedProperties || (tv !== 'undefined'))
    })
    const cbor_keys = {}
    if (this.canonical) {
      // Note: this can't be a normal sort, because 'b' needs to sort before
      // 'aa'
      keys.sort((a, b) => {
        // Always strings, so don't bother to pass options.
        // hold on to the cbor versions, since there's no need
        // to encode more than once
        const a_cbor = cbor_keys[a] || (cbor_keys[a] = Encoder.encode(a))
        const b_cbor = cbor_keys[b] || (cbor_keys[b] = Encoder.encode(b))

        return a_cbor.compare(b_cbor)
      })
    }
    if (opts.indefinite) {
      if (!this._pushUInt8((MT.MAP << 5) | NUMBYTES.INDEFINITE)) {
        return false
      }
    } else if (!this._pushInt(keys.length, MT.MAP)) {
      return false
    }
    let ck = null
    for (let j = 0, len2 = keys.length; j < len2; j++) {
      const k = keys[j]
      if (this.canonical && ((ck = cbor_keys[k]))) {
        if (!this.push(ck)) { // Already a Buffer
          return false
        }
      } else if (!this._pushString(k)) {
        return false
      }
      if (!this.pushAny(obj[k])) {
        return false
      }
    }
    if (opts.indefinite) {
      if (!this.push(BREAK)) {
        return false
      }
    } else if (this.detectLoops) {
      this.detectLoops.delete(obj)
    }
    return true
  }

  /**
   * @param {any[]} objs Array of supported things.
   * @returns {Buffer} Concatenation of encodings for the supported things.
   * @ignore
   */
  _encodeAll(objs) {
    const bs = new NoFilter({highWaterMark: this.readableHighWaterMark})
    this.pipe(bs)
    for (const o of objs) {
      this.pushAny(o)
    }
    this.end()
    return bs.read()
  }

  /**
   * Add an encoding function to the list of supported semantic types.  This
   * is useful for objects for which you can't add an encodeCBOR method.
   *
   * @param {string|Function} type The type to encode.
   * @param {EncodeFunction} fun The encoder to use.
   * @returns {EncodeFunction?} The previous encoder or undefined if there
   *   wasn't one.
   * @throws {TypeError} Invalid function.
   */
  addSemanticType(type, fun) {
    const typeName = (typeof type === 'string') ? type : type.name
    const old = this.semanticTypes[typeName]

    if (fun) {
      if (typeof fun !== 'function') {
        throw new TypeError('fun must be of type function')
      }
      this.semanticTypes[typeName] = fun
    } else if (old) {
      delete this.semanticTypes[typeName]
    }
    return old
  }

  /**
   * Push any supported type onto the encoded stream.
   *
   * @param {any} obj The thing to encode.
   * @returns {boolean} True on success.
   * @throws {TypeError} Unknown type for obj.
   */
  pushAny(obj) {
    switch (typeof obj) {
      case 'number':
        return this._pushNumber(obj)
      case 'bigint':
        return this._pushJSBigint(obj)
      case 'string':
        return this._pushString(obj)
      case 'boolean':
        return this._pushBoolean(obj)
      case 'undefined':
        return this._pushUndefined(obj)
      case 'object':
        return this._pushObject(obj)
      case 'symbol':
        switch (obj) {
          case SYMS.NULL:
            return this._pushNull(null)
          case SYMS.UNDEFINED:
            return this._pushUndefined(undefined)
          // TODO: Add pluggable support for other symbols
          default:
            throw new TypeError(`Unknown symbol: ${obj.toString()}`)
        }
      default:
        throw new TypeError(
          `Unknown type: ${typeof obj}, ${(typeof obj.toString === 'function') ? obj.toString() : ''}`
        )
    }
  }

  /**
   * Encode an array and all of its elements.
   *
   * @param {Encoder} gen Encoder to use.
   * @param {any[]} obj Array to encode.
   * @param {object} [opts] Options.
   * @param {boolean} [opts.indefinite=false] Use indefinite encoding?
   * @returns {boolean} True on success.
   */
  static pushArray(gen, obj, opts) {
    opts = {
      indefinite: false,
      ...opts,
    }
    const len = obj.length
    if (opts.indefinite) {
      if (!gen._pushUInt8((MT.ARRAY << 5) | NUMBYTES.INDEFINITE)) {
        return false
      }
    } else if (!gen._pushInt(len, MT.ARRAY)) {
      return false
    }
    for (let j = 0; j < len; j++) {
      if (!gen.pushAny(obj[j])) {
        return false
      }
    }
    if (opts.indefinite) {
      if (!gen.push(BREAK)) {
        return false
      }
    }
    return true
  }

  /**
   * Remove the loop detector WeakSet for this Encoder.
   *
   * @returns {boolean} True when the Encoder was reset, else false.
   */
  removeLoopDetectors() {
    if (!this.detectLoops) {
      return false
    }
    this.detectLoops = new WeakSet()
    return true
  }

  /**
   * @param {Encoder} gen Encoder.
   * @param {Date} obj Date to encode.
   * @returns {boolean} True on success.
   * @ignore
   */
  static _pushDate(gen, obj) {
    switch (gen.dateType) {
      case 'string':
        return gen._pushTag(TAG.DATE_STRING) &&
          gen._pushString(obj.toISOString())
      case 'int':
        return gen._pushTag(TAG.DATE_EPOCH) &&
          gen._pushIntNum(Math.round(obj.getTime() / 1000))
      case 'float':
        // Force float
        return gen._pushTag(TAG.DATE_EPOCH) &&
          gen._pushFloat(obj.getTime() / 1000)
      case 'number':
      default:
        // If we happen to have an integral number of seconds,
        // use integer.  Otherwise, use float.
        return gen._pushTag(TAG.DATE_EPOCH) &&
          gen.pushAny(obj.getTime() / 1000)
    }
  }

  /**
   * @param {Encoder} gen Encoder.
   * @param {Buffer} obj Buffer to encode.
   * @returns {boolean} True on success.
   * @ignore
   */
  static _pushBuffer(gen, obj) {
    return gen._pushInt(obj.length, MT.BYTE_STRING) && gen.push(obj)
  }

  /**
   * @param {Encoder} gen Encoder.
   * @param {NoFilter} obj Buffer to encode.
   * @returns {boolean} True on success.
   * @ignore
   */
  static _pushNoFilter(gen, obj) {
    return Encoder._pushBuffer(gen, /** @type {Buffer} */ (obj.slice()))
  }

  /**
   * @param {Encoder} gen Encoder.
   * @param {RegExp} obj RegExp to encode.
   * @returns {boolean} True on success.
   * @ignore
   */
  static _pushRegexp(gen, obj) {
    return gen._pushTag(TAG.REGEXP) && gen.pushAny(obj.source)
  }

  /**
   * @param {Encoder} gen Encoder.
   * @param {Set} obj Set to encode.
   * @returns {boolean} True on success.
   * @ignore
   */
  static _pushSet(gen, obj) {
    if (!gen._pushTag(TAG.SET)) {
      return false
    }
    if (!gen._pushInt(obj.size, MT.ARRAY)) {
      return false
    }
    for (const x of obj) {
      if (!gen.pushAny(x)) {
        return false
      }
    }
    return true
  }

  /**
   * @param {Encoder} gen Encoder.
   * @param {URL} obj URL to encode.
   * @returns {boolean} True on success.
   * @ignore
   */
  static _pushURL(gen, obj) {
    return gen._pushTag(TAG.URI) && gen.pushAny(obj.toString())
  }

  /**
   * @param {Encoder} gen Encoder.
   * @param {object} obj Boxed String, Number, or Boolean object to encode.
   * @returns {boolean} True on success.
   * @ignore
   */
  static _pushBoxed(gen, obj) {
    return gen.pushAny(obj.valueOf())
  }

  /**
   * @param {Encoder} gen Encoder.
   * @param {Map} obj Map to encode.
   * @returns {boolean} True on success.
   * @throws {Error} Map key that is undefined.
   * @ignore
   */
  static _pushMap(gen, obj, opts) {
    opts = {
      indefinite: false,
      ...opts,
    }
    let entries = [...obj.entries()]
    if (gen.omitUndefinedProperties) {
      entries = entries.filter(([k, v]) => v !== undefined)
    }
    if (opts.indefinite) {
      if (!gen._pushUInt8((MT.MAP << 5) | NUMBYTES.INDEFINITE)) {
        return false
      }
    } else if (!gen._pushInt(entries.length, MT.MAP)) {
      return false
    }
    // Memoizing the cbor only helps in certain cases, and hurts in most
    // others.  Just avoid it.
    if (gen.canonical) {
      // Keep the key/value pairs together, so we don't have to do odd
      // gets with object keys later
      const enc = new Encoder({
        genTypes: gen.semanticTypes,
        canonical: gen.canonical,
        detectLoops: Boolean(gen.detectLoops), // Give enc its own loop detector
        dateType: gen.dateType,
        disallowUndefinedKeys: gen.disallowUndefinedKeys,
        collapseBigIntegers: gen.collapseBigIntegers,
      })
      const bs = new NoFilter({highWaterMark: gen.readableHighWaterMark})
      enc.pipe(bs)
      entries.sort(([a], [b]) => {
        // Both a and b are the keys
        enc.pushAny(a)
        const a_cbor = bs.read()
        enc.pushAny(b)
        const b_cbor = bs.read()
        return a_cbor.compare(b_cbor)
      })
      for (const [k, v] of entries) {
        if (gen.disallowUndefinedKeys && (typeof k === 'undefined')) {
          throw new Error('Invalid Map key: undefined')
        }
        if (!(gen.pushAny(k) && gen.pushAny(v))) {
          return false
        }
      }
    } else {
      for (const [k, v] of entries) {
        if (gen.disallowUndefinedKeys && (typeof k === 'undefined')) {
          throw new Error('Invalid Map key: undefined')
        }
        if (!(gen.pushAny(k) && gen.pushAny(v))) {
          return false
        }
      }
    }
    if (opts.indefinite) {
      if (!gen.push(BREAK)) {
        return false
      }
    }
    return true
  }

  /**
   * @param {Encoder} gen Encoder.
   * @param {NodeJS.TypedArray} obj Array to encode.
   * @returns {boolean} True on success.
   * @ignore
   */
  static _pushTypedArray(gen, obj) {
    // See https://tools.ietf.org/html/rfc8746

    let typ = 0b01000000
    let sz = obj.BYTES_PER_ELEMENT
    const {name} = obj.constructor

    if (name.startsWith('Float')) {
      typ |= 0b00010000
      sz /= 2
    } else if (!name.includes('U')) {
      typ |= 0b00001000
    }
    if (name.includes('Clamped') || ((sz !== 1) && !utils.isBigEndian())) {
      typ |= 0b00000100
    }
    typ |= {
      1: 0b00,
      2: 0b01,
      4: 0b10,
      8: 0b11,
    }[sz]
    if (!gen._pushTag(typ)) {
      return false
    }
    return Encoder._pushBuffer(
      gen,
      Buffer.from(obj.buffer, obj.byteOffset, obj.byteLength)
    )
  }

  /**
   * @param {Encoder} gen Encoder.
   * @param { ArrayBuffer } obj Array to encode.
   * @returns {boolean} True on success.
   * @ignore
   */
  static _pushArrayBuffer(gen, obj) {
    return Encoder._pushBuffer(gen, Buffer.from(obj))
  }

  /**
   * Encode the given object with indefinite length.  There are apparently
   * some (IMO) broken implementations of poorly-specified protocols that
   * REQUIRE indefinite-encoding.  See the example for how to add this as an
   * `encodeCBOR` function to an object or class to get indefinite encoding.
   *
   * @param {Encoder} gen The encoder to use.
   * @param {string|Buffer|Array|Map|object} [obj] The object to encode.  If
   *   null, use "this" instead.
   * @param {EncodingOptions} [options={}] Options for encoding.
   * @returns {boolean} True on success.
   * @throws {Error} No object to encode or invalid indefinite encoding.
   * @example <caption>Force indefinite encoding:</caption>
   * const o = {
   *   a: true,
   *   encodeCBOR: cbor.Encoder.encodeIndefinite,
   * }
   * const m = []
   * m.encodeCBOR = cbor.Encoder.encodeIndefinite
   * cbor.encodeOne([o, m])
   */
  static encodeIndefinite(gen, obj, options = {}) {
    if (obj == null) {
      if (this == null) {
        throw new Error('No object to encode')
      }
      obj = this
    }

    // TODO: consider other options
    const {chunkSize = 4096} = options

    let ret = true
    const objType = typeof obj
    let buf = null
    if (objType === 'string') {
      // TODO: make sure not to split surrogate pairs at the edges of chunks,
      // since such half-surrogates cannot be legally encoded as UTF-8.
      ret = ret && gen._pushUInt8((MT.UTF8_STRING << 5) | NUMBYTES.INDEFINITE)
      let offset = 0
      while (offset < obj.length) {
        const endIndex = offset + chunkSize
        ret = ret && gen._pushString(obj.slice(offset, endIndex))
        offset = endIndex
      }
      ret = ret && gen.push(BREAK)
    } else if ((buf = utils.bufferishToBuffer(obj))) {
      ret = ret && gen._pushUInt8((MT.BYTE_STRING << 5) | NUMBYTES.INDEFINITE)
      let offset = 0
      while (offset < buf.length) {
        const endIndex = offset + chunkSize
        ret = ret && Encoder._pushBuffer(gen, buf.slice(offset, endIndex))
        offset = endIndex
      }
      ret = ret && gen.push(BREAK)
    } else if (Array.isArray(obj)) {
      ret = ret && Encoder.pushArray(gen, obj, {
        indefinite: true,
      })
    } else if (obj instanceof Map) {
      ret = ret && Encoder._pushMap(gen, obj, {
        indefinite: true,
      })
    } else {
      if (objType !== 'object') {
        throw new Error('Invalid indefinite encoding')
      }
      ret = ret && gen._pushObject(obj, {
        indefinite: true,
        skipTypes: true,
      })
    }
    return ret
  }

  /**
   * Encode one or more JavaScript objects, and return a Buffer containing the
   * CBOR bytes.
   *
   * @param {...any} objs The objects to encode.
   * @returns {Buffer} The encoded objects.
   */
  static encode(...objs) {
    return new Encoder()._encodeAll(objs)
  }

  /**
   * Encode one or more JavaScript objects canonically (slower!), and return
   * a Buffer containing the CBOR bytes.
   *
   * @param {...any} objs The objects to encode.
   * @returns {Buffer} The encoded objects.
   */
  static encodeCanonical(...objs) {
    return new Encoder({
      canonical: true,
    })._encodeAll(objs)
  }

  /**
   * Encode one JavaScript object using the given options.
   *
   * @param {any} obj The object to encode.
   * @param {EncodingOptions} [options={}] Passed to the Encoder constructor.
   * @returns {Buffer} The encoded objects.
   * @static
   */
  static encodeOne(obj, options) {
    return new Encoder(options)._encodeAll([obj])
  }

  /**
   * Encode one JavaScript object using the given options in a way that
   * is more resilient to objects being larger than the highWaterMark
   * number of bytes.  As with the other static encode functions, this
   * will still use a large amount of memory.  Use a stream-based approach
   * directly if you need to process large and complicated inputs.
   *
   * @param {any} obj The object to encode.
   * @param {EncodingOptions} [options={}] Passed to the Encoder constructor.
   * @returns {Promise<Buffer>} A promise for the encoded buffer.
   */
  static encodeAsync(obj, options) {
    return new Promise((resolve, reject) => {
      const bufs = []
      const enc = new Encoder(options)
      enc.on('data', buf => bufs.push(buf))
      enc.on('error', reject)
      enc.on('finish', () => resolve(Buffer.concat(bufs)))
      enc.pushAny(obj)
      enc.end()
    })
  }

  /**
   * The currently supported set of semantic types.  May be modified by plugins.
   *
   * @type {SemanticMap}
   */
  static get SEMANTIC_TYPES() {
    return current_SEMANTIC_TYPES
  }

  static set SEMANTIC_TYPES(val) {
    current_SEMANTIC_TYPES = val
  }

  /**
   * Reset the supported semantic types to the original set, before any
   * plugins modified the list.
   */
  static reset() {
    Encoder.SEMANTIC_TYPES = {...SEMANTIC_TYPES}
  }
}

Object.assign(SEMANTIC_TYPES, {
  Array: Encoder.pushArray,
  Date: Encoder._pushDate,
  Buffer: Encoder._pushBuffer,
  [Buffer.name]: Encoder._pushBuffer, // Might be mangled
  Map: Encoder._pushMap,
  NoFilter: Encoder._pushNoFilter,
  [NoFilter.name]: Encoder._pushNoFilter, // Mßight be mangled
  RegExp: Encoder._pushRegexp,
  Set: Encoder._pushSet,
  ArrayBuffer: Encoder._pushArrayBuffer,
  Uint8ClampedArray: Encoder._pushTypedArray,
  Uint8Array: Encoder._pushTypedArray,
  Uint16Array: Encoder._pushTypedArray,
  Uint32Array: Encoder._pushTypedArray,
  Int8Array: Encoder._pushTypedArray,
  Int16Array: Encoder._pushTypedArray,
  Int32Array: Encoder._pushTypedArray,
  Float32Array: Encoder._pushTypedArray,
  Float64Array: Encoder._pushTypedArray,
  URL: Encoder._pushURL,
  Boolean: Encoder._pushBoxed,
  Number: Encoder._pushBoxed,
  String: Encoder._pushBoxed,
})

// Safari needs to get better.
if (typeof BigUint64Array !== 'undefined') {
  SEMANTIC_TYPES[BigUint64Array.name] = Encoder._pushTypedArray
}
if (typeof BigInt64Array !== 'undefined') {
  SEMANTIC_TYPES[BigInt64Array.name] = Encoder._pushTypedArray
}

Encoder.reset()
module.exports = Encoder
