import Option from './option.mjs'
import Definitions from './option-definitions.mjs'
import camelCase from 'lodash.camelcase'

/**
 * A map of { DefinitionNameString: Option }. By default, an Output has an `_unknown` property and any options with defaultValues.
 */
class Output extends Map {
  constructor (definitions) {
    super()
    /**
     * @type {OptionDefinitions}
     */
    this.definitions = Definitions.from(definitions)

    /* by default, an Output has an `_unknown` property and any options with defaultValues */
    this.set('_unknown', Option.create({ name: '_unknown', multiple: true }))
    for (const def of this.definitions.whereDefaultValueSet()) {
      this.set(def.name, Option.create(def))
    }
  }

  toObject (options) {
    options = options || {}
    const output = {}
    for (const item of this) {
      const name = options.camelCase && item[0] !== '_unknown' ? camelCase(item[0]) : item[0]
      const option = item[1]
      if (name === '_unknown' && !option.get().length) continue
      output[name] = option.get()
    }

    if (options.skipUnknown) delete output._unknown
    return output
  }
}

export default Output
