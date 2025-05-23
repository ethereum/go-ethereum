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
/* eslint-disable max-classes-per-file */
import EventEmitter3 from 'eventemitter3';
/**
 * This class copy the behavior of Node.js EventEmitter class.
 * It is used to provide the same interface for the browser environment.
 */
export class EventEmitter extends EventEmitter3 {
    constructor() {
        super(...arguments);
        // must be defined for backwards compatibility
        this.maxListeners = Number.MAX_SAFE_INTEGER;
    }
    setMaxListeners(maxListeners) {
        this.maxListeners = maxListeners;
        return this;
    }
    getMaxListeners() {
        return this.maxListeners;
    }
}
//# sourceMappingURL=event_emitter.js.map