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
exports.isFilterObject = void 0;
const address_js_1 = require("./address.js");
const block_js_1 = require("./block.js");
const object_js_1 = require("./object.js");
const topic_js_1 = require("./topic.js");
/**
 * First we check if all properties in the provided value are expected,
 * then because all Filter properties are optional, we check if the expected properties
 * are defined. If defined and they're not the expected type, we immediately return false,
 * otherwise we return true after all checks pass.
 */
const isFilterObject = (value) => {
    const expectedFilterProperties = [
        'fromBlock',
        'toBlock',
        'address',
        'topics',
        'blockHash',
    ];
    if ((0, object_js_1.isNullish)(value) || typeof value !== 'object')
        return false;
    if (!Object.keys(value).every(property => expectedFilterProperties.includes(property)))
        return false;
    if ((!(0, object_js_1.isNullish)(value.fromBlock) && !(0, block_js_1.isBlockNumberOrTag)(value.fromBlock)) ||
        (!(0, object_js_1.isNullish)(value.toBlock) && !(0, block_js_1.isBlockNumberOrTag)(value.toBlock)))
        return false;
    if (!(0, object_js_1.isNullish)(value.address)) {
        if (Array.isArray(value.address)) {
            if (!value.address.every(address => (0, address_js_1.isAddress)(address)))
                return false;
        }
        else if (!(0, address_js_1.isAddress)(value.address))
            return false;
    }
    if (!(0, object_js_1.isNullish)(value.topics)) {
        if (!value.topics.every(topic => {
            if ((0, object_js_1.isNullish)(topic))
                return true;
            if (Array.isArray(topic)) {
                return topic.every(nestedTopic => (0, topic_js_1.isTopic)(nestedTopic));
            }
            if ((0, topic_js_1.isTopic)(topic))
                return true;
            return false;
        }))
            return false;
    }
    return true;
};
exports.isFilterObject = isFilterObject;
//# sourceMappingURL=filter.js.map