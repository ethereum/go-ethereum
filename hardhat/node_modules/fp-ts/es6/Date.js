import { IO } from './IO';
/**
 * Returns the current `Date`
 *
 * @since 1.10.0
 */
export var create = new IO(function () { return new Date(); });
/**
 * Returns the number of milliseconds elapsed since January 1, 1970, 00:00:00 UTC
 *
 * @since 1.10.0
 */
export var now = new IO(function () { return new Date().getTime(); });
