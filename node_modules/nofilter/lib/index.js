'use strict'

const stream = require('stream')
const {Buffer} = require('buffer')
const td = new TextDecoder('utf8', {fatal: true, ignoreBOM: true})

/**
 * @typedef {object} NoFilterOptions
 * @property {string|Buffer} [input=null] Input source data.
 * @property {BufferEncoding} [inputEncoding=null] Encoding name for input,
 *   ignored if input is not a String.
 * @property {number} [highWaterMark=16384] The maximum number of bytes to
 *   store in the internal buffer before ceasing to read from the underlying
 *   resource. Default=16kb, or 16 for objectMode streams.
 * @property {BufferEncoding} [encoding=null] If specified, then buffers
 *   will be decoded to strings using the specified encoding.
 * @property {boolean} [objectMode=false] Whether this stream should behave
 *   as a stream of objects. Meaning that stream.read(n) returns a single
 *   value instead of a Buffer of size n.
 * @property {boolean} [decodeStrings=true] Whether or not to decode
 *   strings into Buffers before passing them to _write().
 * @property {boolean} [watchPipe=true] Whether to watch for 'pipe' events,
 *   setting this stream's objectMode based on the objectMode of the input
 *   stream.
 * @property {boolean} [readError=false] If true, when a read() underflows,
 *   throw an error.
 * @property {boolean} [allowHalfOpen=true] If set to false, then the
 *   stream will automatically end the writable side when the readable side
 *   ends.
 * @property {boolean} [autoDestroy=true] Whether this stream should
 *   automatically call .destroy() on itself after ending.
 * @property {BufferEncoding} [defaultEncoding='utf8'] The default encoding
 *   that is used when no encoding is specified as an argument to
 *   stream.write().
 * @property {boolean} [emitClose=true] Whether or not the stream should
 *   emit 'close' after it has been destroyed.
 * @property {number} [readableHighWaterMark] Sets highWaterMark for the
 *   readable side of the stream. Has no effect if highWaterMark is provided.
 * @property {boolean} [readableObjectMode=false] Sets objectMode for
 *   readable side of the stream. Has no effect if objectMode is true.
 * @property {number} [writableHighWaterMark] Sets highWaterMark for the
 *   writable side of the stream. Has no effect if highWaterMark is provided.
 * @property {boolean} [writableObjectMode=false] Sets objectMode for
 *   writable side of the stream. Has no effect if objectMode is true.
 */

/**
 * NoFilter stream.  Can be used to sink or source data to and from
 * other node streams.  Implemented as the "identity" Transform stream
 * (hence the name), but allows for inspecting data that is in-flight.
 *
 * Allows passing in source data (input, inputEncoding) at creation
 * time.  Source data can also be passed in the options object.
 *
 * @example <caption>source and sink</caption>
 * const source = new NoFilter('Zm9v', 'base64')
 * source.pipe(process.stdout)
 * const sink = new Nofilter()
 * // NOTE: 'finish' fires when the input is done writing
 * sink.on('finish', () => console.log(n.toString('base64')))
 * process.stdin.pipe(sink)
 */
class NoFilter extends stream.Transform {
  /**
   * Create an instance of NoFilter.
   *
   * @param {string|Buffer|BufferEncoding|NoFilterOptions} [input] Source data.
   * @param {BufferEncoding|NoFilterOptions} [inputEncoding] Encoding
   *   name for input, ignored if input is not a String.
   * @param {NoFilterOptions} [options] Other options.
   */
  constructor(input, inputEncoding, options = {}) {
    let inp = null
    let inpE = /** @type {BufferEncoding?} */ (null)
    switch (typeof input) {
      case 'object':
        if (Buffer.isBuffer(input)) {
          inp = input
        } else if (input) {
          options = input
        }
        break
      case 'string':
        inp = input
        break
      case 'undefined':
        break
      default:
        throw new TypeError('Invalid input')
    }
    switch (typeof inputEncoding) {
      case 'object':
        if (inputEncoding) {
          options = inputEncoding
        }
        break
      case 'string':
        inpE = /** @type {BufferEncoding} */ (inputEncoding)
        break
      case 'undefined':
        break
      default:
        throw new TypeError('Invalid inputEncoding')
    }
    if (!options || typeof options !== 'object') {
      throw new TypeError('Invalid options')
    }
    if (inp == null) {
      inp = options.input
    }
    if (inpE == null) {
      inpE = options.inputEncoding
    }
    delete options.input
    delete options.inputEncoding
    const watchPipe = options.watchPipe == null ? true : options.watchPipe
    delete options.watchPipe
    const readError = Boolean(options.readError)
    delete options.readError
    super(options)

    this.readError = readError

    if (watchPipe) {
      this.on('pipe', readable => {
        // @ts-ignore: TS2339 (using internal interface)
        const om = readable._readableState.objectMode
        // @ts-ignore: TS2339 (using internal interface)
        if ((this.length > 0) && (om !== this._readableState.objectMode)) {
          throw new Error(
            'Do not switch objectMode in the middle of the stream'
          )
        }

        // @ts-ignore: TS2339 (using internal interface)
        this._readableState.objectMode = om
        // @ts-ignore: TS2339 (using internal interface)
        this._writableState.objectMode = om
      })
    }

    if (inp != null) {
      this.end(inp, inpE)
    }
  }

