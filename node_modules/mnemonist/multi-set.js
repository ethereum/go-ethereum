/**
 * Mnemonist MultiSet
 * ====================
 *
 * JavaScript implementation of a MultiSet.
 */
var Iterator = require('obliterator/iterator'),
    forEach = require('obliterator/foreach'),
    FixedReverseHeap = require('./fixed-reverse-heap.js');

/**
 * Helpers.
 */
var MULTISET_ITEM_COMPARATOR = function(a, b) {
  if (a[1] > b[1])
    return -1;
  if (a[1] < b[1])
    return 1;

  return 0;
};

// TODO: helper functions: union, intersection, sum, difference, subtract

/**
 * MultiSet.
 *
 * @constructor
 */
function MultiSet() {
  this.items = new Map();

  Object.defineProperty(this.items, 'constructor', {
    value: MultiSet,
    enumerable: false
  });

  this.clear();
}

/**
 * Method used to clear the structure.
 *
 * @return {undefined}
 */
MultiSet.prototype.clear = function() {

  // Properties
  this.size = 0;
  this.dimension = 0;
  this.items.clear();
};

/**
 * Method used to add an item to the set.
 *
 * @param  {any}    item  - Item to add.
 * @param  {number} count - Optional count.
 * @return {MultiSet}
 */
MultiSet.prototype.add = function(item, count) {
  if (count === 0)
    return this;

  if (count < 0)
    return this.remove(item, -count);

  count = count || 1;

  if (typeof count !== 'number')
    throw new Error('mnemonist/multi-set.add: given count should be a number.');

  this.size += count;

  const currentCount = this.items.get(item);

  if (currentCount === undefined)
    this.dimension++;
  else
    count += currentCount;

  this.items.set(item, count);

  return this;
};

/**
 * Method used to set the multiplicity of an item in the set.
 *
 * @param  {any}    item  - Target item.
 * @param  {number} count - Desired multiplicity.
 * @return {MultiSet}
 */
MultiSet.prototype.set = function(item, count) {
  var currentCount;

  if (typeof count !== 'number')
    throw new Error('mnemonist/multi-set.set: given count should be a number.');

  // Setting an item to 0 or to a negative number means deleting it from the set
  if (count <= 0) {
    currentCount = this.items.get(item);

    if (typeof currentCount !== 'undefined') {
      this.size -= currentCount;
      this.dimension--;
    }

    this.items.delete(item);
    return this;
  }

  count = count || 1;

  currentCount = this.items.get(item);

  if (typeof currentCount === 'number') {
    this.items.set(item, currentCount + count);
  }
  else {
    this.dimension++;
    this.items.set(item, count);
  }

  this.size += count;

  return this;
};

/**
 * Method used to return whether the item exists in the set.
 *
 * @param  {any} item  - Item to check.
 * @return {boolan}
 */
MultiSet.prototype.has = function(item) {
  return this.items.has(item);
};

/**
 * Method used to delete an item from the set.
 *
 * @param  {any} item  - Item to delete.
 * @return {boolan}
 */
MultiSet.prototype.delete = function(item) {
  var count = this.items.get(item);

  if (count === 0)
    return false;

  this.size -= count;
  this.dimension--;
  this.items.delete(item);

  return true;
};

/**
 * Method used to remove an item from the set.
 *
 * @param  {any} item  - Item to delete.
 * @param  {number} count - Optional count.
 * @return {undefined}
 */
MultiSet.prototype.remove = function(item, count) {
  if (count === 0)
    return;

  if (count < 0)
    return this.add(item, -count);

  count = count || 1;

  if (typeof count !== 'number')
    throw new Error('mnemonist/multi-set.remove: given count should be a number.');

  var currentCount = this.multiplicity(item),
      newCount = Math.max(0, currentCount - count);

  if (newCount === 0) {
    this.delete(item);
  }
  else {
    this.items.set(item, newCount);
    this.size -= (currentCount - newCount);
  }

  return;
};

/**
 * Method used to change a key into another one, merging counts if the target
 * key already exists.
 *
 * @param  {any} a - From key.
 * @param  {any} b - To key.
 * @return {MultiSet}
 */
MultiSet.prototype.edit = function(a, b) {
  var am = this.multiplicity(a);

  // If a does not exist in the set, we can stop right there
  if (am === 0)
    return;

  var bm = this.multiplicity(b);

  this.items.set(b, am + bm);
  this.items.delete(a);

  return this;
};

/**
 * Method used to return the multiplicity of the given item.
 *
 * @param  {any} item  - Item to get.
 * @return {number}
 */
MultiSet.prototype.multiplicity = function(item) {
  var count = this.items.get(item);

  if (typeof count === 'undefined')
    return 0;

  return count;
};
MultiSet.prototype.get = MultiSet.prototype.multiplicity;
MultiSet.prototype.count = MultiSet.prototype.multiplicity;

