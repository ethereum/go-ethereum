/**
 * @file Adapted from https://github.com/purescript/purescript-exceptions
 */
import { left, right } from './Either';
import { IO, io } from './IO';
import { none, some } from './Option';
/**
 * Create a JavaScript error, specifying a message
 *
 * @since 1.0.0
 * @deprecated
 */
export var error = function (message) {
    return new Error(message);
};
/**
 * Get the error message from a JavaScript error
 *
 * @since 1.0.0
 * @deprecated
 */
export var message = function (e) {
    return e.message;
};
/**
 * Get the stack trace from a JavaScript error
 *
 * @since 1.0.0
 * @deprecated
 */
export var stack = function (e) {
    return typeof e.stack === 'string' ? some(e.stack) : none;
};
/**
 * Throw an exception
 *
 * @since 1.0.0
 * @deprecated
 */
export var throwError = function (e) {
    return new IO(function () {
        throw e;
    });
};
/**
 * Catch an exception by providing an exception handler
 *
 * @since 1.0.0
 * @deprecated
 */
export var catchError = function (ma, handler) {
    return new IO(function () {
        try {
            return ma.run();
        }
        catch (e) {
            if (e instanceof Error) {
                return handler(e).run();
            }
            else {
                return handler(new Error(e.toString())).run();
            }
        }
    });
};
/**
 * Runs an IO and returns eventual Exceptions as a `Left` value. If the computation succeeds the result gets wrapped in
 * a `Right`.
 *
 * @since 1.0.0
 * @deprecated
 */
export var tryCatch = function (ma) {
    // tslint:disable-next-line: deprecation
    return catchError(ma.map(right), function (e) { return io.of(left(e)); });
};