  /**
   * Is the given object a {NoFilter}?
   *
   * @param {object} obj The object to test.
   * @returns {boolean} True if obj is a NoFilter.
   */
  static isNoFilter(obj) {
    return obj instanceof this
  }

  /**
   * The same as nf1.compare(nf2). Useful for sorting an Array of NoFilters.
   *
   * @param {NoFilter} nf1 The first object to compare.
   * @param {NoFilter} nf2 The second object to compare.
   * @returns {number} -1, 0, 1 for less, equal, greater.
   * @throws {TypeError} Arguments not NoFilter instances.
   * @example
   * const arr = [new NoFilter('1234'), new NoFilter('0123')]
   * arr.sort(NoFilter.compare)
   */
  static compare(nf1, nf2) {
    if (!(nf1 instanceof this)) {
      throw new TypeError('Arguments must be NoFilters')
    }
    if (nf1 === nf2) {
      return 0
    }
    return nf1.compare(nf2)
  }

  /**
   * Returns a buffer which is the result of concatenating all the
   * NoFilters in the list together. If the list has no items, or if
   * the totalLength is 0, then it returns a zero-length buffer.
   *
   * If length is not provided, it is read from the buffers in the
   * list. However, this adds an additional loop to the function, so
   * it is faster to provide the length explicitly if you already know it.
   *
   * @param {Array<NoFilter>} list Inputs.  Must not be all either in object
   *   mode, or all not in object mode.
   * @param {number} [length=null] Number of bytes or objects to read.
   * @returns {Buffer|Array} The concatenated values as an array if in object
   *   mode, otherwise a Buffer.
   * @throws {TypeError} List not array of NoFilters.
   */
  static concat(list, length) {
    if (!Array.isArray(list)) {
      throw new TypeError('list argument must be an Array of NoFilters')
    }
    if ((list.length === 0) || (length === 0)) {
      return Buffer.alloc(0)
    }
    if ((length == null)) {
      length = list.reduce((tot, nf) => {
        if (!(nf instanceof NoFilter)) {
          throw new TypeError('list argument must be an Array of NoFilters')
        }
        return tot + nf.length
      }, 0)
    }
    let allBufs = true
    let allObjs = true
    const bufs = list.map(nf => {
      if (!(nf instanceof NoFilter)) {
        throw new TypeError('list argument must be an Array of NoFilters')
      }
      const buf = nf.slice()
      if (Buffer.isBuffer(buf)) {
        allObjs = false
      } else {
        allBufs = false
      }
      return buf
    })
    if (allBufs) {
      // @ts-ignore: TS2322, tsc can't see the type checking above
      return Buffer.concat(bufs, length)
    }
    if (allObjs) {
      return [].concat(...bufs).slice(0, length)
    }
    // TODO: maybe coalesce buffers, counting bytes, and flatten in arrays
    // counting objects?  I can't imagine why that would be useful.
    throw new Error('Concatenating mixed object and byte streams not supported')
  }

  /**
   * @ignore
   */
  _transform(chunk, encoding, callback) {
    // @ts-ignore: TS2339 (using internal interface)
    if (!this._readableState.objectMode && !Buffer.isBuffer(chunk)) {
      chunk = Buffer.from(chunk, encoding)
    }
    this.push(chunk)
    callback()
  }

