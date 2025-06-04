/**
 * Mnemonist Fixed Reverse Heap
 * =============================
 *
 * Static heap implementation with fixed capacity. It's a "reverse" heap
 * because it stores the elements in reverse so we can replace the worst
 * item in logarithmic time. As such, one cannot pop this heap but can only
 * consume it at the end. This structure is very efficient when trying to
 * find the n smallest/largest items from a larger query (k nearest neigbors
 * for instance).
 */
var comparators = require('./utils/comparators.js'),
    Heap = require('./heap.js');

var DEFAULT_COMPARATOR = comparators.DEFAULT_COMPARATOR,
    reverseComparator = comparators.reverseComparator;

/**
 * Helper functions.
 */

/**
 * Function used to sift up.
 *
 * @param {function} compare - Comparison function.
 * @param {array}    heap    - Array storing the heap's data.
 * @param {number}   size    - Heap's true size.
 * @param {number}   i       - Index.
 */
function siftUp(compare, heap, size, i) {
  var endIndex = size,
      startIndex = i,
      item = heap[i],
      childIndex = 2 * i + 1,
      rightIndex;

  while (childIndex < endIndex) {
    rightIndex = childIndex + 1;

    if (
      rightIndex < endIndex &&
      compare(heap[childIndex], heap[rightIndex]) >= 0
    ) {
      childIndex = rightIndex;
    }

    heap[i] = heap[childIndex];
    i = childIndex;
    childIndex = 2 * i + 1;
  }

  heap[i] = item;
  Heap.siftDown(compare, heap, startIndex, i);
}

/**
 * Fully consumes the given heap.
 *
 * @param  {function} ArrayClass - Array class to use.
 * @param  {function} compare    - Comparison function.
 * @param  {array}    heap       - Array storing the heap's data.
 * @param  {number}   size       - True size of the heap.
 * @return {array}
 */
function consume(ArrayClass, compare, heap, size) {
  var l = size,
      i = l;

  var array = new ArrayClass(size),
      lastItem,
      item;

  while (i > 0) {
    lastItem = heap[--i];

    if (i !== 0) {
      item = heap[0];
      heap[0] = lastItem;
      siftUp(compare, heap, --size, 0);
      lastItem = item;
    }

    array[i] = lastItem;
  }

  return array;
}

/**
 * Binary Minimum FixedReverseHeap.
 *
 * @constructor
 * @param {function} ArrayClass - The class of array to use.
 * @param {function} comparator - Comparator function.
 * @param {number}   capacity   - Maximum number of items to keep.
 */
function FixedReverseHeap(ArrayClass, comparator, capacity) {

  // Comparator can be omitted
  if (arguments.length === 2) {
    capacity = comparator;
    comparator = null;
  }

  this.ArrayClass = ArrayClass;
  this.capacity = capacity;

  this.items = new ArrayClass(capacity);
  this.clear();
  this.comparator = comparator || DEFAULT_COMPARATOR;

  if (typeof capacity !== 'number' && capacity <= 0)
    throw new Error('mnemonist/FixedReverseHeap.constructor: capacity should be a number > 0.');

  if (typeof this.comparator !== 'function')
    throw new Error('mnemonist/FixedReverseHeap.constructor: given comparator should be a function.');

  this.comparator = reverseComparator(this.comparator);
}

/**
 * Method used to clear the heap.
 *
 * @return {undefined}
 */
FixedReverseHeap.prototype.clear = function() {

  // Properties
  this.size = 0;
};

/**
 * Method used to push an item into the heap.
 *
 * @param  {any}    item - Item to push.
 * @return {number}
 */
FixedReverseHeap.prototype.push = function(item) {

  // Still some place
  if (this.size < this.capacity) {
    this.items[this.size] = item;
    Heap.siftDown(this.comparator, this.items, 0, this.size);
    this.size++;
  }

  // Heap is full, we need to replace worst item
  else {

    if (this.comparator(item, this.items[0]) > 0)
      Heap.replace(this.comparator, this.items, item);
  }

  return this.size;
};

/**
 * Method used to peek the worst item in the heap.
 *
 * @return {any}
 */
FixedReverseHeap.prototype.peek = function() {
  return this.items[0];
};

/**
 * Method used to consume the heap fully and return its items as a sorted array.
 *
 * @return {array}
 */
FixedReverseHeap.prototype.consume = function() {
  var items = consume(this.ArrayClass, this.comparator, this.items, this.size);
  this.size = 0;

  return items;
};

/**
 * Method used to convert the heap to an array. Note that it basically clone
 * the heap and consumes it completely. This is hardly performant.
 *
 * @return {array}
 */
FixedReverseHeap.prototype.toArray = function() {
  return consume(this.ArrayClass, this.comparator, this.items.slice(0, this.size), this.size);
};

/**
 * Convenience known methods.
 */
FixedReverseHeap.prototype.inspect = function() {
  var proxy = this.toArray();

  // Trick so that node displays the name of the constructor
  Object.defineProperty(proxy, 'constructor', {
    value: FixedReverseHeap,
    enumerable: false
  });

  return proxy;
};

if (typeof Symbol !== 'undefined')
  FixedReverseHeap.prototype[Symbol.for('nodejs.util.inspect.custom')] = FixedReverseHeap.prototype.inspect;

/**
 * Exporting.
 */
module.exports = FixedReverseHeap;
