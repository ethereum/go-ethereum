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
import { ERR_CONTRACT, ERR_CONTRACT_ABI_MISSING, ERR_CONTRACT_EXECUTION_REVERTED, ERR_CONTRACT_EVENT_NOT_EXISTS, ERR_CONTRACT_INSTANTIATION, ERR_CONTRACT_MISSING_ADDRESS, ERR_CONTRACT_MISSING_DEPLOY_DATA, ERR_CONTRACT_MISSING_FROM_ADDRESS, ERR_CONTRACT_REQUIRED_CALLBACK, ERR_CONTRACT_RESERVED_EVENT, ERR_CONTRACT_RESOLVER_MISSING, ERR_CONTRACT_TX_DATA_AND_INPUT, } from '../error_codes.js';
import { BaseWeb3Error, InvalidValueError } from '../web3_error_base.js';
export class Web3ContractError extends BaseWeb3Error {
    constructor(message, receipt) {
        super(message);
        this.code = ERR_CONTRACT;
        this.receipt = receipt;
    }
}
export class ResolverMethodMissingError extends BaseWeb3Error {
    constructor(address, name) {
        super(`The resolver at ${address} does not implement requested method: "${name}".`);
        this.address = address;
        this.name = name;
        this.code = ERR_CONTRACT_RESOLVER_MISSING;
    }
    toJSON() {
        return Object.assign(Object.assign({}, super.toJSON()), { address: this.address, name: this.name });
    }
}
export class ContractMissingABIError extends BaseWeb3Error {
    constructor() {
        super('You must provide the json interface of the contract when instantiating a contract object.');
        this.code = ERR_CONTRACT_ABI_MISSING;
    }
}
export class ContractOnceRequiresCallbackError extends BaseWeb3Error {
    constructor() {
        super('Once requires a callback as the second parameter.');
        this.code = ERR_CONTRACT_REQUIRED_CALLBACK;
    }
}
export class ContractEventDoesNotExistError extends BaseWeb3Error {
    constructor(eventName) {
        super(`Event "${eventName}" doesn't exist in this contract.`);
        this.eventName = eventName;
        this.code = ERR_CONTRACT_EVENT_NOT_EXISTS;
    }
    toJSON() {
        return Object.assign(Object.assign({}, super.toJSON()), { eventName: this.eventName });
    }
}
export class ContractReservedEventError extends BaseWeb3Error {
    constructor(type) {
        super(`Event "${type}" doesn't exist in this contract.`);
        this.type = type;
        this.code = ERR_CONTRACT_RESERVED_EVENT;
    }
    toJSON() {
        return Object.assign(Object.assign({}, super.toJSON()), { type: this.type });
    }
}
export class ContractMissingDeployDataError extends BaseWeb3Error {
    constructor() {
        super(`No "data" specified in neither the given options, nor the default options.`);
        this.code = ERR_CONTRACT_MISSING_DEPLOY_DATA;
    }
}
export class ContractNoAddressDefinedError extends BaseWeb3Error {
    constructor() {
        super("This contract object doesn't have address set yet, please set an address first.");
        this.code = ERR_CONTRACT_MISSING_ADDRESS;
    }
}
export class ContractNoFromAddressDefinedError extends BaseWeb3Error {
    constructor() {
        super('No "from" address specified in neither the given options, nor the default options.');
        this.code = ERR_CONTRACT_MISSING_FROM_ADDRESS;
    }
}
export class ContractInstantiationError extends BaseWeb3Error {
    constructor() {
        super(...arguments);
        this.code = ERR_CONTRACT_INSTANTIATION;
    }
}
/**
 * This class is expected to be set as an `cause` inside ContractExecutionError
 * The properties would be typically decoded from the `data` if it was encoded according to EIP-838
 */
export class Eip838ExecutionError extends Web3ContractError {
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
/**
 * Used when an error is raised while executing a function inside a smart contract.
 * The data is expected to be encoded according to EIP-848.
 */
export class ContractExecutionError extends Web3ContractError {
    constructor(rpcError) {
        super('Error happened while trying to execute a function inside a smart contract');
        this.code = ERR_CONTRACT_EXECUTION_REVERTED;
        this.cause = new Eip838ExecutionError(rpcError);
    }
}
export class ContractTransactionDataAndInputError extends InvalidValueError {
    constructor(value) {
        var _a, _b;
        super(`data: ${(_a = value.data) !== null && _a !== void 0 ? _a : 'undefined'}, input: ${(_b = value.input) !== null && _b !== void 0 ? _b : 'undefined'}`, 'You can\'t have "data" and "input" as properties of a contract at the same time, please use either "data" or "input" instead.');
        this.code = ERR_CONTRACT_TX_DATA_AND_INPUT;
    }
}
//# sourceMappingURL=contract_errors.js.map