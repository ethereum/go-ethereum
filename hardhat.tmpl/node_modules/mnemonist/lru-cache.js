/**
 * Mnemonist LRUCache
 * ===================
 *
 * JavaScript implementation of the LRU Cache data structure. To save up
 * memory and allocations this implementation represents its underlying
 * doubly-linked list as static arrays and pointers. Thus, memory is allocated
 * only once at instantiation and JS objects are never created to serve as
 * pointers. This also means this implementation does not trigger too many
 * garbage collections.
 *
 * Note that to save up memory, a LRU Cache can be implemented using a singly
 * linked list by storing predecessors' pointers as hashmap values.
 * However, this means more hashmap lookups and would probably slow the whole
 * thing down. What's more, pointers are not the things taking most space in
 * memory.
 */
var Iterator = require('obliterator/iterator'),
    forEach = require('obliterator/foreach'),
    typed = require('./utils/typed-arrays.js'),
    iterables = require('./utils/iterables.js');

/**
 * LRUCache.
 *
 * @constructor
 * @param {function} Keys     - Array class for storing keys.
 * @param {function} Values   - Array class for storing values.
 * @param {number}   capacity - Desired capacity.
 */
function LRUCache(Keys, Values, capacity) {
  if (arguments.length < 2) {
    capacity = Keys;
    Keys = null;
    Values = null;
  }

  this.capacity = capacity;

  if (typeof this.capacity !== 'number' || this.capacity <= 0)
    throw new Error('mnemonist/lru-cache: capacity should be positive number.');
  else if (!isFinite(this.capacity) || Math.floor(this.capacity) !== this.capacity)
      throw new Error('mnemonist/lru-cache: capacity should be a finite positive integer.');

  var PointerArray = typed.getPointerArray(capacity);

  this.forward = new PointerArray(capacity);
  this.backward = new PointerArray(capacity);
  this.K = typeof Keys === 'function' ? new Keys(capacity) : new Array(capacity);
  this.V = typeof Values === 'function' ? new Values(capacity) : new Array(capacity);

  // Properties
  this.size = 0;
  this.head = 0;
  this.tail = 0;
  this.items = {};
}

/**
 * Method used to clear the structure.
 *
 * @return {undefined}
 */
LRUCache.prototype.clear = function() {
  this.size = 0;
  this.head = 0;
  this.tail = 0;
  this.items = {};
};

/**
 * Method used to splay a value on top.
 *
 * @param  {number}   pointer - Pointer of the value to splay on top.
 * @return {LRUCache}
 */
LRUCache.prototype.splayOnTop = function(pointer) {
  var oldHead = this.head;

  if (this.head === pointer)
    return this;

  var previous = this.backward[pointer],
      next = this.forward[pointer];

  if (this.tail === pointer) {
    this.tail = previous;
  }
  else {
    this.backward[next] = previous;
  }

  this.forward[previous] = next;

  this.backward[oldHead] = pointer;
  this.head = pointer;
  this.forward[pointer] = oldHead;

  return this;
};

/**
 * Method used to set the value for the given key in the cache.
 *
 * @param  {any} key   - Key.
 * @param  {any} value - Value.
 * @return {undefined}
 */
LRUCache.prototype.set = function(key, value) {

  // The key already exists, we just need to update the value and splay on top
  var pointer = this.items[key];

  if (typeof pointer !== 'undefined') {
    this.splayOnTop(pointer);
    this.V[pointer] = value;

    return;
  }

  // The cache is not yet full
  if (this.size < this.capacity) {
    pointer = this.size++;
  }

  // Cache is full, we need to drop the last value
  else {
    pointer = this.tail;
    this.tail = this.backward[pointer];
    delete this.items[this.K[pointer]];
  }

  // Storing key & value
  this.items[key] = pointer;
  this.K[pointer] = key;
  this.V[pointer] = value;

  // Moving the item at the front of the list
  this.forward[pointer] = this.head;
  this.backward[this.head] = pointer;
  this.head = pointer;
};

/**
 * Method used to set the value for the given key in the cache
 *
 * @param  {any} key   - Key.
 * @param  {any} value - Value.
 * @return {{evicted: boolean, key: any, value: any}} An object containing the
 * key and value of an item that was overwritten or evicted in the set
 * operation, as well as a boolean indicating whether it was evicted due to
 * limited capacity. Return value is null if nothing was evicted or overwritten
 * during the set operation.
 */
LRUCache.prototype.setpop = function(key, value) {
  var oldValue = null;
  var oldKey = null;
  // The key already exists, we just need to update the value and splay on top
  var pointer = this.items[key];

  if (typeof pointer !== 'undefined') {
    this.splayOnTop(pointer);
    oldValue = this.V[pointer];
    this.V[pointer] = value;
    return {evicted: false, key: key, value: oldValue};
  }

  // The cache is not yet full
  if (this.size < this.capacity) {
    pointer = this.size++;
  }

  // Cache is full, we need to drop the last value
  else {
    pointer = this.tail;
    this.tail = this.backward[pointer];
    oldValue = this.V[pointer];
    oldKey = this.K[pointer];
    delete this.items[this.K[pointer]];
  }

  // Storing key & value
  this.items[key] = pointer;
  this.K[pointer] = key;
  this.V[pointer] = value;

  // Moving the item at the front of the list
  this.forward[pointer] = this.head;
  this.backward[this.head] = pointer;
  this.head = pointer;

  // Return object if eviction took place, otherwise return null
  if (oldKey) {
    return {evicted: true, key: oldKey, value: oldValue};
  }
  else {
    return null;
  }
};

