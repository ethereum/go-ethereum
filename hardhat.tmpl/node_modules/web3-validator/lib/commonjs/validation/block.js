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
exports.isBlockNumberOrTag = exports.isBlockTag = exports.isBlockNumber = void 0;
const web3_types_1 = require("web3-types");
const numbers_js_1 = require("./numbers.js");
const isBlockNumber = (value) => (0, numbers_js_1.isUInt)(value);
exports.isBlockNumber = isBlockNumber;
/**
 * Returns true if the given blockNumber is 'latest', 'pending', 'earliest, 'safe' or 'finalized'
 */
const isBlockTag = (value) => Object.values(web3_types_1.BlockTags).includes(value);
exports.isBlockTag = isBlockTag;
/**
 * Returns true if given value is valid hex string and not negative, or is a valid BlockTag
 */
const isBlockNumberOrTag = (value) => (0, exports.isBlockTag)(value) || (0, exports.isBlockNumber)(value);
exports.isBlockNumberOrTag = isBlockNumberOrTag;
//# sourceMappingURL=block.js.map