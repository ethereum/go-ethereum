/* eslint no-constant-condition: 0 */
/**
 * Obliterator Consume Function
 * =============================
 *
 * Function consuming the given iterator for n or every steps.
 */

/**
 * Consume.
 *
 * @param  {Iterator} iterator - Target iterator.
 * @param  {number}   [steps]  - Optional steps.
 */
module.exports = function consume(iterator, steps) {
  var step,
    l = arguments.length > 1 ? steps : Infinity,
    i = 0;

  while (true) {
    if (i === l) return;

    step = iterator.next();

    if (step.done) return;

    i++;
  }
};
