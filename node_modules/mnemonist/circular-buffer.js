/**
 * Mnemonist CircularBuffer
 * =========================
 *
 * Circular buffer implementation fit to use as a finite deque.
 */
var iterables = require('./utils/iterables.js'),
    FixedDeque = require('./fixed-deque');

/**
 * CircularBuffer.
 *
 * @constructor
 */
function CircularBuffer(ArrayClass, capacity) {

  if (arguments.length < 2)
    throw new Error('mnemonist/circular-buffer: expecting an Array class and a capacity.');

  if (typeof capacity !== 'number' || capacity <= 0)
    throw new Error('mnemonist/circular-buffer: `capacity` should be a positive number.');

  this.ArrayClass = ArrayClass;
  this.capacity = capacity;
  this.items = new ArrayClass(this.capacity);
  this.clear();
}

/**
 * Pasting most of the prototype from FixedDeque.
 */
function paste(name) {
  CircularBuffer.prototype[name] = FixedDeque.prototype[name];
}

Object.keys(FixedDeque.prototype).forEach(paste);

if (typeof Symbol !== 'undefined')
  Object.getOwnPropertySymbols(FixedDeque.prototype).forEach(paste);

/**
 * Method used to append a value to the buffer.
 *
 * @param  {any}    item - Item to append.
 * @return {number}      - Returns the new size of the buffer.
 */
CircularBuffer.prototype.push = function(item) {
  var index = (this.start + this.size) % this.capacity;

  this.items[index] = item;

  // Overwriting?
  if (this.size === this.capacity) {

    // If start is at the end, we wrap around the buffer
    this.start = (index + 1) % this.capacity;

    return this.size;
  }

  return ++this.size;
};

/**
 * Method used to prepend a value to the buffer.
 *
 * @param  {any}    item - Item to prepend.
 * @return {number}      - Returns the new size of the buffer.
 */
CircularBuffer.prototype.unshift = function(item) {
  var index = this.start - 1;

  if (this.start === 0)
    index = this.capacity - 1;

  this.items[index] = item;

  // Overwriting
  if (this.size === this.capacity) {

    this.start = index;

    return this.size;
  }

  this.start = index;

  return ++this.size;
};

/**
 * Static @.from function taking an arbitrary iterable & converting it into
 * a circular buffer.
 *
 * @param  {Iterable} iterable   - Target iterable.
 * @param  {function} ArrayClass - Array class to use.
 * @param  {number}   capacity   - Desired capacity.
 * @return {FiniteStack}
 */
CircularBuffer.from = function(iterable, ArrayClass, capacity) {
  if (arguments.length < 3) {
    capacity = iterables.guessLength(iterable);

    if (typeof capacity !== 'number')
      throw new Error('mnemonist/circular-buffer.from: could not guess iterable length. Please provide desired capacity as last argument.');
  }

  var buffer = new CircularBuffer(ArrayClass, capacity);

  if (iterables.isArrayLike(iterable)) {
    var i, l;

    for (i = 0, l = iterable.length; i < l; i++)
      buffer.items[i] = iterable[i];

    buffer.size = l;

    return buffer;
  }

  iterables.forEach(iterable, function(value) {
    buffer.push(value);
  });

  return buffer;
};

/**
 * Exporting.
 */
module.exports = CircularBuffer;
