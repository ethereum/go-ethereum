// Tweaked version of nathan7's binary-parse-stream
// (see https://github.com/nathan7/binary-parse-stream)
// Uses NoFilter instead of the readable in the original.  Removes
// the ability to read -1, which was odd and un-needed.
// License for binary-parse-stream: MIT

// binary-parse-stream is now unmaintained, so I have rewritten it as
// more modern JS so I can get tsc to help check types.

'use strict'
const stream = require('stream')
const NoFilter = require('nofilter')

/**
 * BinaryParseStream is a TransformStream that consumes buffers and outputs
 * objects on the other end.  It expects your subclass to implement a `_parse`
 * method that is a generator.  When your generator yields a number, it'll be
 * fed a buffer of that length from the input.  When your generator returns,
 * the return value will be pushed to the output side.
 *
 * @extends stream.Transform
 */
class BinaryParseStream extends stream.Transform {
  /**
   * Creates an instance of BinaryParseStream.
   *
   * @param {stream.TransformOptions} options Stream options.
   * @memberof BinaryParseStream
   */
  constructor(options) {
    super(options)
    // Doesn't work to pass these in as opts, for some reason
    // also, work around typescript not knowing TransformStream internals
    // eslint-disable-next-line dot-notation
    this['_writableState'].objectMode = false
    // eslint-disable-next-line dot-notation
    this['_readableState'].objectMode = true

    this.bs = new NoFilter()
    this.__restart()
  }

  _transform(fresh, encoding, cb) {
    this.bs.write(fresh)

    while (this.bs.length >= this.__needed) {
      let ret = null
      const chunk = (this.__needed === null) ?
        undefined :
        this.bs.read(this.__needed)

      try {
        ret = this.__parser.next(chunk)
      } catch (e) {
        return cb(e)
      }

      if (this.__needed) {
        this.__fresh = false
      }

      if (ret.done) {
        this.push(ret.value)
        this.__restart()
      } else {
        this.__needed = ret.value || Infinity
      }
    }

    return cb()
  }

  /**
   * Subclasses must override this to set their parsing behavior.  Yield a
   * number to receive a Buffer of that many bytes.
   *
   * @abstract
   * @returns {Generator<number, undefined, Buffer>}
   */
  /* istanbul ignore next */
  *_parse() { // eslint-disable-line class-methods-use-this, require-yield
    throw new Error('Must be implemented in subclass')
  }

  __restart() {
    this.__needed = null
    this.__parser = this._parse()
    this.__fresh = true
  }

  _flush(cb) {
    cb(this.__fresh ? null : new Error('unexpected end of input'))
  }
}

module.exports = BinaryParseStream
