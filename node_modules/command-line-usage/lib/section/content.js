const Section = require('../section')
const t = require('typical')
const Table = require('table-layout')
const chalkFormat = require('../chalk-format')

class ContentSection extends Section {
  constructor (section) {
    super()
    this.header(section.header)

    if (section.content) {
      /* add content without indentation or wrapping */
      if (section.raw) {
        const arrayify = require('array-back')
        const content = arrayify(section.content).map(line => chalkFormat(line))
        this.add(content)
      } else {
        this.add(getContentLines(section.content))
      }

      this.add()
    }
  }
}

function getContentLines (content) {
  const defaultPadding = { left: '  ', right: ' ' }

  if (content) {
    /* string content */
    if (t.isString(content)) {
      const table = new Table({ column: chalkFormat(content) }, {
        padding: defaultPadding,
        maxWidth: 80
      })
      return table.renderLines()

    /* array of strings */
    } else if (Array.isArray(content) && content.every(t.isString)) {
      const rows = content.map(string => ({ column: chalkFormat(string) }))
      const table = new Table(rows, {
        padding: defaultPadding,
        maxWidth: 80
      })
      return table.renderLines()

    /* array of objects (use table-layout) */
    } else if (Array.isArray(content) && content.every(t.isPlainObject)) {
      const table = new Table(content.map(row => ansiFormatRow(row)), {
        padding: defaultPadding
      })
      return table.renderLines()

    /* { options: object, data: object[] } */
    } else if (t.isPlainObject(content)) {
      if (!content.options || !content.data) {
        throw new Error('must have an "options" or "data" property\n' + JSON.stringify(content))
      }
      const options = Object.assign(
        { padding: defaultPadding },
        content.options
      )

      /* convert nowrap to noWrap to avoid breaking compatibility */
      if (options.columns) {
        options.columns = options.columns.map(column => {
          if (column.nowrap) {
            column.noWrap = column.nowrap
            delete column.nowrap
          }
          return column
        })
      }

      const table = new Table(
        content.data.map(row => ansiFormatRow(row)),
        options
      )
      return table.renderLines()
    } else {
      const message = `invalid input - 'content' must be a string, array of strings, or array of plain objects:\n\n${JSON.stringify(content)}`
      throw new Error(message)
    }
  }
}

function ansiFormatRow (row) {
  for (const key in row) {
    row[key] = chalkFormat(row[key])
  }
  return row
}

module.exports = ContentSection

/**
 * A Content section comprises a header and one or more lines of content.
 * @typedef module:command-line-usage~content
 * @property header {string} - The section header, always bold and underlined.
 * @property content {string|string[]|object[]} - Overloaded property, accepting data in one of four formats:
 *
 * 1. A single string (one line of text)
 * 2. An array of strings (multiple lines of text)
 * 3. An array of objects (recordset-style data). In this case, the data will be rendered in table format. The property names of each object are not important, so long as they are consistent throughout the array.
 * 4. An object with two properties - `data` and `options`. In this case, the data and options will be passed directly to the underlying [table layout](https://github.com/75lb/table-layout) module for rendering.
 *
 * @property raw {boolean} - Set to true to avoid indentation and wrapping. Useful for banners.
 * @example
 * Simple string of content. For ansi formatting, use [chalk template literal syntax](https://github.com/chalk/chalk#tagged-template-literal).
 * ```js
 * {
 *   header: 'A typical app',
 *   content: 'Generates something {rgb(255,200,0).italic very {underline.bgRed important}}.'
 * }
 * ```
 *
 * An array of strings is interpreted as lines, to be joined by the system newline character.
 * ```js
 * {
 *   header: 'A typical app',
 *   content: [
 *     'First line.',
 *     'Second line.'
 *   ]
 * }
 * ```
 *
 * An array of recordset-style objects are rendered in table layout.
 * ```js
 * {
 *   header: 'A typical app',
 *   content: [
 *     { colA: 'First row, first column.', colB: 'First row, second column.'},
 *     { colA: 'Second row, first column.', colB: 'Second row, second column.'}
 *   ]
 * }
 * ```
 *
 * An object with `data` and `options` properties will be passed directly to the underlying [table layout](https://github.com/75lb/table-layout) module for rendering.
 * ```js
 * {
 *   header: 'A typical app',
 *   content: {
 *     data: [
 *      { colA: 'First row, first column.', colB: 'First row, second column.'},
 *      { colA: 'Second row, first column.', colB: 'Second row, second column.'}
 *     ],
 *     options: {
 *       maxWidth: 60
 *     }
 *   }
 * }
 * ```
 */
