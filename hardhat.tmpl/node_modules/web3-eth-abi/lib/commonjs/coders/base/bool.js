"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.encodeBoolean = encodeBoolean;
exports.decodeBool = decodeBool;
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
const utils_js_1 = require("../utils.js");
const number_js_1 = require("./number.js");
function encodeBoolean(param, input) {
    let value;
    try {
        value = (0, web3_utils_1.toBool)(input);
    }
    catch (e) {
        if (e instanceof web3_errors_1.InvalidBooleanError) {
            throw new web3_errors_1.AbiError('provided input is not valid boolean value', {
                type: param.type,
                value: input,
                name: param.name,
            });
        }
    }
    return (0, number_js_1.encodeNumber)({ type: 'uint8', name: '' }, Number(value));
}
function decodeBool(_param, bytes) {
    const numberResult = (0, number_js_1.decodeNumber)({ type: 'uint8', name: '' }, bytes);
    if (numberResult.result > 1 || numberResult.result < 0) {
        throw new web3_errors_1.AbiError('Invalid boolean value encoded', {
            boolBytes: bytes.subarray(0, utils_js_1.WORD_SIZE),
            numberResult,
        });
    }
    return {
        result: numberResult.result === BigInt(1),
        encoded: numberResult.encoded,
        consumed: utils_js_1.WORD_SIZE,
    };
}
//# sourceMappingURL=bool.js.map