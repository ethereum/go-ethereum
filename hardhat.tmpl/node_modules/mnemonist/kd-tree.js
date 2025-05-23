/**
 * Mnemonist KDTree
 * =================
 *
 * Low-level JavaScript implementation of a k-dimensional tree.
 */
var iterables = require('./utils/iterables.js');
var typed = require('./utils/typed-arrays.js');
var createTupleComparator = require('./utils/comparators.js').createTupleComparator;
var FixedReverseHeap = require('./fixed-reverse-heap.js');
var inplaceQuickSortIndices = require('./sort/quick.js').inplaceQuickSortIndices;

/**
 * Helper function used to compute the squared distance between a query point
 * and an indexed points whose values are stored in a tree's axes.
 *
 * Note that squared distance is used instead of euclidean to avoid
 * costly sqrt computations.
 *
 * @param  {number} dimensions - Number of dimensions.
 * @param  {array}  axes       - Axes data.
 * @param  {number} pivot      - Pivot.
 * @param  {array}  point      - Query point.
 * @return {number}
 */
function squaredDistanceAxes(dimensions, axes, pivot, b) {
  var d;

  var dist = 0,
      step;

  for (d = 0; d < dimensions; d++) {
    step = axes[d][pivot] - b[d];
    dist += step * step;
  }

  return dist;
}

/**
 * Helper function used to reshape input data into low-level axes data.
 *
 * @param  {number} dimensions - Number of dimensions.
 * @param  {array}  data       - Data in the shape [label, [x, y, z...]]
 * @return {object}
 */
function reshapeIntoAxes(dimensions, data) {
  var l = data.length;

  var axes = new Array(dimensions),
      labels = new Array(l),
      axis;

  var PointerArray = typed.getPointerArray(l);

  var ids = new PointerArray(l);

  var d, i, row;

  var f = true;

  for (d = 0; d < dimensions; d++) {
    axis = new Float64Array(l);

    for (i = 0; i < l; i++) {
      row = data[i];
      axis[i] = row[1][d];

      if (f) {
        labels[i] = row[0];
        ids[i] = i;
      }
    }

    f = false;
    axes[d] = axis;
  }

  return {axes: axes, ids: ids, labels: labels};
}

/**
 * Helper function used to build a kd-tree from axes data.
 *
 * @param  {number} dimensions - Number of dimensions.
 * @param  {array}  axes       - Axes.
 * @param  {array}  ids        - Indices to sort.
 * @param  {array}  labels     - Point labels.
 * @return {object}
 */
function buildTree(dimensions, axes, ids, labels) {
  var l = labels.length;

  // NOTE: +1 because we need to keep 0 as null pointer
  var PointerArray = typed.getPointerArray(l + 1);

  // Building the tree
  var pivots = new PointerArray(l),
      lefts = new PointerArray(l),
      rights = new PointerArray(l);

  var stack = [[0, 0, ids.length, -1, 0]],
      step,
      parent,
      direction,
      median,
      pivot,
      lo,
      hi;

  var d, i = 0;

  while (stack.length !== 0) {
    step = stack.pop();

    d = step[0];
    lo = step[1];
    hi = step[2];
    parent = step[3];
    direction = step[4];

    inplaceQuickSortIndices(axes[d], ids, lo, hi);

    l = hi - lo;
    median = lo + (l >>> 1); // Fancy floor(l / 2)
    pivot = ids[median];
    pivots[i] = pivot;

    if (parent > -1) {
      if (direction === 0)
        lefts[parent] = i + 1;
      else
        rights[parent] = i + 1;
    }

    d = (d + 1) % dimensions;

    // Right
    if (median !== lo && median !== hi - 1) {
      stack.push([d, median + 1, hi, i, 1]);
    }

    // Left
    if (median !== lo) {
      stack.push([d, lo, median, i, 0]);
    }

    i++;
  }

  return {
    axes: axes,
    labels: labels,
    pivots: pivots,
    lefts: lefts,
    rights: rights
  };
}

/**
 * KDTree.
 *
 * @constructor
 */
function KDTree(dimensions, build) {
  this.dimensions = dimensions;
  this.visited = 0;

  this.axes = build.axes;
  this.labels = build.labels;

  this.pivots = build.pivots;
  this.lefts = build.lefts;
  this.rights = build.rights;

  this.size = this.labels.length;
}

/**
 * Method returning the query's nearest neighbor.
 *
 * @param  {array}  query - Query point.
 * @return {any}
 */
KDTree.prototype.nearestNeighbor = function(query) {
  var bestDistance = Infinity,
      best = null;

  var dimensions = this.dimensions,
      axes = this.axes,
      pivots = this.pivots,
      lefts = this.lefts,
      rights = this.rights;

  var visited = 0;

  function recurse(d, node) {
    visited++;

    var left = lefts[node],
        right = rights[node],
        pivot = pivots[node];

    var dist = squaredDistanceAxes(
      dimensions,
      axes,
      pivot,
      query
    );

    if (dist < bestDistance) {
      best = pivot;
      bestDistance = dist;

      if (dist === 0)
        return;
    }

    var dx = axes[d][pivot] - query[d];

    d = (d + 1) % dimensions;

    // Going the correct way?
    if (dx > 0) {
      if (left !== 0)
        recurse(d, left - 1);
    }
    else {
      if (right !== 0)
        recurse(d, right - 1);
    }

    // Going the other way?
    if (dx * dx < bestDistance) {
      if (dx > 0) {
        if (right !== 0)
          recurse(d, right - 1);
      }
      else {
        if (left !== 0)
          recurse(d, left - 1);
      }
    }
  }

  recurse(0, 0);

  this.visited = visited;
  return this.labels[best];
};

