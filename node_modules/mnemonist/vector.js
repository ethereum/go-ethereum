/**
 * Mnemonist Vector
 * =================
 *
 * Abstract implementation of a growing array that can be used with JavaScript
 * typed arrays and other array-like structures.
 *
 * Note: should try and use ArrayBuffer.transfer when it will be available.
 */
var Iterator = require('obliterator/iterator'),
    forEach = require('obliterator/foreach'),
    iterables = require('./utils/iterables.js'),
    typed = require('./utils/typed-arrays.js');

/**
 * Defaults.
 */
var DEFAULT_GROWING_POLICY = function(currentCapacity) {
  return Math.max(1, Math.ceil(currentCapacity * 1.5));
};

var pointerArrayFactory = function(capacity) {
  var PointerArray = typed.getPointerArray(capacity);

  return new PointerArray(capacity);
};

/**
 * Vector.
 *
 * @constructor
 * @param {function}      ArrayClass             - An array constructor.
 * @param {number|object} initialCapacityOrOptions - Self-explanatory:
 * @param {number}        initialCapacity          - Initial capacity.
 * @param {number}        initialLength            - Initial length.
 * @param {function}      policy                   - Allocation policy.
 */
function Vector(ArrayClass, initialCapacityOrOptions) {
  if (arguments.length < 1)
    throw new Error('mnemonist/vector: expecting at least a byte array constructor.');

  var initialCapacity = initialCapacityOrOptions || 0,
      policy = DEFAULT_GROWING_POLICY,
      initialLength = 0,
      factory = false;

  if (typeof initialCapacityOrOptions === 'object') {
    initialCapacity = initialCapacityOrOptions.initialCapacity || 0;
    initialLength = initialCapacityOrOptions.initialLength || 0;
    policy = initialCapacityOrOptions.policy || policy;
    factory = initialCapacityOrOptions.factory === true;
  }

  this.factory = factory ? ArrayClass : null;
  this.ArrayClass = ArrayClass;
  this.length = initialLength;
  this.capacity = Math.max(initialLength, initialCapacity);
  this.policy = policy;
  this.array = new ArrayClass(this.capacity);
}

/**
 * Method used to set a value.
 *
 * @param  {number} index - Index to edit.
 * @param  {any}    value - Value.
 * @return {Vector}
 */
Vector.prototype.set = function(index, value) {

  // Out of bounds?
  if (this.length < index)
    throw new Error('Vector(' + this.ArrayClass.name + ').set: index out of bounds.');

  // Updating value
  this.array[index] = value;

  return this;
};

/**
 * Method used to get a value.
 *
 * @param  {number} index - Index to retrieve.
 * @return {any}
 */
Vector.prototype.get = function(index) {
  if (this.length < index)
    return undefined;

  return this.array[index];
};

/**
 * Method used to apply the growing policy.
 *
 * @param  {number} [override] - Override capacity.
 * @return {number}
 */
Vector.prototype.applyPolicy = function(override) {
  var newCapacity = this.policy(override || this.capacity);

  if (typeof newCapacity !== 'number' || newCapacity < 0)
    throw new Error('mnemonist/vector.applyPolicy: policy returned an invalid value (expecting a positive integer).');

  if (newCapacity <= this.capacity)
    throw new Error('mnemonist/vector.applyPolicy: policy returned a less or equal capacity to allocate.');

  // TODO: we should probably check that the returned number is an integer
  return newCapacity;
};

/**
 * Method used to reallocate the underlying array.
 *
 * @param  {number}       capacity - Target capacity.
 * @return {Vector}
 */
Vector.prototype.reallocate = function(capacity) {
  if (capacity === this.capacity)
    return this;

  var oldArray = this.array;

  if (capacity < this.length)
    this.length = capacity;

  if (capacity > this.capacity) {
    if (this.factory === null)
      this.array = new this.ArrayClass(capacity);
    else
      this.array = this.factory(capacity);

    if (typed.isTypedArray(this.array)) {
      this.array.set(oldArray, 0);
    }
    else {
      for (var i = 0, l = this.length; i < l; i++)
        this.array[i] = oldArray[i];
    }
  }
  else {
    this.array = oldArray.slice(0, capacity);
  }

  this.capacity = capacity;

  return this;
};

/**
 * Method used to grow the array.
 *
 * @param  {number}       [capacity] - Optional capacity to match.
 * @return {Vector}
 */
