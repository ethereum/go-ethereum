'use strict'

const stream = require('stream')
const utils = require('./utils')
const Decoder = require('./decoder')
const NoFilter = require('nofilter')
const {MT, NUMBYTES, SYMS} = require('./constants')
const {Buffer} = require('buffer')

function plural(c) {
  if (c > 1) {
    return 's'
  }
  return ''
}

/**
 * @typedef CommentOptions
 * @property {number} [max_depth=10] How many times to indent
 *   the dashes.
 * @property {number} [depth=1] Initial indentation depth.
 * @property {boolean} [no_summary=false] If true, omit the summary
 *   of the full bytes read at the end.
 * @property {object} [tags] Mapping from tag number to function(v),
 *   where v is the decoded value that comes after the tag, and where the
 *   function returns the correctly-created value for that tag.
 * @property {boolean} [preferWeb=false] If true, prefer Uint8Arrays to
 *   be generated instead of node Buffers.  This might turn on some more
 *   changes in the future, so forward-compatibility is not guaranteed yet.
 * @property {BufferEncoding} [encoding='hex'] Encoding to use for input, if it
 *   is a string.
 */
/**
 * @callback commentCallback
 * @param {Error} [error] If one was generated.
 * @param {string} [commented] The comment string.
 * @returns {void}
 */
/**
 * Normalize inputs to the static functions.
 *
 * @param {CommentOptions|commentCallback|string|number} opts Encoding,
 *   max_depth, or callback.
 * @param {commentCallback} [cb] Called on completion.
 * @returns {{options: CommentOptions, cb: commentCallback}} Normalized value.
 * @throws {TypeError} Unknown option type.
 * @private
 */
function normalizeOptions(opts, cb) {
  switch (typeof opts) {
    case 'function':
      return {options: {}, cb: /** @type {commentCallback} */ (opts)}
    case 'string':
      return {options: {encoding: /** @type {BufferEncoding} */ (opts)}, cb}
    case 'number':
      return {options: {max_depth: opts}, cb}
    case 'object':
      return {options: opts || {}, cb}
    default:
      throw new TypeError('Unknown option type')
  }
}

/**
 * Generate the expanded format of RFC 8949, section 3.2.2.
 *
 * @extends stream.Transform
 */
