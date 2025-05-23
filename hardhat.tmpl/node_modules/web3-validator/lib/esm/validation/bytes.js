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
import { hexToUint8Array, parseBaseType } from '../utils.js';
import { isHexStrict } from './string.js';
/**
 * checks input if typeof data is valid Uint8Array input
 */
export const isUint8Array = (data) => { var _a, _b; return data instanceof Uint8Array || ((_a = data === null || data === void 0 ? void 0 : data.constructor) === null || _a === void 0 ? void 0 : _a.name) === 'Uint8Array' || ((_b = data === null || data === void 0 ? void 0 : data.constructor) === null || _b === void 0 ? void 0 : _b.name) === 'Buffer'; };
export const isBytes = (value, options = {
    abiType: 'bytes',
}) => {
    if (typeof value !== 'string' && !Array.isArray(value) && !isUint8Array(value)) {
        return false;
    }
    // isHexStrict also accepts - prefix which can not exists in bytes
    if (typeof value === 'string' && isHexStrict(value) && value.startsWith('-')) {
        return false;
    }
    if (typeof value === 'string' && !isHexStrict(value)) {
        return false;
    }
    let valueToCheck;
    if (typeof value === 'string') {
        if (value.length % 2 !== 0) {
            // odd length hex
            return false;
        }
        valueToCheck = hexToUint8Array(value);
    }
    else if (Array.isArray(value)) {
        if (value.some(d => d < 0 || d > 255 || !Number.isInteger(d))) {
            return false;
        }
        valueToCheck = new Uint8Array(value);
    }
    else {
        valueToCheck = value;
    }
    if (options === null || options === void 0 ? void 0 : options.abiType) {
        const { baseTypeSize } = parseBaseType(options.abiType);
        return baseTypeSize ? valueToCheck.length === baseTypeSize : true;
    }
    if (options === null || options === void 0 ? void 0 : options.size) {
        return valueToCheck.length === (options === null || options === void 0 ? void 0 : options.size);
    }
    return true;
};
//# sourceMappingURL=bytes.js.map