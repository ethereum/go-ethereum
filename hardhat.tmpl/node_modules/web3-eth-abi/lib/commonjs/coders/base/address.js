"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.encodeAddress = encodeAddress;
exports.decodeAddress = decodeAddress;
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
const ADDRESS_BYTES_COUNT = 20;
const ADDRESS_OFFSET = utils_js_1.WORD_SIZE - ADDRESS_BYTES_COUNT;
function encodeAddress(param, input) {
    if (typeof input !== 'string') {
        throw new web3_errors_1.AbiError('address type expects string as input type', {
            value: input,
            name: param.name,
            type: param.type,
        });
    }
    let address = input.toLowerCase();
    if (!address.startsWith('0x')) {
        address = `0x${address}`;
    }
    if (!(0, web3_validator_1.isAddress)(address)) {
        throw new web3_errors_1.AbiError('provided input is not valid address', {
            value: input,
            name: param.name,
            type: param.type,
        });
    }
    // for better performance, we could convert hex to destination bytes directly (encoded var)
    const addressBytes = web3_validator_1.utils.hexToUint8Array(address);
    // expand address to WORD_SIZE
    const encoded = (0, utils_js_1.alloc)(utils_js_1.WORD_SIZE);
    encoded.set(addressBytes, ADDRESS_OFFSET);
    return {
        dynamic: false,
        encoded,
    };
}
function decodeAddress(_param, bytes) {
    const addressBytes = bytes.subarray(ADDRESS_OFFSET, utils_js_1.WORD_SIZE);
    if (addressBytes.length !== ADDRESS_BYTES_COUNT) {
        throw new web3_errors_1.AbiError('Invalid decoding input, not enough bytes to decode address', { bytes });
    }
    const result = web3_validator_1.utils.uint8ArrayToHexString(addressBytes);
    // should we check is decoded value is valid address?
    // if(!isAddress(result)) {
    //     throw new AbiError("encoded data is not valid address", {
    //         address: result,
    //     });
    // }
    return {
        result: (0, web3_utils_1.toChecksumAddress)(result),
        encoded: bytes.subarray(utils_js_1.WORD_SIZE),
        consumed: utils_js_1.WORD_SIZE,
    };
}
//# sourceMappingURL=address.js.map