Vector.prototype.grow = function(capacity) {
  var newCapacity;

  if (typeof capacity === 'number') {

    if (this.capacity >= capacity)
      return this;

    // We need to match the given capacity
    newCapacity = this.capacity;

    while (newCapacity < capacity)
      newCapacity = this.applyPolicy(newCapacity);

    this.reallocate(newCapacity);

    return this;
  }

  // We need to run the policy once
  newCapacity = this.applyPolicy();
  this.reallocate(newCapacity);

  return this;
};

/**
 * Method used to resize the array. Won't deallocate.
 *
 * @param  {number}       length - Target length.
 * @return {Vector}
 */
Vector.prototype.resize = function(length) {
  if (length === this.length)
    return this;

  if (length < this.length) {
    this.length = length;
    return this;
  }

  this.length = length;
  this.reallocate(length);

  return this;
};

/**
 * Method used to push a value into the array.
 *
 * @param  {any}    value - Value to push.
 * @return {number}       - Length of the array.
 */
Vector.prototype.push = function(value) {
  if (this.capacity === this.length)
    this.grow();

  this.array[this.length++] = value;

  return this.length;
};

/**
 * Method used to pop the last value of the array.
 *
 * @return {number} - The popped value.
 */
Vector.prototype.pop = function() {
  if (this.length === 0)
    return;

  return this.array[--this.length];
};

/**
 * Method used to create an iterator over a vector's values.
 *
 * @return {Iterator}
 */
Vector.prototype.values = function() {
  var items = this.array,
      l = this.length,
      i = 0;

  return new Iterator(function() {
    if (i >= l)
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
 * Method used to create an iterator over a vector's entries.
 *
 * @return {Iterator}
 */
Vector.prototype.entries = function() {
  var items = this.array,
      l = this.length,
      i = 0;

  return new Iterator(function() {
    if (i >= l)
      return {
        done: true
      };

    var value = items[i];

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
  Vector.prototype[Symbol.iterator] = Vector.prototype.values;

/**
 * Convenience known methods.
 */
Vector.prototype.inspect = function() {
  var proxy = this.array.slice(0, this.length);

  proxy.type = this.array.constructor.name;
  proxy.items = this.length;
  proxy.capacity = this.capacity;

  // Trick so that node displays the name of the constructor
  Object.defineProperty(proxy, 'constructor', {
    value: Vector,
    enumerable: false
  });

  return proxy;
};

if (typeof Symbol !== 'undefined')
  Vector.prototype[Symbol.for('nodejs.util.inspect.custom')] = Vector.prototype.inspect;

/**
 * Static @.from function taking an arbitrary iterable & converting it into
 * a vector.
 *
 * @param  {Iterable} iterable   - Target iterable.
 * @param  {function} ArrayClass - Byte array class.
 * @param  {number}   capacity   - Desired capacity.
 * @return {Vector}
 */
Vector.from = function(iterable, ArrayClass, capacity) {

  if (arguments.length < 3) {

    // Attempting to guess the needed capacity
    capacity = iterables.guessLength(iterable);

    if (typeof capacity !== 'number')
      throw new Error('mnemonist/vector.from: could not guess iterable length. Please provide desired capacity as last argument.');
  }

  var vector = new Vector(ArrayClass, capacity);

  forEach(iterable, function(value) {
    vector.push(value);
  });

  return vector;
};

/**
 * Exporting.
 */
function subClass(ArrayClass) {
  var SubClass = function(initialCapacityOrOptions) {
    Vector.call(this, ArrayClass, initialCapacityOrOptions);
  };

  for (var k in Vector.prototype) {
    if (Vector.prototype.hasOwnProperty(k))
      SubClass.prototype[k] = Vector.prototype[k];
  }

  SubClass.from = function(iterable, capacity) {
    return Vector.from(iterable, ArrayClass, capacity);
  };

  if (typeof Symbol !== 'undefined')
    SubClass.prototype[Symbol.iterator] = SubClass.prototype.values;

  return SubClass;
}

Vector.Int8Vector = subClass(Int8Array);
Vector.Uint8Vector = subClass(Uint8Array);
Vector.Uint8ClampedVector = subClass(Uint8ClampedArray);
Vector.Int16Vector = subClass(Int16Array);
Vector.Uint16Vector = subClass(Uint16Array);
Vector.Int32Vector = subClass(Int32Array);
Vector.Uint32Vector = subClass(Uint32Array);
Vector.Float32Vector = subClass(Float32Array);
Vector.Float64Vector = subClass(Float64Array);
Vector.PointerVector = subClass(pointerArrayFactory);

module.exports = Vector;
