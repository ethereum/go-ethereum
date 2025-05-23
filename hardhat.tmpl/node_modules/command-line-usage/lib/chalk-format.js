function chalkFormat (str) {
  if (str) {
    str = str.replace(/`/g, '\\`')
    const chalk = require('chalk')
    return chalk(Object.assign([], { raw: [str] }))
  } else {
    return ''
  }
}

module.exports = chalkFormat
