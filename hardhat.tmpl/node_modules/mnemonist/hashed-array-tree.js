/**
 * Mnemonist HashedArrayTree
 * ==========================
 *
 * Abstract implementation of a hashed array tree representing arrays growing
 * dynamically.
 */

/**
 * Defaults.
 */
var DEFAULT_BLOCK_SIZE = 1024;

/**
 * Helpers.
 */
function powerOfTwo(x) {
  return (x & (x - 1)) === 0;
}

/**
 * HashedArrayTree.
 *
 * @constructor
 * @param {function}      ArrayClass           - An array constructor.
 * @param {number|object} initialCapacityOrOptions - Self-explanatory.
 */
function HashedArrayTree(ArrayClass, initialCapacityOrOptions) {
  if (arguments.length < 1)
    throw new Error('mnemonist/hashed-array-tree: expecting at least a byte array constructor.');

  var initialCapacity = initialCapacityOrOptions || 0,
      blockSize = DEFAULT_BLOCK_SIZE,
      initialLength = 0;

  if (typeof initialCapacityOrOptions === 'object') {
    initialCapacity = initialCapacityOrOptions.initialCapacity || 0;
    initialLength = initialCapacityOrOptions.initialLength || 0;
    blockSize = initialCapacityOrOptions.blockSize || DEFAULT_BLOCK_SIZE;
  }

  if (!blockSize || !powerOfTwo(blockSize))
    throw new Error('mnemonist/hashed-array-tree: block size should be a power of two.');

  var capacity = Math.max(initialLength, initialCapacity),
      initialBlocks = Math.ceil(capacity / blockSize);

  this.ArrayClass = ArrayClass;
  this.length = initialLength;
  this.capacity = initialBlocks * blockSize;
  this.blockSize = blockSize;
  this.offsetMask = blockSize - 1;
  this.blockMask = Math.log2(blockSize);

  // Allocating initial blocks
  this.blocks = new Array(initialBlocks);

  for (var i = 0; i < initialBlocks; i++)
    this.blocks[i] = new this.ArrayClass(this.blockSize);
}

/**
 * Method used to set a value.
 *
 * @param  {number} index - Index to edit.
 * @param  {any}    value - Value.
 * @return {HashedArrayTree}
 */
HashedArrayTree.prototype.set = function(index, value) {

  // Out of bounds?
  if (this.length < index)
    throw new Error('HashedArrayTree(' + this.ArrayClass.name + ').set: index out of bounds.');

  var block = index >> this.blockMask,
      i = index & this.offsetMask;

  this.blocks[block][i] = value;

  return this;
};

/**
 * Method used to get a value.
 *
 * @param  {number} index - Index to retrieve.
 * @return {any}
 */
HashedArrayTree.prototype.get = function(index) {
  if (this.length < index)
    return;

  var block = index >> this.blockMask,
      i = index & this.offsetMask;

  return this.blocks[block][i];
};

/**
 * Method used to grow the array.
 *
 * @param  {number}          capacity - Optional capacity to accomodate.
 * @return {HashedArrayTree}
 */
HashedArrayTree.prototype.grow = function(capacity) {
  if (typeof capacity !== 'number')
    capacity = this.capacity + this.blockSize;

  if (this.capacity >= capacity)
    return this;

  while (this.capacity < capacity) {
    this.blocks.push(new this.ArrayClass(this.blockSize));
    this.capacity += this.blockSize;
  }

  return this;
};

/**
 * Method used to resize the array. Won't deallocate.
 *
 * @param  {number}       length - Target length.
 * @return {HashedArrayTree}
 */
HashedArrayTree.prototype.resize = function(length) {
  if (length === this.length)
    return this;

  if (length < this.length) {
    this.length = length;
    return this;
  }

  this.length = length;
  this.grow(length);

  return this;
};

/**
 * Method used to push a value into the array.
 *
 * @param  {any}    value - Value to push.
 * @return {number}       - Length of the array.
 */
HashedArrayTree.prototype.push = function(value) {
  if (this.capacity === this.length)
    this.grow();

  var index = this.length;

  var block = index >> this.blockMask,
      i = index & this.offsetMask;

  this.blocks[block][i] = value;

  return ++this.length;
};

/**
 * Method used to pop the last value of the array.
 *
 * @return {number} - The popped value.
 */
HashedArrayTree.prototype.pop = function() {
  if (this.length === 0)
    return;

  var lastBlock = this.blocks[this.blocks.length - 1];

  var i = (--this.length) & this.offsetMask;

  return lastBlock[i];
};

/**
 * Convenience known methods.
 */
HashedArrayTree.prototype.inspect = function() {
  var proxy = new this.ArrayClass(this.length),
      block;

  for (var i = 0, l = this.length; i < l; i++) {
    block = i >> this.blockMask;
    proxy[i] = this.blocks[block][i & this.offsetMask];
  }

  proxy.type = this.ArrayClass.name;
  proxy.items = this.length;
  proxy.capacity = this.capacity;
  proxy.blockSize = this.blockSize;

  // Trick so that node displays the name of the constructor
  Object.defineProperty(proxy, 'constructor', {
    value: HashedArrayTree,
    enumerable: false
  });

  return proxy;
};

if (typeof Symbol !== 'undefined')
  HashedArrayTree.prototype[Symbol.for('nodejs.util.inspect.custom')] = HashedArrayTree.prototype.inspect;

/**
 * Exporting.
 */
module.exports = HashedArrayTree;
