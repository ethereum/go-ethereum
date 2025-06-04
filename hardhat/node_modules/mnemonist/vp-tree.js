/**
 * Mnemonist Vantage Point Tree
 * =============================
 *
 * JavaScript implementation of the Vantage Point Tree storing the binary
 * tree as a flat byte array.
 *
 * Note that a VPTree has worst cases and is likely not to be perfectly
 * balanced because of median ambiguity. It is therefore not suitable
 * for hairballs and tiny datasets.
 *
 * [Reference]:
 * https://en.wikipedia.org/wiki/Vantage-point_tree
 */
var iterables = require('./utils/iterables.js'),
    typed = require('./utils/typed-arrays.js'),
    inplaceQuickSortIndices = require('./sort/quick.js').inplaceQuickSortIndices,
    lowerBoundIndices = require('./utils/binary-search.js').lowerBoundIndices,
    Heap = require('./heap.js');

var getPointerArray = typed.getPointerArray;

// TODO: implement vantage point selection techniques (by swapping with last)
// TODO: is this required to implement early termination for k <= size?

/**
 * Heap comparator used by the #.nearestNeighbors method.
 */
function comparator(a, b) {
  if (a.distance < b.distance)
    return 1;

  if (a.distance > b.distance)
    return -1;

  return 0;
}

/**
 * Function used to create the binary tree.
 *
 * @param  {function}     distance - Distance function to use.
 * @param  {array}        items    - Items to index (will be mutated).
 * @param  {array}        indices  - Indexes of the items.
 * @return {Float64Array}          - The flat binary tree.
 */
function createBinaryTree(distance, items, indices) {
  var N = indices.length;

  var PointerArray = getPointerArray(N);

  var C = 0,
      nodes = new PointerArray(N),
      lefts = new PointerArray(N),
      rights = new PointerArray(N),
      mus = new Float64Array(N),
      stack = [0, 0, N],
      distances = new Float64Array(N),
      nodeIndex,
      vantagePoint,
      medianIndex,
      lo,
      hi,
      mid,
      mu,
      i,
      l;

  while (stack.length) {
    hi = stack.pop();
    lo = stack.pop();
    nodeIndex = stack.pop();

    // Getting our vantage point
    vantagePoint = indices[hi - 1];
    hi--;

    l = hi - lo;

    // Storing vantage point
    nodes[nodeIndex] = vantagePoint;

    // We are in a leaf
    if (l === 0)
      continue;

    // We only have two elements, the second one has to go right
    if (l === 1) {

      // We put remaining item to the right
      mu = distance(items[vantagePoint], items[indices[lo]]);

      mus[nodeIndex] = mu;

      // Right
      C++;
      rights[nodeIndex] = C;
      nodes[C] = indices[lo];

      continue;
    }

    // Computing distance from vantage point to other points
    for (i = lo; i < hi; i++)
      distances[indices[i]] = distance(items[vantagePoint], items[indices[i]]);

    inplaceQuickSortIndices(distances, indices, lo, hi);

    // Finding median of distances
    medianIndex = lo + (l / 2) - 1;

    // Need to interpolate?
    if (medianIndex === (medianIndex | 0)) {
      mu = (
        distances[indices[medianIndex]] +
        distances[indices[medianIndex + 1]]
      ) / 2;
    }
    else {
      mu = distances[indices[Math.ceil(medianIndex)]];
    }

    // Storing mu
    mus[nodeIndex] = mu;

    mid = lowerBoundIndices(distances, indices, mu, lo, hi);

    // console.log('Vantage point', items[vantagePoint], vantagePoint);
    // console.log('mu =', mu);
    // console.log('lo =', lo);
    // console.log('hi =', hi);
    // console.log('mid =', mid);

    // console.log('need to split', Array.from(indices).slice(lo, hi).map(i => {
    //   return [distances[i], distance(items[vantagePoint], items[i]), items[i]];
    // }));

    // Right
    if (hi - mid > 0) {
      C++;
      rights[nodeIndex] = C;
      stack.push(C, mid, hi);
      // console.log('Went right with ', Array.from(indices).slice(mid, hi).map(i => {
      //   return [distances[i], distance(items[vantagePoint], items[i]), items[i]];
      // }));
    }

    // Left
    if (mid - lo > 0) {
      C++;
      lefts[nodeIndex] = C;
      stack.push(C, lo, mid);
      // console.log('Went left with', Array.from(indices).slice(lo, mid).map(i => {
      //   return [distances[i], distance(items[vantagePoint], items[i]), items[i]];
      // }));
    }

    // console.log();
  }

  return {
    nodes: nodes,
    lefts: lefts,
    rights: rights,
    mus: mus
  };
}

/**
 * VPTree.
 *
 * @constructor
 * @param {function} distance - Distance function to use.
 * @param {Iterable} items    - Items to store.
 */
