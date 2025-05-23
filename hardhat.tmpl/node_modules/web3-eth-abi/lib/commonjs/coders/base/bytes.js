"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.encodeBytes = encodeBytes;
exports.decodeBytes = decodeBytes;
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
const web3_validator_1 = require("web3-validator");
const utils_js_1 = require("../utils.js");
const number_js_1 = require("./number.js");
const MAX_STATIC_BYTES_COUNT = 32;
function encodeBytes(param, input) {
    // hack for odd length hex strings
    if (typeof input === 'string' && input.length % 2 !== 0) {
        // eslint-disable-next-line no-param-reassign
        input += '0';
    }
    if (!(0, web3_validator_1.isBytes)(input)) {
        throw new web3_errors_1.AbiError('provided input is not valid bytes value', {
            type: param.type,
            value: input,
            name: param.name,
        });
    }
    const bytes = (0, web3_utils_1.bytesToUint8Array)(input);
    const [, size] = param.type.split('bytes');
    // fixed size
    if (size) {
        if (Number(size) > MAX_STATIC_BYTES_COUNT || Number(size) < 1) {
            throw new web3_errors_1.AbiError('invalid bytes type. Static byte type can have between 1 and 32 bytes', {
                type: param.type,
            });
        }
        if (Number(size) < bytes.length) {
            throw new web3_errors_1.AbiError('provided input size is different than type size', {
                type: param.type,
                value: input,
                name: param.name,
            });
        }
        const encoded = (0, utils_js_1.alloc)(utils_js_1.WORD_SIZE);
        encoded.set(bytes);
        return {
            dynamic: false,
            encoded,
        };
    }
    const partsLength = Math.ceil(bytes.length / utils_js_1.WORD_SIZE);
    // one word for length of data + WORD for each part of actual data
    const encoded = (0, utils_js_1.alloc)(utils_js_1.WORD_SIZE + partsLength * utils_js_1.WORD_SIZE);
    encoded.set((0, number_js_1.encodeNumber)({ type: 'uint32', name: '' }, bytes.length).encoded);
    encoded.set(bytes, utils_js_1.WORD_SIZE);
    return {
        dynamic: true,
        encoded,
    };
}
function decodeBytes(param, bytes) {
    const [, sizeString] = param.type.split('bytes');
    let size = Number(sizeString);
    let remainingBytes = bytes;
    let partsCount = 1;
    let consumed = 0;
    if (!size) {
        // dynamic bytes
        const result = (0, number_js_1.decodeNumber)({ type: 'uint32', name: '' }, remainingBytes);
        size = Number(result.result);
        consumed += result.consumed;
        remainingBytes = result.encoded;
        partsCount = Math.ceil(size / utils_js_1.WORD_SIZE);
    }
    if (size > bytes.length) {
        throw new web3_errors_1.AbiError('there is not enough data to decode', {
            type: param.type,
            encoded: bytes,
            size,
        });
    }
    return {
        result: (0, web3_utils_1.bytesToHex)(remainingBytes.subarray(0, size)),
        encoded: remainingBytes.subarray(partsCount * utils_js_1.WORD_SIZE),
        consumed: consumed + partsCount * utils_js_1.WORD_SIZE,
    };
}
//# sourceMappingURL=bytes.js.map