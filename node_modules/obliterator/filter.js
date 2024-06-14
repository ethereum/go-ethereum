/**
 * Obliterator Filter Function
 * ===========================
 *
 * Function returning a iterator filtering the given iterator.
 */
var Iterator = require('./iterator.js');
var iter = require('./iter.js');

/**
 * Filter.
 *
 * @param  {Iterable} target    - Target iterable.
 * @param  {function} predicate - Predicate function.
 * @return {Iterator}
 */
module.exports = function filter(target, predicate) {
  var iterator = iter(target);
  var step;

  return new Iterator(function () {
    do {
      step = iterator.next();
    } while (!step.done && !predicate(step.value));

    return step;
  });
};
