/**
 * Mnemonist PassjoinIndex
 * ========================
 *
 * The PassjoinIndex is an index leveraging the "passjoin" algorithm as a mean
 * to index strings for Levenshtein distance queries. It features a complexity
 * related to the Levenshtein query threshold k rather than the number of
 * strings to test (roughly O(k^3)).
 *
 * [References]:
 * Jiang, Yu, Dong Deng, Jiannan Wang, Guoliang Li, et Jianhua Feng.
 * « Efficient Parallel Partition-Based Algorithms for Similarity Search and Join
 * with Edit Distance Constraints ». In Proceedings of the Joint EDBT/ICDT 2013
 * Workshops on - EDBT ’13, 341. Genoa, Italy: ACM Press, 2013.
 * https://doi.org/10.1145/2457317.2457382.
 *
 * Li, Guoliang, Dong Deng, et Jianhua Feng. « A Partition-Based Method for
 * String Similarity Joins with Edit-Distance Constraints ». ACM Transactions on
 * Database Systems 38, no 2 (1 juin 2013): 1‑33.
 * https://doi.org/10.1145/2487259.2487261.
 *
 * [Urls]:
 * http://people.csail.mit.edu/dongdeng/projects/passjoin/index.html
 */
var Iterator = require('obliterator/iterator'),
    forEach = require('obliterator/foreach');

// TODO: leveraging BagDistance as an upper bound of Levenshtein
// TODO: leverage n-grams recursive indexing
// TODO: try the MultiArray as a memory backend
// TODO: what about damerau levenshtein

/**
 * Helpers.
 */

/**
 * Function returning the number of substrings that will be selected by the
 * multi-match-aware selection scheme for theshold `k`, for a string of length
 * `s` to match strings of length `l`.
 *
 * @param   {number} k - Levenshtein distance threshold.
 * @param   {number} s - Length of target strings.
 * @param   {number} l - Length of strings to match.
 * @returns {number}   - The number of selected substrings.
 */
function countSubstringsL(k, s, l) {
  return (((Math.pow(k, 2) - Math.pow(Math.abs(s - l), 2)) / 2) | 0) + k + 1;
}

/**
 * Function returning the minimum number of substrings that will be selected by
 * the multi-match-aware selection scheme for theshold `k`, for a string of
 * length `s` to match any string of relevant length.
 *
 * @param   {number} k - Levenshtein distance threshold.
 * @param   {number} s - Length of target strings.
 * @returns {number}   - The number of selected substrings.
 */
function countKeys(k, s) {
  var c = 0;

  for (var l = 0, m = s + 1; l < m; l++)
    c += countSubstringsL(k, s, l);

  return c;
}

/**
 * Function used to compare two keys in order to sort them first by decreasing
 * length and then alphabetically as per the "4.2 Effective Indexing Strategy"
 * point of the paper.
 *
 * @param   {number} k - Levenshtein distance threshold.
 * @param   {number} s - Length of target strings.
 * @returns {number}   - The number of selected substrings.
 */
function comparator(a, b) {
  if (a.length > b.length)
    return -1;
  if (a.length < b.length)
    return 1;

  if (a < b)
    return -1;
  if (a > b)
    return 1;

  return 0;
}

/**
 * Function partitioning a string into k + 1 uneven segments, the shorter
 * ones, then the longer ones.
 *
 * @param   {number} k - Levenshtein distance threshold.
 * @param   {number} l - Length of the string.
 * @returns {Array}    - The partition tuples (start, length).
 */
function partition(k, l) {
  var m = k + 1,
      a = (l / m) | 0,
      b = a + 1,
      i,
      j;

  var largeSegments = l - a * m,
      smallSegments = m - largeSegments;

  var tuples = new Array(k + 1);

  for (i = 0; i < smallSegments; i++)
    tuples[i] = [i * a, a];

  var offset = (i - 1) * a + a;

  for (j = 0; j < largeSegments; j++)
    tuples[i + j] = [offset + j * b, b];

  return tuples;
}

/**
 * Function yielding a string's k + 1 passjoin segments to index.
 *
 * @param   {number} k      - Levenshtein distance threshold.
 * @param   {string} string - Target string.
 * @returns {Array}         - The string's segments.
 */
function segments(k, string) {
  var l = string.length,
      m = k + 1,
      a = (l / m) | 0,
      b = a + 1,
      o,
      i,
      j;

  var largeSegments = l - a * m,
      smallSegments = m - largeSegments;

  var S = new Array(k + 1);

  for (i = 0; i < smallSegments; i++) {
    o = i * a;
    S[i] = string.slice(o, o + a);
  }

  var offset = (i - 1) * a + a;

  for (j = 0; j < largeSegments; j++) {
    o = offset + j * b;
    S[i + j] = string.slice(o, o + b);
  }

  return S;
}

