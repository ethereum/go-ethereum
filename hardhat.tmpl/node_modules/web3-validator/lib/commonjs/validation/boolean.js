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
exports.isBoolean = void 0;
const string_js_1 = require("./string.js");
const isBoolean = (value) => {
    if (!['number', 'string', 'boolean'].includes(typeof value)) {
        return false;
    }
    if (typeof value === 'boolean') {
        return true;
    }
    if (typeof value === 'string' && !(0, string_js_1.isHexStrict)(value)) {
        return value === '1' || value === '0';
    }
    if (typeof value === 'string' && (0, string_js_1.isHexStrict)(value)) {
        return value === '0x1' || value === '0x0';
    }
    // type === number
    return value === 1 || value === 0;
};
exports.isBoolean = isBoolean;
//# sourceMappingURL=boolean.js.map