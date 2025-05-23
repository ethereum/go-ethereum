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
exports.isTopicInBloom = exports.isTopic = void 0;
const bloom_js_1 = require("./bloom.js");
/**
 * Checks if its a valid topic
 */
const isTopic = (topic) => {
    if (typeof topic !== 'string') {
        return false;
    }
    if (!/^(0x)?[0-9a-f]{64}$/i.test(topic)) {
        return false;
    }
    if (/^(0x)?[0-9a-f]{64}$/.test(topic) || /^(0x)?[0-9A-F]{64}$/.test(topic)) {
        return true;
    }
    return false;
};
exports.isTopic = isTopic;
/**
 * Returns true if the topic is part of the given bloom.
 * note: false positives are possible.
 */
const isTopicInBloom = (bloom, topic) => {
    if (!(0, bloom_js_1.isBloom)(bloom)) {
        return false;
    }
    if (!(0, exports.isTopic)(topic)) {
        return false;
    }
    return (0, bloom_js_1.isInBloom)(bloom, topic);
};
exports.isTopicInBloom = isTopicInBloom;
//# sourceMappingURL=topic.js.map