// TODO: jsdocs
function segmentPos(k, i, string) {
  if (i === 0)
    return 0;

  var l = string.length;

  var m = k + 1,
      a = (l / m) | 0,
      b = a + 1;

  var largeSegments = l - a * m,
      smallSegments = m - largeSegments;

  if (i <= smallSegments - 1)
    return i * a;

  var offset = i - smallSegments;

  return smallSegments * a + offset * b;
}

/**
 * Function returning the interval of relevant substrings to lookup using the
 * multi-match-aware substring selection scheme described in the paper.
 *
 * @param   {number} k      - Levenshtein distance threshold.
 * @param   {number} delta  - Signed length difference between both considered strings.
 * @param   {number} i      - k + 1 segment index.
 * @param   {number} s      - String's length.
 * @param   {number} pi     - k + 1 segment position in target string.
 * @param   {number} li     - k + 1 segment length.
 * @returns {Array}         - The interval (start, stop).
 */
function multiMatchAwareInterval(k, delta, i, s, pi, li) {
  var start1 = pi - i,
      end1 = pi + i;

  var o = k - i;

  var start2 = pi + delta - o,
      end2 = pi + delta + o;

  var end3 = s - li;

  return [Math.max(0, start1, start2), Math.min(end1, end2, end3)];
}

/**
 * Function yielding relevant substrings to lookup using the multi-match-aware
 * substring selection scheme described in the paper.
 *
 * @param   {number} k      - Levenshtein distance threshold.
 * @param   {string} string  - Target string.
 * @param   {number} l      - Length of strings to match.
 * @param   {number} i      - k + 1 segment index.
 * @param   {number} pi     - k + 1 segment position in target string.
 * @param   {number} li     - k + 1 segment length.
 * @returns {Array}         - The contiguous substrings.
 */
function multiMatchAwareSubstrings(k, string, l, i, pi, li) {
  var s = string.length;

  // Note that we need to keep the non-absolute delta for this function
  // to work in both directions, up & down
  var delta = s - l;

  var interval = multiMatchAwareInterval(k, delta, i, s, pi, li);

  var start = interval[0],
      stop = interval[1];

  var currentSubstring = '';

  var substrings = [];

  var substring, j, m;

  for (j = start, m = stop + 1; j < m; j++) {
    substring = string.slice(j, j + li);

    // We skip identical consecutive substrings (to avoid repetition in case
    // of contiguous letter duplication)
    if (substring === currentSubstring)
      continue;

    substrings.push(substring);

    currentSubstring = substring;
  }

  return substrings;
}

/**
 * PassjoinIndex.
 *
 * @note I tried to apply the paper's optimizations regarding Levenshtein
 * distance computations but it did not provide a performance boost, quite
 * the contrary. This is because since we are mostly using the index for small k
 * here, most of the strings we work on are quite small and the bookkeeping
 * induced by Ukkonen's method and the paper's one are slowing us down more than
 * they actually help us go faster.
 *
 * @note This implementation does not try to ensure that you add the same string
 * more than once.
 *
 * @constructor
 * @param {function} levenshtein - Levenshtein distance function.
 * @param {number}   k           - Levenshtein distance threshold.
 */
function PassjoinIndex(levenshtein, k) {
  if (typeof levenshtein !== 'function')
    throw new Error('mnemonist/passjoin-index: `levenshtein` should be a function returning edit distance between two strings.');

  if (typeof k !== 'number' || k < 1)
    throw new Error('mnemonist/passjoin-index: `k` should be a number > 0');

  this.levenshtein = levenshtein;
  this.k = k;
  this.clear();
}

/**
 * Method used to clear the structure.
 *
 * @return {undefined}
 */
PassjoinIndex.prototype.clear = function() {

  // Properties
  this.size = 0;
  this.strings = [];
  this.invertedIndices = {};
};

/**
 * Method used to add a new value to the index.
 *
 * @param  {string|Array} value - Value to add.
 * @return {PassjoinIndex}
 */