  /**
   * @returns {Buffer[]} The current internal buffers.  They are layed out
   *   end to end.
   * @ignore
   */
  _bufArray() {
    // @ts-ignore: TS2339 (using internal interface)
    let bufs = this._readableState.buffer
    // HACK: replace with something else one day.  This is what I get for
    // relying on internals.
    if (!Array.isArray(bufs)) {
      let b = bufs.head
      bufs = []
      while (b != null) {
        bufs.push(b.data)
        b = b.next
      }
    }
    return bufs
  }

  /**
   * Pulls some data out of the internal buffer and returns it.
   * If there is no data available, then it will return null.
   *
   * If you pass in a size argument, then it will return that many bytes. If
   * size bytes are not available, then it will return null, unless we've
   * ended, in which case it will return the data remaining in the buffer.
   *
   * If you do not specify a size argument, then it will return all the data in
   * the internal buffer.
   *
   * @param {number} [size=null] Number of bytes to read.
   * @returns {string|Buffer|null} If no data or not enough data, null.  If
   *   decoding output a string, otherwise a Buffer.
   * @throws Error If readError is true and there was underflow.
   * @fires NoFilter#read When read from.
   */
  read(size) {
    const buf = super.read(size)
    if (buf != null) {
      /**
       * Read event. Fired whenever anything is read from the stream.
       *
       * @event NoFilter#read
       * @param {Buffer|string|object} buf What was read.
       */
      this.emit('read', buf)
      if (this.readError && (buf.length < size)) {
        throw new Error(`Read ${buf.length}, wanted ${size}`)
      }
    } else if (this.readError) {
      throw new Error(`No data available, wanted ${size}`)
    }
    return buf
  }

  /**
   * Read the full number of bytes asked for, no matter how long it takes.
   * Fail if an error occurs in the meantime, or if the stream finishes before
   * enough data is available.
   *
   * Note: This function won't work fully correctly if you are using
   * stream-browserify (for example, on the Web).
   *
   * @param {number} size The number of bytes to read.
   * @returns {Promise<string|Buffer>} A promise for the data read.
   */
  readFull(size) {
    let onReadable = null
    let onFinish = null
    let onError = null
    return new Promise((resolve, reject) => {
      if (this.length >= size) {
        resolve(this.read(size))
        return
      }

      // Added in Node 12.19.  This won't work with stream-browserify yet.
      // If it's needed, file a bug, and I'll do a work-around.
      if (this.writableFinished) {
        // Already finished writing, so no more coming.
        reject(new Error(`Stream finished before ${size} bytes were available`))
        return
      }

      onReadable = chunk => {
        if (this.length >= size) {
          resolve(this.read(size))
        }
      }
      onFinish = () => {
        reject(new Error(`Stream finished before ${size} bytes were available`))
      }
      onError = reject
      this.on('readable', onReadable)
      this.on('error', onError)
      this.on('finish', onFinish)
    }).finally(() => {
      if (onReadable) {
        this.removeListener('readable', onReadable)
        this.removeListener('error', onError)
        this.removeListener('finish', onFinish)
      }
    })
  }

  /**
   * Return a promise fulfilled with the full contents, after the 'finish'
   * event fires.  Errors on the stream cause the promise to be rejected.
   *
   * @param {Function} [cb=null] Finished/error callback used in *addition*
   *   to the promise.
   * @returns {Promise<Buffer|string>} Fulfilled when complete.
   */
  promise(cb) {
    let done = false
    return new Promise((resolve, reject) => {
      this.on('finish', () => {
        const data = this.read()
        if ((cb != null) && !done) {
          done = true
          cb(null, data)
        }
        resolve(data)
      })
      this.on('error', er => {
        if ((cb != null) && !done) {
          done = true
          cb(er)
        }
        reject(er)
      })
    })
  }

  /**
   * Returns a number indicating whether this comes before or after or is the
   * same as the other NoFilter in sort order.
   *
   * @param {NoFilter} other The other object to compare.
   * @returns {number} -1, 0, 1 for less, equal, greater.
   * @throws {TypeError} Arguments must be NoFilters.
   */
  compare(other) {
    if (!(other instanceof NoFilter)) {
      throw new TypeError('Arguments must be NoFilters')
    }
    if (this === other) {
      return 0
    }

    const buf1 = this.slice()
    const buf2 = other.slice()
    // These will both be buffers because of the check above.
    if (Buffer.isBuffer(buf1) && Buffer.isBuffer(buf2)) {
      return buf1.compare(buf2)
    }
    throw new Error('Cannot compare streams in object mode')
  }

