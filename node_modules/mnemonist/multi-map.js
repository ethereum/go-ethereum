/**
 * Mnemonist MultiMap
 * ===================
 *
 * Implementation of a MultiMap with custom container.
 */
var Iterator = require('obliterator/iterator'),
    forEach = require('obliterator/foreach');

/**
 * MultiMap.
 *
 * @constructor
 */
function MultiMap(Container) {

  this.Container = Container || Array;
  this.items = new Map();
  this.clear();

  Object.defineProperty(this.items, 'constructor', {
    value: MultiMap,
    enumerable: false
  });
}

/**
 * Method used to clear the structure.
 *
 * @return {undefined}
 */
MultiMap.prototype.clear = function() {

  // Properties
  this.size = 0;
  this.dimension = 0;
  this.items.clear();
};

/**
 * Method used to set a value.
 *
 * @param  {any}      key   - Key.
 * @param  {any}      value - Value to add.
 * @return {MultiMap}
 */
MultiMap.prototype.set = function(key, value) {
  var container = this.items.get(key),
      sizeBefore;

  if (!container) {
    this.dimension++;
    container = new this.Container();
    this.items.set(key, container);
  }

  if (this.Container === Set) {
    sizeBefore = container.size;
    container.add(value);

    if (sizeBefore < container.size)
      this.size++;
  }
  else {
    container.push(value);
    this.size++;
  }

  return this;
};

/**
 * Method used to delete the given key.
 *
 * @param  {any}     key - Key to delete.
 * @return {boolean}
 */
MultiMap.prototype.delete = function(key) {
  var container = this.items.get(key);

  if (!container)
    return false;

  this.size -= (this.Container === Set ? container.size : container.length);
  this.dimension--;
  this.items.delete(key);

  return true;
};

/**
 * Method used to delete the remove an item in the container stored at the
 * given key.
 *
 * @param  {any}     key - Key to delete.
 * @return {boolean}
 */
MultiMap.prototype.remove = function(key, value) {
  var container = this.items.get(key),
      wasDeleted,
      index;

  if (!container)
    return false;

  if (this.Container === Set) {
    wasDeleted = container.delete(value);

    if (wasDeleted)
      this.size--;

    if (container.size === 0) {
      this.items.delete(key);
      this.dimension--;
    }

    return wasDeleted;
  }
  else {
    index = container.indexOf(value);

    if (index === -1)
      return false;

    this.size--;

    if (container.length === 1) {
      this.items.delete(key);
      this.dimension--;

      return true;
    }

    container.splice(index, 1);

    return true;
  }
};

/**
 * Method used to return whether the given keys exists in the map.
 *
 * @param  {any}     key - Key to check.
 * @return {boolean}
 */
MultiMap.prototype.has = function(key) {
  return this.items.has(key);
};

/**
 * Method used to return the container stored at the given key or `undefined`.
 *
 * @param  {any}     key - Key to get.
 * @return {boolean}
 */
MultiMap.prototype.get = function(key) {
  return this.items.get(key);
};

/**
 * Method used to return the multiplicity of the given key, meaning the number
 * of times it is set, or, more trivially, the size of the attached container.
 *
 * @param  {any}     key - Key to check.
 * @return {number}
 */
MultiMap.prototype.multiplicity = function(key) {
  var container = this.items.get(key);

  if (typeof container === 'undefined')
    return 0;

  return this.Container === Set ? container.size : container.length;
};
MultiMap.prototype.count = MultiMap.prototype.multiplicity;

/**
 * Method used to iterate over each of the key/value pairs.
 *
 * @param  {function}  callback - Function to call for each item.
 * @param  {object}    scope    - Optional scope.
 * @return {undefined}
 */
MultiMap.prototype.forEach = function(callback, scope) {
  scope = arguments.length > 1 ? scope : this;

  // Inner iteration function is created here to avoid creating it in the loop
  var key;
  function inner(value) {
    callback.call(scope, value, key);
  }

  this.items.forEach(function(container, k) {
    key = k;
    container.forEach(inner);
  });
};

