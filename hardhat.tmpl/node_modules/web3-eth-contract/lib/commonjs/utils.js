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
exports.create2ContractAddress = exports.createContractAddress = exports.getCreateAccessListParams = exports.isWeb3ContractContext = exports.getEstimateGasParams = exports.getEthTxCallParams = exports.getSendTxParams = void 0;
const rlp_1 = require("@ethereumjs/rlp");
const web3_errors_1 = require("web3-errors");
const web3_utils_1 = require("web3-utils");
const web3_validator_1 = require("web3-validator");
const encoding_js_1 = require("./encoding.js");
const dataInputEncodeMethodHelper = (txParams, abi, params, dataInputFill) => {
    var _a, _b;
    const tx = {};
    if (!(0, web3_utils_1.isNullish)(txParams.data) || dataInputFill === 'both') {
        tx.data = (0, encoding_js_1.encodeMethodABI)(abi, params, ((_a = txParams.data) !== null && _a !== void 0 ? _a : txParams.input));
    }
    if (!(0, web3_utils_1.isNullish)(txParams.input) || dataInputFill === 'both') {
        tx.input = (0, encoding_js_1.encodeMethodABI)(abi, params, ((_b = txParams.input) !== null && _b !== void 0 ? _b : txParams.data));
    }
    // if input and data is empty, use web3config default
    if ((0, web3_utils_1.isNullish)(tx.input) && (0, web3_utils_1.isNullish)(tx.data)) {
        tx[dataInputFill] = (0, encoding_js_1.encodeMethodABI)(abi, params);
    }
    return { data: tx.data, input: tx.input };
};
const getSendTxParams = ({ abi, params, options, contractOptions, }) => {
    var _a, _b, _c;
    const deploymentCall = (_c = (_b = (_a = options === null || options === void 0 ? void 0 : options.input) !== null && _a !== void 0 ? _a : options === null || options === void 0 ? void 0 : options.data) !== null && _b !== void 0 ? _b : contractOptions.input) !== null && _c !== void 0 ? _c : contractOptions.data;
    if (!deploymentCall && !(options === null || options === void 0 ? void 0 : options.to) && !contractOptions.address) {
        throw new web3_errors_1.Web3ContractError('Contract address not specified');
    }
    if (!(options === null || options === void 0 ? void 0 : options.from) && !contractOptions.from) {
        throw new web3_errors_1.Web3ContractError('Contract "from" address not specified');
    }
    let txParams = (0, web3_utils_1.mergeDeep)({
        to: contractOptions.address,
        gas: contractOptions.gas,
        gasPrice: contractOptions.gasPrice,
        from: contractOptions.from,
        input: contractOptions.input,
        maxPriorityFeePerGas: contractOptions.maxPriorityFeePerGas,
        maxFeePerGas: contractOptions.maxFeePerGas,
        data: contractOptions.data,
    }, options);
    const dataInput = dataInputEncodeMethodHelper(txParams, abi, params, options === null || options === void 0 ? void 0 : options.dataInputFill);
    txParams = Object.assign(Object.assign({}, txParams), { data: dataInput.data, input: dataInput.input });
    return txParams;
};
exports.getSendTxParams = getSendTxParams;
const getEthTxCallParams = ({ abi, params, options, contractOptions, }) => {
    if (!(options === null || options === void 0 ? void 0 : options.to) && !contractOptions.address) {
        throw new web3_errors_1.Web3ContractError('Contract address not specified');
    }
    let txParams = (0, web3_utils_1.mergeDeep)({
        to: contractOptions.address,
        gas: contractOptions.gas,
        gasPrice: contractOptions.gasPrice,
        from: contractOptions.from,
        input: contractOptions.input,
        maxPriorityFeePerGas: contractOptions.maxPriorityFeePerGas,
        maxFeePerGas: contractOptions.maxFeePerGas,
        data: contractOptions.data,
    }, options);
    const dataInput = dataInputEncodeMethodHelper(txParams, abi, params, options === null || options === void 0 ? void 0 : options.dataInputFill);
    txParams = Object.assign(Object.assign({}, txParams), { data: dataInput.data, input: dataInput.input });
    return txParams;
};
exports.getEthTxCallParams = getEthTxCallParams;
const getEstimateGasParams = ({ abi, params, options, contractOptions, }) => {
    let txParams = (0, web3_utils_1.mergeDeep)({
        to: contractOptions.address,
        gas: contractOptions.gas,
        gasPrice: contractOptions.gasPrice,
        from: contractOptions.from,
        input: contractOptions.input,
        data: contractOptions.data,
    }, options);
    const dataInput = dataInputEncodeMethodHelper(txParams, abi, params, options === null || options === void 0 ? void 0 : options.dataInputFill);
    txParams = Object.assign(Object.assign({}, txParams), { data: dataInput.data, input: dataInput.input });
    return txParams;
};
exports.getEstimateGasParams = getEstimateGasParams;
const isWeb3ContractContext = (options) => typeof options === 'object' &&
    !(0, web3_utils_1.isNullish)(options) &&
    Object.keys(options).length !== 0 &&
    !(0, web3_utils_1.isContractInitOptions)(options);
