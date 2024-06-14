const through = require('through2')
const split = require('split2')
const { EOL } = require('os')
const stringify = require('json-stringify-safe')

module.exports.stringify = (opts) =>
  through.obj(opts, (obj, _, cb) => {
    cb(null, stringify(obj) + EOL)
  })

module.exports.parse = (opts) => {
  opts = opts || {}
  opts.strict = opts.strict !== false

  function parseRow (row) {
    try {
      if (row) return JSON.parse(row)
    } catch (e) {
      if (opts.strict) {
        this.emit('error', new Error('Could not parse row ' + row.slice(0, 50) + '...'))
      }
    }
  }

  return split(parseRow, opts)
}