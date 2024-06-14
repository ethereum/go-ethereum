/**
 * Obliterator Range Function
 * ===========================
 *
 * Function returning a range iterator.
 */
var Iterator = require('./iterator.js');

/**
 * Range.
 *
 * @param  {number} start - Start.
 * @param  {number} end   - End.
 * @param  {number} step  - Step.
 * @return {Iterator}
 */
module.exports = function range(start, end, step) {
  if (arguments.length === 1) {
    end = start;
    start = 0;
  }

  if (arguments.length < 3) step = 1;

  var i = start;

  var iterator = new Iterator(function () {
    if (i < end) {
      var value = i;

      i += step;

      return {value: value, done: false};
    }

    return {done: true};
  });

  iterator.start = start;
  iterator.end = end;
  iterator.step = step;

  return iterator;
};
