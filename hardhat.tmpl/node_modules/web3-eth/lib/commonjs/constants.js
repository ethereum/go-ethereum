"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.NUMBER_DATA_FORMAT = exports.ALL_EVENTS_ABI = exports.ALL_EVENTS = void 0;
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
const web3_types_1 = require("web3-types");
exports.ALL_EVENTS = 'ALLEVENTS';
exports.ALL_EVENTS_ABI = {
    name: exports.ALL_EVENTS,
    signature: '',
    type: 'event',
    inputs: [],
};
exports.NUMBER_DATA_FORMAT = { bytes: web3_types_1.FMT_BYTES.HEX, number: web3_types_1.FMT_NUMBER.NUMBER };
//# sourceMappingURL=constants.js.map