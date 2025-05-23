"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.encodeString = encodeString;
exports.decodeString = decodeString;
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
const web3_errors_1 = require("web3-errors");
const web3_utils_1 = require("web3-utils");
const bytes_js_1 = require("./bytes.js");
function encodeString(_param, input) {
    if (typeof input !== 'string') {
        throw new web3_errors_1.AbiError('invalid input, should be string', { input });
    }
    const bytes = (0, web3_utils_1.utf8ToBytes)(input);
    return (0, bytes_js_1.encodeBytes)({ type: 'bytes', name: '' }, bytes);
}
function decodeString(_param, bytes) {
    const r = (0, bytes_js_1.decodeBytes)({ type: 'bytes', name: '' }, bytes);
    return {
        result: (0, web3_utils_1.hexToUtf8)(r.result),
        encoded: r.encoded,
        consumed: r.consumed,
    };
}
//# sourceMappingURL=string.js.map