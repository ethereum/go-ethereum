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
exports.mergeDeep = void 0;
const web3_types_1 = require("web3-types");
const web3_validator_1 = require("web3-validator");
const isIterable = (item) => typeof item === 'object' &&
    !(0, web3_validator_1.isNullish)(item) &&
    !Array.isArray(item) &&
    !(item instanceof web3_types_1.TypedArray);
// The following code is a derivative work of the code from the "LiskHQ/lisk-sdk" project,
// which is licensed under Apache version 2.
/**
 * Deep merge two objects.
 * @param destination - The destination object.
 * @param sources - An array of source objects.
 * @returns - The merged object.
 */
const mergeDeep = (destination, ...sources) => {
    if (!isIterable(destination)) {
        return destination;
    }
    const result = Object.assign({}, destination); // clone deep here
    for (const src of sources) {
        // const src = { ..._src };
        // eslint-disable-next-line no-restricted-syntax
        for (const key in src) {
            if (isIterable(src[key])) {
                if (!result[key]) {
                    result[key] = {};
                }
                result[key] = (0, exports.mergeDeep)(result[key], src[key]);
            }
            else if (!(0, web3_validator_1.isNullish)(src[key]) && Object.hasOwnProperty.call(src, key)) {
                if (Array.isArray(src[key]) || src[key] instanceof web3_types_1.TypedArray) {
                    result[key] = src[key].slice(0);
                }
                else {
                    result[key] = src[key];
                }
            }
        }
    }
    return result;
};
exports.mergeDeep = mergeDeep;
//# sourceMappingURL=objects.js.map