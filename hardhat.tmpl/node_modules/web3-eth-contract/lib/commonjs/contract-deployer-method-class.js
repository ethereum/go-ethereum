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
exports.DeployerMethodClass = void 0;
const web3_errors_1 = require("web3-errors");
const web3_eth_1 = require("web3-eth");
const web3_eth_abi_1 = require("web3-eth-abi");
const web3_types_1 = require("web3-types");
const web3_utils_1 = require("web3-utils");
const web3_validator_1 = require("web3-validator");
const encoding_js_1 = require("./encoding.js");
const utils_js_1 = require("./utils.js");
/*
 * This class is only supposed to be used for the return of `new Contract(...).deploy(...)` method.
 */
class DeployerMethodClass {
    _contractMethodDeploySend(tx) {
        // eslint-disable-next-line no-use-before-define
        const returnTxOptions = {
            transactionResolver: (receipt) => {
                if (receipt.status === BigInt(0)) {
                    throw new web3_errors_1.Web3ContractError("code couldn't be stored", receipt);
                }
                const newContract = this.parent.clone();
                // eslint-disable-next-line @typescript-eslint/no-unsafe-member-access
                newContract.options.address = receipt.contractAddress;
                return newContract;
            },
            contractAbi: this.parent.options.jsonInterface,
            // TODO Should make this configurable by the user
            checkRevertBeforeSending: false,
        };
        return (0, web3_validator_1.isNullish)(this.parent.getTransactionMiddleware())
            ? (0, web3_eth_1.sendTransaction)(this.parent, tx, this.parent.defaultReturnFormat, returnTxOptions) // not calling this with undefined Middleware because it will not break if Eth package is not updated
            : (0, web3_eth_1.sendTransaction)(this.parent, tx, this.parent.defaultReturnFormat, returnTxOptions, this.parent.getTransactionMiddleware());
    }
    constructor(
    // eslint-disable-next-line no-use-before-define
    parent, deployOptions) {
        this.parent = parent;
        this.deployOptions = deployOptions;
        const { args, abi, contractOptions, deployData } = this.calculateDeployParams();
        this.args = args;
        this.constructorAbi = abi;
        this.contractOptions = contractOptions;
        this.deployData = deployData;
    }
    send(options) {
        const modifiedOptions = Object.assign({}, options);
        const tx = this.populateTransaction(modifiedOptions);
        return this._contractMethodDeploySend(tx);
    }
    populateTransaction(txOptions) {
        var _a, _b;
        const modifiedContractOptions = Object.assign(Object.assign({}, this.contractOptions), { from: (_b = (_a = this.contractOptions.from) !== null && _a !== void 0 ? _a : this.parent.defaultAccount) !== null && _b !== void 0 ? _b : undefined });
        // args, abi, contractOptions, deployData
        const tx = (0, utils_js_1.getSendTxParams)({
            abi: this.constructorAbi,
            params: this.args,
            options: Object.assign(Object.assign({}, txOptions), { dataInputFill: this.parent.contractDataInputFill }),
            contractOptions: modifiedContractOptions,
        });
        // @ts-expect-error remove unnecessary field
        if (tx.dataInputFill) {
            // @ts-expect-error remove unnecessary field
            delete tx.dataInputFill;
        }
        return tx;
    }
    calculateDeployParams() {
        var _a, _b, _c, _d, _e, _f;
        let abi = this.parent.options.jsonInterface.find(j => j.type === 'constructor');
        if (!abi) {
            abi = {
                type: 'constructor',
                stateMutability: '',
            };
        }
        const _input = (0, web3_utils_1.format)({ format: 'bytes' }, (_b = (_a = this.deployOptions) === null || _a === void 0 ? void 0 : _a.input) !== null && _b !== void 0 ? _b : this.parent.options.input, web3_types_1.DEFAULT_RETURN_FORMAT);
        const _data = (0, web3_utils_1.format)({ format: 'bytes' }, (_d = (_c = this.deployOptions) === null || _c === void 0 ? void 0 : _c.data) !== null && _d !== void 0 ? _d : this.parent.options.data, web3_types_1.DEFAULT_RETURN_FORMAT);
        if ((!_input || _input.trim() === '0x') && (!_data || _data.trim() === '0x')) {
            throw new web3_errors_1.Web3ContractError('contract creation without any data provided.');
        }
        const args = (_f = (_e = this.deployOptions) === null || _e === void 0 ? void 0 : _e.arguments) !== null && _f !== void 0 ? _f : [];
        const contractOptions = Object.assign(Object.assign({}, this.parent.options), { input: _input, data: _data });
        const deployData = _input !== null && _input !== void 0 ? _input : _data;
        return { args, abi, contractOptions, deployData };
    }
    estimateGas(options_1) {
        return __awaiter(this, arguments, void 0, function* (options, returnFormat = this.parent.defaultReturnFormat) {
            const modifiedOptions = Object.assign({}, options);
            return this.parent.contractMethodEstimateGas({
                abi: this.constructorAbi,
                params: this.args,
                returnFormat,
                options: modifiedOptions,
                contractOptions: this.contractOptions,
            });
        });
    }
    encodeABI() {
        return (0, encoding_js_1.encodeMethodABI)(this.constructorAbi, this.args, (0, web3_utils_1.format)({ format: 'bytes' }, this.deployData, this.parent.defaultReturnFormat));
    }
    decodeData(data) {
        return Object.assign(Object.assign({}, (0, web3_eth_abi_1.decodeFunctionCall)(this.constructorAbi, data.replace(this.deployData, ''), false)), { __method__: this.constructorAbi.type });
    }
}
exports.DeployerMethodClass = DeployerMethodClass;
//# sourceMappingURL=contract-deployer-method-class.js.map