  /**
   * Do these NoFilter's contain the same bytes?  Doesn't work if either is
   * in object mode.
   *
   * @param {NoFilter} other Other NoFilter to compare against.
   * @returns {boolean} Equal?
   */
  equals(other) {
    return this.compare(other) === 0
  }

  /**
   * Read bytes or objects without consuming them.  Useful for diagnostics.
   * Note: as a side-effect, concatenates multiple writes together into what
   * looks like a single write, so that this concat doesn't have to happen
   * multiple times when you're futzing with the same NoFilter.
   *
   * @param {number} [start=0] Beginning offset.
   * @param {number} [end=length] Ending offset.
   * @returns {Buffer|Array} If in object mode, an array of objects.  Otherwise,
   *   concatenated array of contents.
   */
  slice(start, end) {
    // @ts-ignore: TS2339 (using internal interface)
    if (this._readableState.objectMode) {
      return this._bufArray().slice(start, end)
    }
    const bufs = this._bufArray()
    switch (bufs.length) {
      case 0: return Buffer.alloc(0)
      case 1: return bufs[0].slice(start, end)
      default: {
        const b = Buffer.concat(bufs)
        // TODO: store the concatented bufs back
        // @_readableState.buffer = [b]
        return b.slice(start, end)
      }
    }
  }

  /**
   * Get a byte by offset.  I didn't want to get into metaprogramming
   * to give you the `NoFilter[0]` syntax.
   *
   * @param {number} index The byte to retrieve.
   * @returns {number} 0-255.
   */
  get(index) {
    return this.slice()[index]
  }

  /**
   * Return an object compatible with Buffer's toJSON implementation, so that
   * round-tripping will produce a Buffer.
   *
   * @returns {string|Array|{type: 'Buffer',data: number[]}} If in object mode,
   *   the objects.  Otherwise, JSON text.
   * @example <caption>output for 'foo', not in object mode</caption>
   * ({
   *   type: 'Buffer',
   *   data: [102, 111, 111],
   * })
   */
  toJSON() {
    const b = this.slice()
    if (Buffer.isBuffer(b)) {
      return b.toJSON()
    }
    return b
  }

  /**
   * Decodes and returns a string from buffer data encoded using the specified
   * character set encoding. If encoding is undefined or null, then encoding
   * defaults to 'utf8'. The start and end parameters default to 0 and
   * NoFilter.length when undefined.
   *
   * @param {BufferEncoding} [encoding='utf8'] Which to use for decoding?
   * @param {number} [start=0] Start offset.
   * @param {number} [end=length] End offset.
   * @returns {string} String version of the contents.
   */
  toString(encoding, start, end) {
    const buf = this.slice(start, end)
    if (!Buffer.isBuffer(buf)) {
      return JSON.stringify(buf)
    }
    if (!encoding || (encoding === 'utf8')) {
      return td.decode(buf)
    }
    return buf.toString(encoding)
  }

  /**
   * @ignore
   */
  [Symbol.for('nodejs.util.inspect.custom')](depth, options) {
    const bufs = this._bufArray()
    const hex = bufs.map(b => {
      if (Buffer.isBuffer(b)) {
        return options.stylize(b.toString('hex'), 'string')
      }
      return JSON.stringify(b)
    }).join(', ')
    return `${this.constructor.name} [${hex}]`
  }

  /**
   * Current readable length, in bytes.
   *
   * @returns {number} Length of the contents.
   */
  get length() {
    // @ts-ignore: TS2339 (using internal interface)
    return this._readableState.length
  }

  /**
   * Write a JavaScript BigInt to the stream.  Negative numbers will be
   * written as their 2's complement version.
   *
   * @param {bigint} val The value to write.
   * @returns {boolean} True on success.
   */
  writeBigInt(val) {
    let str = val.toString(16)
    if (val < 0) {
      // Two's complement
      // Note: str always starts with '-' here.
      const sz = BigInt(Math.floor(str.length / 2))
      const mask = BigInt(1) << (sz * BigInt(8))
      val = mask + val
      str = val.toString(16)
    }
    if (str.length % 2) {
      str = `0${str}`
    }
    return this.push(Buffer.from(str, 'hex'))
  }

