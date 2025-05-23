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
exports.ContractTransactionDataAndInputError = exports.ContractExecutionError = exports.Eip838ExecutionError = exports.ContractInstantiationError = exports.ContractNoFromAddressDefinedError = exports.ContractNoAddressDefinedError = exports.ContractMissingDeployDataError = exports.ContractReservedEventError = exports.ContractEventDoesNotExistError = exports.ContractOnceRequiresCallbackError = exports.ContractMissingABIError = exports.ResolverMethodMissingError = exports.Web3ContractError = void 0;
const error_codes_js_1 = require("../error_codes.js");
const web3_error_base_js_1 = require("../web3_error_base.js");
class Web3ContractError extends web3_error_base_js_1.BaseWeb3Error {
    constructor(message, receipt) {
        super(message);
        this.code = error_codes_js_1.ERR_CONTRACT;
        this.receipt = receipt;
    }
}
exports.Web3ContractError = Web3ContractError;
class ResolverMethodMissingError extends web3_error_base_js_1.BaseWeb3Error {
    constructor(address, name) {
        super(`The resolver at ${address} does not implement requested method: "${name}".`);
        this.address = address;
        this.name = name;
        this.code = error_codes_js_1.ERR_CONTRACT_RESOLVER_MISSING;
    }
    toJSON() {
        return Object.assign(Object.assign({}, super.toJSON()), { address: this.address, name: this.name });
    }
}
exports.ResolverMethodMissingError = ResolverMethodMissingError;
class ContractMissingABIError extends web3_error_base_js_1.BaseWeb3Error {
    constructor() {
        super('You must provide the json interface of the contract when instantiating a contract object.');
        this.code = error_codes_js_1.ERR_CONTRACT_ABI_MISSING;
    }
}
exports.ContractMissingABIError = ContractMissingABIError;
class ContractOnceRequiresCallbackError extends web3_error_base_js_1.BaseWeb3Error {
    constructor() {
        super('Once requires a callback as the second parameter.');
        this.code = error_codes_js_1.ERR_CONTRACT_REQUIRED_CALLBACK;
    }
}
exports.ContractOnceRequiresCallbackError = ContractOnceRequiresCallbackError;
class ContractEventDoesNotExistError extends web3_error_base_js_1.BaseWeb3Error {
    constructor(eventName) {
        super(`Event "${eventName}" doesn't exist in this contract.`);
        this.eventName = eventName;
        this.code = error_codes_js_1.ERR_CONTRACT_EVENT_NOT_EXISTS;
    }
    toJSON() {
        return Object.assign(Object.assign({}, super.toJSON()), { eventName: this.eventName });
    }
}
exports.ContractEventDoesNotExistError = ContractEventDoesNotExistError;
class ContractReservedEventError extends web3_error_base_js_1.BaseWeb3Error {
    constructor(type) {
        super(`Event "${type}" doesn't exist in this contract.`);
        this.type = type;
        this.code = error_codes_js_1.ERR_CONTRACT_RESERVED_EVENT;
    }
    toJSON() {
        return Object.assign(Object.assign({}, super.toJSON()), { type: this.type });
    }
}
exports.ContractReservedEventError = ContractReservedEventError;
class ContractMissingDeployDataError extends web3_error_base_js_1.BaseWeb3Error {
    constructor() {
        super(`No "data" specified in neither the given options, nor the default options.`);
        this.code = error_codes_js_1.ERR_CONTRACT_MISSING_DEPLOY_DATA;
    }
}
exports.ContractMissingDeployDataError = ContractMissingDeployDataError;
class ContractNoAddressDefinedError extends web3_error_base_js_1.BaseWeb3Error {
    constructor() {
        super("This contract object doesn't have address set yet, please set an address first.");
        this.code = error_codes_js_1.ERR_CONTRACT_MISSING_ADDRESS;
    }
}
exports.ContractNoAddressDefinedError = ContractNoAddressDefinedError;
class ContractNoFromAddressDefinedError extends web3_error_base_js_1.BaseWeb3Error {
    constructor() {
        super('No "from" address specified in neither the given options, nor the default options.');
        this.code = error_codes_js_1.ERR_CONTRACT_MISSING_FROM_ADDRESS;
    }
}
exports.ContractNoFromAddressDefinedError = ContractNoFromAddressDefinedError;
class ContractInstantiationError extends web3_error_base_js_1.BaseWeb3Error {
    constructor() {
        super(...arguments);
        this.code = error_codes_js_1.ERR_CONTRACT_INSTANTIATION;
    }
}
exports.ContractInstantiationError = ContractInstantiationError;
/**
 * This class is expected to be set as an `cause` inside ContractExecutionError
 * The properties would be typically decoded from the `data` if it was encoded according to EIP-838
 */
class Eip838ExecutionError extends Web3ContractError {
    constructor(error) {
        super(error.message || 'Error');
        this.name = ('name' in error && error.name) || this.constructor.name;
        // eslint-disable-next-line @typescript-eslint/prefer-nullish-coalescing
        this.stack = ('stack' in error && error.stack) || undefined;
        this.code = error.code;
        // get embedded error details got from some providers like MetaMask
        // and set this.data from the inner error data for easier read.
        // note: the data is a hex string inside either:
        //	 error.data, error.data.data or error.data.originalError.data (https://github.com/web3/web3.js/issues/4454#issuecomment-1485953455)
        if (typeof error.data === 'object') {
            let originalError;
            if (error.data && 'originalError' in error.data) {
                originalError = error.data.originalError;
            }
            else {
                // Ganache has no `originalError` sub-object unlike others
                originalError = error.data;
            }
            this.data = originalError.data;
            this.cause = new Eip838ExecutionError(originalError);
        }
        else {
            this.data = error.data;
        }
    }
    setDecodedProperties(errorName, errorSignature, errorArgs) {
        this.errorName = errorName;
        this.errorSignature = errorSignature;
        this.errorArgs = errorArgs;
    }
    toJSON() {
        let json = Object.assign(Object.assign({}, super.toJSON()), { data: this.data });
        if (this.errorName) {
            json = Object.assign(Object.assign({}, json), { errorName: this.errorName, errorSignature: this.errorSignature, errorArgs: this.errorArgs });
        }
        return json;
    }
}
exports.Eip838ExecutionError = Eip838ExecutionError;
/**
 * Used when an error is raised while executing a function inside a smart contract.
 * The data is expected to be encoded according to EIP-848.
 */
class ContractExecutionError extends Web3ContractError {
    constructor(rpcError) {
        super('Error happened while trying to execute a function inside a smart contract');
        this.code = error_codes_js_1.ERR_CONTRACT_EXECUTION_REVERTED;
        this.cause = new Eip838ExecutionError(rpcError);
    }
}
exports.ContractExecutionError = ContractExecutionError;
class ContractTransactionDataAndInputError extends web3_error_base_js_1.InvalidValueError {
    constructor(value) {
        var _a, _b;
        super(`data: ${(_a = value.data) !== null && _a !== void 0 ? _a : 'undefined'}, input: ${(_b = value.input) !== null && _b !== void 0 ? _b : 'undefined'}`, 'You can\'t have "data" and "input" as properties of a contract at the same time, please use either "data" or "input" instead.');
        this.code = error_codes_js_1.ERR_CONTRACT_TX_DATA_AND_INPUT;
    }
}
exports.ContractTransactionDataAndInputError = ContractTransactionDataAndInputError;
//# sourceMappingURL=contract_errors.js.map