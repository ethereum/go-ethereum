'use strict'

exports.Commented = require('./commented')
exports.Diagnose = require('./diagnose')
exports.Decoder = require('./decoder')
exports.Encoder = require('./encoder')
exports.Simple = require('./simple')
exports.Tagged = require('./tagged')
exports.Map = require('./map')

/**
 * Convenience name for {@linkcode Commented.comment}.
 */
exports.comment = exports.Commented.comment

/**
 * Convenience name for {@linkcode Decoder.decodeAll}.
 */
exports.decodeAll = exports.Decoder.decodeAll

/**
 * Convenience name for {@linkcode Decoder.decodeFirst}.
 */
exports.decodeFirst = exports.Decoder.decodeFirst

/**
 * Convenience name for {@linkcode Decoder.decodeAllSync}.
 */
exports.decodeAllSync = exports.Decoder.decodeAllSync

/**
 * Convenience name for {@linkcode Decoder.decodeFirstSync}.
 */
exports.decodeFirstSync = exports.Decoder.decodeFirstSync

/**
 * Convenience name for {@linkcode Diagnose.diagnose}.
 */
exports.diagnose = exports.Diagnose.diagnose

/**
 * Convenience name for {@linkcode Encoder.encode}.
 */
exports.encode = exports.Encoder.encode

/**
 * Convenience name for {@linkcode Encoder.encodeCanonical}.
 */
exports.encodeCanonical = exports.Encoder.encodeCanonical

/**
 * Convenience name for {@linkcode Encoder.encodeOne}.
 */
exports.encodeOne = exports.Encoder.encodeOne

/**
 * Convenience name for {@linkcode Encoder.encodeAsync}.
 */
exports.encodeAsync = exports.Encoder.encodeAsync

/**
 * Convenience name for {@linkcode Decoder.decodeFirstSync}.
 */
exports.decode = exports.Decoder.decodeFirstSync

/**
 * The codec information for
 * {@link https://github.com/Level/encoding-down encoding-down}, which is a
 * codec framework for leveldb.  CBOR is a particularly convenient format for
 * both keys and values, as it can deal with a lot of types that JSON can't
 * handle without losing type information.
 *
 * @example
 * const level = require('level')
 * const cbor = require('cbor')
 *
 * async function putget() {
 *   const db = level('./db', {
 *     keyEncoding: cbor.leveldb,
 *     valueEncoding: cbor.leveldb,
 *   })
 *
 *   await db.put({a: 1}, 9857298342094820394820394820398234092834n)
 *   const val = await db.get({a: 1})
 * }
 */
exports.leveldb = {
  decode: exports.Decoder.decodeFirstSync,
  encode: exports.Encoder.encode,
  buffer: true,
  name: 'cbor',
}

/**
 * Reset everything that we can predict a plugin might have altered in good
 * faith.  For now that includes the default set of tags that decoding and
 * encoding will use.
 */
exports.reset = function reset() {
  exports.Encoder.reset()
  exports.Tagged.reset()
}
