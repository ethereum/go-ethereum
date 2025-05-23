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
var _a;
Object.defineProperty(exports, "__esModule", { value: true });
exports.Web3DeferredPromise = void 0;
const web3_errors_1 = require("web3-errors");
/**
 * The class is a simple implementation of a deferred promise with optional timeout functionality,
 * which can be useful when dealing with asynchronous tasks.
 *
 */
class Web3DeferredPromise {
    /**
     *
     * @param timeout - (optional) The timeout in milliseconds.
     * @param eagerStart - (optional) If true, the timer starts as soon as the promise is created.
     * @param timeoutMessage - (optional) The message to include in the timeout erro that is thrown when the promise times out.
     */
    constructor({ timeout, eagerStart, timeoutMessage, } = {
        timeout: 0,
        eagerStart: false,
        timeoutMessage: 'DeferredPromise timed out',
    }) {
        // public tag to treat object as promise by different libs
        // eslint-disable-next-line @typescript-eslint/prefer-as-const
        this[_a] = 'Promise';
        this._state = 'pending';
        this._promise = new Promise((resolve, reject) => {
            this._resolve = resolve;
            this._reject = reject;
        });
        this._timeoutMessage = timeoutMessage;
        this._timeoutInterval = timeout;
        if (eagerStart) {
            this.startTimer();
        }
    }
    /**
     * Returns the current state of the promise.
     * @returns 'pending' | 'fulfilled' | 'rejected'
     */
    get state() {
        return this._state;
    }
    /**
     *
     * @param onfulfilled - (optional) The callback to execute when the promise is fulfilled.
     * @param onrejected  - (optional) The callback to execute when the promise is rejected.
     * @returns
     */
    then(onfulfilled, onrejected) {
        return __awaiter(this, void 0, void 0, function* () {
            return this._promise.then(onfulfilled, onrejected);
        });
    }
    /**
     *
     * @param onrejected - (optional) The callback to execute when the promise is rejected.
     * @returns
     */
    catch(
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    onrejected) {
        return __awaiter(this, void 0, void 0, function* () {
            return this._promise.catch(onrejected);
        });
    }
    /**
     *
     * @param onfinally - (optional) The callback to execute when the promise is settled (fulfilled or rejected).
     * @returns
     */
    finally(onfinally) {
        return __awaiter(this, void 0, void 0, function* () {
            return this._promise.finally(onfinally);
        });
    }
    /**
     * Resolves the current promise.
     * @param value - The value to resolve the promise with.
     */
    resolve(value) {
        this._resolve(value);
        this._state = 'fulfilled';
        this._clearTimeout();
    }
    /**
     * Rejects the current promise.
     * @param reason - The reason to reject the promise with.
     */
    reject(reason) {
        this._reject(reason);
        this._state = 'rejected';
        this._clearTimeout();
    }
    /**
     * Starts the timeout timer for the promise.
     */
    startTimer() {
        if (this._timeoutInterval && this._timeoutInterval > 0) {
            this._timeoutId = setTimeout(this._checkTimeout.bind(this), this._timeoutInterval);
        }
    }
    _checkTimeout() {
        if (this._state === 'pending' && this._timeoutId) {
            this.reject(new web3_errors_1.OperationTimeoutError(this._timeoutMessage));
        }
    }
    _clearTimeout() {
        if (this._timeoutId) {
            clearTimeout(this._timeoutId);
        }
    }
}
exports.Web3DeferredPromise = Web3DeferredPromise;
_a = Symbol.toStringTag;
//# sourceMappingURL=web3_deferred_promise.js.map