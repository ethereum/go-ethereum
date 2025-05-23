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
exports.decodeParameters = decodeParameters;
const web3_validator_1 = require("web3-validator");
const tuple_js_1 = require("./base/tuple.js");
const utils_js_1 = require("./utils.js");
function decodeParameters(abis, bytes, _loose) {
    const abiParams = (0, utils_js_1.toAbiParams)(abis);
    const bytesArray = web3_validator_1.utils.hexToUint8Array(bytes);
    return (0, tuple_js_1.decodeTuple)({ type: 'tuple', name: '', components: abiParams }, bytesArray).result;
}
//# sourceMappingURL=decode.js.map