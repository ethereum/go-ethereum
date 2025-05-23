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
Object.defineProperty(exports, "__esModule", { value: true });
exports.Web3EventEmitter = void 0;
const web3_utils_1 = require("web3-utils");
class Web3EventEmitter {
    constructor() {
        this._emitter = new web3_utils_1.EventEmitter();
    }
    on(eventName, fn) {
        // eslint-disable-next-line @typescript-eslint/no-misused-promises
        this._emitter.on(eventName, fn);
    }
    once(eventName, fn) {
        // eslint-disable-next-line @typescript-eslint/no-misused-promises
        this._emitter.once(eventName, fn);
    }
    off(eventName, fn) {
        // eslint-disable-next-line @typescript-eslint/no-misused-promises
        this._emitter.off(eventName, fn);
    }
    emit(eventName, params) {
        this._emitter.emit(eventName, params);
    }
    listenerCount(eventName) {
        return this._emitter.listenerCount(eventName);
    }
    listeners(eventName) {
        return this._emitter.listeners(eventName);
    }
    eventNames() {
        return this._emitter.eventNames();
    }
    removeAllListeners() {
        return this._emitter.removeAllListeners();
    }
    setMaxListenerWarningThreshold(maxListenersWarningThreshold) {
        this._emitter.setMaxListeners(maxListenersWarningThreshold);
    }
    getMaxListeners() {
        return this._emitter.getMaxListeners();
    }
}
exports.Web3EventEmitter = Web3EventEmitter;
//# sourceMappingURL=web3_event_emitter.js.map