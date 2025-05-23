/*
 * Mnemonist StaticIntervalTree
 * =============================
 *
 * JavaScript implementation of a static interval tree. This tree is static in
 * that you are required to know all its items beforehand and to built it
 * from an iterable.
 *
 * This implementation represents the interval tree as an augmented balanced
 * binary search tree. It works by sorting the intervals by startpoint first
 * then proceeds building the augmented balanced BST bottom-up from the
 * sorted list.
 *
 * Note that this implementation considers every given intervals as closed for
 * simplicity's sake.
 *
 * For more information: https://en.wikipedia.org/wiki/Interval_tree
 */
var iterables = require('./utils/iterables.js'),
    typed = require('./utils/typed-arrays.js');

var FixedStack = require('./fixed-stack.js');


// TODO: pass index to getters
// TODO: custom comparison
// TODO: possibility to pass offset buffer

// TODO: intervals() => Symbol.iterator
// TODO: dfs()

/**
 * Helpers.
 */

/**
 * Recursive function building the BST from the sorted list of interval
 * indices.
 *
 * @param  {array}    intervals     - Array of intervals to index.
 * @param  {function} endGetter     - Getter function for end of intervals.
 * @param  {array}    sortedIndices - Sorted indices of the intervals.
 * @param  {array}    tree          - BST memory.
 * @param  {array}    augmentations - Array of node augmentations.
 * @param  {number}   i             - BST index of current node.
 * @param  {number}   low           - Dichotomy low index.
 * @param  {number}   high          - Dichotomy high index.
 * @return {number}                 - Created node augmentation value.
 */
function buildBST(
  intervals,
  endGetter,
  sortedIndices,
  tree,
  augmentations,
  i,
  low,
  high
) {
  var mid = (low + (high - low) / 2) | 0,
      midMinusOne = ~-mid,
      midPlusOne = -~mid;

  var current = sortedIndices[mid];
  tree[i] = current + 1;

  var end = endGetter ? endGetter(intervals[current]) : intervals[current][1];

  var left = i * 2 + 1,
      right = i * 2 + 2;

  var leftEnd = -Infinity,
      rightEnd = -Infinity;

  if (low <= midMinusOne) {
    leftEnd = buildBST(
      intervals,
      endGetter,
      sortedIndices,
      tree,
      augmentations,
      left,
      low,
      midMinusOne
    );
  }

  if (midPlusOne <= high) {
    rightEnd = buildBST(
      intervals,
      endGetter,
      sortedIndices,
      tree,
      augmentations,
      right,
      midPlusOne,
      high
    );
  }

  var augmentation = Math.max(end, leftEnd, rightEnd);

  var augmentationPointer = current;

  if (augmentation === leftEnd)
    augmentationPointer = augmentations[tree[left] - 1];
  else if (augmentation === rightEnd)
    augmentationPointer = augmentations[tree[right] - 1];

  augmentations[current] = augmentationPointer;

  return augmentation;
}

/**
 * StaticIntervalTree.
 *
 * @constructor
 * @param {array}           intervals - Array of intervals to index.
 * @param {array<function>} getters   - Optional getters.
 */
function StaticIntervalTree(intervals, getters) {

  // Properties
  this.size = intervals.length;
  this.intervals = intervals;

  var startGetter = null,
      endGetter = null;

  if (Array.isArray(getters)) {
    startGetter = getters[0];
    endGetter = getters[1];
  }

  // Building the indices array
  var length = intervals.length;

  var IndicesArray = typed.getPointerArray(length + 1);

  var indices = new IndicesArray(length);

  var i;

  for (i = 1; i < length; i++)
    indices[i] = i;

  // Sorting indices array
  // TODO: check if some version of radix sort can outperform this part
  indices.sort(function(a, b) {
    a = intervals[a];
    b = intervals[b];

    if (startGetter) {
      a = startGetter(a);
      b = startGetter(b);
    }
    else {
      a = a[0];
      b = b[0];
    }

    if (a < b)
      return -1;

    if (a > b)
      return 1;

    // TODO: use getters
    // TODO: this ordering has the following invariant: if query interval
    // contains [nodeStart, max], then whole right subtree can be collected
    // a = a[1];
    // b = b[1];

    // if (a < b)
    //   return 1;

    // if (a > b)
    //   return -1;

    return 0;
  });

  // Building the binary tree
  var height = Math.ceil(Math.log2(length + 1)),
      treeSize = Math.pow(2, height) - 1;

  var tree = new IndicesArray(treeSize);

  var augmentations = new IndicesArray(length);

  buildBST(
    intervals,
    endGetter,
    indices,
    tree,
    augmentations,
    0,
    0,
    length - 1
  );

  // Dropping indices
  indices = null;

  // Storing necessary information
  this.height = height;
  this.tree = tree;
  this.augmentations = augmentations;
  this.startGetter = startGetter;
  this.endGetter = endGetter;

  // Initializing DFS stack
  this.stack = new FixedStack(IndicesArray, this.height);
}

