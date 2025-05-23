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
exports.isObject = exports.isNullish = void 0;
const web3_types_1 = require("web3-types");
// Explicitly check for the
// eslint-disable-next-line @typescript-eslint/ban-types
const isNullish = (item) => 
// Using "null" value intentionally for validation
// eslint-disable-next-line no-null/no-null
item === undefined || item === null;
exports.isNullish = isNullish;
const isObject = (item) => typeof item === 'object' &&
    !(0, exports.isNullish)(item) &&
    !Array.isArray(item) &&
    !(item instanceof web3_types_1.TypedArray);
exports.isObject = isObject;
//# sourceMappingURL=object.js.map