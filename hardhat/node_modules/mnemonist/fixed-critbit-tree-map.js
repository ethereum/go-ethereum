/* eslint no-constant-condition: 0 */

/* eslint-disable */

/**
 * Mnemonist FixedFixedCritBitTreeMap
 * ===================================
 *
 * TODO...
 *
 * [References]:
 * https://cr.yp.to/critbit.html
 * https://www.imperialviolet.org/binary/critbit.pdf
 */
var bitwise = require('./utils/bitwise.js'),
    typed = require('./utils/typed-arrays.js');

/**
 * Helpers.
 */

/**
 * Helper returning the direction we need to take given a key and an
 * encoded critbit.
 *
 * @param  {string} key     - Target key.
 * @param  {number} critbit - Packed address of byte + mask.
 * @return {number}         - 0, left or 1, right.
 */
function getDirection(key, critbit) {
  var byteIndex = critbit >> 8;

  if (byteIndex > key.length - 1)
    return 0;

  var byte = key.charCodeAt(byteIndex),
      mask = critbit & 0xff;

  return byte & mask;
}

/**
 * Helper returning the packed address of byte + mask or -1 if strings
 * are identical.
 *
 * @param  {string} a      - First key.
 * @param  {string} b      - Second key.
 * @return {number}        - Packed address of byte + mask.
 */
function findCriticalBit(a, b) {
  var i = 0,
      tmp;

  // Swapping so a is the shortest
  if (a.length > b.length) {
    tmp = b;
    b = a;
    a = tmp;
  }

  var l = a.length,
      mask;

  while (i < l) {
    if (a[i] !== b[i]) {
      mask = bitwise.msb8(
        a.charCodeAt(i) ^ b.charCodeAt(i)
      );

      return (i << 8) | mask;
    }

    i++;
  }

  // Strings are identical
  if (a.length === b.length)
    return -1;

  // NOTE: x ^ 0 is the same as x
  mask = bitwise.msb8(b.charCodeAt(i));

  return (i << 8) | mask;
}

/**
 * FixedCritBitTreeMap.
 *
 * @constructor
 */
function FixedCritBitTreeMap(capacity) {

  if (typeof capacity !== 'number' || capacity <= 0)
    throw new Error('mnemonist/fixed-critbit-tree-map: `capacity` should be a positive number.');

  // Properties
  this.capacity = capacity;
  this.offset = 0;
  this.root = 0;
  this.size = 0;

  var PointerArray = typed.getSignedPointerArray(capacity + 1);

  this.keys = new Array(capacity);
  this.values = new Array(capacity);
  this.lefts = new PointerArray(capacity - 1);
  this.rights = new PointerArray(capacity - 1);
  this.critbits = new Uint32Array(capacity);
}

/**
 * Method used to clear the FixedCritBitTreeMap.
 *
 * @return {undefined}
 */
FixedCritBitTreeMap.prototype.clear = function() {

  // Properties
  // TODO...
  this.root = null;
  this.size = 0;
};

/**
 * Method used to set the value of the given key in the trie.
 *
 * @param  {string}         key   - Key to set.
 * @param  {any}            value - Arbitrary value.
 * @return {FixedCritBitTreeMap}
 */
