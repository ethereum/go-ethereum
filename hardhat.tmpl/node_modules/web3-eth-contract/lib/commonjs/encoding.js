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
exports.decodeMethodReturn = exports.decodeMethodParams = exports.encodeMethodABI = exports.encodeEventABI = exports.decodeEventABI = void 0;
const web3_utils_1 = require("web3-utils");
const web3_types_1 = require("web3-types");
const web3_eth_abi_1 = require("web3-eth-abi");
const web3_eth_1 = require("web3-eth");
const web3_errors_1 = require("web3-errors");
var web3_eth_2 = require("web3-eth");
Object.defineProperty(exports, "decodeEventABI", { enumerable: true, get: function () { return web3_eth_2.decodeEventABI; } });
const encodeEventABI = ({ address }, event, options) => {
    var _a, _b;
    const topics = options === null || options === void 0 ? void 0 : options.topics;
    const filter = (_a = options === null || options === void 0 ? void 0 : options.filter) !== null && _a !== void 0 ? _a : {};
    const opts = {};
    if (!(0, web3_utils_1.isNullish)(options === null || options === void 0 ? void 0 : options.fromBlock)) {
        opts.fromBlock = (0, web3_utils_1.format)(web3_eth_1.blockSchema.properties.number, options === null || options === void 0 ? void 0 : options.fromBlock, {
            number: web3_types_1.FMT_NUMBER.HEX,
            bytes: web3_types_1.FMT_BYTES.HEX,
        });
    }
    if (!(0, web3_utils_1.isNullish)(options === null || options === void 0 ? void 0 : options.toBlock)) {
        opts.toBlock = (0, web3_utils_1.format)(web3_eth_1.blockSchema.properties.number, options === null || options === void 0 ? void 0 : options.toBlock, {
            number: web3_types_1.FMT_NUMBER.HEX,
            bytes: web3_types_1.FMT_BYTES.HEX,
        });
    }
    if (topics && Array.isArray(topics)) {
        opts.topics = [...topics];
    }
    else {
        opts.topics = [];
        // add event signature
        if (event && !event.anonymous && ![web3_eth_1.ALL_EVENTS, 'allEvents'].includes(event.name)) {
            opts.topics.push((_b = event.signature) !== null && _b !== void 0 ? _b : (0, web3_eth_abi_1.encodeEventSignature)((0, web3_eth_abi_1.jsonInterfaceMethodToString)(event)));
        }
        // add event topics (indexed arguments)
        if (![web3_eth_1.ALL_EVENTS, 'allEvents'].includes(event.name) && event.inputs) {
            for (const input of event.inputs) {
                if (!input.indexed) {
                    continue;
                }
                const value = filter[input.name];
                if (!value) {
                    // eslint-disable-next-line no-null/no-null
                    opts.topics.push(null);
                    continue;
                }
                // TODO: https://github.com/ethereum/web3.js/issues/344
                // TODO: deal properly with components
                if (Array.isArray(value)) {
                    opts.topics.push(value.map(v => (0, web3_eth_abi_1.encodeParameter)(input.type, v)));
                }
                else if (input.type === 'string') {
                    opts.topics.push((0, web3_utils_1.keccak256)(value));
                }
                else {
                    opts.topics.push((0, web3_eth_abi_1.encodeParameter)(input.type, value));
                }
            }
        }
    }
    if (!opts.topics.length)
        delete opts.topics;
    if (address) {
        opts.address = address.toLowerCase();
    }
    return opts;
};
exports.encodeEventABI = encodeEventABI;
const encodeMethodABI = (abi, args, deployData) => {
    const inputLength = Array.isArray(abi.inputs) ? abi.inputs.length : 0;
    if (abi.inputs && inputLength !== args.length) {
        throw new web3_errors_1.Web3ContractError(`The number of arguments is not matching the methods required number. You need to pass ${inputLength} arguments.`);
    }
    let params;
    if (abi.inputs) {
        params = (0, web3_eth_abi_1.encodeParameters)(Array.isArray(abi.inputs) ? abi.inputs : [], args).replace('0x', '');
    }
    else {
        params = (0, web3_eth_abi_1.inferTypesAndEncodeParameters)(args).replace('0x', '');
    }
    if ((0, web3_eth_abi_1.isAbiConstructorFragment)(abi)) {
        if (!deployData)
            throw new web3_errors_1.Web3ContractError('The contract has no contract data option set. This is necessary to append the constructor parameters.');
        if (!deployData.startsWith('0x')) {
            return `0x${deployData}${params}`;
        }
        return `${deployData}${params}`;
    }
    return `${(0, web3_eth_abi_1.encodeFunctionSignature)(abi)}${params}`;
};
exports.encodeMethodABI = encodeMethodABI;
/** @deprecated import `decodeFunctionCall` from ''web3-eth-abi' instead. */
exports.decodeMethodParams = web3_eth_abi_1.decodeFunctionCall;
/** @deprecated import `decodeFunctionReturn` from ''web3-eth-abi' instead. */
exports.decodeMethodReturn = web3_eth_abi_1.decodeFunctionReturn;
//# sourceMappingURL=encoding.js.map