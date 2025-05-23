/**
 * Obliterator Split Function
 * ===========================
 *
 * Function returning an iterator over the pieces of a regex split.
 */
var Iterator = require('./iterator.js');

/**
 * Function used to make the given pattern global.
 *
 * @param  {RegExp} pattern - Regular expression to make global.
 * @return {RegExp}
 */
function makeGlobal(pattern) {
  var flags = 'g';

  if (pattern.multiline) flags += 'm';
  if (pattern.ignoreCase) flags += 'i';
  if (pattern.sticky) flags += 'y';
  if (pattern.unicode) flags += 'u';

  return new RegExp(pattern.source, flags);
}

/**
 * Split.
 *
 * @param  {RegExp}   pattern - Regular expression to use.
 * @param  {string}   string  - Target string.
 * @return {Iterator}
 */
module.exports = function split(pattern, string) {
  if (!(pattern instanceof RegExp))
    throw new Error(
      'obliterator/split: invalid pattern. Expecting a regular expression.'
    );

  if (typeof string !== 'string')
    throw new Error('obliterator/split: invalid target. Expecting a string.');

  // NOTE: cloning the pattern has a performance cost but side effects for not
  // doing so might be worse.
  pattern = makeGlobal(pattern);

  var consumed = false,
    current = 0;

  return new Iterator(function () {
    if (consumed) return {done: true};

    var match = pattern.exec(string),
      value,
      length;

    if (match) {
      length = match.index + match[0].length;

      value = string.slice(current, match.index);
      current = length;
    } else {
      consumed = true;
      value = string.slice(current);
    }

    return {value: value, done: false};
  });
};
