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
exports.isContractAddressInBloom = exports.isUserEthereumAddressInBloom = exports.isInBloom = exports.isBloom = void 0;
const keccak_js_1 = require("ethereum-cryptography/keccak.js");
const utils_js_1 = require("../utils.js");
const address_js_1 = require("./address.js");
const string_js_1 = require("./string.js");
/**
 * Returns true if the bloom is a valid bloom
 * https://github.com/joshstevens19/ethereum-bloom-filters/blob/fbeb47b70b46243c3963fe1c2988d7461ef17236/src/index.ts#L7
 */
const isBloom = (bloom) => {
    if (typeof bloom !== 'string') {
        return false;
    }
    if (!/^(0x)?[0-9a-f]{512}$/i.test(bloom)) {
        return false;
    }
    if (/^(0x)?[0-9a-f]{512}$/.test(bloom) || /^(0x)?[0-9A-F]{512}$/.test(bloom)) {
        return true;
    }
    return false;
};
exports.isBloom = isBloom;
/**
 * Returns true if the value is part of the given bloom
 * note: false positives are possible.
 */
const isInBloom = (bloom, value) => {
    if (typeof value === 'string' && !(0, string_js_1.isHexStrict)(value)) {
        return false;
    }
    if (!(0, exports.isBloom)(bloom)) {
        return false;
    }
    const uint8Array = typeof value === 'string' ? (0, utils_js_1.hexToUint8Array)(value) : value;
    const hash = (0, utils_js_1.uint8ArrayToHexString)((0, keccak_js_1.keccak256)(uint8Array)).slice(2);
    for (let i = 0; i < 12; i += 4) {
        // calculate bit position in bloom filter that must be active
        const bitpos = 
        // eslint-disable-next-line no-bitwise
        ((parseInt(hash.slice(i, i + 2), 16) << 8) + parseInt(hash.slice(i + 2, i + 4), 16)) &
            2047;
        // test if bitpos in bloom is active
        const code = (0, utils_js_1.codePointToInt)(bloom.charCodeAt(bloom.length - 1 - Math.floor(bitpos / 4)));
        // eslint-disable-next-line no-bitwise
        const offset = 1 << bitpos % 4;
        // eslint-disable-next-line no-bitwise
        if ((code & offset) !== offset) {
            return false;
        }
    }
    return true;
};
exports.isInBloom = isInBloom;
/**
 * Returns true if the ethereum users address is part of the given bloom note: false positives are possible.
 */
const isUserEthereumAddressInBloom = (bloom, ethereumAddress) => {
    if (!(0, exports.isBloom)(bloom)) {
        return false;
    }
    if (!(0, address_js_1.isAddress)(ethereumAddress)) {
        return false;
    }
    // you have to pad the ethereum address to 32 bytes
    // else the bloom filter does not work
    // this is only if your matching the USERS
    // ethereum address. Contract address do not need this
    // hence why we have 2 methods
    // (0x is not in the 2nd parameter of padleft so 64 chars is fine)
    const address = (0, utils_js_1.padLeft)(ethereumAddress, 64);
    return (0, exports.isInBloom)(bloom, address);
};
exports.isUserEthereumAddressInBloom = isUserEthereumAddressInBloom;
/**
 * Returns true if the contract address is part of the given bloom.
 * note: false positives are possible.
 */
const isContractAddressInBloom = (bloom, contractAddress) => {
    if (!(0, exports.isBloom)(bloom)) {
        return false;
    }
    if (!(0, address_js_1.isAddress)(contractAddress)) {
        return false;
    }
    return (0, exports.isInBloom)(bloom, contractAddress);
};
exports.isContractAddressInBloom = isContractAddressInBloom;
//# sourceMappingURL=bloom.js.map