FixedCritBitTreeMap.prototype.set = function(key, value) {
  var pointer;

  // TODO: yell if capacity is already full!

  // Tree is empty
  if (this.size === 0) {
    this.keys[0] = key;
    this.values[0] = value;

    this.size++;

    this.root = -1;

    return this;
  }

  // Walk state
  var pointer = this.root,
      newPointer,
      leftOrRight,
      opposite,
      ancestors = [],
      path = [],
      ancestor,
      parent,
      child,
      critbit,
      internal,
      best,
      dir,
      i,
      l;

  // Walking the tree
  while (true) {

    // Traversing an internal node
    if (pointer > 0) {
      pointer -= 1;

      // Choosing the correct direction
      dir = getDirection(key, this.critbits[pointer]);

      leftOrRight = dir === 0 ? this.lefts : this.rights;
      newPointer = leftOrRight[pointer];

      if (newPointer === 0) {

        // Creating a fitting external node
        pointer = this.size++;
        leftOrRight[newPointer] = -(pointer + 1);
        this.keys[pointer] = key;
        this.values[pointer] = value;
        return this;
      }

      ancestors.push(pointer);
      path.push(dir);
      pointer = newPointer;
    }

    // Reaching an external node
    else {
      pointer = -pointer;
      pointer -= 1;

      // 1. Creating a new external node
      critbit = findCriticalBit(key, this.keys[pointer]);

      // Key is identical, we just replace the value
      if (critbit === -1) {
        this.values[pointer] = value;
        return this;
      }

      internal = this.offset++;
      newPointer = this.size++;

      this.keys[newPointer] = key;
      this.values[newPointer] = value;

      this.critbits[internal] = critbit;

      dir = getDirection(key, critbit);
      leftOrRight = dir === 0 ? this.lefts : this.rights;
      opposite = dir === 0 ? this.rights : this.lefts;

      leftOrRight[internal] = -(newPointer + 1);
      opposite[internal] = -(pointer + 1);

      // 2. Bubbling up
      best = -1;
      l = ancestors.length;

      for (i = l - 1; i >= 0; i--) {
        ancestor = ancestors[i];

        // TODO: this can be made faster
        if ((this.critbits[ancestor] >> 8) > (critbit >> 8)) {
          continue;
        }
        else if ((this.critbits[ancestor] >> 8) === (critbit >> 8)) {
          if ((this.critbits[ancestor] & 0xff) < (critbit & 0xff))
            continue;
        }

        best = i;
        break;
      }

      // Do we need to attach to the root?
      if (best < 0) {
        this.root = internal + 1;

        // Need to rewire parent as child?
        if (l > 0) {
          parent = ancestors[0];

          opposite[internal] = parent + 1;
        }
      }

      // Simple case without rotation
      else if (best === l - 1) {
        parent = ancestors[best];
        dir = path[best];

        leftOrRight = dir === 0 ? this.lefts : this.rights;

        leftOrRight[parent] = internal + 1;
      }

      // Full rotation
      else {
        parent = ancestors[best];
        dir = path[best];
        child = ancestors[best + 1];

        opposite[internal] = child + 1;

        leftOrRight = dir === 0 ? this.lefts : this.rights;

        leftOrRight[parent] = internal + 1;
      }

      return this;
    }
  }
};

/**
 * Method used to get the value attached to the given key in the tree or
 * undefined if not found.
 *
 * @param  {string} key   - Key to get.
 * @return {any}
 */
FixedCritBitTreeMap.prototype.get = function(key) {

  // Walk state
  var pointer = this.root,
      dir;

  // Walking the tree
  while (true) {

    // Dead end
    if (pointer === 0)
      return;

    // Traversing an internal node
    if (pointer > 0) {
      pointer -= 1;
      dir = getDirection(key, this.critbits[pointer]);

      pointer = dir === 0 ? this.lefts[pointer] : this.rights[pointer];
    }

    // Reaching an external node
    else {
      pointer = -pointer;
      pointer -= 1;

      if (this.keys[pointer] !== key)
        return;

      return this.values[pointer];
    }
  }
};

/**
 * Method used to return whether the given key exists in the tree.
 *
 * @param  {string} key - Key to test.
 * @return {boolean}
 */
FixedCritBitTreeMap.prototype.has = function(key) {

  // Walk state
  var pointer = this.root,
      dir;

  // Walking the tree
  while (true) {

    // Dead end
    if (pointer === 0)
      return false;

    // Traversing an internal node
    if (pointer > 0) {
      pointer -= 1;
      dir = getDirection(key, this.critbits[pointer]);

      pointer = dir === 0 ? this.lefts[pointer] : this.rights[pointer];
    }

    // Reaching an external node
    else {
      pointer = -pointer;
      pointer -= 1;

      return this.keys[pointer] === key;
    }
  }
};

/**
 * Method used to iterate over the tree in key order.
 *
 * @param  {function}  callback - Function to call for each item.
 * @param  {object}    scope    - Optional scope.
 * @return {undefined}
 */
FixedCritBitTreeMap.prototype.forEach = function(callback, scope) {
  scope = arguments.length > 1 ? scope : this;

  // Inorder traversal of the tree
  var current = this.root,
      stack = [],
      p;

  while (true) {

    if (current !== 0) {
      stack.push(current);

      current = current > 0 ? this.lefts[current - 1] : 0;
    }

    else {
      if (stack.length > 0) {
        current = stack.pop();

        if (current < 0) {
          p = -current;
          p -= 1;

          callback.call(scope, this.values[p], this.keys[p]);
        }

        current = current > 0 ? this.rights[current - 1] : 0;
      }
      else {
        break;
      }
    }
  }
};

/**
 * Convenience known methods.
 */
FixedCritBitTreeMap.prototype.inspect = function() {
  return this;
};

if (typeof Symbol !== 'undefined')
  FixedCritBitTreeMap.prototype[Symbol.for('nodejs.util.inspect.custom')] = FixedCritBitTreeMap.prototype.inspect;

/**
 * Static @.from function taking an arbitrary iterable & converting it into
 * a FixedCritBitTreeMap.
 *
 * @param  {Iterable} iterable - Target iterable.
 * @return {FixedCritBitTreeMap}
 */
// FixedCritBitTreeMap.from = function(iterable) {

// };

/**
 * Exporting.
 */
module.exports = FixedCritBitTreeMap;