exports.isWeb3ContractContext = isWeb3ContractContext;
const getCreateAccessListParams = ({ abi, params, options, contractOptions, }) => {
    if (!(options === null || options === void 0 ? void 0 : options.to) && !contractOptions.address) {
        throw new web3_errors_1.Web3ContractError('Contract address not specified');
    }
    if (!(options === null || options === void 0 ? void 0 : options.from) && !contractOptions.from) {
        throw new web3_errors_1.Web3ContractError('Contract "from" address not specified');
    }
    let txParams = (0, web3_utils_1.mergeDeep)({
        to: contractOptions.address,
        gas: contractOptions.gas,
        gasPrice: contractOptions.gasPrice,
        from: contractOptions.from,
        input: contractOptions.input,
        maxPriorityFeePerGas: contractOptions.maxPriorityFeePerGas,
        maxFeePerGas: contractOptions.maxFeePerGas,
        data: contractOptions.data,
    }, options);
    const dataInput = dataInputEncodeMethodHelper(txParams, abi, params, options === null || options === void 0 ? void 0 : options.dataInputFill);
    txParams = Object.assign(Object.assign({}, txParams), { data: dataInput.data, input: dataInput.input });
    return txParams;
};
exports.getCreateAccessListParams = getCreateAccessListParams;
const createContractAddress = (from, nonce) => {
    if (!(0, web3_validator_1.isAddress)(from))
        throw new web3_errors_1.InvalidAddressError(`Invalid address given ${from}`);
    let nonceValue = nonce;
    if (typeof nonce === 'string' && (0, web3_validator_1.isHexString)(nonce))
        nonceValue = (0, web3_utils_1.hexToNumber)(nonce);
    else if (typeof nonce === 'string' && !(0, web3_validator_1.isHexString)(nonce))
        throw new web3_errors_1.InvalidNumberError('Invalid nonce value format');
    const rlpEncoded = rlp_1.RLP.encode([from, nonceValue]);
    const result = (0, web3_utils_1.keccak256)(rlpEncoded);
    const contractAddress = '0x'.concat(result.substring(26));
    return (0, web3_utils_1.toChecksumAddress)(contractAddress);
};
exports.createContractAddress = createContractAddress;
const create2ContractAddress = (from, salt, initCode) => {
    if (!(0, web3_validator_1.isAddress)(from))
        throw new web3_errors_1.InvalidAddressError(`Invalid address given ${from}`);
    if (!(0, web3_validator_1.isHexString)(salt))
        throw new web3_errors_1.InvalidMethodParamsError(`Invalid salt value ${salt}`);
    if (!(0, web3_validator_1.isHexString)(initCode))
        throw new web3_errors_1.InvalidMethodParamsError(`Invalid initCode value ${initCode}`);
    const initCodeHash = (0, web3_utils_1.keccak256)(initCode);
    const initCodeHashPadded = initCodeHash.padStart(64, '0'); // Pad to 32 bytes (64 hex characters)
    const create2Params = ['0xff', from, salt, initCodeHashPadded].map(x => x.replace(/0x/, ''));
    const create2Address = `0x${create2Params.join('')}`;
    return (0, web3_utils_1.toChecksumAddress)(`0x${(0, web3_utils_1.keccak256)(create2Address).slice(26)}`); // Slice to get the last 20 bytes (40 hex characters) & checksum
};
exports.create2ContractAddress = create2ContractAddress;
//# sourceMappingURL=utils.js.map