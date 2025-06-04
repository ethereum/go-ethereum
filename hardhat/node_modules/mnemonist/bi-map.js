/**
 * Mnemonist BiMap
 * ================
 *
 * JavaScript implementation of a BiMap.
 */
var forEach = require('obliterator/foreach');

/**
 * Inverse Map.
 *
 * @constructor
 */
function InverseMap(original) {

  this.size = 0;
  this.items = new Map();
  this.inverse = original;
}

/**
 * BiMap.
 *
 * @constructor
 */
function BiMap() {

  this.size = 0;
  this.items = new Map();
  this.inverse = new InverseMap(this);
}

/**
 * Method used to clear the map.
 *
 * @return {undefined}
 */
function clear() {
  this.size = 0;
  this.items.clear();
  this.inverse.items.clear();
}

BiMap.prototype.clear = clear;
InverseMap.prototype.clear = clear;

/**
 * Method used to set a relation.
 *
 * @param  {any} key - Key.
 * @param  {any} value - Value.
 * @return {BiMap|InverseMap}
 */
function set(key, value) {

  // First we need to attempt to see if the relation is not flawed
  if (this.items.has(key)) {
    var currentValue = this.items.get(key);

    // The relation already exists, we do nothing
    if (currentValue === value)
      return this;
    else
      this.inverse.items.delete(currentValue);
  }

  if (this.inverse.items.has(value)) {
    var currentKey = this.inverse.items.get(value);

    if (currentKey === key)
      return this;
    else
      this.items.delete(currentKey);
  }

  // Here we actually add the relation
  this.items.set(key, value);
  this.inverse.items.set(value, key);

  // Size
  this.size = this.items.size;
  this.inverse.size = this.inverse.items.size;

  return this;
}

BiMap.prototype.set = set;
InverseMap.prototype.set = set;

/**
 * Method used to delete a relation.
 *
 * @param  {any} key - Key.
 * @return {boolean}
 */
function del(key) {
  if (this.items.has(key)) {
    var currentValue = this.items.get(key);

    this.items.delete(key);
    this.inverse.items.delete(currentValue);

    // Size
    this.size = this.items.size;
    this.inverse.size = this.inverse.items.size;

    return true;
  }

  return false;
}

BiMap.prototype.delete = del;
InverseMap.prototype.delete = del;

/**
 * Mapping some Map prototype function unto our two classes.
 */
var METHODS = ['has', 'get', 'forEach', 'keys', 'values', 'entries'];

METHODS.forEach(function(name) {
  BiMap.prototype[name] = InverseMap.prototype[name] = function() {
    return Map.prototype[name].apply(this.items, arguments);
  };
});

/**
 * Attaching the #.values method to Symbol.iterator if possible.
 */
if (typeof Symbol !== 'undefined') {
  BiMap.prototype[Symbol.iterator] = BiMap.prototype.entries;
  InverseMap.prototype[Symbol.iterator] = InverseMap.prototype.entries;
}

/**
 * Convenience known methods.
 */
BiMap.prototype.inspect = function() {
  var dummy = {
    left: this.items,
    right: this.inverse.items
  };

  // Trick so that node displays the name of the constructor
  Object.defineProperty(dummy, 'constructor', {
    value: BiMap,
    enumerable: false
  });

  return dummy;
};

if (typeof Symbol !== 'undefined')
  BiMap.prototype[Symbol.for('nodejs.util.inspect.custom')] = BiMap.prototype.inspect;

InverseMap.prototype.inspect = function() {
  var dummy = {
    left: this.inverse.items,
    right: this.items
  };

  // Trick so that node displays the name of the constructor
  Object.defineProperty(dummy, 'constructor', {
    value: InverseMap,
    enumerable: false
  });

  return dummy;
};

if (typeof Symbol !== 'undefined')
  InverseMap.prototype[Symbol.for('nodejs.util.inspect.custom')] = InverseMap.prototype.inspect;


/**
 * Static @.from function taking an arbitrary iterable & converting it into
 * a bimap.
 *
 * @param  {Iterable} iterable - Target iterable.
 * @return {BiMap}
 */
BiMap.from = function(iterable) {
  var bimap = new BiMap();

  forEach(iterable, function(value, key) {
    bimap.set(key, value);
  });

  return bimap;
};

/**
 * Exporting.
 */
module.exports = BiMap;
