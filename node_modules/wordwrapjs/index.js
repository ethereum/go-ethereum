const os = require('os')
const t = require('typical')

/**
 * @module wordwrapjs
 */

const re = {
  chunk: /[^\s-]+?-\b|\S+|\s+|\r\n?|\n/g,
  ansiEscapeSequence: /\u001b.*?m/g
}

/**
 * @alias module:wordwrapjs
 * @typicalname wordwrap
 */
class WordWrap {
  constructor (text, options) {
    options = options || {}
    if (!t.isDefined(text)) text = ''

    this._lines = String(text).split(/\r\n|\n/g)
    this.options = options
    this.options.width = options.width === undefined ? 30 : options.width
  }

  lines () {
    const flatten = require('reduce-flatten')

    /* trim each line of the supplied text */
    return this._lines.map(trimLine.bind(this))

      /* split each line into an array of chunks, else mark it empty */
      .map(line => line.match(re.chunk) || [ '~~empty~~' ])

      /* optionally, break each word on the line into pieces */
      .map(lineWords => {
        if (this.options.break) {
          return lineWords.map(breakWord.bind(this))
        } else {
          return lineWords
        }
      })
      .map(lineWords => lineWords.reduce(flatten, []))

      /* transforming the line of words to one or more new lines wrapped to size */
      .map(lineWords => {
        return lineWords
          .reduce((lines, word) => {
            let currentLine = lines[lines.length - 1]
            if (replaceAnsi(word).length + replaceAnsi(currentLine).length > this.options.width) {
              lines.push(word)
            } else {
              lines[lines.length - 1] += word
            }
            return lines
          }, [ '' ])
      })
      .reduce(flatten, [])

      /* trim the wrapped lines */
      .map(trimLine.bind(this))

      /* filter out empty lines */
      .filter(line => line.trim())

      /* restore the user's original empty lines */
      .map(line => line.replace('~~empty~~', ''))
  }

  wrap () {
    return this.lines().join(os.EOL)
  }

  toString () {
    return this.wrap()
  }

  /**
   * @param {string} - the input text to wrap
   * @param [options] {object} - optional configuration
   * @param [options.width] {number} - the max column width in characters (defaults to 30).
   * @param [options.break] {boolean} - if true, words exceeding the specified `width` will be forcefully broken
   * @param [options.noTrim] {boolean} - By default, each line output is trimmed. If `noTrim` is set, no line-trimming occurs - all whitespace from the input text is left in.
   * @return {string}
   */
  static wrap (text, options) {
    const block = new this(text, options)
    return block.wrap()
  }

  /**
   * Wraps the input text, returning an array of strings (lines).
   * @param {string} - input text
   * @param {object} - Accepts same options as constructor.
   */
  static lines (text, options) {
    const block = new this(text, options)
    return block.lines()
  }

  /**
   * Returns true if the input text would be wrapped if passed into `.wrap()`.
   * @param {string} - input text
   * @return {boolean}
   */
  static isWrappable (text) {
    if (t.isDefined(text)) {
      text = String(text)
      var matches = text.match(re.chunk)
      return matches ? matches.length > 1 : false
    }
  }

  /**
   * Splits the input text into an array of words and whitespace.
   * @param {string} - input text
   * @returns {string[]}
   */
  static getChunks (text) {
    return text.match(re.chunk) || []
  }
}

function trimLine (line) {
  return this.options.noTrim ? line : line.trim()
}

function replaceAnsi (string) {
  return string.replace(re.ansiEscapeSequence, '')
}

/* break a word into several pieces */
function breakWord (word) {
  if (replaceAnsi(word).length > this.options.width) {
    const letters = word.split('')
    let piece
    const pieces = []
    while ((piece = letters.splice(0, this.options.width)).length) {
      pieces.push(piece.join(''))
    }
    return pieces
  } else {
    return word
  }
}

module.exports = WordWrap
