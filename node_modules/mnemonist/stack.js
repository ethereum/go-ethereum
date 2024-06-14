/**
 * Mnemonist Stack
 * ================
 *
 * Stack implementation relying on JavaScript arrays, which are fast enough &
 * correctly optimized for this kind of work.
 */
var Iterator = require('obliterator/iterator'),
    forEach = require('obliterator/foreach');

/**
 * Stack
 *
 * @constructor
 */
function Stack() {
  this.clear();
}

/**
 * Method used to clear the stack.
 *
 * @return {undefined}
 */
Stack.prototype.clear = function() {

  // Properties
  this.items = [];
  this.size = 0;
};

/**
 * Method used to add an item to the stack.
 *
 * @param  {any}    item - Item to add.
 * @return {number}
 */
Stack.prototype.push = function(item) {
  this.items.push(item);
  return ++this.size;
};

/**
 * Method used to retrieve & remove the last item of the stack.
 *
 * @return {any}
 */
Stack.prototype.pop = function() {
  if (this.size === 0)
    return;

  this.size--;
  return this.items.pop();
};

/**
 * Method used to get the last item of the stack.
 *
 * @return {any}
 */
Stack.prototype.peek = function() {
  return this.items[this.size - 1];
};

/**
 * Method used to iterate over the stack.
 *
 * @param  {function}  callback - Function to call for each item.
 * @param  {object}    scope    - Optional scope.
 * @return {undefined}
 */
Stack.prototype.forEach = function(callback, scope) {
  scope = arguments.length > 1 ? scope : this;

  for (var i = 0, l = this.items.length; i < l; i++)
    callback.call(scope, this.items[l - i - 1], i, this);
};

/**
 * Method used to convert the stack to a JavaScript array.
 *
 * @return {array}
 */
Stack.prototype.toArray = function() {
  var array = new Array(this.size),
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
Stack.prototype.values = function() {
  var items = this.items,
      l = items.length,
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
Stack.prototype.entries = function() {
  var items = this.items,
      l = items.length,
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
  Stack.prototype[Symbol.iterator] = Stack.prototype.values;


/**
 * Convenience known methods.
 */
Stack.prototype.toString = function() {
  return this.toArray().join(',');
};

Stack.prototype.toJSON = function() {
  return this.toArray();
};

Stack.prototype.inspect = function() {
  var array = this.toArray();

  // Trick so that node displays the name of the constructor
  Object.defineProperty(array, 'constructor', {
    value: Stack,
    enumerable: false
  });

  return array;
};

if (typeof Symbol !== 'undefined')
  Stack.prototype[Symbol.for('nodejs.util.inspect.custom')] = Stack.prototype.inspect;

/**
 * Static @.from function taking an arbitrary iterable & converting it into
 * a stack.
 *
 * @param  {Iterable} iterable   - Target iterable.
 * @return {Stack}
 */
Stack.from = function(iterable) {
  var stack = new Stack();

  forEach(iterable, function(value) {
    stack.push(value);
  });

  return stack;
};

/**
 * Static @.of function taking an arbitrary number of arguments & converting it
 * into a stack.
 *
 * @param  {...any} args
 * @return {Stack}
 */
Stack.of = function() {
  return Stack.from(arguments);
};

/**
 * Exporting.
 */
module.exports = Stack;
