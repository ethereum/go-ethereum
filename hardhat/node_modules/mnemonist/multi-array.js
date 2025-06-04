/**
 * Mnemonist MultiArray
 * =====================
 *
 * Memory-efficient representation of an array of arrays. In JavaScript and
 * most high-level languages, creating objects has a cost. This implementation
 * is therefore able to represent nested containers without needing to create
 * objects. This works by storing singly linked lists in a single flat array.
 * However, this means that this structure comes with some read/write
 * overhead but consume very few memory.
 *
 * This structure should be particularly suited to indices that will need to
 * merge arrays anyway when queried and that are quite heavily hit (such as
 * an inverted index or a quad tree).
 *
 * Note: the implementation does not require to keep track of head pointers
 * but this comes with some advantages such as not needing to offset pointers
 * by 1 and being able to perform in-order iteration. This remains quite lean
 * in memory and does not hinder performance whatsoever.
 */
var typed = require('./utils/typed-arrays.js'),
    Vector = require('./vector.js'),
    Iterator = require('obliterator/iterator');

var PointerVector = Vector.PointerVector;

/**
 * MultiArray.
 *
 * @constructor
 */
function MultiArray(Container, capacity) {
  this.capacity = capacity || null;
  this.Container = Container || Array;
  this.hasFixedCapacity = this.capacity !== null;

  if (typeof this.Container !== 'function')
    throw new Error('mnemonist/multi-array.constructor: container should be a function.');

  this.clear();
}

/**
 * Method used to clear the structure.
 *
 * @return {undefined}
 */
MultiArray.prototype.clear = function() {

  // Properties
  this.size = 0;
  this.dimension = 0;

  // NOTE: #.heads, #.tails & #.lengths have a length equal to the dimension of
  // the array, while #.pointers has a length equal to its size.

  // Storage
  if (this.hasFixedCapacity) {
    var capacity = this.capacity;

    var PointerArray = typed.getPointerArray(capacity);

    var policy = function(currentCapacity) {
      var newCapacity = Math.max(1, Math.ceil(currentCapacity * 1.5));

      // Clamping max allocation
      return Math.min(newCapacity, capacity);
    };

    var initialCapacity = Math.max(8, capacity);

    this.tails = new Vector(PointerArray, {policy: policy, initialCapacity: initialCapacity});
    this.lengths = new Vector(PointerArray, {policy: policy, initialCapacity: initialCapacity});
    this.pointers = new PointerArray(capacity);

    this.items = new this.Container(capacity);
  }
  else {

    this.tails = new PointerVector();
    this.lengths = new PointerVector();
    this.pointers = new PointerVector();

    this.items = new this.Container();
  }
};

/**
 * Method used to add an item to the container at the given index.
 *
 * @param  {number} index - Index of the container.
 * @param  {any}    item  - Item to add.
 * @return {MultiArray}
 */
MultiArray.prototype.set = function(index, item) {
  var pointer = this.size;

  // TODO: this can be factorized!

  if (this.hasFixedCapacity) {

    if (index >= this.capacity || this.size === this.capacity)
      throw new Error('mnemonist/multi-array: attempting to allocate further than capacity.');

    // This linked list does not exist yet. Let's create it
    if (index >= this.dimension) {

      // We may be required to grow the vectors
      this.dimension = index + 1;
      this.tails.grow(this.dimension);
      this.lengths.grow(this.dimension);

      this.tails.resize(this.dimension);
      this.lengths.resize(this.dimension);

      this.lengths.array[index] = 1;
    }

    // Appending to the list
    else {
      this.pointers[pointer] = this.tails.array[index];
      this.lengths.array[index]++;
    }

    this.tails.array[index] = pointer;
    this.items[pointer] = item;
  }
  else {

    // This linked list does not exist yet. Let's create it
    if (index >= this.dimension) {

      // We may be required to grow the vectors
      this.dimension = index + 1;
      this.tails.grow(this.dimension);
      this.lengths.grow(this.dimension);

      this.tails.resize(this.dimension);
      this.lengths.resize(this.dimension);

      this.pointers.push(0);
      this.lengths.array[index] = 1;
    }

    // Appending to the list
    else {
      this.pointers.push(this.tails.array[index]);
      this.lengths.array[index]++;
    }

    this.tails.array[index] = pointer;
    this.items.push(item);
  }

  this.size++;

  return this;
};

/**
 * Method used to push a new container holding the given value.
 * Note: it might be useful to make this function able to take an iterable
 * or variadic someday. For the time being it's just a convenience for
 * implementing compact multi maps and such.
 *
 * @param  {any} item  - Item to add.
 * @return {MultiArray}
 */
MultiArray.prototype.push = function(item) {
  var pointer = this.size,
      index = this.dimension;

  if (this.hasFixedCapacity) {

    if (index >= this.capacity || this.size === this.capacity)
      throw new Error('mnemonist/multi-array: attempting to allocate further than capacity.');

    this.items[pointer] = item;
  }
  else {
    this.items.push(item);
    this.pointers.push(0);
  }

  this.lengths.push(1);
  this.tails.push(pointer);

  this.dimension++;
  this.size++;

  return this;
};

