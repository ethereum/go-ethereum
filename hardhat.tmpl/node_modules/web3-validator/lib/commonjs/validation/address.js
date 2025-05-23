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
exports.isAddress = exports.checkAddressCheckSum = void 0;
const keccak_js_1 = require("ethereum-cryptography/keccak.js");
const utils_js_1 = require("ethereum-cryptography/utils.js");
const utils_js_2 = require("../utils.js");
const string_js_1 = require("./string.js");
const bytes_js_1 = require("./bytes.js");
/**
 * Checks the checksum of a given address. Will also return false on non-checksum addresses.
 */
const checkAddressCheckSum = (data) => {
    if (!/^(0x)?[0-9a-f]{40}$/i.test(data))
        return false;
    const address = data.slice(2);
    const updatedData = (0, utils_js_1.utf8ToBytes)(address.toLowerCase());
    const addressHash = (0, utils_js_2.uint8ArrayToHexString)((0, keccak_js_1.keccak256)((0, utils_js_2.ensureIfUint8Array)(updatedData))).slice(2);
    for (let i = 0; i < 40; i += 1) {
        // the nth letter should be uppercase if the nth digit of casemap is 1
        if ((parseInt(addressHash[i], 16) > 7 && address[i].toUpperCase() !== address[i]) ||
            (parseInt(addressHash[i], 16) <= 7 && address[i].toLowerCase() !== address[i])) {
            return false;
        }
    }
    return true;
};
exports.checkAddressCheckSum = checkAddressCheckSum;
/**
 * Checks if a given string is a valid Ethereum address. It will also check the checksum, if the address has upper and lowercase letters.
 */
const isAddress = (value, checkChecksum = true) => {
    if (typeof value !== 'string' && !(0, bytes_js_1.isUint8Array)(value)) {
        return false;
    }
    let valueToCheck;
    if ((0, bytes_js_1.isUint8Array)(value)) {
        valueToCheck = (0, utils_js_2.uint8ArrayToHexString)(value);
    }
    else if (typeof value === 'string' && !(0, string_js_1.isHexStrict)(value)) {
        valueToCheck = value.toLowerCase().startsWith('0x') ? value : `0x${value}`;
    }
    else {
        valueToCheck = value;
    }
    // check if it has the basic requirements of an address
    if (!/^(0x)?[0-9a-f]{40}$/i.test(valueToCheck)) {
        return false;
    }
    // If it's ALL lowercase or ALL upppercase
    if (/^(0x|0X)?[0-9a-f]{40}$/.test(valueToCheck) ||
        /^(0x|0X)?[0-9A-F]{40}$/.test(valueToCheck)) {
        return true;
        // Otherwise check each case
    }
    return checkChecksum ? (0, exports.checkAddressCheckSum)(valueToCheck) : true;
};
exports.isAddress = isAddress;
//# sourceMappingURL=address.js.map