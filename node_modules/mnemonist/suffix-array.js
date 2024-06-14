/**
 * Mnemonist Suffix Array
 * =======================
 *
 * Linear time implementation of a suffix array using the recursive
 * method by Karkkainen and Sanders.
 *
 * [References]:
 * https://www.cs.helsinki.fi/u/tpkarkka/publications/jacm05-revised.pdf
 * http://people.mpi-inf.mpg.de/~sanders/programs/suffix/
 * http://citeseerx.ist.psu.edu/viewdoc/download?doi=10.1.1.184.442&rep=rep1&type=pdf
 *
 * [Article]:
 * "Simple Linear Work Suffix Array Construction", Karkkainen and Sanders.
 *
 * [Note]:
 * A paper by Simon J. Puglisi, William F. Smyth & Andrew Turpin named
 * "The Performance of Linear Time Suffix Sorting Algorithms" seems to
 * prove that supralinear algorithm are in fact better faring for
 * "real" world use cases. It would be nice to check this out in JavaScript
 * because the high level of the language could change a lot to the fact.
 *
 * The current code is largely inspired by the following:
 * https://github.com/tixxit/suffixarray/blob/master/suffixarray.js
 */

/**
 * Constants.
 */
var SEPARATOR = '\u0001';

/**
 * Function used to sort the triples.
 *
 * @param {string|array} string - Padded sequence.
 * @param {array}        array  - Array to sort (will be mutated).
 * @param {number}       offset - Index offset.
 */
function sort(string, array, offset) {
  var l = array.length,
      buckets = [],
      i = l,
      j = -1,
      b,
      d = 0,
      bits;

  while (i--)
    j = Math.max(string[array[i] + offset], j);

  bits = j >> 24 && 32 || j >> 16 && 24 || j >> 8 && 16 || 8;

  for (; d < bits; d += 4) {
    for (i = 16; i--;)
      buckets[i] = [];
    for (i = l; i--;)
      buckets[((string[array[i] + offset]) >> d) & 15].push(array[i]);
    for (b = 0; b < 16; b++) {
      for (j = buckets[b].length; j--;)
        array[++i] = buckets[b][j];
    }
  }
}

/**
 * Comparison helper.
 */
function compare(string, lookup, m, n) {
  return (
    (string[m] - string[n]) ||
    (m % 3 === 2 ?
      (string[m + 1] - string[n + 1]) || (lookup[m + 2] - lookup[n + 2]) :
      (lookup[m + 1] - lookup[n + 1]))
  );
}

/**
 * Recursive function used to build the suffix tree in linear time.
 *
 * @param  {string|array} string - Padded sequence.
 * @param  {number}       l      - True length of sequence (unpadded).
 * @return {array}
 */
function build(string, l) {
  var a = [],
      b = [],
      al = (2 * l / 3) | 0,
      bl = l - al,
      r = (al + 1) >> 1,
      i = al,
      j = 0,
      k,
      lookup = [],
      result = [];

  if (l === 1)
    return [0];

  while (i--)
    a[i] = ((i * 3) >> 1) + 1;

  for (i = 3; i--;)
    sort(string, a, i);

  j = b[((a[0] / 3) | 0) + (a[0] % 3 === 1 ? 0 : r)] = 1;

  for (i = 1; i < al; i++) {
    if (string[a[i]] !== string[a[i - 1]] ||
        string[a[i] + 1] !== string[a[i - 1] + 1] ||
        string[a[i] + 2] !== string[a[i - 1] + 2])
      j++;

    b[((a[i] / 3) | 0) + (a[i] % 3 === 1 ? 0 : r)] = j;
  }

  if (j < al) {
    b = build(b, al);

    for (i = al; i--;)
      a[i] = b[i] < r ? b[i] * 3 + 1 : ((b[i] - r) * 3 + 2);
  }

  for (i = al; i--;)
    lookup[a[i]] = i;
  lookup[l] = -1;
  lookup[l + 1] = -2;

  b = l % 3 === 1 ? [l - 1] : [];

  for (i = 0; i < al; i++) {
    if (a[i] % 3 === 1)
      b.push(a[i] - 1);
  }

  sort(string, b, 0);

  for (i = 0, j = 0, k = 0; i < al && j < bl;)
    result[k++] = (
      compare(string, lookup, a[i], b[j]) < 0 ?
        a[i++] :
        b[j++]
    );

  while (i < al)
    result[k++] = a[i++];

  while (j < bl)
    result[k++] = b[j++];

  return result;
}

/**
 * Function used to create the array we are going to work on.
 *
 * @param  {string|array} target - Target sequence.
 * @return {array}
 */