/**
 * Method returning a list of intervals containing the given point.
 *
 * @param  {any}   point - Target point.
 * @return {array}
 */
StaticIntervalTree.prototype.intervalsContainingPoint = function(point) {
  var matches = [];

  var stack = this.stack;

  stack.clear();
  stack.push(0);

  var l = this.tree.length;

  var bstIndex,
      intervalIndex,
      interval,
      maxInterval,
      start,
      end,
      max,
      left,
      right;

  while (stack.size) {
    bstIndex = stack.pop();
    intervalIndex = this.tree[bstIndex] - 1;
    interval = this.intervals[intervalIndex];
    maxInterval = this.intervals[this.augmentations[intervalIndex]];

    max = this.endGetter ? this.endGetter(maxInterval) : maxInterval[1];

    // No possible match, point is farther right than the max end value
    if (point > max)
      continue;

    // Searching left
    left = bstIndex * 2 + 1;

    if (left < l && this.tree[left] !== 0)
      stack.push(left);

    start = this.startGetter ? this.startGetter(interval) : interval[0];
    end = this.endGetter ? this.endGetter(interval) : interval[1];

    // Checking current node
    if (point >= start && point <= end)
      matches.push(interval);

    // If the point is to the left of the start of the current interval,
    // then it cannot be in the right child
    if (point < start)
      continue;

    // Searching right
    right = bstIndex * 2 + 2;

    if (right < l && this.tree[right] !== 0)
      stack.push(right);
  }

  return matches;
};

/**
 * Method returning a list of intervals overlapping the given interval.
 *
 * @param  {any}   interval - Target interval.
 * @return {array}
 */
StaticIntervalTree.prototype.intervalsOverlappingInterval = function(interval) {
  var intervalStart = this.startGetter ? this.startGetter(interval) : interval[0],
      intervalEnd = this.endGetter ? this.endGetter(interval) : interval[1];

  var matches = [];

  var stack = this.stack;

  stack.clear();
  stack.push(0);

  var l = this.tree.length;

  var bstIndex,
      intervalIndex,
      currentInterval,
      maxInterval,
      start,
      end,
      max,
      left,
      right;

  while (stack.size) {
    bstIndex = stack.pop();
    intervalIndex = this.tree[bstIndex] - 1;
    currentInterval = this.intervals[intervalIndex];
    maxInterval = this.intervals[this.augmentations[intervalIndex]];

    max = this.endGetter ? this.endGetter(maxInterval) : maxInterval[1];

    // No possible match, start is farther right than the max end value
    if (intervalStart > max)
      continue;

    // Searching left
    left = bstIndex * 2 + 1;

    if (left < l && this.tree[left] !== 0)
      stack.push(left);

    start = this.startGetter ? this.startGetter(currentInterval) : currentInterval[0];
    end = this.endGetter ? this.endGetter(currentInterval) : currentInterval[1];

    // Checking current node
    if (intervalEnd >= start && intervalStart <= end)
      matches.push(currentInterval);

    // If the end is to the left of the start of the current interval,
    // then it cannot be in the right child
    if (intervalEnd < start)
      continue;

    // Searching right
    right = bstIndex * 2 + 2;

    if (right < l && this.tree[right] !== 0)
      stack.push(right);
  }

  return matches;
};

/**
 * Convenience known methods.
 */
StaticIntervalTree.prototype.inspect = function() {
  var proxy = this.intervals.slice();

  // Trick so that node displays the name of the constructor
  Object.defineProperty(proxy, 'constructor', {
    value: StaticIntervalTree,
    enumerable: false
  });

  return proxy;
};

if (typeof Symbol !== 'undefined')
  StaticIntervalTree.prototype[Symbol.for('nodejs.util.inspect.custom')] = StaticIntervalTree.prototype.inspect;

/**
 * Static @.from function taking an arbitrary iterable & converting it into
 * a structure.
 *
 * @param  {Iterable} iterable - Target iterable.
 * @return {StaticIntervalTree}
 */
StaticIntervalTree.from = function(iterable, getters) {
  if (iterables.isArrayLike(iterable))
    return new StaticIntervalTree(iterable, getters);

  return new StaticIntervalTree(Array.from(iterable), getters);
};

/**
 * Exporting.
 */
module.exports = StaticIntervalTree;
