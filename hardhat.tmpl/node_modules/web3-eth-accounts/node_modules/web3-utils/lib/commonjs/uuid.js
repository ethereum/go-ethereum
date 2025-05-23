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
exports.uuidV4 = void 0;
/**
 * @module Utils
 */
const converters_js_1 = require("./converters.js");
const random_js_1 = require("./random.js");
/**
 * Generate a version 4 (random) uuid
 * https://github.com/uuidjs/uuid/blob/main/src/v4.js#L5
 * @returns - A version 4 uuid of the form xxxxxxxx-xxxx-4xxx-xxxx-xxxxxxxxxxxx
 * @example
 * ```ts
 * console.log(web3.utils.uuidV4());
 * > "1b9d6bcd-bbfd-4b2d-9b5d-ab8dfbbd4bed"
 * ```
 */
const uuidV4 = () => {
    const bytes = (0, random_js_1.randomBytes)(16);
    // https://github.com/ethers-io/ethers.js/blob/ce8f1e4015c0f27bf178238770b1325136e3351a/packages/json-wallets/src.ts/utils.ts#L54
    // Section: 4.1.3:
    // - time_hi_and_version[12:16] = 0b0100
    /* eslint-disable-next-line */
    bytes[6] = (bytes[6] & 0x0f) | 0x40;
    // Section 4.4
    // - clock_seq_hi_and_reserved[6] = 0b0
    // - clock_seq_hi_and_reserved[7] = 0b1
    /* eslint-disable-next-line */
    bytes[8] = (bytes[8] & 0x3f) | 0x80;
    const hexString = (0, converters_js_1.bytesToHex)(bytes);
    return [
        hexString.substring(2, 10),
        hexString.substring(10, 14),
        hexString.substring(14, 18),
        hexString.substring(18, 22),
        hexString.substring(22, 34),
    ].join('-');
};
exports.uuidV4 = uuidV4;
//# sourceMappingURL=uuid.js.map