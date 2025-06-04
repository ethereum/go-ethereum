/**
 * Mnemonist FixedStack
 * =====================
 *
 * The fixed stack is a stack whose capacity is defined beforehand and that
 * cannot be exceeded. This class is really useful when combined with
 * byte arrays to save up some memory and avoid memory re-allocation, hence
 * speeding up computations.
 *
 * This has however a downside: you need to know the maximum size you stack
 * can have during your iteration (which is not too difficult to compute when
 * performing, say, a DFS on a balanced binary tree).
 */
var Iterator = require('obliterator/iterator'),
    iterables = require('./utils/iterables.js');

/**
 * FixedStack
 *
 * @constructor
 * @param {function} ArrayClass - Array class to use.
 * @param {number}   capacity   - Desired capacity.
 */
function FixedStack(ArrayClass, capacity) {

  if (arguments.length < 2)
    throw new Error('mnemonist/fixed-stack: expecting an Array class and a capacity.');

  if (typeof capacity !== 'number' || capacity <= 0)
    throw new Error('mnemonist/fixed-stack: `capacity` should be a positive number.');

  this.capacity = capacity;
  this.ArrayClass = ArrayClass;
  this.items = new this.ArrayClass(this.capacity);
  this.clear();
}

/**
 * Method used to clear the stack.
 *
 * @return {undefined}
 */
FixedStack.prototype.clear = function() {

  // Properties
  this.size = 0;
};

/**
 * Method used to add an item to the stack.
 *
 * @param  {any}    item - Item to add.
 * @return {number}
 */
FixedStack.prototype.push = function(item) {
  if (this.size === this.capacity)
    throw new Error('mnemonist/fixed-stack.push: stack capacity (' + this.capacity + ') exceeded!');

  this.items[this.size++] = item;
  return this.size;
};

/**
 * Method used to retrieve & remove the last item of the stack.
 *
 * @return {any}
 */
FixedStack.prototype.pop = function() {
  if (this.size === 0)
    return;

  return this.items[--this.size];
};

/**
 * Method used to get the last item of the stack.
 *
 * @return {any}
 */
FixedStack.prototype.peek = function() {
  return this.items[this.size - 1];
};

/**
 * Method used to iterate over the stack.
 *
 * @param  {function}  callback - Function to call for each item.
 * @param  {object}    scope    - Optional scope.
 * @return {undefined}
 */
FixedStack.prototype.forEach = function(callback, scope) {
  scope = arguments.length > 1 ? scope : this;

  for (var i = 0, l = this.items.length; i < l; i++)
    callback.call(scope, this.items[l - i - 1], i, this);
};

/**
 * Method used to convert the stack to a JavaScript array.
 *
 * @return {array}
 */
FixedStack.prototype.toArray = function() {
  var array = new this.ArrayClass(this.size),
      l = this.size - 1,
      i = this.size;

  while (i--)
    array[i] = this.items[l - i];

  return array;
};

/**
 * Method used to create an iterator over a stack's values.
 *
 * @return {Iterator}
 */
FixedStack.prototype.values = function() {
  var items = this.items,
      l = this.size,
      i = 0;

  return new Iterator(function() {
    if (i >= l)
      return {
        done: true
      };

    var value = items[l - i - 1];
    i++;

    return {
      value: value,
      done: false
    };
  });
};

/**
 * Method used to create an iterator over a stack's entries.
 *
 * @return {Iterator}
 */
FixedStack.prototype.entries = function() {
  var items = this.items,
      l = this.size,
      i = 0;

  return new Iterator(function() {
    if (i >= l)
      return {
        done: true
      };

    var value = items[l - i - 1];

    return {
      value: [i++, value],
      done: false
    };
  });
};

/**
 * Attaching the #.values method to Symbol.iterator if possible.
 */
if (typeof Symbol !== 'undefined')
  FixedStack.prototype[Symbol.iterator] = FixedStack.prototype.values;


/**
 * Convenience known methods.
 */
FixedStack.prototype.toString = function() {
  return this.toArray().join(',');
};

FixedStack.prototype.toJSON = function() {
  return this.toArray();
};

FixedStack.prototype.inspect = function() {
  var array = this.toArray();

  array.type = this.ArrayClass.name;
  array.capacity = this.capacity;

  // Trick so that node displays the name of the constructor
  Object.defineProperty(array, 'constructor', {
    value: FixedStack,
    enumerable: false
  });

  return array;
};

if (typeof Symbol !== 'undefined')
  FixedStack.prototype[Symbol.for('nodejs.util.inspect.custom')] = FixedStack.prototype.inspect;

/**
 * Static @.from function taking an arbitrary iterable & converting it into
 * a stack.
 *
 * @param  {Iterable} iterable   - Target iterable.
 * @param  {function} ArrayClass - Array class to use.
 * @param  {number}   capacity   - Desired capacity.
 * @return {FixedStack}
 */
FixedStack.from = function(iterable, ArrayClass, capacity) {

  if (arguments.length < 3) {
    capacity = iterables.guessLength(iterable);

    if (typeof capacity !== 'number')
      throw new Error('mnemonist/fixed-stack.from: could not guess iterable length. Please provide desired capacity as last argument.');
  }

  var stack = new FixedStack(ArrayClass, capacity);

  if (iterables.isArrayLike(iterable)) {
    var i, l;

    for (i = 0, l = iterable.length; i < l; i++)
      stack.items[i] = iterable[i];

    stack.size = l;

    return stack;
  }

  iterables.forEach(iterable, function(value) {
    stack.push(value);
  });

  return stack;
};

/**
 * Exporting.
 */
module.exports = FixedStack;
