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
import { isAddress } from './address.js';
import { isBlockNumberOrTag } from './block.js';
import { isNullish } from './object.js';
import { isTopic } from './topic.js';
/**
 * First we check if all properties in the provided value are expected,
 * then because all Filter properties are optional, we check if the expected properties
 * are defined. If defined and they're not the expected type, we immediately return false,
 * otherwise we return true after all checks pass.
 */
export const isFilterObject = (value) => {
    const expectedFilterProperties = [
        'fromBlock',
        'toBlock',
        'address',
        'topics',
        'blockHash',
    ];
    if (isNullish(value) || typeof value !== 'object')
        return false;
    if (!Object.keys(value).every(property => expectedFilterProperties.includes(property)))
        return false;
    if ((!isNullish(value.fromBlock) && !isBlockNumberOrTag(value.fromBlock)) ||
        (!isNullish(value.toBlock) && !isBlockNumberOrTag(value.toBlock)))
        return false;
    if (!isNullish(value.address)) {
        if (Array.isArray(value.address)) {
            if (!value.address.every(address => isAddress(address)))
                return false;
        }
        else if (!isAddress(value.address))
            return false;
    }
    if (!isNullish(value.topics)) {
        if (!value.topics.every(topic => {
            if (isNullish(topic))
                return true;
            if (Array.isArray(topic)) {
                return topic.every(nestedTopic => isTopic(nestedTopic));
            }
            if (isTopic(topic))
                return true;
            return false;
        }))
            return false;
    }
    return true;
};
//# sourceMappingURL=filter.js.map