function VPTree(distance, items) {
  if (typeof distance !== 'function')
    throw new Error('mnemonist/VPTree.constructor: given `distance` must be a function.');

  if (!items)
    throw new Error('mnemonist/VPTree.constructor: you must provide items to the tree. A VPTree cannot be updated after its creation.');

  // Properties
  this.distance = distance;
  this.heap = new Heap(comparator);
  this.D = 0;

  var arrays = iterables.toArrayWithIndices(items);
  this.items = arrays[0];
  var indices = arrays[1];

  // Creating the binary tree
  this.size = indices.length;

  var result = createBinaryTree(distance, this.items, indices);

  this.nodes = result.nodes;
  this.lefts = result.lefts;
  this.rights = result.rights;
  this.mus = result.mus;
}

/**
 * Function used to retrieve the k nearest neighbors of the query.
 *
 * @param  {number} k     - Number of neighbors to retrieve.
 * @param  {any}    query - The query.
 * @return {array}
 */
VPTree.prototype.nearestNeighbors = function(k, query) {
  var neighbors = this.heap,
      stack = [0],
      tau = Infinity,
      nodeIndex,
      itemIndex,
      vantagePoint,
      leftIndex,
      rightIndex,
      mu,
      d;

  this.D = 0;

  while (stack.length) {
    nodeIndex = stack.pop();
    itemIndex = this.nodes[nodeIndex];
    vantagePoint = this.items[itemIndex];

    // Distance between query & the current vantage point
    d = this.distance(vantagePoint, query);
    this.D++;

    if (d < tau) {
      neighbors.push({distance: d, item: vantagePoint});

      // Trimming
      if (neighbors.size > k)
        neighbors.pop();

      // Adjusting tau (only if we already have k items, else it stays Infinity)
      if (neighbors.size >= k)
       tau = neighbors.peek().distance;
    }

    leftIndex = this.lefts[nodeIndex];
    rightIndex = this.rights[nodeIndex];

    // We are a leaf
    if (!leftIndex && !rightIndex)
      continue;

    mu = this.mus[nodeIndex];

    if (d < mu) {
      if (leftIndex && d < mu + tau)
        stack.push(leftIndex);
      if (rightIndex && d >= mu - tau) // Might not be necessary to test d
        stack.push(rightIndex);
    }
    else {
      if (rightIndex && d >= mu - tau)
        stack.push(rightIndex);
      if (leftIndex && d < mu + tau) // Might not be necessary to test d
        stack.push(leftIndex);
    }
  }

  var array = new Array(neighbors.size);

  for (var i = neighbors.size - 1; i >= 0; i--)
    array[i] = neighbors.pop();

  return array;
};

/**
 * Function used to retrieve every neighbors of query in the given radius.
 *
 * @param  {number} radius - Radius.
 * @param  {any}    query  - The query.
 * @return {array}
 */
VPTree.prototype.neighbors = function(radius, query) {
  var neighbors = [],
      stack = [0],
      nodeIndex,
      itemIndex,
      vantagePoint,
      leftIndex,
      rightIndex,
      mu,
      d;

  this.D = 0;

  while (stack.length) {
    nodeIndex = stack.pop();
    itemIndex = this.nodes[nodeIndex];
    vantagePoint = this.items[itemIndex];

    // Distance between query & the current vantage point
    d = this.distance(vantagePoint, query);
    this.D++;

    if (d <= radius)
      neighbors.push({distance: d, item: vantagePoint});

    leftIndex = this.lefts[nodeIndex];
    rightIndex = this.rights[nodeIndex];

    // We are a leaf
    if (!leftIndex && !rightIndex)
      continue;

    mu = this.mus[nodeIndex];

    if (d < mu) {
      if (leftIndex && d < mu + radius)
        stack.push(leftIndex);
      if (rightIndex && d >= mu - radius) // Might not be necessary to test d
        stack.push(rightIndex);
    }
    else {
      if (rightIndex && d >= mu - radius)
        stack.push(rightIndex);
      if (leftIndex && d < mu + radius) // Might not be necessary to test d
        stack.push(leftIndex);
    }
  }

  return neighbors;
};

/**
 * Convenience known methods.
 */
VPTree.prototype.inspect = function() {
  var array = this.items.slice();

  // Trick so that node displays the name of the constructor
  Object.defineProperty(array, 'constructor', {
    value: VPTree,
    enumerable: false
  });

  return array;
};

if (typeof Symbol !== 'undefined')
  VPTree.prototype[Symbol.for('nodejs.util.inspect.custom')] = VPTree.prototype.inspect;

/**
 * Static @.from function taking an arbitrary iterable & converting it into
 * a tree.
 *
 * @param  {Iterable} iterable - Target iterable.
 * @param  {function} distance - Distance function to use.
 * @return {VPTree}
 */
VPTree.from = function(iterable, distance) {
  return new VPTree(distance, iterable);
};

/**
 * Exporting.
 */
module.exports = VPTree;
