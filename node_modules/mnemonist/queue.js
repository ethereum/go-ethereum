/**
 * Mnemonist Queue
 * ================
 *
 * Queue implementation based on the ideas of Queue.js that seems to beat
 * a LinkedList one in performance.
 */
var Iterator = require('obliterator/iterator'),
    forEach = require('obliterator/foreach');

/**
 * Queue
 *
 * @constructor
 */
function Queue() {
  this.clear();
}

/**
 * Method used to clear the queue.
 *
 * @return {undefined}
 */
Queue.prototype.clear = function() {

  // Properties
  this.items = [];
  this.offset = 0;
  this.size = 0;
};

/**
 * Method used to add an item to the queue.
 *
 * @param  {any}    item - Item to enqueue.
 * @return {number}
 */
Queue.prototype.enqueue = function(item) {

  this.items.push(item);
  return ++this.size;
};

/**
 * Method used to retrieve & remove the first item of the queue.
 *
 * @return {any}
 */
Queue.prototype.dequeue = function() {
  if (!this.size)
    return;

  var item = this.items[this.offset];

  if (++this.offset * 2 >= this.items.length) {
    this.items = this.items.slice(this.offset);
    this.offset = 0;
  }

  this.size--;

  return item;
};

/**
 * Method used to retrieve the first item of the queue.
 *
 * @return {any}
 */
Queue.prototype.peek = function() {
  if (!this.size)
    return;

  return this.items[this.offset];
};

/**
 * Method used to iterate over the queue.
 *
 * @param  {function}  callback - Function to call for each item.
 * @param  {object}    scope    - Optional scope.
 * @return {undefined}
 */
Queue.prototype.forEach = function(callback, scope) {
  scope = arguments.length > 1 ? scope : this;

  for (var i = this.offset, j = 0, l = this.items.length; i < l; i++, j++)
    callback.call(scope, this.items[i], j, this);
};

/*
 * Method used to convert the queue to a JavaScript array.
 *
 * @return {array}
 */
Queue.prototype.toArray = function() {
  return this.items.slice(this.offset);
};

/**
 * Method used to create an iterator over a queue's values.
 *
 * @return {Iterator}
 */
Queue.prototype.values = function() {
  var items = this.items,
      i = this.offset;

  return new Iterator(function() {
    if (i >= items.length)
      return {
        done: true
      };

    var value = items[i];
    i++;

    return {
      value: value,
      done: false
    };
  });
};

/**
 * Method used to create an iterator over a queue's entries.
 *
 * @return {Iterator}
 */
Queue.prototype.entries = function() {
  var items = this.items,
      i = this.offset,
      j = 0;

  return new Iterator(function() {
    if (i >= items.length)
      return {
        done: true
      };

    var value = items[i];
    i++;

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
  Queue.prototype[Symbol.iterator] = Queue.prototype.values;

/**
 * Convenience known methods.
 */
Queue.prototype.toString = function() {
  return this.toArray().join(',');
};

Queue.prototype.toJSON = function() {
  return this.toArray();
};

Queue.prototype.inspect = function() {
  var array = this.toArray();

  // Trick so that node displays the name of the constructor
  Object.defineProperty(array, 'constructor', {
    value: Queue,
    enumerable: false
  });

  return array;
};

if (typeof Symbol !== 'undefined')
  Queue.prototype[Symbol.for('nodejs.util.inspect.custom')] = Queue.prototype.inspect;

/**
 * Static @.from function taking an arbitrary iterable & converting it into
 * a queue.
 *
 * @param  {Iterable} iterable   - Target iterable.
 * @return {Queue}
 */
Queue.from = function(iterable) {
  var queue = new Queue();

  forEach(iterable, function(value) {
    queue.enqueue(value);
  });

  return queue;
};

/**
 * Static @.of function taking an arbitrary number of arguments & converting it
 * into a queue.
 *
 * @param  {...any} args
 * @return {Queue}
 */
Queue.of = function() {
  return Queue.from(arguments);
};

/**
 * Exporting.
 */
module.exports = Queue;
