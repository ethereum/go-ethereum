/**
 * Mnemonist FixedDeque
 * =====================
 *
 * Fixed capacity double-ended queue implemented as ring deque.
 */
var iterables = require('./utils/iterables.js'),
    Iterator = require('obliterator/iterator');

/**
 * FixedDeque.
 *
 * @constructor
 */
function FixedDeque(ArrayClass, capacity) {

  if (arguments.length < 2)
    throw new Error('mnemonist/fixed-deque: expecting an Array class and a capacity.');

  if (typeof capacity !== 'number' || capacity <= 0)
    throw new Error('mnemonist/fixed-deque: `capacity` should be a positive number.');

  this.ArrayClass = ArrayClass;
  this.capacity = capacity;
  this.items = new ArrayClass(this.capacity);
  this.clear();
}

/**
 * Method used to clear the structure.
 *
 * @return {undefined}
 */
FixedDeque.prototype.clear = function() {

  // Properties
  this.start = 0;
  this.size = 0;
};

/**
 * Method used to append a value to the deque.
 *
 * @param  {any}    item - Item to append.
 * @return {number}      - Returns the new size of the deque.
 */
FixedDeque.prototype.push = function(item) {
  if (this.size === this.capacity)
    throw new Error('mnemonist/fixed-deque.push: deque capacity (' + this.capacity + ') exceeded!');

  var index = (this.start + this.size) % this.capacity;

  this.items[index] = item;

  return ++this.size;
};

/**
 * Method used to prepend a value to the deque.
 *
 * @param  {any}    item - Item to prepend.
 * @return {number}      - Returns the new size of the deque.
 */
FixedDeque.prototype.unshift = function(item) {
  if (this.size === this.capacity)
    throw new Error('mnemonist/fixed-deque.unshift: deque capacity (' + this.capacity + ') exceeded!');

  var index = this.start - 1;

  if (this.start === 0)
    index = this.capacity - 1;

  this.items[index] = item;
  this.start = index;

  return ++this.size;
};

/**
 * Method used to pop the deque.
 *
 * @return {any} - Returns the popped item.
 */
FixedDeque.prototype.pop = function() {
  if (this.size === 0)
    return;

  const index = (this.start + this.size - 1) % this.capacity;

  this.size--;

  return this.items[index];
};

/**
 * Method used to shift the deque.
 *
 * @return {any} - Returns the shifted item.
 */
FixedDeque.prototype.shift = function() {
  if (this.size === 0)
    return;

  var index = this.start;

  this.size--;
  this.start++;

  if (this.start === this.capacity)
    this.start = 0;

  return this.items[index];
};

/**
 * Method used to peek the first value of the deque.
 *
 * @return {any}
 */
FixedDeque.prototype.peekFirst = function() {
  if (this.size === 0)
    return;

  return this.items[this.start];
};

/**
 * Method used to peek the last value of the deque.
 *
 * @return {any}
 */
FixedDeque.prototype.peekLast = function() {
  if (this.size === 0)
    return;

  var index = this.start + this.size - 1;

  if (index > this.capacity)
    index -= this.capacity;

  return this.items[index];
};

/**
 * Method used to get the desired value of the deque.
 *
 * @param  {number} index
 * @return {any}
 */
FixedDeque.prototype.get = function(index) {
  if (this.size === 0)
    return;

  index = this.start + index;

  if (index > this.capacity)
    index -= this.capacity;

  return this.items[index];
};

/**
 * Method used to iterate over the deque.
 *
 * @param  {function}  callback - Function to call for each item.
 * @param  {object}    scope    - Optional scope.
 * @return {undefined}
 */
FixedDeque.prototype.forEach = function(callback, scope) {
  scope = arguments.length > 1 ? scope : this;

  var c = this.capacity,
      l = this.size,
      i = this.start,
      j = 0;

  while (j < l) {
    callback.call(scope, this.items[i], j, this);
    i++;
    j++;

    if (i === c)
      i = 0;
  }
};

/**
 * Method used to convert the deque to a JavaScript array.
 *
 * @return {array}
 */
// TODO: optional array class as argument?
FixedDeque.prototype.toArray = function() {

  // Optimization
  var offset = this.start + this.size;

  if (offset < this.capacity)
    return this.items.slice(this.start, offset);

  var array = new this.ArrayClass(this.size),
      c = this.capacity,
      l = this.size,
      i = this.start,
      j = 0;

  while (j < l) {
    array[j] = this.items[i];
    i++;
    j++;

    if (i === c)
      i = 0;
  }

  return array;
};

/**
 * Method used to create an iterator over the deque's values.
 *
 * @return {Iterator}
 */
FixedDeque.prototype.values = function() {
  var items = this.items,
      c = this.capacity,
      l = this.size,
      i = this.start,
      j = 0;

  return new Iterator(function() {
    if (j >= l)
      return {
        done: true
      };

    var value = items[i];

    i++;
    j++;

    if (i === c)
      i = 0;

    return {
      value: value,
      done: false
    };
  });
};

/**
 * Method used to create an iterator over the deque's entries.
 *
 * @return {Iterator}
 */
FixedDeque.prototype.entries = function() {
  var items = this.items,
      c = this.capacity,
      l = this.size,
      i = this.start,
      j = 0;

  return new Iterator(function() {
    if (j >= l)
      return {
        done: true
      };

    var value = items[i];

    i++;

    if (i === c)
      i = 0;

    return {
      value: [j++, value],
      done: false
    };
  });
};

/**
 * Attaching the #.values method to Symbol.iterator if possible.
 */
if (typeof Symbol !== 'undefined')
  FixedDeque.prototype[Symbol.iterator] = FixedDeque.prototype.values;

/**
 * Convenience known methods.
 */
FixedDeque.prototype.inspect = function() {
  var array = this.toArray();

  array.type = this.ArrayClass.name;
  array.capacity = this.capacity;

  // Trick so that node displays the name of the constructor
  Object.defineProperty(array, 'constructor', {
    value: FixedDeque,
    enumerable: false
  });

  return array;
};

if (typeof Symbol !== 'undefined')
  FixedDeque.prototype[Symbol.for('nodejs.util.inspect.custom')] = FixedDeque.prototype.inspect;

/**
 * Static @.from function taking an arbitrary iterable & converting it into
 * a deque.
 *
 * @param  {Iterable} iterable   - Target iterable.
 * @param  {function} ArrayClass - Array class to use.
 * @param  {number}   capacity   - Desired capacity.
 * @return {FiniteStack}
 */
FixedDeque.from = function(iterable, ArrayClass, capacity) {
  if (arguments.length < 3) {
    capacity = iterables.guessLength(iterable);

    if (typeof capacity !== 'number')
      throw new Error('mnemonist/fixed-deque.from: could not guess iterable length. Please provide desired capacity as last argument.');
  }

  var deque = new FixedDeque(ArrayClass, capacity);

  if (iterables.isArrayLike(iterable)) {
    var i, l;

    for (i = 0, l = iterable.length; i < l; i++)
      deque.items[i] = iterable[i];

    deque.size = l;

    return deque;
  }

  iterables.forEach(iterable, function(value) {
    deque.push(value);
  });

  return deque;
};

/**
 * Exporting.
 */
module.exports = FixedDeque;
