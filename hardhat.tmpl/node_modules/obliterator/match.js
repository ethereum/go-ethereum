/**
 * Obliterator Match Function
 * ===========================
 *
 * Function returning an iterator over the matches of the given regex on the
 * target string.
 */
var Iterator = require('./iterator.js');

/**
 * Match.
 *
 * @param  {RegExp}   pattern - Regular expression to use.
 * @param  {string}   string  - Target string.
 * @return {Iterator}
 */
module.exports = function match(pattern, string) {
  var executed = false;

  if (!(pattern instanceof RegExp))
    throw new Error(
      'obliterator/match: invalid pattern. Expecting a regular expression.'
    );

  if (typeof string !== 'string')
    throw new Error('obliterator/match: invalid target. Expecting a string.');

  return new Iterator(function () {
    if (executed && !pattern.global) {
      pattern.lastIndex = 0;
      return {done: true};
    }

    executed = true;

    var m = pattern.exec(string);

    if (m) return {value: m};

    pattern.lastIndex = 0;
    return {done: true};
  });
};
