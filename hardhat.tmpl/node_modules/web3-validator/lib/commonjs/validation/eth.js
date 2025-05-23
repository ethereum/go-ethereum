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
exports.isValidEthBaseType = void 0;
const utils_js_1 = require("../utils.js");
const isValidEthBaseType = (type) => {
    const { baseType, baseTypeSize } = (0, utils_js_1.parseBaseType)(type);
    if (!baseType) {
        return false;
    }
    if (baseType === type) {
        return true;
    }
    if ((baseType === 'int' || baseType === 'uint') && baseTypeSize) {
        if (!(baseTypeSize <= 256 && baseTypeSize % 8 === 0)) {
            return false;
        }
    }
    if (baseType === 'bytes' && baseTypeSize) {
        if (!(baseTypeSize >= 1 && baseTypeSize <= 32)) {
            return false;
        }
    }
    return true;
};
exports.isValidEthBaseType = isValidEthBaseType;
//# sourceMappingURL=eth.js.map