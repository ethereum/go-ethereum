'use strict'

const Commented = require('./commented')
const Diagnose = require('./diagnose')
const Decoder = require('./decoder')
const Encoder = require('./encoder')
const Simple = require('./simple')
const Tagged = require('./tagged')
const Map = require('./map')
const SharedValueEncoder = require('./sharedValueEncoder')

module.exports = {
  Commented,
  Diagnose,
  Decoder,
  Encoder,
  Simple,
  Tagged,
  Map,
  SharedValueEncoder,

  /**
   * Convenience name for {@linkcode Commented.comment}.
   */
  comment: Commented.comment,

  /**
   * Convenience name for {@linkcode Decoder.decodeAll}.
   */
  decodeAll: Decoder.decodeAll,

  /**
   * Convenience name for {@linkcode Decoder.decodeFirst}.
   */
  decodeFirst: Decoder.decodeFirst,

  /**
   * Convenience name for {@linkcode Decoder.decodeAllSync}.
   */
  decodeAllSync: Decoder.decodeAllSync,

  /**
   * Convenience name for {@linkcode Decoder.decodeFirstSync}.
   */
  decodeFirstSync: Decoder.decodeFirstSync,

  /**
   * Convenience name for {@linkcode Diagnose.diagnose}.
   */
  diagnose: Diagnose.diagnose,

  /**
   * Convenience name for {@linkcode Encoder.encode}.
   */
  encode: Encoder.encode,

  /**
   * Convenience name for {@linkcode Encoder.encodeCanonical}.
   */
  encodeCanonical: Encoder.encodeCanonical,

  /**
   * Convenience name for {@linkcode Encoder.encodeOne}.
   */
  encodeOne: Encoder.encodeOne,

  /**
   * Convenience name for {@linkcode Encoder.encodeAsync}.
   */
  encodeAsync: Encoder.encodeAsync,

  /**
   * Convenience name for {@linkcode Decoder.decodeFirstSync}.
   */
  decode: Decoder.decodeFirstSync,

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
  leveldb: {
    decode: Decoder.decodeFirstSync,
    encode: Encoder.encode,
    buffer: true,
    name: 'cbor',
  },

  /**
   * Reset everything that we can predict a plugin might have altered in good
   * faith.  For now that includes the default set of tags that decoding and
   * encoding will use.
   */
  reset() {
    Encoder.reset()
    Tagged.reset()
  },
}