PassjoinIndex.prototype.add = function(value) {
  var l = value.length;

  var stringIndex = this.size;

  this.strings.push(value);
  this.size++;

  var S = segments(this.k, value);

  var Ll = this.invertedIndices[l];

  if (typeof Ll === 'undefined') {
    Ll = {};
    this.invertedIndices[l] = Ll;
  }

  var segment,
      matches,
      key,
      i,
      m;

  for (i = 0, m = S.length; i < m; i++) {
    segment = S[i];
    key = segment + i;
    matches = Ll[key];

    if (typeof matches === 'undefined') {
      matches = [stringIndex];
      Ll[key] = matches;
    }
    else {
      matches.push(stringIndex);
    }
  }

  return this;
};

/**
 * Method used to search for string matching the given query.
 *
 * @param  {string|Array} query - Query string.
 * @return {Array}
 */
PassjoinIndex.prototype.search = function(query) {
  var s = query.length,
      k = this.k;

  var M = new Set();

  var candidates,
      candidate,
      queryPos,
      querySegmentLength,
      key,
      S,
      P,
      l,
      m,
      i,
      n1,
      j,
      n2,
      y,
      n3;

  for (l = Math.max(0, s - k), m = s + k + 1; l < m; l++) {
    var Ll = this.invertedIndices[l];

    if (typeof Ll === 'undefined')
      continue;

    P = partition(k, l);

    for (i = 0, n1 = P.length; i < n1; i++) {
      queryPos = P[i][0];
      querySegmentLength = P[i][1];

      S = multiMatchAwareSubstrings(
        k,
        query,
        l,
        i,
        queryPos,
        querySegmentLength
      );

      // Empty string edge case
      if (!S.length)
        S = [''];

      for (j = 0, n2 = S.length; j < n2; j++) {
        key = S[j] + i;
        candidates = Ll[key];

        if (typeof candidates === 'undefined')
          continue;

        for (y = 0, n3 = candidates.length; y < n3; y++) {
          candidate = this.strings[candidates[y]];

          // NOTE: first condition is here not to compute Levenshtein
          // distance for tiny strings

          // NOTE: maintaining a Set of rejected candidate is not really useful
          // because it consumes more memory and because non-matches are
          // less likely to be candidates agains
          if (
            s <= k && l <= k ||
            (
              !M.has(candidate) &&
              this.levenshtein(query, candidate) <= k
            )
          )
            M.add(candidate);
        }
      }
    }
  }

  return M;
};

/**
 * Method used to iterate over the index.
 *
 * @param  {function}  callback - Function to call for each item.
 * @param  {object}    scope    - Optional scope.
 * @return {undefined}
 */
PassjoinIndex.prototype.forEach = function(callback, scope) {
  scope = arguments.length > 1 ? scope : this;

  for (var i = 0, l = this.strings.length; i < l; i++)
    callback.call(scope, this.strings[i], i, this);
};

/**
 * Method used to create an iterator over a index's values.
 *
 * @return {Iterator}
 */
PassjoinIndex.prototype.values = function() {
  var strings = this.strings,
      l = strings.length,
      i = 0;

  return new Iterator(function() {
    if (i >= l)
      return {
        done: true
      };

    var value = strings[i];
    i++;

    return {
      value: value,
      done: false
    };
  });
};

/**
 * Attaching the #.values method to Symbol.iterator if possible.
 */
if (typeof Symbol !== 'undefined')
  PassjoinIndex.prototype[Symbol.iterator] = PassjoinIndex.prototype.values;

/**
 * Convenience known methods.
 */
PassjoinIndex.prototype.inspect = function() {
  var array = this.strings.slice();

  // Trick so that node displays the name of the constructor
  Object.defineProperty(array, 'constructor', {
    value: PassjoinIndex,
    enumerable: false
  });

  return array;
};

if (typeof Symbol !== 'undefined')
  PassjoinIndex.prototype[Symbol.for('nodejs.util.inspect.custom')] = PassjoinIndex.prototype.inspect;

/**
 * Static @.from function taking an arbitrary iterable & converting it into
 * a structure.
 *
 * @param  {Iterable} iterable - Target iterable.
 * @return {PassjoinIndex}
 */
PassjoinIndex.from = function(iterable, levenshtein, k) {
  var index = new PassjoinIndex(levenshtein, k);

  forEach(iterable, function(string) {
    index.add(string);
  });

  return index;
};

/**
 * Exporting.
 */
PassjoinIndex.countKeys = countKeys;
PassjoinIndex.comparator = comparator;
PassjoinIndex.partition = partition;
PassjoinIndex.segments = segments;
PassjoinIndex.segmentPos = segmentPos;
PassjoinIndex.multiMatchAwareInterval = multiMatchAwareInterval;
PassjoinIndex.multiMatchAwareSubstrings = multiMatchAwareSubstrings;

module.exports = PassjoinIndex;
