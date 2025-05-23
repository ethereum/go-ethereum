/**
 * Obliterator Some Function
 * ==========================
 *
 * Function taking an iterable and a predicate and returning whether a
 * matching item can be found.
 */
var iter = require('./iter.js');

/**
 * Some.
 *
 * @param  {Iterable} iterable  - Target iterable.
 * @param  {function} predicate - Predicate function.
 * @return {boolean}
 */
module.exports = function some(iterable, predicate) {
  var iterator = iter(iterable);

  var step;

  while (((step = iterator.next()), !step.done)) {
    if (predicate(step.value)) return true;
  }

  return false;
};
