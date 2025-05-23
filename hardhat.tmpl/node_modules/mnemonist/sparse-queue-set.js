/**
 * Mnemonist SparseQueueSet
 * =========================
 *
 * JavaScript sparse queue set implemented on top of byte arrays.
 *
 * [Reference]: https://research.swtch.com/sparse
 */
var Iterator = require('obliterator/iterator'),
    getPointerArray = require('./utils/typed-arrays.js').getPointerArray;

/**
 * SparseQueueSet.
 *
 * @constructor
 */
function SparseQueueSet(capacity) {

  var ByteArray = getPointerArray(capacity);

  // Properties
  this.start = 0;
  this.size = 0;
  this.capacity = capacity;
  this.dense = new ByteArray(capacity);
  this.sparse = new ByteArray(capacity);
}

/**
 * Method used to clear the structure.
 *
 * @return {undefined}
 */
SparseQueueSet.prototype.clear = function() {
  this.start = 0;
  this.size = 0;
};

/**
 * Method used to check the existence of a member in the queue.
 *
 * @param  {number} member - Member to test.
 * @return {SparseQueueSet}
 */
SparseQueueSet.prototype.has = function(member) {
  if (this.size === 0)
    return false;

  var index = this.sparse[member];

  var inBounds = (
    index < this.capacity &&
    (
      index >= this.start &&
      index < this.start + this.size
    ) ||
    (
      index < ((this.start + this.size) % this.capacity)
    )
  );

  return (
    inBounds &&
    this.dense[index] === member
  );
};

/**
 * Method used to add a member to the queue.
 *
 * @param  {number} member - Member to add.
 * @return {SparseQueueSet}
 */
SparseQueueSet.prototype.enqueue = function(member) {
  var index = this.sparse[member];

  if (this.size !== 0) {
    var inBounds = (
      index < this.capacity &&
      (
        index >= this.start &&
        index < this.start + this.size
      ) ||
      (
        index < ((this.start + this.size) % this.capacity)
      )
    );

    if (inBounds && this.dense[index] === member)
      return this;
  }

  index = (this.start + this.size) % this.capacity;

  this.dense[index] = member;
  this.sparse[member] = index;
  this.size++;

  return this;
};

/**
 * Method used to remove the next member from the queue.
 *
 * @param  {number} member - Member to delete.
 * @return {boolean}
 */
SparseQueueSet.prototype.dequeue = function() {
  if (this.size === 0)
    return;

  var index = this.start;

  this.size--;
  this.start++;

  if (this.start === this.capacity)
    this.start = 0;

  var member = this.dense[index];

  this.sparse[member] = this.capacity;

  return member;
};

/**
 * Method used to iterate over the queue's values.
 *
 * @param  {function}  callback - Function to call for each item.
 * @param  {object}    scope    - Optional scope.
 * @return {undefined}
 */
SparseQueueSet.prototype.forEach = function(callback, scope) {
  scope = arguments.length > 1 ? scope : this;

  var c = this.capacity,
      l = this.size,
      i = this.start,
      j = 0;

  while (j < l) {
    callback.call(scope, this.dense[i], j, this);
    i++;
    j++;

    if (i === c)
      i = 0;
  }
};

/**
 * Method used to create an iterator over a set's values.
 *
 * @return {Iterator}
 */
SparseQueueSet.prototype.values = function() {
  var dense = this.dense,
      c = this.capacity,
      l = this.size,
      i = this.start,
      j = 0;

  return new Iterator(function() {
    if (j >= l)
      return {
        done: true
      };

    var value = dense[i];

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
 * Attaching the #.values method to Symbol.iterator if possible.
 */
if (typeof Symbol !== 'undefined')
  SparseQueueSet.prototype[Symbol.iterator] = SparseQueueSet.prototype.values;

/**
 * Convenience known methods.
 */
SparseQueueSet.prototype.inspect = function() {
  var proxy = [];

  this.forEach(function(member) {
    proxy.push(member);
  });

  // Trick so that node displays the name of the constructor
  Object.defineProperty(proxy, 'constructor', {
    value: SparseQueueSet,
    enumerable: false
  });

  proxy.capacity = this.capacity;

  return proxy;
};

if (typeof Symbol !== 'undefined')
  SparseQueueSet.prototype[Symbol.for('nodejs.util.inspect.custom')] = SparseQueueSet.prototype.inspect;

/**
 * Exporting.
 */
module.exports = SparseQueueSet;
