/* eslint no-constant-condition: 0 */
/**
 * Mnemonist StaticDisjointSet
 * ============================
 *
 * JavaScript implementation of a static disjoint set (union-find).
 *
 * Note that to remain performant, this implementation needs to know a size
 * beforehand.
 */
var helpers = require('./utils/typed-arrays.js');

/**
 * StaticDisjointSet.
 *
 * @constructor
 */
function StaticDisjointSet(size) {

  // Optimizing the typed array types
  var ParentsTypedArray = helpers.getPointerArray(size),
      RanksTypedArray = helpers.getPointerArray(Math.log2(size));

  // Properties
  this.size = size;
  this.dimension = size;
  this.parents = new ParentsTypedArray(size);
  this.ranks = new RanksTypedArray(size);

  // Initializing parents
  for (var i = 0; i < size; i++)
    this.parents[i] = i;
}

/**
 * Method used to find the root of the given item.
 *
 * @param  {number} x - Target item.
 * @return {number}
 */
StaticDisjointSet.prototype.find = function(x) {
  var y = x;

  var c, p;

  while (true) {
    c = this.parents[y];

    if (y === c)
      break;

    y = c;
  }

  // Path compression
  while (true) {
    p = this.parents[x];

    if (p === y)
      break;

    this.parents[x] = y;
    x = p;
  }

  return y;
};

/**
 * Method used to perform the union of two items.
 *
 * @param  {number} x - First item.
 * @param  {number} y - Second item.
 * @return {StaticDisjointSet}
 */
StaticDisjointSet.prototype.union = function(x, y) {
  var xRoot = this.find(x),
      yRoot = this.find(y);

  // x and y are already in the same set
  if (xRoot === yRoot)
    return this;

  this.dimension--;

  // x and y are not in the same set, we merge them
  var xRank = this.ranks[x],
      yRank = this.ranks[y];

  if (xRank < yRank) {
    this.parents[xRoot] = yRoot;
  }
  else if (xRank > yRank) {
    this.parents[yRoot] = xRoot;
  }
  else {
    this.parents[yRoot] = xRoot;
    this.ranks[xRoot]++;
  }

  return this;
};

/**
 * Method returning whether two items are connected.
 *
 * @param  {number} x - First item.
 * @param  {number} y - Second item.
 * @return {boolean}
 */
StaticDisjointSet.prototype.connected = function(x, y) {
  var xRoot = this.find(x);

  return xRoot === this.find(y);
};

/**
 * Method returning the set mapping.
 *
 * @return {TypedArray}
 */
StaticDisjointSet.prototype.mapping = function() {
  var MappingClass = helpers.getPointerArray(this.dimension);

  var ids = {},
      mapping = new MappingClass(this.size),
      c = 0;

  var r;

  for (var i = 0, l = this.parents.length; i < l; i++) {
    r = this.find(i);

    if (typeof ids[r] === 'undefined') {
      mapping[i] = c;
      ids[r] = c++;
    }
    else {
      mapping[i] = ids[r];
    }
  }

  return mapping;
};

/**
 * Method used to compile the disjoint set into an array of arrays.
 *
 * @return {array}
 */
StaticDisjointSet.prototype.compile = function() {
  var ids = {},
      result = new Array(this.dimension),
      c = 0;

  var r;

  for (var i = 0, l = this.parents.length; i < l; i++) {
    r = this.find(i);

    if (typeof ids[r] === 'undefined') {
      result[c] = [i];
      ids[r] = c++;
    }
    else {
      result[ids[r]].push(i);
    }
  }

  return result;
};

/**
 * Convenience known methods.
 */
StaticDisjointSet.prototype.inspect = function() {
  var array = this.compile();

  // Trick so that node displays the name of the constructor
  Object.defineProperty(array, 'constructor', {
    value: StaticDisjointSet,
    enumerable: false
  });

  return array;
};

if (typeof Symbol !== 'undefined')
  StaticDisjointSet.prototype[Symbol.for('nodejs.util.inspect.custom')] = StaticDisjointSet.prototype.inspect;


/**
 * Exporting.
 */
module.exports = StaticDisjointSet;
