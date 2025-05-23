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
exports.namehash = exports.normalize = void 0;
const web3_utils_1 = require("web3-utils");
// eslint-disable-next-line camelcase
const ens_normalize_1 = require("@adraffy/ens-normalize");
const normalize = (name) => (0, ens_normalize_1.ens_normalize)(name);
exports.normalize = normalize;
const namehash = (inputName) => {
    // Reject empty names:
    let node = '';
    for (let i = 0; i < 32; i += 1) {
        node += '00';
    }
    if (inputName) {
        const name = (0, exports.normalize)(inputName);
        const labels = name.split('.');
        for (let i = labels.length - 1; i >= 0; i -= 1) {
            const labelSha = (0, web3_utils_1.sha3Raw)(labels[i]).slice(2);
            node = (0, web3_utils_1.sha3Raw)(`0x${node}${labelSha}`).slice(2);
        }
    }
    return `0x${node}`;
};
exports.namehash = namehash;
//# sourceMappingURL=utils.js.map