/**
 * Method used to iterate over each of the associations.
 *
 * @param  {function}  callback - Function to call for each item.
 * @param  {object}    scope    - Optional scope.
 * @return {undefined}
 */
MultiMap.prototype.forEachAssociation = function(callback, scope) {
  scope = arguments.length > 1 ? scope : this;

  this.items.forEach(callback, scope);
};

/**
 * Method returning an iterator over the map's keys.
 *
 * @return {Iterator}
 */
MultiMap.prototype.keys = function() {
  return this.items.keys();
};

/**
 * Method returning an iterator over the map's keys.
 *
 * @return {Iterator}
 */
MultiMap.prototype.values = function() {
  var iterator = this.items.values(),
      inContainer = false,
      countainer,
      step,
      i,
      l;

  if (this.Container === Set)
    return new Iterator(function next() {
      if (!inContainer) {
        step = iterator.next();

        if (step.done)
          return {done: true};

        inContainer = true;
        countainer = step.value.values();
      }

      step = countainer.next();

      if (step.done) {
        inContainer = false;
        return next();
      }

      return {
        done: false,
        value: step.value
      };
    });

  return new Iterator(function next() {
    if (!inContainer) {
      step = iterator.next();

      if (step.done)
        return {done: true};

      inContainer = true;
      countainer = step.value;
      i = 0;
      l = countainer.length;
    }

    if (i >= l) {
      inContainer = false;
      return next();
    }

    return {
      done: false,
      value: countainer[i++]
    };
  });
};

/**
 * Method returning an iterator over the map's entries.
 *
 * @return {Iterator}
 */
MultiMap.prototype.entries = function() {
  var iterator = this.items.entries(),
      inContainer = false,
      countainer,
      step,
      key,
      i,
      l;

  if (this.Container === Set)
    return new Iterator(function next() {
      if (!inContainer) {
        step = iterator.next();

        if (step.done)
          return {done: true};

        inContainer = true;
        key = step.value[0];
        countainer = step.value[1].values();
      }

      step = countainer.next();

      if (step.done) {
        inContainer = false;
        return next();
      }

      return {
        done: false,
        value: [key, step.value]
      };
    });

  return new Iterator(function next() {
    if (!inContainer) {
      step = iterator.next();

      if (step.done)
        return {done: true};

      inContainer = true;
      key = step.value[0];
      countainer = step.value[1];
      i = 0;
      l = countainer.length;
    }

    if (i >= l) {
      inContainer = false;
      return next();
    }

    return {
      done: false,
      value: [key, countainer[i++]]
    };
  });
};

/**
 * Method returning an iterator over the map's containers.
 *
 * @return {Iterator}
 */
MultiMap.prototype.containers = function() {
  return this.items.values();
};

/**
 * Method returning an iterator over the map's associations.
 *
 * @return {Iterator}
 */
MultiMap.prototype.associations = function() {
  return this.items.entries();
};

/**
 * Attaching the #.entries method to Symbol.iterator if possible.
 */
if (typeof Symbol !== 'undefined')
  MultiMap.prototype[Symbol.iterator] = MultiMap.prototype.entries;

/**
 * Convenience known methods.
 */
MultiMap.prototype.inspect = function() {
  return this.items;
};

if (typeof Symbol !== 'undefined')
  MultiMap.prototype[Symbol.for('nodejs.util.inspect.custom')] = MultiMap.prototype.inspect;
MultiMap.prototype.toJSON = function() {
  return this.items;
};

/**
 * Static @.from function taking an arbitrary iterable & converting it into
 * a structure.
 *
 * @param  {Iterable} iterable  - Target iterable.
 * @param  {Class}    Container - Container.
 * @return {MultiMap}
 */
MultiMap.from = function(iterable, Container) {
  var map = new MultiMap(Container);

  forEach(iterable, function(value, key) {
    map.set(key, value);
  });

  return map;
};

/**
 * Exporting.
 */
module.exports = MultiMap;