function convert(target) {

  // Creating the alphabet array
  var length = target.length,
      paddingOffset = length % 3,
      array = new Array(length + paddingOffset),
      l,
      i;

  // If we have an arbitrary sequence, we need to transform it
  if (typeof target !== 'string') {
    var uniqueTokens = Object.create(null);

    for (i = 0; i < length; i++) {
      if (!uniqueTokens[target[i]])
        uniqueTokens[target[i]] = true;
    }

    var alphabet = Object.create(null),
        sortedUniqueTokens = Object.keys(uniqueTokens).sort();

    for (i = 0, l = sortedUniqueTokens.length; i < l; i++)
      alphabet[sortedUniqueTokens[i]] = i + 1;

    for (i = 0; i < length; i++) {
      array[i] = alphabet[target[i]];
    }
  }
  else {
    for (i = 0; i < length; i++)
      array[i] = target.charCodeAt(i);
  }

  // Padding the array
  for (; i < paddingOffset; i++)
    array[i] = 0;

  return array;
}

/**
 * Suffix Array.
 *
 * @constructor
 * @param {string|array} string - Sequence for which to build the suffix array.
 */
function SuffixArray(string) {

  // Properties
  this.hasArbitrarySequence = typeof string !== 'string';
  this.string = string;
  this.length = string.length;

  // Building the array
  this.array = build(convert(string), this.length);
}

/**
 * Convenience known methods.
 */
SuffixArray.prototype.toString = function() {
  return this.array.join(',');
};

SuffixArray.prototype.toJSON = function() {
  return this.array;
};

SuffixArray.prototype.inspect = function() {
  var array = new Array(this.length);

  for (var i = 0; i < this.length; i++)
    array[i] = this.string.slice(this.array[i]);

  // Trick so that node displays the name of the constructor
  Object.defineProperty(array, 'constructor', {
    value: SuffixArray,
    enumerable: false
  });

  return array;
};

if (typeof Symbol !== 'undefined')
  SuffixArray.prototype[Symbol.for('nodejs.util.inspect.custom')] = SuffixArray.prototype.inspect;

/**
 * Generalized Suffix Array.
 *
 * @constructor
 */
function GeneralizedSuffixArray(strings) {

  // Properties
  this.hasArbitrarySequence = typeof strings[0] !== 'string';
  this.size = strings.length;

  if (this.hasArbitrarySequence) {
    this.text = [];

    for (var i = 0, l = this.size; i < l; i++) {
      this.text.push.apply(this.text, strings[i]);

      if (i < l - 1)
        this.text.push(SEPARATOR);
    }
  }
  else {
    this.text = strings.join(SEPARATOR);
  }

  this.firstLength = strings[0].length;
  this.length = this.text.length;

  // Building the array
  this.array = build(convert(this.text), this.length);
}

/**
 * Method used to retrieve the longest common subsequence of the generalized
 * suffix array.
 *
 * @return {string|array}
 */
GeneralizedSuffixArray.prototype.longestCommonSubsequence = function() {
  var lcs = this.hasArbitrarySequence ? [] : '',
      lcp,
      i,
      j,
      s,
      t;

  for (i = 1; i < this.length; i++) {
    s = this.array[i];
    t = this.array[i - 1];

    if (s < this.firstLength &&
        t < this.firstLength)
      continue;

    if (s > this.firstLength &&
        t > this.firstLength)
      continue;

    lcp = Math.min(this.length - s, this.length - t);

    for (j = 0; j < lcp; j++) {
      if (this.text[s + j] !== this.text[t + j]) {
        lcp = j;
        break;
      }
    }

    if (lcp > lcs.length)
      lcs = this.text.slice(s, s + lcp);
  }

  return lcs;
};

/**
 * Convenience known methods.
 */
GeneralizedSuffixArray.prototype.toString = function() {
  return this.array.join(',');
};

GeneralizedSuffixArray.prototype.toJSON = function() {
  return this.array;
};

GeneralizedSuffixArray.prototype.inspect = function() {
  var array = new Array(this.length);

  for (var i = 0; i < this.length; i++)
    array[i] = this.text.slice(this.array[i]);

  // Trick so that node displays the name of the constructor
  Object.defineProperty(array, 'constructor', {
    value: GeneralizedSuffixArray,
    enumerable: false
  });

  return array;
};

if (typeof Symbol !== 'undefined')
  GeneralizedSuffixArray.prototype[Symbol.for('nodejs.util.inspect.custom')] = GeneralizedSuffixArray.prototype.inspect;

/**
 * Exporting.
 */
SuffixArray.GeneralizedSuffixArray = GeneralizedSuffixArray;
module.exports = SuffixArray;
