/**
 * Obliterator Combinations Function
 * ==================================
 *
 * Iterator returning combinations of the given array.
 */
var Iterator = require('./iterator.js');

/**
 * Helper mapping indices to items.
 */
function indicesToItems(target, items, indices, r) {
  for (var i = 0; i < r; i++) target[i] = items[indices[i]];
}

/**
 * Combinations.
 *
 * @param  {array}    array - Target array.
 * @param  {number}   r     - Size of the subsequences.
 * @return {Iterator}
 */
module.exports = function combinations(array, r) {
  if (!Array.isArray(array))
    throw new Error(
      'obliterator/combinations: first argument should be an array.'
    );

  var n = array.length;

  if (typeof r !== 'number')
    throw new Error(
      'obliterator/combinations: second argument should be omitted or a number.'
    );

  if (r > n)
    throw new Error(
      'obliterator/combinations: the size of the subsequences should not exceed the length of the array.'
    );

  if (r === n) return Iterator.of(array.slice());

  var indices = new Array(r),
    subsequence = new Array(r),
    first = true,
    i;

  for (i = 0; i < r; i++) indices[i] = i;

  return new Iterator(function next() {
    if (first) {
      first = false;

      indicesToItems(subsequence, array, indices, r);
      return {value: subsequence, done: false};
    }

    if (indices[r - 1]++ < n - 1) {
      indicesToItems(subsequence, array, indices, r);
      return {value: subsequence, done: false};
    }

    i = r - 2;

    while (i >= 0 && indices[i] >= n - (r - i)) --i;

    if (i < 0) return {done: true};

    indices[i]++;

    while (++i < r) indices[i] = indices[i - 1] + 1;

    indicesToItems(subsequence, array, indices, r);
    return {value: subsequence, done: false};
  });
};
