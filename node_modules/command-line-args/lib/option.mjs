import arrayify from '../node_modules/array-back/index.mjs'
import t from '../node_modules/typical/index.mjs'
import Definition from './option-definition.mjs'
const _value = new WeakMap()

/**
 * Encapsulates behaviour (defined by an OptionDefinition) when setting values
 */
class Option {
  constructor (definition) {
    this.definition = new Definition(definition)
    this.state = null /* set or default */
    this.resetToDefault()
  }

  get () {
    return _value.get(this)
  }

  set (val) {
    this._set(val, 'set')
  }

  _set (val, state) {
    const def = this.definition
    if (def.isMultiple()) {
      /* don't add null or undefined to a multiple */
      if (val !== null && val !== undefined) {
        const arr = this.get()
        if (this.state === 'default') arr.length = 0
        arr.push(def.type(val))
        this.state = state
      }
    } else {
      /* throw if already set on a singlar defaultOption */
      if (!def.isMultiple() && this.state === 'set') {
        const err = new Error(`Singular option already set [${this.definition.name}=${this.get()}]`)
        err.name = 'ALREADY_SET'
        err.value = val
        err.optionName = def.name
        throw err
      } else if (val === null || val === undefined) {
        _value.set(this, val)
        // /* required to make 'partial: defaultOption with value equal to defaultValue 2' pass */
        // if (!(def.defaultOption && !def.isMultiple())) {
        //   this.state = state
        // }
      } else {
        _value.set(this, def.type(val))
        this.state = state
      }
    }
  }

  resetToDefault () {
    if (t.isDefined(this.definition.defaultValue)) {
      if (this.definition.isMultiple()) {
        _value.set(this, arrayify(this.definition.defaultValue).slice())
      } else {
        _value.set(this, this.definition.defaultValue)
      }
    } else {
      if (this.definition.isMultiple()) {
        _value.set(this, [])
      } else {
        _value.set(this, null)
      }
    }
    this.state = 'default'
  }

  static create (definition) {
    definition = new Definition(definition)
    if (definition.isBoolean()) {
      return FlagOption.create(definition)
    } else {
      return new this(definition)
    }
  }
}

class FlagOption extends Option {
  set (val) {
    super.set(true)
  }

  static create (def) {
    return new this(def)
  }
}

export default Option
