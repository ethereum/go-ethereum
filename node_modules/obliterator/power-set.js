/**
 * Obliterator Power Set Function
 * ===============================
 *
 * Iterator returning the power set of the given array.
 */
var Iterator = require('./iterator.js'),
  combinations = require('./combinations.js'),
  chain = require('./chain.js');

/**
 * Power set.
 *
 * @param  {array}    array - Target array.
 * @return {Iterator}
 */
module.exports = function powerSet(array) {
  var n = array.length;

  var iterators = new Array(n + 1);

  iterators[0] = Iterator.of([]);

  for (var i = 1; i < n + 1; i++) iterators[i] = combinations(array, i);

  return chain.apply(null, iterators);
};