/**
 * Method used to return the frequency of the given item in the set.
 *
 * @param  {any} item - Item to get.
 * @return {number}
 */
MultiSet.prototype.frequency = function(item) {
  if (this.size === 0)
    return 0;

  var count = this.multiplicity(item);

  return count / this.size;
};

/**
 * Method used to return the n most common items from the set.
 *
 * @param  {number} n - Number of items to retrieve.
 * @return {array}
 */
MultiSet.prototype.top = function(n) {
  if (typeof n !== 'number' || n <= 0)
    throw new Error('mnemonist/multi-set.top: n must be a number > 0.');

  var heap = new FixedReverseHeap(Array, MULTISET_ITEM_COMPARATOR, n);

  var iterator = this.items.entries(),
      step;

  while ((step = iterator.next(), !step.done))
    heap.push(step.value);

  return heap.consume();
};

/**
 * Method used to iterate over the set's values.
 *
 * @param  {function}  callback - Function to call for each item.
 * @param  {object}    scope    - Optional scope.
 * @return {undefined}
 */
MultiSet.prototype.forEach = function(callback, scope) {
  scope = arguments.length > 1 ? scope : this;

  var i;

  this.items.forEach(function(multiplicity, value) {

    for (i = 0; i < multiplicity; i++)
      callback.call(scope, value, value);
  });
};

/**
 * Method used to iterate over the set's multiplicities.
 *
 * @param  {function}  callback - Function to call for each multiplicity.
 * @param  {object}    scope    - Optional scope.
 * @return {undefined}
 */
MultiSet.prototype.forEachMultiplicity = function(callback, scope) {
  scope = arguments.length > 1 ? scope : this;

  this.items.forEach(callback, scope);
};

/**
 * Method returning an iterator over the set's keys. I.e. its unique values,
 * in a sense.
 *
 * @return {Iterator}
 */
MultiSet.prototype.keys = function() {
  return this.items.keys();
};

/**
 * Method returning an iterator over the set's values.
 *
 * @return {Iterator}
 */
MultiSet.prototype.values = function() {
  var iterator = this.items.entries(),
      inContainer = false,
      step,
      value,
      multiplicity,
      i;

  return new Iterator(function next() {
    if (!inContainer) {
      step = iterator.next();

      if (step.done)
        return {done: true};

      inContainer = true;
      value = step.value[0];
      multiplicity = step.value[1];
      i = 0;
    }

    if (i >= multiplicity) {
      inContainer = false;
      return next();
    }

    i++;

    return {
      done: false,
      value: value
    };
  });
};

/**
 * Method returning an iterator over the set's multiplicities.
 *
 * @return {Iterator}
 */
MultiSet.prototype.multiplicities = function() {
  return this.items.entries();
};

/**
 * Attaching the #.entries method to Symbol.iterator if possible.
 */
if (typeof Symbol !== 'undefined')
  MultiSet.prototype[Symbol.iterator] = MultiSet.prototype.values;

/**
 * Convenience known methods.
 */
MultiSet.prototype.inspect = function() {
  return this.items;
};

if (typeof Symbol !== 'undefined')
  MultiSet.prototype[Symbol.for('nodejs.util.inspect.custom')] = MultiSet.prototype.inspect;
MultiSet.prototype.toJSON = function() {
  return this.items;
};

/**
 * Static @.from function taking an arbitrary iterable & converting it into
 * a structure.
 *
 * @param  {Iterable} iterable - Target iterable.
 * @return {MultiSet}
 */
MultiSet.from = function(iterable) {
  var set = new MultiSet();

  forEach(iterable, function(value) {
    set.add(value);
  });

  return set;
};

/**
 * Function returning whether the multiset A is a subset of the multiset B.
 *
 * @param  {MultiSet} A - First set.
 * @param  {MultiSet} B - Second set.
 * @return {boolean}
 */
MultiSet.isSubset = function(A, B) {
  var iterator = A.multiplicities(),
      step,
      key,
      mA;

  // Shortcuts
  if (A === B)
    return true;

  if (A.dimension > B.dimension)
    return false;

  while ((step = iterator.next(), !step.done)) {
    key = step.value[0];
    mA = step.value[1];

    if (B.multiplicity(key) < mA)
      return false;
  }

  return true;
};

/**
 * Function returning whether the multiset A is a superset of the multiset B.
 *
 * @param  {MultiSet} A - First set.
 * @param  {MultiSet} B - Second set.
 * @return {boolean}
 */
MultiSet.isSuperset = function(A, B) {
  return MultiSet.isSubset(B, A);
};

/**
 * Exporting.
 */
module.exports = MultiSet;
