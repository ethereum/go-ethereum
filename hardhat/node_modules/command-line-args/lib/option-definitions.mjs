import arrayify from '../node_modules/array-back/index.mjs'
import * as argvTools from './argv-tools.mjs'
import t from '../node_modules/typical/index.mjs'
import Definition from './option-definition.mjs'

/**
 * @module option-definitions
 */

/**
 * @alias module:option-definitions
 */
class Definitions extends Array {
  /**
   * validate option definitions
   * @param {boolean} [caseInsensitive=false] - whether arguments will be parsed in a case insensitive manner
   * @returns {string}
   */
  validate (caseInsensitive) {
    const someHaveNoName = this.some(def => !def.name)
    if (someHaveNoName) {
      halt(
        'INVALID_DEFINITIONS',
        'Invalid option definitions: the `name` property is required on each definition'
      )
    }

    const someDontHaveFunctionType = this.some(def => def.type && typeof def.type !== 'function')
    if (someDontHaveFunctionType) {
      halt(
        'INVALID_DEFINITIONS',
        'Invalid option definitions: the `type` property must be a setter fuction (default: `Boolean`)'
      )
    }

    let invalidOption

    const numericAlias = this.some(def => {
      invalidOption = def
      return t.isDefined(def.alias) && t.isNumber(def.alias)
    })
    if (numericAlias) {
      halt(
        'INVALID_DEFINITIONS',
        'Invalid option definition: to avoid ambiguity an alias cannot be numeric [--' + invalidOption.name + ' alias is -' + invalidOption.alias + ']'
      )
    }

    const multiCharacterAlias = this.some(def => {
      invalidOption = def
      return t.isDefined(def.alias) && def.alias.length !== 1
    })
    if (multiCharacterAlias) {
      halt(
        'INVALID_DEFINITIONS',
        'Invalid option definition: an alias must be a single character'
      )
    }

    const hypenAlias = this.some(def => {
      invalidOption = def
      return def.alias === '-'
    })
    if (hypenAlias) {
      halt(
        'INVALID_DEFINITIONS',
        'Invalid option definition: an alias cannot be "-"'
      )
    }

    const duplicateName = hasDuplicates(this.map(def => caseInsensitive ? def.name.toLowerCase() : def.name))
    if (duplicateName) {
      halt(
        'INVALID_DEFINITIONS',
        'Two or more option definitions have the same name'
      )
    }

    const duplicateAlias = hasDuplicates(this.map(def => caseInsensitive && t.isDefined(def.alias) ? def.alias.toLowerCase() : def.alias))
    if (duplicateAlias) {
      halt(
        'INVALID_DEFINITIONS',
        'Two or more option definitions have the same alias'
      )
    }

    const duplicateDefaultOption = this.filter(def => def.defaultOption === true).length > 1;
    if (duplicateDefaultOption) {
      halt(
        'INVALID_DEFINITIONS',
        'Only one option definition can be the defaultOption'
      )
    }

    const defaultBoolean = this.some(def => {
      invalidOption = def
      return def.isBoolean() && def.defaultOption
    })
    if (defaultBoolean) {
      halt(
        'INVALID_DEFINITIONS',
        `A boolean option ["${invalidOption.name}"] can not also be the defaultOption.`
      )
    }
  }

  /**
   * Get definition by option arg (e.g. `--one` or `-o`)
   * @param {string} [arg] the argument name to get the definition for
   * @param {boolean} [caseInsensitive] whether to use case insensitive comparisons when finding the appropriate definition
   * @returns {Definition}
   */
  get (arg, caseInsensitive) {
    if (argvTools.isOption(arg)) {
      if (argvTools.re.short.test(arg)) {
        const shortOptionName = argvTools.getOptionName(arg)
        if (caseInsensitive) {
          const lowercaseShortOptionName = shortOptionName.toLowerCase()
          return this.find(def => t.isDefined(def.alias) && def.alias.toLowerCase() === lowercaseShortOptionName)
        } else {
          return this.find(def => def.alias === shortOptionName)
        }
      } else {
        const optionName = argvTools.getOptionName(arg)
        if (caseInsensitive) {
          const lowercaseOptionName = optionName.toLowerCase()
          return this.find(def => def.name.toLowerCase() === lowercaseOptionName)
        } else {
          return this.find(def => def.name === optionName)
        }
      }
    } else {
      return this.find(def => def.name === arg)
    }
  }

  getDefault () {
    return this.find(def => def.defaultOption === true)
  }

  isGrouped () {
    return this.some(def => def.group)
  }

  whereGrouped () {
    return this.filter(containsValidGroup)
  }

  whereNotGrouped () {
    return this.filter(def => !containsValidGroup(def))
  }

  whereDefaultValueSet () {
    return this.filter(def => t.isDefined(def.defaultValue))
  }

  static from (definitions, caseInsensitive) {
    if (definitions instanceof this) return definitions
    const result = super.from(arrayify(definitions), def => Definition.create(def))
    result.validate(caseInsensitive)
    return result
  }
}

function halt (name, message) {
  const err = new Error(message)
  err.name = name
  throw err
}

function containsValidGroup (def) {
  return arrayify(def.group).some(group => group)
}

function hasDuplicates (array) {
  const items = {}
  for (let i = 0; i < array.length; i++) {
    const value = array[i]
    if (items[value]) {
      return true
    } else {
      if (t.isDefined(value)) items[value] = true
    }
  }
}

export default Definitions
