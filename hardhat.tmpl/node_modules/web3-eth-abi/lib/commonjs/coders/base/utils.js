"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.encodeDynamicParams = encodeDynamicParams;
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
const web3_utils_1 = require("web3-utils");
const utils_js_1 = require("../utils.js");
const number_js_1 = require("./number.js");
function encodeDynamicParams(encodedParams) {
    let staticSize = 0;
    let dynamicSize = 0;
    const staticParams = [];
    const dynamicParams = [];
    // figure out static size
    for (const encodedParam of encodedParams) {
        if (encodedParam.dynamic) {
            staticSize += utils_js_1.WORD_SIZE;
        }
        else {
            staticSize += encodedParam.encoded.length;
        }
    }
    for (const encodedParam of encodedParams) {
        if (encodedParam.dynamic) {
            staticParams.push((0, number_js_1.encodeNumber)({ type: 'uint256', name: '' }, staticSize + dynamicSize));
            dynamicParams.push(encodedParam);
            dynamicSize += encodedParam.encoded.length;
        }
        else {
            staticParams.push(encodedParam);
        }
    }
    return (0, web3_utils_1.uint8ArrayConcat)(...staticParams.map(p => p.encoded), ...dynamicParams.map(p => p.encoded));
}
//# sourceMappingURL=utils.js.map