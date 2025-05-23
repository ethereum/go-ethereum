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
var __awaiter = (this && this.__awaiter) || function (thisArg, _arguments, P, generator) {
    function adopt(value) { return value instanceof P ? value : new P(function (resolve) { resolve(value); }); }
    return new (P || (P = Promise))(function (resolve, reject) {
        function fulfilled(value) { try { step(generator.next(value)); } catch (e) { reject(e); } }
        function rejected(value) { try { step(generator["throw"](value)); } catch (e) { reject(e); } }
        function step(result) { result.done ? resolve(result.value) : adopt(result.value).then(fulfilled, rejected); }
        step((generator = generator.apply(thisArg, _arguments || [])).next());
    });
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.createNewPendingTransactionFilter = createNewPendingTransactionFilter;
exports.createNewFilter = createNewFilter;
exports.createNewBlockFilter = createNewBlockFilter;
exports.uninstallFilter = uninstallFilter;
exports.getFilterChanges = getFilterChanges;
exports.getFilterLogs = getFilterLogs;
const web3_rpc_methods_1 = require("web3-rpc-methods");
const web3_utils_1 = require("web3-utils");
const web3_validator_1 = require("web3-validator");
const schemas_js_1 = require("./schemas.js");
/**
 * View additional documentations here: {@link Web3Eth.createNewPendingTransactionFilter}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 * @param returnFormat ({@link DataFormat}) Return format
 */
function createNewPendingTransactionFilter(web3Context, returnFormat) {
    return __awaiter(this, void 0, void 0, function* () {
        const response = yield web3_rpc_methods_1.ethRpcMethods.newPendingTransactionFilter(web3Context.requestManager);
        return (0, web3_utils_1.format)({ format: 'uint' }, response, returnFormat !== null && returnFormat !== void 0 ? returnFormat : web3Context.defaultReturnFormat);
    });
}
/**
 * View additional documentations here: {@link Web3Eth.createNewFilter}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 * @param filter ({@link FilterParam}) Filter param optional having from-block to-block address or params
 * @param returnFormat ({@link DataFormat}) Return format
 */
function createNewFilter(web3Context, filter, returnFormat) {
    return __awaiter(this, void 0, void 0, function* () {
        // format type bigint or number toBlock and fromBlock to hexstring.
        let { toBlock, fromBlock } = filter;
        if (!(0, web3_validator_1.isNullish)(toBlock)) {
            if (typeof toBlock === 'number' || typeof toBlock === 'bigint') {
                toBlock = (0, web3_utils_1.numberToHex)(toBlock);
            }
        }
        if (!(0, web3_validator_1.isNullish)(fromBlock)) {
            if (typeof fromBlock === 'number' || typeof fromBlock === 'bigint') {
                fromBlock = (0, web3_utils_1.numberToHex)(fromBlock);
            }
        }
        const formattedFilter = Object.assign(Object.assign({}, filter), { fromBlock, toBlock });
        const response = yield web3_rpc_methods_1.ethRpcMethods.newFilter(web3Context.requestManager, formattedFilter);
        return (0, web3_utils_1.format)({ format: 'uint' }, response, returnFormat !== null && returnFormat !== void 0 ? returnFormat : web3Context.defaultReturnFormat);
    });
}
/**
 * View additional documentations here: {@link Web3Eth.createNewBlockFilter}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 * @param returnFormat ({@link DataFormat}) Return format
 */
function createNewBlockFilter(web3Context, returnFormat) {
    return __awaiter(this, void 0, void 0, function* () {
        const response = yield web3_rpc_methods_1.ethRpcMethods.newBlockFilter(web3Context.requestManager);
        return (0, web3_utils_1.format)({ format: 'uint' }, response, returnFormat !== null && returnFormat !== void 0 ? returnFormat : web3Context.defaultReturnFormat);
    });
}
/**
 * View additional documentations here: {@link Web3Eth.uninstallFilter}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 * @param filterIdentifier ({@link Numbers}) filter id
 */
function uninstallFilter(web3Context, filterIdentifier) {
    return __awaiter(this, void 0, void 0, function* () {
        const response = yield web3_rpc_methods_1.ethRpcMethods.uninstallFilter(web3Context.requestManager, (0, web3_utils_1.numberToHex)(filterIdentifier));
        return response;
    });
}
/**
 * View additional documentations here: {@link Web3Eth.getFilterChanges}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 * @param filterIdentifier ({@link Numbers}) filter id
 */
function getFilterChanges(web3Context, filterIdentifier, returnFormat) {
    return __awaiter(this, void 0, void 0, function* () {
        const response = yield web3_rpc_methods_1.ethRpcMethods.getFilterChanges(web3Context.requestManager, (0, web3_utils_1.numberToHex)(filterIdentifier));
        const result = response.map(res => {
            if (typeof res === 'string') {
                return res;
            }
            return (0, web3_utils_1.format)(schemas_js_1.logSchema, res, returnFormat !== null && returnFormat !== void 0 ? returnFormat : web3Context.defaultReturnFormat);
        });
        return result;
    });
}
/**
 * View additional documentations here: {@link Web3Eth.getFilterLogs}
 * @param web3Context ({@link Web3Context}) Web3 configuration object that contains things such as the provider, request manager, wallet, etc.
 * @param filterIdentifier ({@link Numbers}) filter id
 */
function getFilterLogs(web3Context, filterIdentifier, returnFormat) {
    return __awaiter(this, void 0, void 0, function* () {
        const response = yield web3_rpc_methods_1.ethRpcMethods.getFilterLogs(web3Context.requestManager, (0, web3_utils_1.numberToHex)(filterIdentifier));
        const result = response.map(res => {
            if (typeof res === 'string') {
                return res;
            }
            return (0, web3_utils_1.format)(schemas_js_1.logSchema, res, returnFormat !== null && returnFormat !== void 0 ? returnFormat : web3Context.defaultReturnFormat);
        });
        return result;
    });
}
//# sourceMappingURL=filtering_rpc_method_wrappers.js.map