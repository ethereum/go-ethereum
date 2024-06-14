/**
 * @module command-line-usage
 */

/**
 * Generates a usage guide suitable for a command-line app.
 * @param {Section|Section[]} - One or more section objects ({@link module:command-line-usage~content} or {@link module:command-line-usage~optionList}).
 * @returns {string}
 * @alias module:command-line-usage
 */
function commandLineUsage (sections) {
  const arrayify = require('array-back')
  sections = arrayify(sections)
  if (sections.length) {
    const OptionList = require('./lib/section/option-list')
    const ContentSection = require('./lib/section/content')
    const output = sections.map(section => {
      if (section.optionList) {
        return new OptionList(section)
      } else {
        return new ContentSection(section)
      }
    })
    return '\n' + output.join('\n')
  } else {
    return ''
  }
}

module.exports = commandLineUsage
