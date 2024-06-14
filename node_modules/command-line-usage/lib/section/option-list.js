const Section = require('../section')
const Table = require('table-layout')
const chalk = require('../chalk-format')
const t = require('typical')
const arrayify = require('array-back')

class OptionList extends Section {
  constructor (data) {
    super()
    let definitions = arrayify(data.optionList)
    const hide = arrayify(data.hide)
    const groups = arrayify(data.group)

    /* filter out hidden definitions */
    if (hide.length) {
      definitions = definitions.filter(definition => {
        return hide.indexOf(definition.name) === -1
      })
    }

    if (data.header) this.header(data.header)

    if (groups.length) {
      definitions = definitions.filter(def => {
        const noGroupMatch = groups.indexOf('_none') > -1 && !t.isDefined(def.group)
        const groupMatch = intersect(arrayify(def.group), groups)
        if (noGroupMatch || groupMatch) return def
      })
    }

    const rows = definitions.map(def => {
      return {
        option: getOptionNames(def, data.reverseNameOrder),
        description: chalk(def.description)
      }
    })

    const tableOptions = data.tableOptions || {
      padding: { left: '  ', right: ' ' },
      columns: [
        { name: 'option', noWrap: true },
        { name: 'description', maxWidth: 80 }
      ]
    }
    const table = new Table(rows, tableOptions)
    this.add(table.renderLines())

    this.add()
  }
}

function getOptionNames (definition, reverseNameOrder) {
  let type = definition.type ? definition.type.name.toLowerCase() : 'string'
  const multiple = (definition.multiple || definition.lazyMultiple) ? '[]' : ''
  if (type) {
    type = type === 'boolean' ? '' : `{underline ${type}${multiple}}`
  }
  type = chalk(definition.typeLabel || type)

  let result = ''
  if (definition.alias) {
    if (definition.name) {
      if (reverseNameOrder) {
        result = chalk(`{bold --${definition.name}}, {bold -${definition.alias}} ${type}`)
      } else {
        result = chalk(`{bold -${definition.alias}}, {bold --${definition.name}} ${type}`)
      }
    } else {
      if (reverseNameOrder) {
        result = chalk(`{bold -${definition.alias}} ${type}`)
      } else {
        result = chalk(`{bold -${definition.alias}} ${type}`)
      }
    }
  } else {
    result = chalk(`{bold --${definition.name}} ${type}`)
  }
  return result
}

function intersect (arr1, arr2) {
  return arr1.some(function (item1) {
    return arr2.some(function (item2) {
      return item1 === item2
    })
  })
}

module.exports = OptionList

/**
 * An OptionList section adds a table displaying the supplied option definitions.
 * @typedef module:command-line-usage~optionList
 * @property {string} [header] - The section header, always bold and underlined.
 * @property optionList {OptionDefinition[]} - An array of [option definition](https://github.com/75lb/command-line-args/blob/master/doc/option-definition.md) objects. In addition to the regular definition properties, command-line-usage will look for:
 *
 * - `description` - a string describing the option.
 * - `typeLabel` - a string to replace the default type string (e.g. `<string>`). It's often more useful to set a more descriptive type label, like `<ms>`, `<files>`, `<command>` etc.
 * @property {string|string[]} [group] - If specified, only options from this particular group will be printed. [Example](https://github.com/75lb/command-line-usage/blob/master/example/groups.js).
 * @property {string|string[]} [hide] - The names of one of more option definitions to hide from the option list. [Example](https://github.com/75lb/command-line-usage/blob/master/example/hide.js).
 * @property {boolean} [reverseNameOrder] - If true, the option alias will be displayed after the name, i.e. `--verbose, -v` instead of `-v, --verbose`).
 * @property {object} [tableOptions] - An options object suitable for passing into [table-layout](https://github.com/75lb/table-layout#table-). See [here for an example](https://github.com/75lb/command-line-usage/blob/master/example/option-list-options.js).
 *
 * @example
 * {
 *   header: 'Options',
 *   optionList: [
 *     {
 *       name: 'help',
 *       alias: 'h',
 *       description: 'Display this usage guide.'
 *     },
 *     {
 *       name: 'src',
 *       description: 'The input files to process',
 *       multiple: true,
 *       defaultOption: true,
 *       typeLabel: '{underline file} ...'
 *     },
 *     {
 *       name: 'timeout',
 *       description: 'Timeout value in ms.',
 *       alias: 't',
 *       typeLabel: '{underline ms}'
 *     }
 *   ]
 * }
 */
