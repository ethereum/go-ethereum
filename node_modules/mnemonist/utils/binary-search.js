/**
 * Mnemonist Binary Search Helpers
 * ================================
 *
 * Typical binary search functions.
 */

/**
 * Function returning the index of the search value in the array or `-1` if
 * not found.
 *
 * @param  {array} array - Haystack.
 * @param  {any}   value - Needle.
 * @return {number}
 */
exports.search = function(array, value, lo, hi) {
  var mid = 0;

  lo = typeof lo !== 'undefined' ? lo : 0;
  hi = typeof hi !== 'undefined' ? hi : array.length;

  hi--;

  var current;

  while (lo <= hi) {
    mid = (lo + hi) >>> 1;

    current = array[mid];

    if (current > value) {
      hi = ~-mid;
    }
    else if (current < value) {
      lo = -~mid;
    }
    else {
      return mid;
    }
  }

  return -1;
};

/**
 * Same as above, but can use a custom comparator function.
 *
 * @param  {function} comparator - Custom comparator function.
 * @param  {array}    array      - Haystack.
 * @param  {any}      value      - Needle.
 * @return {number}
 */
exports.searchWithComparator = function(comparator, array, value) {
  var mid = 0,
      lo = 0,
      hi = ~-array.length,
      comparison;

  while (lo <= hi) {
    mid = (lo + hi) >>> 1;

    comparison = comparator(array[mid], value);

    if (comparison > 0) {
      hi = ~-mid;
    }
    else if (comparison < 0) {
      lo = -~mid;
    }
    else {
      return mid;
    }
  }

  return -1;
};

/**
 * Function returning the lower bound of the given value in the array.
 *
 * @param  {array}  array - Haystack.
 * @param  {any}    value - Needle.
 * @param  {number} [lo] - Start index.
 * @param  {numner} [hi] - End index.
 * @return {number}
 */
exports.lowerBound = function(array, value, lo, hi) {
  var mid = 0;

  lo = typeof lo !== 'undefined' ? lo : 0;
  hi = typeof hi !== 'undefined' ? hi : array.length;

  while (lo < hi) {
    mid = (lo + hi) >>> 1;

    if (value <= array[mid]) {
      hi = mid;
    }
    else {
      lo = -~mid;
    }
  }

  return lo;
};

/**
 * Same as above, but can use a custom comparator function.
 *
 * @param  {function} comparator - Custom comparator function.
 * @param  {array}    array      - Haystack.
 * @param  {any}      value      - Needle.
 * @return {number}
 */
exports.lowerBoundWithComparator = function(comparator, array, value) {
  var mid = 0,
      lo = 0,
      hi = array.length;

  while (lo < hi) {
    mid = (lo + hi) >>> 1;

    if (comparator(value, array[mid]) <= 0) {
      hi = mid;
    }
    else {
      lo = -~mid;
    }
  }

  return lo;
};

/**
 * Same as above, but can work on sorted indices.
 *
 * @param  {array}    array - Haystack.
 * @param  {array}    array - Indices.
 * @param  {any}      value - Needle.
 * @return {number}
 */
exports.lowerBoundIndices = function(array, indices, value, lo, hi) {
  var mid = 0;

  lo = typeof lo !== 'undefined' ? lo : 0;
  hi = typeof hi !== 'undefined' ? hi : array.length;

  while (lo < hi) {
    mid = (lo + hi) >>> 1;

    if (value <= array[indices[mid]]) {
      hi = mid;
    }
    else {
      lo = -~mid;
    }
  }

  return lo;
};

/**
 * Function returning the upper bound of the given value in the array.
 *
 * @param  {array}  array - Haystack.
 * @param  {any}    value - Needle.
 * @param  {number} [lo] - Start index.
 * @param  {numner} [hi] - End index.
 * @return {number}
 */
exports.upperBound = function(array, value, lo, hi) {
  var mid = 0;

  lo = typeof lo !== 'undefined' ? lo : 0;
  hi = typeof hi !== 'undefined' ? hi : array.length;

  while (lo < hi) {
    mid = (lo + hi) >>> 1;

    if (value >= array[mid]) {
      lo = -~mid;
    }
    else {
      hi = mid;
    }
  }

  return lo;
};

/**
 * Same as above, but can use a custom comparator function.
 *
 * @param  {function} comparator - Custom comparator function.
 * @param  {array}    array      - Haystack.
 * @param  {any}      value      - Needle.
 * @return {number}
 */
exports.upperBoundWithComparator = function(comparator, array, value) {
  var mid = 0,
      lo = 0,
      hi = array.length;

  while (lo < hi) {
    mid = (lo + hi) >>> 1;

    if (comparator(value, array[mid]) >= 0) {
      lo = -~mid;
    }
    else {
      hi = mid;
    }
  }

  return lo;
};