/**
 * Method used to check whether the key exists in the cache.
 *
 * @param  {any} key   - Key.
 * @return {boolean}
 */
LRUCache.prototype.has = function(key) {
  return key in this.items;
};

/**
 * Method used to get the value attached to the given key. Will move the
 * related key to the front of the underlying linked list.
 *
 * @param  {any} key   - Key.
 * @return {any}
 */
LRUCache.prototype.get = function(key) {
  var pointer = this.items[key];

  if (typeof pointer === 'undefined')
    return;

  this.splayOnTop(pointer);

  return this.V[pointer];
};

/**
 * Method used to get the value attached to the given key. Does not modify
 * the ordering of the underlying linked list.
 *
 * @param  {any} key   - Key.
 * @return {any}
 */
LRUCache.prototype.peek = function(key) {
    var pointer = this.items[key];

    if (typeof pointer === 'undefined')
        return;

    return this.V[pointer];
};

/**
 * Method used to iterate over the cache's entries using a callback.
 *
 * @param  {function}  callback - Function to call for each item.
 * @param  {object}    scope    - Optional scope.
 * @return {undefined}
 */
LRUCache.prototype.forEach = function(callback, scope) {
  scope = arguments.length > 1 ? scope : this;

  var i = 0,
      l = this.size;

  var pointer = this.head,
      keys = this.K,
      values = this.V,
      forward = this.forward;

  while (i < l) {

    callback.call(scope, values[pointer], keys[pointer], this);
    pointer = forward[pointer];

    i++;
  }
};

/**
 * Method used to create an iterator over the cache's keys from most
 * recently used to least recently used.
 *
 * @return {Iterator}
 */
LRUCache.prototype.keys = function() {
  var i = 0,
      l = this.size;

  var pointer = this.head,
      keys = this.K,
      forward = this.forward;

  return new Iterator(function() {
    if (i >= l)
      return {done: true};

    var key = keys[pointer];

    i++;

    if (i < l)
      pointer = forward[pointer];

    return {
      done: false,
      value: key
    };
  });
};

/**
 * Method used to create an iterator over the cache's values from most
 * recently used to least recently used.
 *
 * @return {Iterator}
 */
LRUCache.prototype.values = function() {
  var i = 0,
      l = this.size;

  var pointer = this.head,
      values = this.V,
      forward = this.forward;

  return new Iterator(function() {
    if (i >= l)
      return {done: true};

    var value = values[pointer];

    i++;

    if (i < l)
      pointer = forward[pointer];

    return {
      done: false,
      value: value
    };
  });
};

/**
 * Method used to create an iterator over the cache's entries from most
 * recently used to least recently used.
 *
 * @return {Iterator}
 */
LRUCache.prototype.entries = function() {
  var i = 0,
      l = this.size;

  var pointer = this.head,
      keys = this.K,
      values = this.V,
      forward = this.forward;

  return new Iterator(function() {
    if (i >= l)
      return {done: true};

    var key = keys[pointer],
        value = values[pointer];

    i++;

    if (i < l)
      pointer = forward[pointer];

    return {
      done: false,
      value: [key, value]
    };
  });
};

/**
 * Attaching the #.entries method to Symbol.iterator if possible.
 */
if (typeof Symbol !== 'undefined')
  LRUCache.prototype[Symbol.iterator] = LRUCache.prototype.entries;

/**
 * Convenience known methods.
 */
LRUCache.prototype.inspect = function() {
  var proxy = new Map();

  var iterator = this.entries(),
      step;

  while ((step = iterator.next(), !step.done))
    proxy.set(step.value[0], step.value[1]);

  // Trick so that node displays the name of the constructor
  Object.defineProperty(proxy, 'constructor', {
    value: LRUCache,
    enumerable: false
  });

  return proxy;
};

if (typeof Symbol !== 'undefined')
  LRUCache.prototype[Symbol.for('nodejs.util.inspect.custom')] = LRUCache.prototype.inspect;

/**
 * Static @.from function taking an arbitrary iterable & converting it into
 * a structure.
 *
 * @param  {Iterable} iterable - Target iterable.
 * @param  {function} Keys     - Array class for storing keys.
 * @param  {function} Values   - Array class for storing values.
 * @param  {number}   capacity - Cache's capacity.
 * @return {LRUCache}
 */
LRUCache.from = function(iterable, Keys, Values, capacity) {
  if (arguments.length < 2) {
    capacity = iterables.guessLength(iterable);

    if (typeof capacity !== 'number')
      throw new Error('mnemonist/lru-cache.from: could not guess iterable length. Please provide desired capacity as last argument.');
  }
  else if (arguments.length === 2) {
    capacity = Keys;
    Keys = null;
    Values = null;
  }

  var cache = new LRUCache(Keys, Values, capacity);

  forEach(iterable, function(value, key) {
    cache.set(key, value);
  });

  return cache;
};

/**
 * Exporting.
 */
module.exports = LRUCache;
