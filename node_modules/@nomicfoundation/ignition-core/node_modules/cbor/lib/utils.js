'use strict'

const {Buffer} = require('buffer')
const NoFilter = require('nofilter')
const stream = require('stream')
const constants = require('./constants')
const {NUMBYTES, SHIFT32, BI, SYMS} = constants
const MAX_SAFE_HIGH = 0x1fffff

/**
 * Convert a UTF8-encoded Buffer to a JS string.  If possible, throw an error
 * on invalid UTF8.  Byte Order Marks are not looked at or stripped.
 *
 * @private
 */
const td = new TextDecoder('utf8', {fatal: true, ignoreBOM: true})
exports.utf8 = buf => td.decode(buf)
exports.utf8.checksUTF8 = true

function isReadable(s) {
  // Is this a readable stream?  In the webpack version, instanceof isn't
  // working correctly.
  if (s instanceof stream.Readable) {
    return true
  }
  return ['read', 'on', 'pipe'].every(f => typeof s[f] === 'function')
}

exports.isBufferish = function isBufferish(b) {
  return b &&
    (typeof b === 'object') &&
    ((Buffer.isBuffer(b)) ||
      (b instanceof Uint8Array) ||
      (b instanceof Uint8ClampedArray) ||
      (b instanceof ArrayBuffer) ||
      (b instanceof DataView))
}

exports.bufferishToBuffer = function bufferishToBuffer(b) {
  if (Buffer.isBuffer(b)) {
    return b
  } else if (ArrayBuffer.isView(b)) {
    return Buffer.from(b.buffer, b.byteOffset, b.byteLength)
  } else if (b instanceof ArrayBuffer) {
    return Buffer.from(b)
  }
  return null
}

exports.parseCBORint = function parseCBORint(ai, buf) {
  switch (ai) {
    case NUMBYTES.ONE:
      return buf.readUInt8(0)
    case NUMBYTES.TWO:
      return buf.readUInt16BE(0)
    case NUMBYTES.FOUR:
      return buf.readUInt32BE(0)
    case NUMBYTES.EIGHT: {
      const f = buf.readUInt32BE(0)
      const g = buf.readUInt32BE(4)
      if (f > MAX_SAFE_HIGH) {
        return (BigInt(f) * BI.SHIFT32) + BigInt(g)
      }
      return (f * SHIFT32) + g
    }
    default:
      throw new Error(`Invalid additional info for int: ${ai}`)
  }
}

exports.writeHalf = function writeHalf(buf, half) {
  // Assume 0, -0, NaN, Infinity, and -Infinity have already been caught

  // HACK: everyone settle in.  This isn't going to be pretty.
  // Translate cn-cbor's C code (from Carsten Borman):

  // uint32_t be32;
  // uint16_t be16, u16;
  // union {
  //   float f;
  //   uint32_t u;
  // } u32;
  // u32.f = float_val;

  const u32 = Buffer.allocUnsafe(4)
  u32.writeFloatBE(half, 0)
  const u = u32.readUInt32BE(0)

  // If ((u32.u & 0x1FFF) == 0) { /* worth trying half */

  // hildjj: If the lower 13 bits aren't 0,
  // we will lose precision in the conversion.
  // mant32 = 24bits, mant16 = 11bits, 24-11 = 13
  if ((u & 0x1FFF) !== 0) {
    return false
  }

  // Sign, exponent, mantissa
  //   int s16 = (u32.u >> 16) & 0x8000;
  //   int exp = (u32.u >> 23) & 0xff;
  //   int mant = u32.u & 0x7fffff;

  let s16 = (u >> 16) & 0x8000 // Top bit is sign
  const exp = (u >> 23) & 0xff // Then 5 bits of exponent
  const mant = u & 0x7fffff

  // Hildjj: zeros already handled.  Assert if you don't believe me.
  //   if (exp == 0 && mant == 0)
  //     ;              /* 0.0, -0.0 */

  //   else if (exp >= 113 && exp <= 142) /* normalized */
  //     s16 += ((exp - 112) << 10) + (mant >> 13);

  if ((exp >= 113) && (exp <= 142)) {
    s16 += ((exp - 112) << 10) + (mant >> 13)
  } else if ((exp >= 103) && (exp < 113)) {
    // Denormalized numbers
    //   else if (exp >= 103 && exp < 113) { /* denorm, exp16 = 0 */
    //     if (mant & ((1 << (126 - exp)) - 1))
    //       goto float32;         /* loss of precision */
    //     s16 += ((mant + 0x800000) >> (126 - exp));

    if (mant & ((1 << (126 - exp)) - 1)) {
      return false
    }
    s16 += ((mant + 0x800000) >> (126 - exp))
  } else {
  //   } else if (exp == 255 && mant == 0) { /* Inf */
  //     s16 += 0x7c00;

    // hildjj: Infinity already handled

    //   } else
    //     goto float32;           /* loss of range */

    return false
  }

  // Done
  //   ensure_writable(3);
  //   u16 = s16;
  //   be16 = hton16p((const uint8_t*)&u16);
  buf.writeUInt16BE(s16)
  return true
}

