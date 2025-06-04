/**
 * Obliterator Permutations Function
 * ==================================
 *
 * Iterator returning permutations of the given array.
 */
var Iterator = require('./iterator.js');

/**
 * Helper mapping indices to items.
 */
function indicesToItems(target, items, indices, r) {
  for (var i = 0; i < r; i++) target[i] = items[indices[i]];
}

/**
 * Permutations.
 *
 * @param  {array}    array - Target array.
 * @param  {number}   r     - Size of the subsequences.
 * @return {Iterator}
 */
module.exports = function permutations(array, r) {
  if (!Array.isArray(array))
    throw new Error(
      'obliterator/permutations: first argument should be an array.'
    );

  var n = array.length;

  if (arguments.length < 2) r = n;

  if (typeof r !== 'number')
    throw new Error(
      'obliterator/permutations: second argument should be omitted or a number.'
    );

  if (r > n)
    throw new Error(
      'obliterator/permutations: the size of the subsequences should not exceed the length of the array.'
    );

  var indices = new Uint32Array(n),
    subsequence = new Array(r),
    cycles = new Uint32Array(r),
    first = true,
    i;

  for (i = 0; i < n; i++) {
    indices[i] = i;

    if (i < r) cycles[i] = n - i;
  }

  i = r;

  return new Iterator(function next() {
    if (first) {
      first = false;
      indicesToItems(subsequence, array, indices, r);
      return {value: subsequence, done: false};
    }

    var tmp, j;

    i--;

    if (i < 0) return {done: true};

    cycles[i]--;

    if (cycles[i] === 0) {
      tmp = indices[i];

      for (j = i; j < n - 1; j++) indices[j] = indices[j + 1];

      indices[n - 1] = tmp;

      cycles[i] = n - i;
      return next();
    } else {
      j = cycles[i];
      tmp = indices[i];

      indices[i] = indices[n - j];
      indices[n - j] = tmp;

      i = r;

      indicesToItems(subsequence, array, indices, r);
      return {value: subsequence, done: false};
    }
  });
};
