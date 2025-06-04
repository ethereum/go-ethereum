/**
 * @file Adapted from https://github.com/purescript/purescript-random
 */
import { IO } from './IO';
/**
 * Returns a random number between 0 (inclusive) and 1 (exclusive). This is a direct wrapper around JavaScript's
 * `Math.random()`.
 *
 * @since 1.0.0
 */
export var random = new IO(function () { return Math.random(); });
/**
 * Takes a range specified by `low` (the first argument) and `high` (the second), and returns a random integer uniformly
 * distributed in the closed interval `[low, high]`. It is unspecified what happens if `low > high`, or if either of
 * `low` or `high` is not an integer.
 *
 * @since 1.0.0
 */
export var randomInt = function (low, high) {
    return random.map(function (n) { return Math.floor((high - low + 1) * n + low); });
};
/**
 * Returns a random number between a minimum value (inclusive) and a maximum value (exclusive). It is unspecified what
 * happens if `maximum < minimum`.
 *
 * @since 1.0.0
 */
export var randomRange = function (min, max) {
    return random.map(function (n) { return (max - min) * n + min; });
};
/**
 * Returns a random boolean value with an equal chance of being `true` or `false`
 *
 * @since 1.0.0
 */
export var randomBool = random.map(function (n) { return n < 0.5; });
