/**
 * Obliterator Includes Function
 * ==============================
 *
 * Function taking an iterable and returning whether the given item can be
 * found in it.
 */
var iter = require('./iter.js');

/**
 * Includes.
 *
 * @param  {Iterable} iterable  - Target iterable.
 * @param  {function} value     - Searched value.
 * @return {boolean}
 */
module.exports = function includes(iterable, value) {
  var iterator = iter(iterable);

  var step;

  while (((step = iterator.next()), !step.done)) {
    if (step.value === value) return true;
  }

  return false;
};