  /**
   * Read a variable-sized JavaScript unsigned BigInt from the stream.
   *
   * @param {number} [len=null] Number of bytes to read or all remaining
   *   if null.
   * @returns {bigint} A BigInt.
   */
  readUBigInt(len) {
    const b = this.read(len)
    if (!Buffer.isBuffer(b)) {
      return null
    }
    return BigInt(`0x${b.toString('hex')}`)
  }

  /**
   * Read a variable-sized JavaScript signed BigInt from the stream in 2's
   * complement format.
   *
   * @param {number} [len=null] Number of bytes to read or all remaining
   *   if null.
   * @returns {bigint} A BigInt.
   */
  readBigInt(len) {
    const b = this.read(len)
    if (!Buffer.isBuffer(b)) {
      return null
    }
    let ret = BigInt(`0x${b.toString('hex')}`)
    // Negative?
    if (b[0] & 0x80) {
      // Two's complement
      const mask = BigInt(1) << (BigInt(b.length) * BigInt(8))
      ret -= mask
    }
    return ret
  }

  /**
   * Write an 8-bit unsigned integer to the stream.  Adds 1 byte.
   *
   * @param {number} value 0..255.
   * @returns {boolean} True on success.
   */
  writeUInt8(value) {
    const b = Buffer.from([value])
    return this.push(b)
  }

  /**
   * Write a little-endian 16-bit unsigned integer to the stream.  Adds
   * 2 bytes.
   *
   * @param {number} value 0..65535.
   * @returns {boolean} True on success.
   */
  writeUInt16LE(value) {
    const b = Buffer.alloc(2)
    b.writeUInt16LE(value)
    return this.push(b)
  }

  /**
   * Write a big-endian 16-bit unsigned integer to the stream.  Adds
   * 2 bytes.
   *
   * @param {number} value 0..65535.
   * @returns {boolean} True on success.
   */
  writeUInt16BE(value) {
    const b = Buffer.alloc(2)
    b.writeUInt16BE(value)
    return this.push(b)
  }

  /**
   * Write a little-endian 32-bit unsigned integer to the stream.  Adds
   * 4 bytes.
   *
   * @param {number} value 0..2**32-1.
   * @returns {boolean} True on success.
   */
  writeUInt32LE(value) {
    const b = Buffer.alloc(4)
    b.writeUInt32LE(value)
    return this.push(b)
  }

  /**
   * Write a big-endian 32-bit unsigned integer to the stream.  Adds
   * 4 bytes.
   *
   * @param {number} value 0..2**32-1.
   * @returns {boolean} True on success.
   */
  writeUInt32BE(value) {
    const b = Buffer.alloc(4)
    b.writeUInt32BE(value)
    return this.push(b)
  }

  /**
   * Write a signed 8-bit integer to the stream.  Adds 1 byte.
   *
   * @param {number} value (-128)..127.
   * @returns {boolean} True on success.
   */
  writeInt8(value) {
    const b = Buffer.from([value])
    return this.push(b)
  }

  /**
   * Write a signed little-endian 16-bit integer to the stream.  Adds 2 bytes.
   *
   * @param {number} value (-32768)..32767.
   * @returns {boolean} True on success.
   */
  writeInt16LE(value) {
    const b = Buffer.alloc(2)
    b.writeUInt16LE(value)
    return this.push(b)
  }

  /**
   * Write a signed big-endian 16-bit integer to the stream.  Adds 2 bytes.
   *
   * @param {number} value (-32768)..32767.
   * @returns {boolean} True on success.
   */
  writeInt16BE(value) {
    const b = Buffer.alloc(2)
    b.writeUInt16BE(value)
    return this.push(b)
  }

  /**
   * Write a signed little-endian 32-bit integer to the stream.  Adds 4 bytes.
   *
   * @param {number} value (-2**31)..(2**31-1).
   * @returns {boolean} True on success.
   */
  writeInt32LE(value) {
    const b = Buffer.alloc(4)
    b.writeUInt32LE(value)
    return this.push(b)
  }

