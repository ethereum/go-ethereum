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
exports.encodeEventSignature = void 0;
/**
 *
 *  @module ABI
 */
const web3_utils_1 = require("web3-utils");
const web3_errors_1 = require("web3-errors");
const utils_js_1 = require("../utils.js");
/**
 * Encodes the event name to its ABI signature, which are the sha3 hash of the event name including input types.
 * @param functionName - The event name to encode, or the {@link AbiEventFragment} object of the event. If string, it has to be in the form of `eventName(param1Type,param2Type,...)`. eg: myEvent(uint256,bytes32).
 * @returns - The ABI signature of the event.
 *
 * @example
 * ```ts
 * const event = web3.eth.abi.encodeEventSignature({
 *   name: "myEvent",
 *   type: "event",
 *   inputs: [
 *     {
 *       type: "uint256",
 *       name: "myNumber",
 *     },
 *     {
 *       type: "bytes32",
 *       name: "myBytes",
 *     },
 *   ],
 * });
 * console.log(event);
 * > 0xf2eeb729e636a8cb783be044acf6b7b1e2c5863735b60d6daae84c366ee87d97
 *
 *  const event = web3.eth.abi.encodeEventSignature({
 *   inputs: [
 *     {
 *       indexed: true,
 *       name: "from",
 *       type: "address",
 *     },
 *     {
 *       indexed: true,
 *       name: "to",
 *       type: "address",
 *     },
 *     {
 *       indexed: false,
 *       name: "value",
 *       type: "uint256",
 *     },
 *   ],
 *   name: "Transfer",
 *   type: "event",
 * });
 * console.log(event);
 * > 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef
 * ```
 */
const encodeEventSignature = (functionName) => {
    if (typeof functionName !== 'string' && !(0, utils_js_1.isAbiEventFragment)(functionName)) {
        throw new web3_errors_1.AbiError('Invalid parameter value in encodeEventSignature');
    }
    let name;
    if (functionName && (typeof functionName === 'function' || typeof functionName === 'object')) {
        name = (0, utils_js_1.jsonInterfaceMethodToString)(functionName);
    }
    else {
        name = functionName;
    }
    return (0, web3_utils_1.sha3Raw)(name);
};
exports.encodeEventSignature = encodeEventSignature;
//# sourceMappingURL=events_api.js.map