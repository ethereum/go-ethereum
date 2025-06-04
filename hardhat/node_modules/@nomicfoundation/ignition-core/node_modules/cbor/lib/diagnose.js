'use strict'

const stream = require('stream')
const Decoder = require('./decoder')
const utils = require('./utils')
const NoFilter = require('nofilter')
const {MT, SYMS} = require('./constants')

/**
 * Things that can act as inputs, from which a NoFilter can be created.
 *
 * @typedef {string|Buffer|ArrayBuffer|Uint8Array|Uint8ClampedArray
 *   |DataView|stream.Readable} BufferLike
 */

/**
 * @typedef DiagnoseOptions
 * @property {string} [separator='\n'] Output between detected objects.
 * @property {boolean} [stream_errors=false] Put error info into the
 *   output stream.
 * @property {number} [max_depth=-1] The maximum depth to parse.
 *   Use -1 for "until you run out of memory".  Set this to a finite
 *   positive number for un-trusted inputs.  Most standard inputs won't nest
 *   more than 100 or so levels; I've tested into the millions before
 *   running out of memory.
 * @property {object} [tags] Mapping from tag number to function(v),
 *   where v is the decoded value that comes after the tag, and where the
 *   function returns the correctly-created value for that tag.
 * @property {boolean} [preferWeb=false] If true, prefer Uint8Arrays to
 *   be generated instead of node Buffers.  This might turn on some more
 *   changes in the future, so forward-compatibility is not guaranteed yet.
 * @property {BufferEncoding} [encoding='hex'] The encoding of input, ignored if
 *   input is not string.
 */
/**
 * @callback diagnoseCallback
 * @param {Error} [error] If one was generated.
 * @param {string} [value] The diagnostic value.
 * @returns {void}
 */
/**
 * @param {DiagnoseOptions|diagnoseCallback|string} opts Options,
 *   the callback, or input incoding.
 * @param {diagnoseCallback} [cb] Called on completion.
 * @returns {{options: DiagnoseOptions, cb: diagnoseCallback}} Normalized.
 * @throws {TypeError} Unknown option type.
 * @private
 */
function normalizeOptions(opts, cb) {
  switch (typeof opts) {
    case 'function':
      return {options: {}, cb: /** @type {diagnoseCallback} */ (opts)}
    case 'string':
      return {options: {encoding: /** @type {BufferEncoding} */ (opts)}, cb}
    case 'object':
      return {options: opts || {}, cb}
    default:
      throw new TypeError('Unknown option type')
  }
}

/**
 * Output the diagnostic format from a stream of CBOR bytes.
 *
 * @extends stream.Transform
 */
class Diagnose extends stream.Transform {
  /**
   * Creates an instance of Diagnose.
   *
   * @param {DiagnoseOptions} [options={}] Options for creation.
   */
  constructor(options = {}) {
    const {
      separator = '\n',
      stream_errors = false,
      // Decoder options
      tags,
      max_depth,
      preferWeb,
      encoding,
      // Stream.Transform options
      ...superOpts
    } = options
    super({
      ...superOpts,
      readableObjectMode: false,
      writableObjectMode: false,
    })

    this.float_bytes = -1
    this.separator = separator
    this.stream_errors = stream_errors
    this.parser = new Decoder({
      tags,
      max_depth,
      preferWeb,
      encoding,
    })
    this.parser.on('more-bytes', this._on_more.bind(this))
    this.parser.on('value', this._on_value.bind(this))
    this.parser.on('start', this._on_start.bind(this))
    this.parser.on('stop', this._on_stop.bind(this))
    this.parser.on('data', this._on_data.bind(this))
    this.parser.on('error', this._on_error.bind(this))
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
    this.parser.write(fresh, encoding, cb)
  }

  /**
   * Flushing.
   *
   * @param {stream.TransformCallback} cb Callback when done.
   * @ignore
   */
  _flush(cb) {
    this.parser._flush(er => {
      if (this.stream_errors) {
        if (er) {
          this._on_error(er)
        }
        return cb()
      }
      return cb(er)
    })
  }

  /**
   * Convenience function to return a string in diagnostic format.
   *
   * @param {BufferLike} input The CBOR bytes to format.
   * @param {DiagnoseOptions |diagnoseCallback|string} [options={}]
   *   Options, the callback, or the input encoding.
   * @param {diagnoseCallback} [cb] Callback.
   * @returns {Promise} If callback not specified.
   * @throws {TypeError} Input not provided.
   */
  static diagnose(input, options = {}, cb = null) {
    if (input == null) {
      throw new TypeError('input required')
    }
    ({options, cb} = normalizeOptions(options, cb))
    const {encoding = 'hex', ...opts} = options

    const bs = new NoFilter()
    const d = new Diagnose(opts)
    let p = null
    if (typeof cb === 'function') {
      d.on('end', () => cb(null, bs.toString('utf8')))
      d.on('error', cb)
    } else {
      p = new Promise((resolve, reject) => {
        d.on('end', () => resolve(bs.toString('utf8')))
        d.on('error', reject)
      })
    }
    d.pipe(bs)
    utils.guessEncoding(input, encoding).pipe(d)
    return p
  }

  /**
   * @ignore
   */
  _on_error(er) {
    if (this.stream_errors) {
      this.push(er.toString())
    } else {
      this.emit('error', er)
    }
  }

  /** @private */
  _on_more(mt, len, parent_mt, pos) {
    if (mt === MT.SIMPLE_FLOAT) {
      this.float_bytes = {
        2: 1,
        4: 2,
        8: 3,
      }[len]
    }
  }

  /** @private */
  _fore(parent_mt, pos) {
    switch (parent_mt) {
      case MT.BYTE_STRING:
      case MT.UTF8_STRING:
      case MT.ARRAY:
        if (pos > 0) {
          this.push(', ')
        }
        break
      case MT.MAP:
        if (pos > 0) {
          if (pos % 2) {
            this.push(': ')
          } else {
            this.push(', ')
          }
        }
    }
  }

  /** @private */
  _on_value(val, parent_mt, pos) {
    if (val === SYMS.BREAK) {
      return
    }
    this._fore(parent_mt, pos)
    const fb = this.float_bytes
    this.float_bytes = -1
    this.push(utils.cborValueToString(val, fb))
  }

  /** @private */
  _on_start(mt, tag, parent_mt, pos) {
    this._fore(parent_mt, pos)
    switch (mt) {
      case MT.TAG:
        this.push(`${tag}(`)
        break
      case MT.ARRAY:
        this.push('[')
        break
      case MT.MAP:
        this.push('{')
        break
      case MT.BYTE_STRING:
      case MT.UTF8_STRING:
        this.push('(')
        break
    }
    if (tag === SYMS.STREAM) {
      this.push('_ ')
    }
  }

  /** @private */
  _on_stop(mt) {
    switch (mt) {
      case MT.TAG:
        this.push(')')
        break
      case MT.ARRAY:
        this.push(']')
        break
      case MT.MAP:
        this.push('}')
        break
      case MT.BYTE_STRING:
      case MT.UTF8_STRING:
        this.push(')')
        break
    }
  }

  /** @private */
  _on_data() {
    this.push(this.separator)
  }
}

module.exports = Diagnose