  /**
   * Write a signed big-endian 32-bit integer to the stream.  Adds 4 bytes.
   *
   * @param {number} value (-2**31)..(2**31-1).
   * @returns {boolean} True on success.
   */
  writeInt32BE(value) {
    const b = Buffer.alloc(4)
    b.writeUInt32BE(value)
    return this.push(b)
  }

  /**
   * Write a little-endian 32-bit float to the stream.  Adds 4 bytes.
   *
   * @param {number} value 32-bit float.
   * @returns {boolean} True on success.
   */
  writeFloatLE(value) {
    const b = Buffer.alloc(4)
    b.writeFloatLE(value)
    return this.push(b)
  }

  /**
   * Write a big-endian 32-bit float to the stream.  Adds 4 bytes.
   *
   * @param {number} value 32-bit float.
   * @returns {boolean} True on success.
   */
  writeFloatBE(value) {
    const b = Buffer.alloc(4)
    b.writeFloatBE(value)
    return this.push(b)
  }

  /**
   * Write a little-endian 64-bit double to the stream.  Adds 8 bytes.
   *
   * @param {number} value 64-bit float.
   * @returns {boolean} True on success.
   */
  writeDoubleLE(value) {
    const b = Buffer.alloc(8)
    b.writeDoubleLE(value)
    return this.push(b)
  }

  /**
   * Write a big-endian 64-bit float to the stream.  Adds 8 bytes.
   *
   * @param {number} value 64-bit float.
   * @returns {boolean} True on success.
   */
  writeDoubleBE(value) {
    const b = Buffer.alloc(8)
    b.writeDoubleBE(value)
    return this.push(b)
  }

  /**
   * Write a signed little-endian 64-bit BigInt to the stream.  Adds 8 bytes.
   *
   * @param {bigint} value BigInt.
   * @returns {boolean} True on success.
   */
  writeBigInt64LE(value) {
    const b = Buffer.alloc(8)
    b.writeBigInt64LE(value)
    return this.push(b)
  }

  /**
   * Write a signed big-endian 64-bit BigInt to the stream.  Adds 8 bytes.
   *
   * @param {bigint} value BigInt.
   * @returns {boolean} True on success.
   */
  writeBigInt64BE(value) {
    const b = Buffer.alloc(8)
    b.writeBigInt64BE(value)
    return this.push(b)
  }

  /**
   * Write an unsigned little-endian 64-bit BigInt to the stream.  Adds 8 bytes.
   *
   * @param {bigint} value Non-negative BigInt.
   * @returns {boolean} True on success.
   */
  writeBigUInt64LE(value) {
    const b = Buffer.alloc(8)
    b.writeBigUInt64LE(value)
    return this.push(b)
  }

  /**
   * Write an unsigned big-endian 64-bit BigInt to the stream.  Adds 8 bytes.
   *
   * @param {bigint} value Non-negative BigInt.
   * @returns {boolean} True on success.
   */
  writeBigUInt64BE(value) {
    const b = Buffer.alloc(8)
    b.writeBigUInt64BE(value)
    return this.push(b)
  }

  /**
   * Read an unsigned 8-bit integer from the stream.  Consumes 1 byte.
   *
   * @returns {number} Value read.
   */
  readUInt8() {
    const b = this.read(1)
    if (!Buffer.isBuffer(b)) {
      return null
    }
    return b.readUInt8()
  }

  /**
   * Read a little-endian unsigned 16-bit integer from the stream.
   * Consumes 2 bytes.
   *
   * @returns {number} Value read.
   */
  readUInt16LE() {
    const b = this.read(2)
    if (!Buffer.isBuffer(b)) {
      return null
    }
    return b.readUInt16LE()
  }

  /**
   * Read a little-endian unsigned 16-bit integer from the stream.
   * Consumes 2 bytes.
   *
   * @returns {number} Value read.
   */
  readUInt16BE() {
    const b = this.read(2)
    if (!Buffer.isBuffer(b)) {
      return null
    }
    return b.readUInt16BE()
  }

  /**
   * Read a little-endian unsigned 32-bit integer from the stream.
   * Consumes 4 bytes.
   *
   * @returns {number} Value read.
   */
  readUInt32LE() {
    const b = this.read(4)
    if (!Buffer.isBuffer(b)) {
      return null
    }
    return b.readUInt32LE()
  }

