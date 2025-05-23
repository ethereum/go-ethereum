"use strict";
/*
This file is part of web3.js.

web3.js is free software: you can redistribute it and/or modify
it under the terms of the GNU Lesser General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

web3.js is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Lesser General Public License for more details.

You should have received a copy of the GNU Lesser General Public License
along with web3.js.  If not, see <http://www.gnu.org/licenses/>.
*/
var __awaiter = (this && this.__awaiter) || function (thisArg, _arguments, P, generator) {
    function adopt(value) { return value instanceof P ? value : new P(function (resolve) { resolve(value); }); }
    return new (P || (P = Promise))(function (resolve, reject) {
        function fulfilled(value) { try { step(generator.next(value)); } catch (e) { reject(e); } }
        function rejected(value) { try { step(generator["throw"](value)); } catch (e) { reject(e); } }
        function step(result) { result.done ? resolve(result.value) : adopt(result.value).then(fulfilled, rejected); }
        step((generator = generator.apply(thisArg, _arguments || [])).next());
    });
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.isPromise = isPromise;
exports.waitWithTimeout = waitWithTimeout;
exports.pollTillDefinedAndReturnIntervalId = pollTillDefinedAndReturnIntervalId;
exports.pollTillDefined = pollTillDefined;
exports.rejectIfTimeout = rejectIfTimeout;
exports.rejectIfConditionAtInterval = rejectIfConditionAtInterval;
const web3_validator_1 = require("web3-validator");
/**
 * An alternative to the node function `isPromise` that exists in `util/types` because it is not available on the browser.
 * @param object - to check if it is a `Promise`
 * @returns `true` if it is an `object` or a `function` that has a `then` function. And returns `false` otherwise.
 */
function isPromise(object) {
    return ((typeof object === 'object' || typeof object === 'function') &&
        // eslint-disable-next-line @typescript-eslint/no-unsafe-member-access
        typeof object.then === 'function');
}
/**
 * Wait for a promise but interrupt it if it did not resolve within a given timeout.
 * If the timeout reached, before the promise code resolve, either throw an error if an error object was provided, or return `undefined`.
 * @param awaitable - The promise or function to wait for.
 * @param timeout - The timeout in milliseconds.
 * @param error - (Optional) The error to throw if the timeout reached.
 */
function waitWithTimeout(awaitable, timeout, error) {
    return __awaiter(this, void 0, void 0, function* () {
        let timeoutId;
        const result = yield Promise.race([
            awaitable instanceof Promise ? awaitable : awaitable(),
            new Promise((resolve, reject) => {
                timeoutId = setTimeout(() => (error ? reject(error) : resolve(undefined)), timeout);
            }),
        ]);
        if (timeoutId) {
            clearTimeout(timeoutId);
        }
        if (result instanceof Error) {
            throw result;
        }
        return result;
    });
}
/**
 * Repeatedly calls an async function with a given interval until the result of the function is defined (not undefined or null),
 * or until a timeout is reached. It returns promise and intervalId.
 * @param func - The function to call.
 * @param interval - The interval in milliseconds.
 */
function pollTillDefinedAndReturnIntervalId(func, interval) {
    let intervalId;
    const polledRes = new Promise((resolve, reject) => {
        intervalId = setInterval((function intervalCallbackFunc() {
            (() => __awaiter(this, void 0, void 0, function* () {
                try {
                    const res = yield waitWithTimeout(func, interval);
                    if (!(0, web3_validator_1.isNullish)(res)) {
                        clearInterval(intervalId);
                        resolve(res);
                    }
                }
                catch (error) {
                    clearInterval(intervalId);
                    reject(error);
                }
            }))();
            return intervalCallbackFunc;
        })(), // this will immediate invoke first call
        interval);
    });
    return [polledRes, intervalId];
}
/**
 * Repeatedly calls an async function with a given interval until the result of the function is defined (not undefined or null),
 * or until a timeout is reached.
 * pollTillDefinedAndReturnIntervalId() function should be used instead of pollTillDefined if you need IntervalId in result.
 * This function will be deprecated in next major release so use pollTillDefinedAndReturnIntervalId().
 * @param func - The function to call.
 * @param interval - The interval in milliseconds.
 */
function pollTillDefined(func, interval) {
    return __awaiter(this, void 0, void 0, function* () {
        return pollTillDefinedAndReturnIntervalId(func, interval)[0];
    });
}
/**
 * Enforce a timeout on a promise, so that it can be rejected if it takes too long to complete
 * @param timeout - The timeout to enforced in milliseconds.
 * @param error - The error to throw if the timeout is reached.
 * @returns A tuple of the timeout id and the promise that will be rejected if the timeout is reached.
 *
 * @example
 * ```ts
 * const [timerId, promise] = web3.utils.rejectIfTimeout(100, new Error('time out'));
 * ```
 */
function rejectIfTimeout(timeout, error) {
    let timeoutId;
    const rejectOnTimeout = new Promise((_, reject) => {
        timeoutId = setTimeout(() => {
            reject(error);
        }, timeout);
    });
    return [timeoutId, rejectOnTimeout];
}
/**
 * Sets an interval that repeatedly executes the given cond function with the specified interval between each call.
 * If the condition is met, the interval is cleared and a Promise that rejects with the returned value is returned.
 * @param cond - The function/condition to call.
 * @param interval - The interval in milliseconds.
 * @returns - an array with the interval ID and the Promise.
 */
function rejectIfConditionAtInterval(cond, interval) {
    let intervalId;
    const rejectIfCondition = new Promise((_, reject) => {
        intervalId = setInterval(() => {
            (() => __awaiter(this, void 0, void 0, function* () {
                const error = yield cond();
                if (error) {
                    clearInterval(intervalId);
                    reject(error);
                }
            }))();
        }, interval);
    });
    return [intervalId, rejectIfCondition];
}
//# sourceMappingURL=promise_helpers.js.map