class Commented extends stream.Transform {
  /**
   * Create a CBOR commenter.
   *
   * @param {CommentOptions} [options={}] Stream options.
   */
  constructor(options = {}) {
    const {
      depth = 1,
      max_depth = 10,
      no_summary = false,
      // Decoder options
      tags = {},
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

    this.depth = depth
    this.max_depth = max_depth
    this.all = new NoFilter()

    if (!tags[24]) {
      tags[24] = this._tag_24.bind(this)
    }
    this.parser = new Decoder({
      tags,
      max_depth,
      preferWeb,
      encoding,
    })
    this.parser.on('value', this._on_value.bind(this))
    this.parser.on('start', this._on_start.bind(this))
    this.parser.on('start-string', this._on_start_string.bind(this))
    this.parser.on('stop', this._on_stop.bind(this))
    this.parser.on('more-bytes', this._on_more.bind(this))
    this.parser.on('error', this._on_error.bind(this))
    if (!no_summary) {
      this.parser.on('data', this._on_data.bind(this))
    }
    this.parser.bs.on('read', this._on_read.bind(this))
  }

  /**
   * @param {Buffer} v Descend into embedded CBOR.
   * @private
   */
  _tag_24(v) {
    const c = new Commented({depth: this.depth + 1, no_summary: true})

    c.on('data', b => this.push(b))
    c.on('error', er => this.emit('error', er))
    c.end(v)
  }

  _transform(fresh, encoding, cb) {
    this.parser.write(fresh, encoding, cb)
  }

  _flush(cb) {
    // TODO: find the test that covers this, and look at the return value
    return this.parser._flush(cb)
  }

  /**
   * Comment on an input Buffer or string, creating a string passed to the
   * callback.  If callback not specified, a promise is returned.
   *
   * @static
   * @param {string|Buffer|ArrayBuffer|Uint8Array|Uint8ClampedArray
   *   |DataView|stream.Readable} input Something to parse.
   * @param {CommentOptions|commentCallback|string|number} [options={}]
   *   Encoding, max_depth, or callback.
   * @param {commentCallback} [cb] If specified, called on completion.
   * @returns {Promise} If cb not specified.
   * @throws {Error} Input required.
   */
  static comment(input, options = {}, cb = null) {
    if (input == null) {
      throw new Error('input required')
    }
    ({options, cb} = normalizeOptions(options, cb))
    const bs = new NoFilter()
    const {encoding = 'hex', ...opts} = options
    const d = new Commented(opts)
    let p = null

    if (typeof cb === 'function') {
      d.on('end', () => {
        cb(null, bs.toString('utf8'))
      })
      d.on('error', cb)
    } else {
      p = new Promise((resolve, reject) => {
        d.on('end', () => {
          resolve(bs.toString('utf8'))
        })
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
    this.push('ERROR: ')
    this.push(er.toString())
    this.push('\n')
  }

  /**
   * @ignore
   */
  _on_read(buf) {
    this.all.write(buf)
    const hex = buf.toString('hex')

    this.push(new Array(this.depth + 1).join('  '))
    this.push(hex)

    let ind = ((this.max_depth - this.depth) * 2) - hex.length
    if (ind < 1) {
      ind = 1
    }
    this.push(new Array(ind + 1).join(' '))
    this.push('-- ')
  }

  /**
   * @ignore
   */
  _on_more(mt, len, parent_mt, pos) {
    let desc = ''

    this.depth++
    switch (mt) {
      case MT.POS_INT:
        desc = 'Positive number,'
        break
      case MT.NEG_INT:
        desc = 'Negative number,'
        break
      case MT.ARRAY:
        desc = 'Array, length'
        break
      case MT.MAP:
        desc = 'Map, count'
        break
      case MT.BYTE_STRING:
        desc = 'Bytes, length'
        break
      case MT.UTF8_STRING:
        desc = 'String, length'
        break
      case MT.SIMPLE_FLOAT:
        if (len === 1) {
          desc = 'Simple value,'
        } else {
          desc = 'Float,'
        }
        break
    }
    this.push(`${desc} next ${len} byte${plural(len)}\n`)
  }

  /**
   * @ignore
   */
  _on_start_string(mt, len, parent_mt, pos) {
    let desc = ''

    this.depth++
    switch (mt) {
      case MT.BYTE_STRING:
        desc = `Bytes, length: ${len}`
        break
      case MT.UTF8_STRING:
        desc = `String, length: ${len.toString()}`
        break
    }
    this.push(`${desc}\n`)
  }

  /**
   * @ignore
   */
  _on_start(mt, tag, parent_mt, pos) {
    this.depth++
    switch (parent_mt) {
      case MT.ARRAY:
        this.push(`[${pos}], `)
        break
      case MT.MAP:
        if (pos % 2) {
          this.push(`{Val:${Math.floor(pos / 2)}}, `)
        } else {
          this.push(`{Key:${Math.floor(pos / 2)}}, `)
        }
        break
    }
    switch (mt) {
      case MT.TAG:
        this.push(`Tag #${tag}`)
        if (tag === 24) {
          this.push(' Encoded CBOR data item')
        }
        break
      case MT.ARRAY:
        if (tag === SYMS.STREAM) {
          this.push('Array (streaming)')
        } else {
          this.push(`Array, ${tag} item${plural(tag)}`)
        }
        break
      case MT.MAP:
        if (tag === SYMS.STREAM) {
          this.push('Map (streaming)')
        } else {
          this.push(`Map, ${tag} pair${plural(tag)}`)
        }
        break
      case MT.BYTE_STRING:
        this.push('Bytes (streaming)')
        break
      case MT.UTF8_STRING:
        this.push('String (streaming)')
        break
    }
    this.push('\n')
  }

  /**
   * @ignore
   */
  _on_stop(mt) {
    this.depth--
  }

  /**
   * @private
   */
  _on_value(val, parent_mt, pos, ai) {
    if (val !== SYMS.BREAK) {
      switch (parent_mt) {
        case MT.ARRAY:
          this.push(`[${pos}], `)
          break
        case MT.MAP:
          if (pos % 2) {
            this.push(`{Val:${Math.floor(pos / 2)}}, `)
          } else {
            this.push(`{Key:${Math.floor(pos / 2)}}, `)
          }
          break
      }
    }
    const str = utils.cborValueToString(val, -Infinity)

    if ((typeof val === 'string') ||
        (Buffer.isBuffer(val))) {
      if (val.length > 0) {
        this.push(str)
        this.push('\n')
      }
      this.depth--
    } else {
      this.push(str)
      this.push('\n')
    }

    switch (ai) {
      case NUMBYTES.ONE:
      case NUMBYTES.TWO:
      case NUMBYTES.FOUR:
      case NUMBYTES.EIGHT:
        this.depth--
    }
  }

  /**
   * @ignore
   */
  _on_data() {
    this.push('0x')
    this.push(this.all.read().toString('hex'))
    this.push('\n')
  }
}

module.exports = Commented