  /**
   * Read a little-endian unsigned 16-bit integer from the stream.
   * Consumes 4 bytes.
   *
   * @returns {number} Value read.
   */
  readUInt32BE() {
    const b = this.read(4)
    if (!Buffer.isBuffer(b)) {
      return null
    }
    return b.readUInt32BE()
  }

  /**
   * Read a signed 8-bit integer from the stream.  Consumes 1 byte.
   *
   * @returns {number} Value read.
   */
  readInt8() {
    const b = this.read(1)
    if (!Buffer.isBuffer(b)) {
      return null
    }
    return b.readInt8()
  }

  /**
   * Read a little-endian signed 16-bit integer from the stream.
   * Consumes 2 bytes.
   *
   * @returns {number} Value read.
   */
  readInt16LE() {
    const b = this.read(2)
    if (!Buffer.isBuffer(b)) {
      return null
    }
    return b.readInt16LE()
  }

  /**
   * Read a little-endian signed 16-bit integer from the stream.
   * Consumes 2 bytes.
   *
   * @returns {number} Value read.
   */
  readInt16BE() {
    const b = this.read(2)
    if (!Buffer.isBuffer(b)) {
      return null
    }
    return b.readInt16BE()
  }

  /**
   * Read a little-endian signed 32-bit integer from the stream.
   * Consumes 4 bytes.
   *
   * @returns {number} Value read.
   */
  readInt32LE() {
    const b = this.read(4)
    if (!Buffer.isBuffer(b)) {
      return null
    }
    return b.readInt32LE()
  }

  /**
   * Read a little-endian signed 16-bit integer from the stream.
   * Consumes 4 bytes.
   *
   * @returns {number} Value read.
   */
  readInt32BE() {
    const b = this.read(4)
    if (!Buffer.isBuffer(b)) {
      return null
    }
    return b.readInt32BE()
  }

  /**
   * Read a 32-bit little-endian float from the stream.
   * Consumes 4 bytes.
   *
   * @returns {number} Value read.
   */
  readFloatLE() {
    const b = this.read(4)
    if (!Buffer.isBuffer(b)) {
      return null
    }
    return b.readFloatLE()
  }

  /**
   * Read a 32-bit big-endian float from the stream.
   * Consumes 4 bytes.
   *
   * @returns {number} Value read.
   */
  readFloatBE() {
    const b = this.read(4)
    if (!Buffer.isBuffer(b)) {
      return null
    }
    return b.readFloatBE()
  }

  /**
   * Read a 64-bit little-endian float from the stream.
   * Consumes 8 bytes.
   *
   * @returns {number} Value read.
   */
  readDoubleLE() {
    const b = this.read(8)
    if (!Buffer.isBuffer(b)) {
      return null
    }
    return b.readDoubleLE()
  }

  /**
   * Read a 64-bit big-endian float from the stream.
   * Consumes 8 bytes.
   *
   * @returns {number} Value read.
   */
  readDoubleBE() {
    const b = this.read(8)
    if (!Buffer.isBuffer(b)) {
      return null
    }
    return b.readDoubleBE()
  }

  /**
   * Read a signed 64-bit little-endian BigInt from the stream.
   * Consumes 8 bytes.
   *
   * @returns {bigint} Value read.
   */
  readBigInt64LE() {
    const b = this.read(8)
    if (!Buffer.isBuffer(b)) {
      return null
    }
    return b.readBigInt64LE()
  }

  /**
   * Read a signed 64-bit big-endian BigInt from the stream.
   * Consumes 8 bytes.
   *
   * @returns {bigint} Value read.
   */
  readBigInt64BE() {
    const b = this.read(8)
    if (!Buffer.isBuffer(b)) {
      return null
    }
    return b.readBigInt64BE()
  }

  /**
   * Read an unsigned 64-bit little-endian BigInt from the stream.
   * Consumes 8 bytes.
   *
   * @returns {bigint} Value read.
   */
  readBigUInt64LE() {
    const b = this.read(8)
    if (!Buffer.isBuffer(b)) {
      return null
    }
    return b.readBigUInt64LE()
  }

  /**
   * Read an unsigned 64-bit big-endian BigInt from the stream.
   * Consumes 8 bytes.
   *
   * @returns {bigint} Value read.
   */
  readBigUInt64BE() {
    const b = this.read(8)
    if (!Buffer.isBuffer(b)) {
      return null
    }
    return b.readBigUInt64BE()
  }
}

module.exports = NoFilter
