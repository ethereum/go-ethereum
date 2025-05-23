'use strict'

/**
 * @enum {number}
 */
exports.MT = {
  POS_INT: 0,
  NEG_INT: 1,
  BYTE_STRING: 2,
  UTF8_STRING: 3,
  ARRAY: 4,
  MAP: 5,
  TAG: 6,
  SIMPLE_FLOAT: 7,
}

/**
 * @enum {number}
 */
exports.TAG = {
  DATE_STRING: 0,
  DATE_EPOCH: 1,
  POS_BIGINT: 2,
  NEG_BIGINT: 3,
  DECIMAL_FRAC: 4,
  BIGFLOAT: 5,
  BASE64URL_EXPECTED: 21,
  BASE64_EXPECTED: 22,
  BASE16_EXPECTED: 23,
  CBOR: 24,
  URI: 32,
  BASE64URL: 33,
  BASE64: 34,
  REGEXP: 35,
  MIME: 36,
  // https://github.com/input-output-hk/cbor-sets-spec/blob/master/CBOR_SETS.md
  SET: 258,
}

/**
 * @enum {number}
 */
exports.NUMBYTES = {
  ZERO: 0,
  ONE: 24,
  TWO: 25,
  FOUR: 26,
  EIGHT: 27,
  INDEFINITE: 31,
}

/**
 * @enum {number}
 */
exports.SIMPLE = {
  FALSE: 20,
  TRUE: 21,
  NULL: 22,
  UNDEFINED: 23,
}

exports.SYMS = {
  NULL: Symbol.for('github.com/hildjj/node-cbor/null'),
  UNDEFINED: Symbol.for('github.com/hildjj/node-cbor/undef'),
  PARENT: Symbol.for('github.com/hildjj/node-cbor/parent'),
  BREAK: Symbol.for('github.com/hildjj/node-cbor/break'),
  STREAM: Symbol.for('github.com/hildjj/node-cbor/stream'),
}

exports.SHIFT32 = 0x100000000

exports.BI = {
  MINUS_ONE: BigInt(-1),
  NEG_MAX: BigInt(-1) - BigInt(Number.MAX_SAFE_INTEGER),
  MAXINT32: BigInt('0xffffffff'),
  MAXINT64: BigInt('0xffffffffffffffff'),
  SHIFT32: BigInt(exports.SHIFT32),
}