exports.parseHalf = function parseHalf(buf) {
  const sign = buf[0] & 0x80 ? -1 : 1
  const exp = (buf[0] & 0x7C) >> 2
  const mant = ((buf[0] & 0x03) << 8) | buf[1]
  if (!exp) {
    return sign * 5.9604644775390625e-8 * mant
  } else if (exp === 0x1f) {
    return sign * (mant ? NaN : Infinity)
  }
  return sign * (2 ** (exp - 25)) * (1024 + mant)
}

exports.parseCBORfloat = function parseCBORfloat(buf) {
  switch (buf.length) {
    case 2:
      return exports.parseHalf(buf)
    case 4:
      return buf.readFloatBE(0)
    case 8:
      return buf.readDoubleBE(0)
    default:
      throw new Error(`Invalid float size: ${buf.length}`)
  }
}

exports.hex = function hex(s) {
  return Buffer.from(s.replace(/^0x/, ''), 'hex')
}

exports.bin = function bin(s) {
  s = s.replace(/\s/g, '')
  let start = 0
  let end = (s.length % 8) || 8
  const chunks = []
  while (end <= s.length) {
    chunks.push(parseInt(s.slice(start, end), 2))
    start = end
    end += 8
  }
  return Buffer.from(chunks)
}

exports.arrayEqual = function arrayEqual(a, b) {
  if ((a == null) && (b == null)) {
    return true
  }
  if ((a == null) || (b == null)) {
    return false
  }
  return (a.length === b.length) && a.every((elem, i) => elem === b[i])
}

exports.bufferToBigInt = function bufferToBigInt(buf) {
  return BigInt(`0x${buf.toString('hex')}`)
}

exports.cborValueToString = function cborValueToString(val, float_bytes = -1) {
  switch (typeof val) {
    case 'symbol': {
      switch (val) {
        case SYMS.NULL:
          return 'null'
        case SYMS.UNDEFINED:
          return 'undefined'
        case SYMS.BREAK:
          return 'BREAK'
      }
      // Impossible in node 10
      /* istanbul ignore if */
      if (val.description) {
        return val.description
      }
      // On node10, Symbol doesn't have description.  Parse it out of the
      // toString value, which looks like `Symbol(foo)`.
      const s = val.toString()
      const m = s.match(/^Symbol\((?<name>.*)\)/)
      /* istanbul ignore if */
      if (m && m.groups.name) {
        // Impossible in node 12+
        /* istanbul ignore next */
        return m.groups.name
      }
      return 'Symbol'
    }
    case 'string':
      return JSON.stringify(val)
    case 'bigint':
      return val.toString()
    case 'number': {
      const s = Object.is(val, -0) ? '-0' : String(val)
      return (float_bytes > 0) ? `${s}_${float_bytes}` : s
    }
    case 'object': {
      if (!val) {
        return 'null'
      }
      const buf = exports.bufferishToBuffer(val)
      if (buf) {
        const hex = buf.toString('hex')
        return (float_bytes === -Infinity) ? hex : `h'${hex}'`
      }
      if (val && typeof val[Symbol.for('nodejs.util.inspect.custom')] === 'function') {
        return val[Symbol.for('nodejs.util.inspect.custom')]()
      }
      // Shouldn't get non-empty arrays here
      if (Array.isArray(val)) {
        return '[]'
      }
      // This should be all that is left
      return '{}'
    }
  }
  return String(val)
}

exports.guessEncoding = function guessEncoding(input, encoding) {
  if (typeof input === 'string') {
    return new NoFilter(input, (encoding == null) ? 'hex' : encoding)
  }
  const buf = exports.bufferishToBuffer(input)
  if (buf) {
    return new NoFilter(buf)
  }
  if (isReadable(input)) {
    return input
  }
  throw new Error('Unknown input type')
}

const B64URL_SWAPS = {
  '=': '',
  '+': '-',
  '/': '_',
}

/**
 * @param {Buffer|Uint8Array|Uint8ClampedArray|ArrayBuffer|DataView} buf
 *   Buffer to convert.
 * @returns {string} Base64url string.
 * @private
 */
exports.base64url = function base64url(buf) {
  return exports.bufferishToBuffer(buf)
    .toString('base64')
    .replace(/[=+/]/g, c => B64URL_SWAPS[c])
}

/**
 * @param {Buffer|Uint8Array|Uint8ClampedArray|ArrayBuffer|DataView} buf
 *   Buffer to convert.
 * @returns {string} Base64 string.
 * @private
 */
exports.base64 = function base64(buf) {
  return exports.bufferishToBuffer(buf).toString('base64')
}

exports.isBigEndian = function isBigEndian() {
  const array = new Uint8Array(4)
  const view = new Uint32Array(array.buffer)
  return !((view[0] = 1) & array[0])
}
