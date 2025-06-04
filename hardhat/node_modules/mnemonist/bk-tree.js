/* eslint no-constant-condition: 0 */
/**
 * Mnemonist BK Tree
 * ==================
 *
 * Implementation of a Burkhard-Keller tree, allowing fast lookups of words
 * that lie within a specified distance of the query word.
 *
 * [Reference]:
 * https://en.wikipedia.org/wiki/BK-tree
 *
 * [Article]:
 * W. Burkhard and R. Keller. Some approaches to best-match file searching,
 * CACM, 1973
 */
var forEach = require('obliterator/foreach');

/**
 * BK Tree.
 *
 * @constructor
 * @param {function} distance - Distance function to use.
 */
function BKTree(distance) {

  if (typeof distance !== 'function')
    throw new Error('mnemonist/BKTree.constructor: given `distance` should be a function.');

  this.distance = distance;
  this.clear();
}

/**
 * Method used to add an item to the tree.
 *
 * @param  {any} item - Item to add.
 * @return {BKTree}
 */
BKTree.prototype.add = function(item) {

  // Initializing the tree with the first given word
  if (!this.root) {
    this.root = {
      item: item,
      children: {}
    };

    this.size++;
    return this;
  }

  var node = this.root,
      d;

  while (true) {
    d = this.distance(item, node.item);

    if (!node.children[d])
      break;

    node = node.children[d];
  }

  node.children[d] = {
    item: item,
    children: {}
  };

  this.size++;
  return this;
};

/**
 * Method used to query the tree.
 *
 * @param  {number} n     - Maximum distance between query & item.
 * @param  {any}    query - Query
 * @return {BKTree}
 */
BKTree.prototype.search = function(n, query) {
  if (!this.root)
    return [];

  var found = [],
      stack = [this.root],
      node,
      child,
      d,
      i,
      l;

  while (stack.length) {
    node = stack.pop();
    d = this.distance(query, node.item);

    if (d <= n)
      found.push({item: node.item, distance: d});

    for (i = d - n, l = d + n + 1; i < l; i++) {
      child = node.children[i];

      if (child)
        stack.push(child);
    }
  }

  return found;
};

/**
 * Method used to clear the tree.
 *
 * @return {undefined}
 */
BKTree.prototype.clear = function() {

  // Properties
  this.size = 0;
  this.root = null;
};

/**
 * Convenience known methods.
 */
BKTree.prototype.toJSON = function() {
  return this.root;
};

BKTree.prototype.inspect = function() {
  var array = [],
      stack = [this.root],
      node,
      d;

  while (stack.length) {
    node = stack.pop();

    if (!node)
      continue;

    array.push(node.item);

    for (d in node.children)
      stack.push(node.children[d]);
  }

  // Trick so that node displays the name of the constructor
  Object.defineProperty(array, 'constructor', {
    value: BKTree,
    enumerable: false
  });

  return array;
};

if (typeof Symbol !== 'undefined')
  BKTree.prototype[Symbol.for('nodejs.util.inspect.custom')] = BKTree.prototype.inspect;

/**
 * Static @.from function taking an arbitrary iterable & converting it into
 * a tree.
 *
 * @param  {Iterable} iterable - Target iterable.
 * @param  {function} distance - Distance function.
 * @return {Heap}
 */
BKTree.from = function(iterable, distance) {
  var tree = new BKTree(distance);

  forEach(iterable, function(value) {
    tree.add(value);
  });

  return tree;
};

/**
 * Exporting.
 */
module.exports = BKTree;
