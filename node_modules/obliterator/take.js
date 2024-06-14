/* eslint no-constant-condition: 0 */
/**
 * Obliterator Take Function
 * ==========================
 *
 * Function taking n or every value of the given iterator and returns them
 * into an array.
 */
var iter = require('./iter.js');

/**
 * Take.
 *
 * @param  {Iterable} iterable - Target iterable.
 * @param  {number}   [n]      - Optional number of items to take.
 * @return {array}
 */
module.exports = function take(iterable, n) {
  var l = arguments.length > 1 ? n : Infinity,
    array = l !== Infinity ? new Array(l) : [],
    step,
    i = 0;

  var iterator = iter(iterable);

  while (true) {
    if (i === l) return array;

    step = iterator.next();

    if (step.done) {
      if (i !== n) array.length = i;

      return array;
    }

    array[i++] = step.value;
  }
};