/**
 * Method used to get the desired container.
 *
 * @param  {number} index - Index of the container.
 * @return {array}
 */
MultiArray.prototype.get = function(index) {
  if (index >= this.dimension)
    return;

  var pointers = this.hasFixedCapacity ? this.pointers : this.pointers.array;

  var pointer = this.tails.array[index],
      length = this.lengths.array[index],
      i = length;

  var array = new this.Container(length);

  while (i !== 0) {
    array[--i] = this.items[pointer];
    pointer = pointers[pointer];
  }

  return array;
};

/**
 * Method used to check if a container exists at the given index.
 *
 * @param  {number} index - Index of the container.
 * @return {boolean}
 */
MultiArray.prototype.has = function(index) {
  return index < this.dimension;
};

/**
 * Method used to get the size of the container stored at given index.
 *
 * @param  {number} index - Index of the container.
 * @return {number}
 */
MultiArray.prototype.multiplicity = function(index) {
  if (index >= this.dimension)
    return 0;

  return this.lengths.array[index];
};
MultiArray.prototype.count = MultiArray.prototype.multiplicity;

/**
 * Method used to iterate over the structure's containers.
 *
 * @return {Iterator}
 */
MultiArray.prototype.containers = function() {
  var self = this,
      l = this.dimension,
      i = 0;

  return new Iterator(function() {
    if (i >= l)
      return {done: true};

    return {value: self.get(i++)};
  });
};

/**
 * Method used to iterate over the structure's associations.
 *
 * @return {Iterator}
 */
MultiArray.prototype.associations = function() {
  var self = this,
      l = this.dimension,
      i = 0;

  return new Iterator(function() {
    if (i >= l)
      return {done: true};

    var data = {value: [i, self.get(i)]};

    i++;

    return data;
  });
};

/**
 * Method used to iterate over the structure's values in the global insertion
 * order.
 *
 * @param  {number}   [index] - Optionally, iterate over the values of a single
 *                              container at index.
 * @return {Iterator}
 */
MultiArray.prototype.values = function(index) {
  var items = this.items,
      length,
      i = 0;

  if (typeof index === 'number') {
    if (index >= this.dimension)
      return Iterator.empty();

    length = this.lengths.array[index];
    items = this.items;

    var pointers = this.hasFixedCapacity ? this.pointers : this.pointers.array;

    if (length === 0)
      return Iterator.empty();

    var pointer = this.tails.array[index],
        v;

    return new Iterator(function() {
      if (i === length)
        return {done: true};

      i++;
      v = items[pointer];
      pointer = pointers[pointer];

      return {done: false, value: v};
    });
  }

  length = this.size;

  return new Iterator(function() {
    if (i >= length)
      return {done: true};

    return {done: false, value: items[i++]};
  });
};

/**
 * Method used to iterate over the structure's entries.
 *
 * @return {Iterator}
 */
MultiArray.prototype.entries = function() {
  if (this.size === 0)
    return Iterator.empty();

  var inContainer = false,
      pointer,
      length,
      i = 0,
      j = 0,
      l = this.dimension,
      v;

  var pointers = this.hasFixedCapacity ? this.pointers : this.pointers.array,
      items = this.items,
      tails = this.tails.array,
      lengths = this.lengths.array;

  var iterator = new Iterator(function next() {
    if (!inContainer) {

      if (i >= l)
        return {done: true};

      length = lengths[i];
      pointer = tails[i];
      i++;

      if (length === 0)
        return next();

      j = 0;
      inContainer = true;
    }

    if (j === length) {
      inContainer = false;
      return next();
    }

    v = items[pointer];

    // TODO: guard for out-of-bounds
    pointer = pointers[pointer];

    j++;

    return {
      done: false,
      value: [i - 1, v]
    };
  });

  return iterator;
};

/**
 * Method used to iterate over the structure's keys.
 *
 * @return {Iterator}
 */
MultiArray.prototype.keys = function() {
  var i = 0,
      l = this.dimension;

  return new Iterator(function() {
    if (i >= l)
      return {done: true};

    return {done: false, value: i++};
  });
};

/**
 * Convenience known methods.
 */
MultiArray.prototype.inspect = function() {
  var proxy = new Array(this.dimension),
      i,
      l;

  for (i = 0, l = this.dimension; i < l; i++)
    proxy[i] = Array.from(this.get(i));

  if (this.hasFixedCapacity) {
    proxy.type = this.Container.name;
    proxy.capacity = this.capacity;
  }

  proxy.size = this.size;
  proxy.dimension = this.dimension;

  // Trick so that node displays the name of the constructor
  Object.defineProperty(proxy, 'constructor', {
    value: MultiArray,
    enumerable: false
  });

  return proxy;
};

if (typeof Symbol !== 'undefined')
  MultiArray.prototype[Symbol.for('nodejs.util.inspect.custom')] = MultiArray.prototype.inspect;

// TODO: .from

/**
 * Exporting.
 */
module.exports = MultiArray;
