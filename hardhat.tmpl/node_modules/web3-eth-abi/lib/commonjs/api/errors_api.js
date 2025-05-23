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
exports.encodeErrorSignature = void 0;
/**
 *
 *  @module ABI
 */
const web3_utils_1 = require("web3-utils");
const web3_errors_1 = require("web3-errors");
const utils_js_1 = require("../utils.js");
/**
 * Encodes the error name to its ABI signature, which are the sha3 hash of the error name including input types.
 */
const encodeErrorSignature = (functionName) => {
    if (typeof functionName !== 'string' && !(0, utils_js_1.isAbiErrorFragment)(functionName)) {
        throw new web3_errors_1.AbiError('Invalid parameter value in encodeErrorSignature');
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
exports.encodeErrorSignature = encodeErrorSignature;
//# sourceMappingURL=errors_api.js.map