import arrayify from '../node_modules/array-back/index.mjs'
import findReplace from '../node_modules/find-replace/dist/index.mjs'

/**
 * Some useful tools for working with `process.argv`.
 *
 * @module argv-tools
 * @typicalName argvTools
 * @example
 * const argvTools = require('argv-tools')
 */

/**
 * Regular expressions for matching option formats.
 * @static
 */
const re = {
  short: /^-([^\d-])$/,
  long: /^--(\S+)/,
  combinedShort: /^-[^\d-]{2,}$/,
  optEquals: /^(--\S+?)=(.*)/
}

/**
 * Array subclass encapsulating common operations on `process.argv`.
 * @static
 */
class ArgvArray extends Array {
  /**
   * Clears the array has loads the supplied input.
   * @param {string[]} argv - The argv list to load. Defaults to `process.argv`.
   */
  load (argv) {
    this.clear()
    if (argv && argv !== process.argv) {
      argv = arrayify(argv)
    } else {
      /* if no argv supplied, assume we are parsing process.argv */
      argv = process.argv.slice(0)
      const deleteCount = process.execArgv.some(isExecArg) ? 1 : 2
      argv.splice(0, deleteCount)
    }
    argv.forEach(arg => this.push(String(arg)))
  }

  /**
   * Clear the array.
   */
  clear () {
    this.length = 0
  }

  /**
   * expand ``--option=value` style args.
   */
  expandOptionEqualsNotation () {
    if (this.some(arg => re.optEquals.test(arg))) {
      const expandedArgs = []
      this.forEach(arg => {
        const matches = arg.match(re.optEquals)
        if (matches) {
          expandedArgs.push(matches[1], matches[2])
        } else {
          expandedArgs.push(arg)
        }
      })
      this.clear()
      this.load(expandedArgs)
    }
  }

  /**
   * expand getopt-style combinedShort options.
   */
  expandGetoptNotation () {
    if (this.hasCombinedShortOptions()) {
      findReplace(this, re.combinedShort, expandCombinedShortArg)
    }
  }

  /**
   * Returns true if the array contains combined short options (e.g. `-ab`).
   * @returns {boolean}
   */
  hasCombinedShortOptions () {
    return this.some(arg => re.combinedShort.test(arg))
  }

  static from (argv) {
    const result = new this()
    result.load(argv)
    return result
  }
}

/**
 * Expand a combined short option.
 * @param {string} - the string to expand, e.g. `-ab`
 * @returns {string[]}
 * @static
 */
function expandCombinedShortArg (arg) {
  /* remove initial hypen */
  arg = arg.slice(1)
  return arg.split('').map(letter => '-' + letter)
}

/**
 * Returns true if the supplied arg matches `--option=value` notation.
 * @param {string} - the arg to test, e.g. `--one=something`
 * @returns {boolean}
 * @static
 */
function isOptionEqualsNotation (arg) {
  return re.optEquals.test(arg)
}

/**
 * Returns true if the supplied arg is in either long (`--one`) or short (`-o`) format.
 * @param {string} - the arg to test, e.g. `--one`
 * @returns {boolean}
 * @static
 */
function isOption (arg) {
  return (re.short.test(arg) || re.long.test(arg)) && !re.optEquals.test(arg)
}

/**
 * Returns true if the supplied arg is in long (`--one`) format.
 * @param {string} - the arg to test, e.g. `--one`
 * @returns {boolean}
 * @static
 */
function isLongOption (arg) {
  return re.long.test(arg) && !isOptionEqualsNotation(arg)
}

/**
 * Returns the name from a long, short or `--options=value` arg.
 * @param {string} - the arg to inspect, e.g. `--one`
 * @returns {string}
 * @static
 */
function getOptionName (arg) {
  if (re.short.test(arg)) {
    return arg.match(re.short)[1]
  } else if (isLongOption(arg)) {
    return arg.match(re.long)[1]
  } else if (isOptionEqualsNotation(arg)) {
    return arg.match(re.optEquals)[1].replace(/^--/, '')
  } else {
    return null
  }
}

function isValue (arg) {
  return !(isOption(arg) || re.combinedShort.test(arg) || re.optEquals.test(arg))
}

function isExecArg (arg) {
  return ['--eval', '-e'].indexOf(arg) > -1 || arg.startsWith('--eval=')
}

export {
  expandCombinedShortArg,
  re,
  ArgvArray,
  getOptionName,
  isOption,
  isLongOption,
  isOptionEqualsNotation,
  isValue
}