var KNN_HEAP_COMPARATOR_3 = createTupleComparator(3);
var KNN_HEAP_COMPARATOR_2 = createTupleComparator(2);

/**
 * Method returning the query's k nearest neighbors.
 *
 * @param  {number} k     - Number of nearest neighbor to retrieve.
 * @param  {array}  query - Query point.
 * @return {array}
 */

// TODO: can do better by improving upon static-kdtree here
KDTree.prototype.kNearestNeighbors = function(k, query) {
  if (k <= 0)
    throw new Error('mnemonist/kd-tree.kNearestNeighbors: k should be a positive number.');

  k = Math.min(k, this.size);

  if (k === 1)
    return [this.nearestNeighbor(query)];

  var heap = new FixedReverseHeap(Array, KNN_HEAP_COMPARATOR_3, k);

  var dimensions = this.dimensions,
      axes = this.axes,
      pivots = this.pivots,
      lefts = this.lefts,
      rights = this.rights;

  var visited = 0;

  function recurse(d, node) {
    var left = lefts[node],
        right = rights[node],
        pivot = pivots[node];

    var dist = squaredDistanceAxes(
      dimensions,
      axes,
      pivot,
      query
    );

    heap.push([dist, visited++, pivot]);

    var point = query[d],
        split = axes[d][pivot],
        dx = point - split;

    d = (d + 1) % dimensions;

    // Going the correct way?
    if (point < split) {
      if (left !== 0) {
        recurse(d, left - 1);
      }
    }
    else {
      if (right !== 0) {
        recurse(d, right - 1);
      }
    }

    // Going the other way?
    if (dx * dx < heap.peek()[0] || heap.size < k) {
      if (point < split) {
        if (right !== 0) {
          recurse(d, right - 1);
        }
      }
      else {
        if (left !== 0) {
          recurse(d, left - 1);
        }
      }
    }
  }

  recurse(0, 0);

  this.visited = visited;

  var best = heap.consume();

  for (var i = 0; i < best.length; i++)
    best[i] = this.labels[best[i][2]];

  return best;
};

/**
 * Method returning the query's k nearest neighbors by linear search.
 *
 * @param  {number} k     - Number of nearest neighbor to retrieve.
 * @param  {array}  query - Query point.
 * @return {array}
 */
KDTree.prototype.linearKNearestNeighbors = function(k, query) {
  if (k <= 0)
    throw new Error('mnemonist/kd-tree.kNearestNeighbors: k should be a positive number.');

  k = Math.min(k, this.size);

  var heap = new FixedReverseHeap(Array, KNN_HEAP_COMPARATOR_2, k);

  var i, l, dist;

  for (i = 0, l = this.size; i < l; i++) {
    dist = squaredDistanceAxes(
      this.dimensions,
      this.axes,
      this.pivots[i],
      query
    );

    heap.push([dist, i]);
  }

  var best = heap.consume();

  for (i = 0; i < best.length; i++)
    best[i] = this.labels[this.pivots[best[i][1]]];

  return best;
};

/**
 * Convenience known methods.
 */
KDTree.prototype.inspect = function() {
  var dummy = new Map();

  dummy.dimensions = this.dimensions;

  Object.defineProperty(dummy, 'constructor', {
    value: KDTree,
    enumerable: false
  });

  var i, j, point;

  for (i = 0; i < this.size; i++) {
    point = new Array(this.dimensions);

    for (j = 0; j < this.dimensions; j++)
      point[j] = this.axes[j][i];

    dummy.set(this.labels[i], point);
  }

  return dummy;
};

if (typeof Symbol !== 'undefined')
  KDTree.prototype[Symbol.for('nodejs.util.inspect.custom')] = KDTree.prototype.inspect;

/**
 * Static @.from function taking an arbitrary iterable & converting it into
 * a structure.
 *
 * @param  {Iterable} iterable   - Target iterable.
 * @param  {number}   dimensions - Space dimensions.
 * @return {KDTree}
 */
KDTree.from = function(iterable, dimensions) {
  var data = iterables.toArray(iterable);

  var reshaped = reshapeIntoAxes(dimensions, data);

  var result = buildTree(dimensions, reshaped.axes, reshaped.ids, reshaped.labels);

  return new KDTree(dimensions, result);
};

/**
 * Static @.from function building a KDTree from given axes.
 *
 * @param  {Iterable} iterable   - Target iterable.
 * @param  {number}   dimensions - Space dimensions.
 * @return {KDTree}
 */
KDTree.fromAxes = function(axes, labels) {
  if (!labels)
    labels = typed.indices(axes[0].length);

  var dimensions = axes.length;

  var result = buildTree(axes.length, axes, typed.indices(labels.length), labels);

  return new KDTree(dimensions, result);
};

/**
 * Exporting.
 */
module.exports = KDTree;
