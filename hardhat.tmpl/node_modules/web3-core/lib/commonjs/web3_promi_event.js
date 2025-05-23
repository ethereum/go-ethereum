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
exports.Web3PromiEvent = void 0;
const web3_event_emitter_js_1 = require("./web3_event_emitter.js");
class Web3PromiEvent extends web3_event_emitter_js_1.Web3EventEmitter {
    constructor(executor) {
        super();
        // public tag to treat object as promise by different libs
        // eslint-disable-next-line @typescript-eslint/prefer-as-const
        this[_a] = 'Promise';
        this._promise = new Promise(executor);
    }
    then(onfulfilled, onrejected) {
        return __awaiter(this, void 0, void 0, function* () {
            return this._promise.then(onfulfilled, onrejected);
        });
    }
    catch(onrejected) {
        return __awaiter(this, void 0, void 0, function* () {
            return this._promise.catch(onrejected);
        });
    }
    finally(onfinally) {
        return __awaiter(this, void 0, void 0, function* () {
            return this._promise.finally(onfinally);
        });
    }
    on(eventName, fn) {
        super.on(eventName, fn);
        return this;
    }
    once(eventName, fn) {
        super.once(eventName, fn);
        return this;
    }
}
exports.Web3PromiEvent = Web3PromiEvent;
_a = Symbol.toStringTag;
//# sourceMappingURL=web3_promi_event.js.map