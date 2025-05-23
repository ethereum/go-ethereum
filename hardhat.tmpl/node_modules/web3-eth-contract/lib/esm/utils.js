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
import { RLP } from '@ethereumjs/rlp';
import { InvalidAddressError, InvalidMethodParamsError, InvalidNumberError, Web3ContractError, } from 'web3-errors';
import { isNullish, mergeDeep, isContractInitOptions, keccak256, toChecksumAddress, hexToNumber, } from 'web3-utils';
import { isAddress, isHexString } from 'web3-validator';
import { encodeMethodABI } from './encoding.js';
const dataInputEncodeMethodHelper = (txParams, abi, params, dataInputFill) => {
    var _a, _b;
    const tx = {};
    if (!isNullish(txParams.data) || dataInputFill === 'both') {
        tx.data = encodeMethodABI(abi, params, ((_a = txParams.data) !== null && _a !== void 0 ? _a : txParams.input));
    }
    if (!isNullish(txParams.input) || dataInputFill === 'both') {
        tx.input = encodeMethodABI(abi, params, ((_b = txParams.input) !== null && _b !== void 0 ? _b : txParams.data));
    }
    // if input and data is empty, use web3config default
    if (isNullish(tx.input) && isNullish(tx.data)) {
        tx[dataInputFill] = encodeMethodABI(abi, params);
    }
    return { data: tx.data, input: tx.input };
};
export const getSendTxParams = ({ abi, params, options, contractOptions, }) => {
    var _a, _b, _c;
    const deploymentCall = (_c = (_b = (_a = options === null || options === void 0 ? void 0 : options.input) !== null && _a !== void 0 ? _a : options === null || options === void 0 ? void 0 : options.data) !== null && _b !== void 0 ? _b : contractOptions.input) !== null && _c !== void 0 ? _c : contractOptions.data;
    if (!deploymentCall && !(options === null || options === void 0 ? void 0 : options.to) && !contractOptions.address) {
        throw new Web3ContractError('Contract address not specified');
    }
    if (!(options === null || options === void 0 ? void 0 : options.from) && !contractOptions.from) {
        throw new Web3ContractError('Contract "from" address not specified');
    }
    let txParams = mergeDeep({
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
export const getEthTxCallParams = ({ abi, params, options, contractOptions, }) => {
    if (!(options === null || options === void 0 ? void 0 : options.to) && !contractOptions.address) {
        throw new Web3ContractError('Contract address not specified');
    }
    let txParams = mergeDeep({
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
export const getEstimateGasParams = ({ abi, params, options, contractOptions, }) => {
    let txParams = mergeDeep({
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
export const isWeb3ContractContext = (options) => typeof options === 'object' &&
    !isNullish(options) &&
    Object.keys(options).length !== 0 &&
    !isContractInitOptions(options);
export const getCreateAccessListParams = ({ abi, params, options, contractOptions, }) => {
    if (!(options === null || options === void 0 ? void 0 : options.to) && !contractOptions.address) {
        throw new Web3ContractError('Contract address not specified');
    }
    if (!(options === null || options === void 0 ? void 0 : options.from) && !contractOptions.from) {
        throw new Web3ContractError('Contract "from" address not specified');
    }
    let txParams = mergeDeep({
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
export const createContractAddress = (from, nonce) => {
    if (!isAddress(from))
        throw new InvalidAddressError(`Invalid address given ${from}`);
    let nonceValue = nonce;
    if (typeof nonce === 'string' && isHexString(nonce))
        nonceValue = hexToNumber(nonce);
    else if (typeof nonce === 'string' && !isHexString(nonce))
        throw new InvalidNumberError('Invalid nonce value format');
    const rlpEncoded = RLP.encode([from, nonceValue]);
    const result = keccak256(rlpEncoded);
    const contractAddress = '0x'.concat(result.substring(26));
    return toChecksumAddress(contractAddress);
};
export const create2ContractAddress = (from, salt, initCode) => {
    if (!isAddress(from))
        throw new InvalidAddressError(`Invalid address given ${from}`);
    if (!isHexString(salt))
        throw new InvalidMethodParamsError(`Invalid salt value ${salt}`);
    if (!isHexString(initCode))
        throw new InvalidMethodParamsError(`Invalid initCode value ${initCode}`);
    const initCodeHash = keccak256(initCode);
    const initCodeHashPadded = initCodeHash.padStart(64, '0'); // Pad to 32 bytes (64 hex characters)
    const create2Params = ['0xff', from, salt, initCodeHashPadded].map(x => x.replace(/0x/, ''));
    const create2Address = `0x${create2Params.join('')}`;
    return toChecksumAddress(`0x${keccak256(create2Address).slice(26)}`); // Slice to get the last 20 bytes (40 hex characters) & checksum
};
//# sourceMappingURL=utils.js.map