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
import { format } from 'web3-utils';
import { DEFAULT_RETURN_FORMAT, } from 'web3-types';
import { decodeLog } from 'web3-eth-abi';
import { logSchema } from '../schemas.js';
import { ALL_EVENTS } from '../constants.js';
export const decodeEventABI = (event, data, jsonInterface, returnFormat = DEFAULT_RETURN_FORMAT) => {
    var _a, _b, _c, _d, _e, _f;
    let modifiedEvent = Object.assign({}, event);
    const result = format(logSchema, data, returnFormat);
    // if allEvents get the right event
    if ([ALL_EVENTS, 'allEvents'].includes(modifiedEvent.name)) {
        const matchedEvent = jsonInterface.find(j => j.signature === data.topics[0]);
        if (matchedEvent) {
            modifiedEvent = matchedEvent;
        }
        else {
            modifiedEvent = { anonymous: true };
        }
    }
    // create empty inputs if none are present (e.g. anonymous events on allEvents)
    modifiedEvent.inputs = (_b = (_a = modifiedEvent.inputs) !== null && _a !== void 0 ? _a : event.inputs) !== null && _b !== void 0 ? _b : [];
    // Handle case where an event signature shadows the current ABI with non-identical
    // arg indexing. If # of topics doesn't match, event is anon.
    if (!modifiedEvent.anonymous) {
        let indexedInputs = 0;
        ((_c = modifiedEvent.inputs) !== null && _c !== void 0 ? _c : []).forEach(input => {
            if (input.indexed) {
                indexedInputs += 1;
            }
        });
        if (indexedInputs > 0 && (data === null || data === void 0 ? void 0 : data.topics) && (data === null || data === void 0 ? void 0 : data.topics.length) !== indexedInputs + 1) {
            // checks if event is anonymous
            modifiedEvent = Object.assign(Object.assign({}, modifiedEvent), { anonymous: true, inputs: [] });
        }
    }
    const argTopics = modifiedEvent.anonymous ? data.topics : ((_d = data.topics) !== null && _d !== void 0 ? _d : []).slice(1);
    return Object.assign(Object.assign({}, result), { returnValues: decodeLog([...((_e = modifiedEvent.inputs) !== null && _e !== void 0 ? _e : [])], data.data, argTopics), event: modifiedEvent.name, signature: !modifiedEvent.anonymous && ((_f = data.topics) === null || _f === void 0 ? void 0 : _f.length) > 0 && data.topics[0]
            ? data.topics[0]
            : undefined, raw: {
            data: data.data,
            topics: data.topics,
        } });
};
//# sourceMappingURL=decoding.js.map