/* eslint no-constant-condition: 0 */
/**
 * Obliterator Take Into Function
 * ===============================
 *
 * Same as the take function but enables the user to select an array class
 * in which to insert the retrieved values.
 */
var iter = require('./iter.js');

/**
 * Take Into.
 *
 * @param  {function} ArrayClass - Array class to use.
 * @param  {Iterable} iterable   - Target iterable.
 * @param  {number}   n          - Number of items to take.
 * @return {array}
 */
module.exports = function takeInto(ArrayClass, iterable, n) {
  var array = new ArrayClass(n),
    step,
    i = 0;

  var iterator = iter(iterable);

  while (true) {
    if (i === n) return array;

    step = iterator.next();

    if (step.done) {
      if (i !== n) return array.slice(0, i);

      return array;
    }

    array[i++] = step.value;
  }
};
