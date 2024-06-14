'use strict'

const Encoder = require('./encoder')
const ObjectRecorder = require('./objectRecorder')
const {Buffer} = require('buffer')

/**
 * Implement value sharing.
 *
 * @see {@link cbor.schmorp.de/value-sharing}
 */
class SharedValueEncoder extends Encoder {
  constructor(opts) {
    super(opts)
    this.valueSharing = new ObjectRecorder()
  }

  /**
   * @param {object} obj Object to encode.
   * @param {import('./encoder').ObjectOptions} [opts] Options for encoding
   *   this object.
   * @returns {boolean} True on success.
   * @throws {Error} Loop detected.
   * @ignore
   */
  _pushObject(obj, opts) {
    if (obj !== null) {
      const shared = this.valueSharing.check(obj)
      switch (shared) {
        case ObjectRecorder.FIRST:
          // Prefix with tag 28
          this._pushTag(28)
          break
        case ObjectRecorder.NEVER:
          // Do nothing
          break
        default:
          return this._pushTag(29) && this._pushIntNum(shared)
      }
    }
    return super._pushObject(obj, opts)
  }

  /**
   * Between encoding runs, stop recording, and start outputing correct tags.
   */
  stopRecording() {
    this.valueSharing.stop()
  }

  /**
   * Remove the existing recording and start over.  Do this between encoding
   * pairs.
   */
  clearRecording() {
    this.valueSharing.clear()
  }

  /**
   * Encode one or more JavaScript objects, and return a Buffer containing the
   * CBOR bytes.
   *
   * @param {...any} objs The objects to encode.
   * @returns {Buffer} The encoded objects.
   */
  static encode(...objs) {
    const enc = new SharedValueEncoder()
    // eslint-disable-next-line no-empty-function
    enc.on('data', () => {}) // Sink all writes

    for (const o of objs) {
      enc.pushAny(o)
    }
    enc.stopRecording()
    enc.removeAllListeners('data')
    return enc._encodeAll(objs)
  }

  // eslint-disable-next-line jsdoc/require-returns-check
  /**
   * Encode one or more JavaScript objects canonically (slower!), and return
   * a Buffer containing the CBOR bytes.
   *
   * @param {...any} objs The objects to encode.
   * @returns {Buffer} Never.
   * @throws {Error} Always.  This combination doesn't work at the moment.
   */
  static encodeCanonical(...objs) {
    throw new Error('Cannot encode canonically in a SharedValueEncoder, which serializes objects multiple times.')
  }

  /**
   * Encode one JavaScript object using the given options.
   *
   * @param {any} obj The object to encode.
   * @param {import('./encoder').EncodingOptions} [options={}]
   *   Passed to the Encoder constructor.
   * @returns {Buffer} The encoded objects.
   * @static
   */
  static encodeOne(obj, options) {
    const enc = new SharedValueEncoder(options)
    // eslint-disable-next-line no-empty-function
    enc.on('data', () => {}) // Sink all writes
    enc.pushAny(obj)
    enc.stopRecording()
    enc.removeAllListeners('data')
    return enc._encodeAll([obj])
  }

  /**
   * Encode one JavaScript object using the given options in a way that
   * is more resilient to objects being larger than the highWaterMark
   * number of bytes.  As with the other static encode functions, this
   * will still use a large amount of memory.  Use a stream-based approach
   * directly if you need to process large and complicated inputs.
   *
   * @param {any} obj The object to encode.
   * @param {import('./encoder').EncodingOptions} [options={}]
   *   Passed to the Encoder constructor.
   * @returns {Promise<Buffer>} A promise for the encoded buffer.
   */
  static encodeAsync(obj, options) {
    return new Promise((resolve, reject) => {
      /** @type {Buffer[]} */
      const bufs = []
      const enc = new SharedValueEncoder(options)
      // eslint-disable-next-line no-empty-function
      enc.on('data', () => {})
      enc.on('error', reject)
      enc.on('finish', () => resolve(Buffer.concat(bufs)))
      enc.pushAny(obj)
      enc.stopRecording()
      enc.removeAllListeners('data')
      enc.on('data', buf => bufs.push(buf))
      enc.pushAny(obj)
      enc.end()
    })
  }
}

module.exports = SharedValueEncoder
