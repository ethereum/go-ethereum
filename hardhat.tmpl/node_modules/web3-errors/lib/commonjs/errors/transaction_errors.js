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
exports.InvalidPropertiesForTransactionTypeError = exports.LocalWalletNotAvailableError = exports.TransactionSigningError = exports.TransactionReceiptMissingBlockNumberError = exports.TransactionMissingReceiptOrBlockHashError = exports.TransactionBlockTimeoutError = exports.TransactionPollingTimeoutError = exports.TransactionSendTimeoutError = exports.TransactionDataAndInputError = exports.UnsupportedTransactionTypeError = exports.Eip1559NotSupportedError = exports.UnableToPopulateNonceError = exports.InvalidNonceOrChainIdError = exports.InvalidTransactionObjectError = exports.UnsupportedFeeMarketError = exports.Eip1559GasPriceError = exports.InvalidMaxPriorityFeePerGasOrMaxFeePerGas = exports.InvalidGasOrGasPrice = exports.TransactionGasMismatchError = exports.TransactionGasMismatchInnerError = exports.MissingGasError = exports.MissingGasInnerError = exports.MissingChainOrHardforkError = exports.CommonOrChainAndHardforkError = exports.HardforkMismatchError = exports.ChainMismatchError = exports.ChainIdMismatchError = exports.MissingCustomChainIdError = exports.MissingCustomChainError = exports.InvalidTransactionCall = exports.InvalidTransactionWithReceiver = exports.InvalidTransactionWithSender = exports.TransactionNotFound = exports.UndefinedRawTransactionError = exports.TransactionOutOfGasError = exports.TransactionRevertedWithoutReasonError = exports.ContractCodeNotStoredError = exports.NoContractAddressFoundError = exports.TransactionRevertWithCustomError = exports.TransactionRevertInstructionError = exports.RevertInstructionError = exports.TransactionError = void 0;
const error_codes_js_1 = require("../error_codes.js");
const web3_error_base_js_1 = require("../web3_error_base.js");
class TransactionError extends web3_error_base_js_1.BaseWeb3Error {
    constructor(message, receipt) {
        super(message);
        this.receipt = receipt;
        this.code = error_codes_js_1.ERR_TX;
    }
    toJSON() {
        return Object.assign(Object.assign({}, super.toJSON()), { receipt: this.receipt });
    }
}
exports.TransactionError = TransactionError;
class RevertInstructionError extends web3_error_base_js_1.BaseWeb3Error {
    constructor(reason, signature) {
        super(`Your request got reverted with the following reason string: ${reason}`);
        this.reason = reason;
        this.signature = signature;
        this.code = error_codes_js_1.ERR_TX_REVERT_INSTRUCTION;
    }
    toJSON() {
        return Object.assign(Object.assign({}, super.toJSON()), { reason: this.reason, signature: this.signature });
    }
}
exports.RevertInstructionError = RevertInstructionError;
class TransactionRevertInstructionError extends web3_error_base_js_1.BaseWeb3Error {
    constructor(reason, signature, receipt, data) {
        super(`Transaction has been reverted by the EVM${receipt === undefined ? '' : `:\n ${web3_error_base_js_1.BaseWeb3Error.convertToString(receipt)}`}`);
        this.reason = reason;
        this.signature = signature;
        this.receipt = receipt;
        this.data = data;
        this.code = error_codes_js_1.ERR_TX_REVERT_TRANSACTION;
    }
    toJSON() {
        return Object.assign(Object.assign({}, super.toJSON()), { reason: this.reason, signature: this.signature, receipt: this.receipt, data: this.data });
    }
}
exports.TransactionRevertInstructionError = TransactionRevertInstructionError;
/**
 * This error is used when a transaction to a smart contract fails and
 * a custom user error (https://blog.soliditylang.org/2021/04/21/custom-errors/)
 * is able to be parsed from the revert reason
 */
class TransactionRevertWithCustomError extends TransactionRevertInstructionError {
    constructor(reason, customErrorName, customErrorDecodedSignature, customErrorArguments, signature, receipt, data) {
        super(reason);
        this.reason = reason;
        this.customErrorName = customErrorName;
        this.customErrorDecodedSignature = customErrorDecodedSignature;
        this.customErrorArguments = customErrorArguments;
        this.signature = signature;
        this.receipt = receipt;
        this.data = data;
        this.code = error_codes_js_1.ERR_TX_REVERT_TRANSACTION_CUSTOM_ERROR;
    }
    toJSON() {
        return Object.assign(Object.assign({}, super.toJSON()), { reason: this.reason, customErrorName: this.customErrorName, customErrorDecodedSignature: this.customErrorDecodedSignature, customErrorArguments: this.customErrorArguments, signature: this.signature, receipt: this.receipt, data: this.data });
    }
}
exports.TransactionRevertWithCustomError = TransactionRevertWithCustomError;
class NoContractAddressFoundError extends TransactionError {
    constructor(receipt) {
        super("The transaction receipt didn't contain a contract address.", receipt);
        this.code = error_codes_js_1.ERR_TX_NO_CONTRACT_ADDRESS;
    }
    toJSON() {
        return Object.assign(Object.assign({}, super.toJSON()), { receipt: this.receipt });
    }
}
exports.NoContractAddressFoundError = NoContractAddressFoundError;
class ContractCodeNotStoredError extends TransactionError {
    constructor(receipt) {
        super("The contract code couldn't be stored, please check your gas limit.", receipt);
        this.code = error_codes_js_1.ERR_TX_CONTRACT_NOT_STORED;
    }
}
exports.ContractCodeNotStoredError = ContractCodeNotStoredError;
class TransactionRevertedWithoutReasonError extends TransactionError {
    constructor(receipt) {
        super(`Transaction has been reverted by the EVM${receipt === undefined ? '' : `:\n ${web3_error_base_js_1.BaseWeb3Error.convertToString(receipt)}`}`, receipt);
        this.code = error_codes_js_1.ERR_TX_REVERT_WITHOUT_REASON;
    }
}
exports.TransactionRevertedWithoutReasonError = TransactionRevertedWithoutReasonError;
class TransactionOutOfGasError extends TransactionError {
    constructor(receipt) {
        super(`Transaction ran out of gas. Please provide more gas:\n ${JSON.stringify(receipt, undefined, 2)}`, receipt);
        this.code = error_codes_js_1.ERR_TX_OUT_OF_GAS;
    }
}
exports.TransactionOutOfGasError = TransactionOutOfGasError;
class UndefinedRawTransactionError extends TransactionError {
    constructor() {
        super(`Raw transaction undefined`);
        this.code = error_codes_js_1.ERR_RAW_TX_UNDEFINED;
    }
}
exports.UndefinedRawTransactionError = UndefinedRawTransactionError;
class TransactionNotFound extends TransactionError {
    constructor() {
        super('Transaction not found');
        this.code = error_codes_js_1.ERR_TX_NOT_FOUND;
    }
}
exports.TransactionNotFound = TransactionNotFound;
class InvalidTransactionWithSender extends web3_error_base_js_1.InvalidValueError {
    constructor(value) {
        super(value, 'invalid transaction with invalid sender');
        this.code = error_codes_js_1.ERR_TX_INVALID_SENDER;
    }
}
exports.InvalidTransactionWithSender = InvalidTransactionWithSender;
class InvalidTransactionWithReceiver extends web3_error_base_js_1.InvalidValueError {
    constructor(value) {
        super(value, 'invalid transaction with invalid receiver');
        this.code = error_codes_js_1.ERR_TX_INVALID_RECEIVER;
    }
}
exports.InvalidTransactionWithReceiver = InvalidTransactionWithReceiver;
class InvalidTransactionCall extends web3_error_base_js_1.InvalidValueError {
    constructor(value) {
        super(value, 'invalid transaction call');
        this.code = error_codes_js_1.ERR_TX_INVALID_CALL;
    }
}
exports.InvalidTransactionCall = InvalidTransactionCall;
class MissingCustomChainError extends web3_error_base_js_1.InvalidValueError {
    constructor() {
        super('MissingCustomChainError', 'If tx.common is provided it must have tx.common.customChain');
        this.code = error_codes_js_1.ERR_TX_MISSING_CUSTOM_CHAIN;
    }
}
exports.MissingCustomChainError = MissingCustomChainError;
class MissingCustomChainIdError extends web3_error_base_js_1.InvalidValueError {
    constructor() {
        super('MissingCustomChainIdError', 'If tx.common is provided it must have tx.common.customChain and tx.common.customChain.chainId');
        this.code = error_codes_js_1.ERR_TX_MISSING_CUSTOM_CHAIN_ID;
    }
}
exports.MissingCustomChainIdError = MissingCustomChainIdError;
class ChainIdMismatchError extends web3_error_base_js_1.InvalidValueError {
    constructor(value) {
        super(JSON.stringify(value), 
        // https://github.com/ChainSafe/web3.js/blob/8783f4d64e424456bdc53b34ef1142d0a7cee4d7/packages/web3-eth-accounts/src/index.js#L176
        'Chain Id doesnt match in tx.chainId tx.common.customChain.chainId');
        this.code = error_codes_js_1.ERR_TX_CHAIN_ID_MISMATCH;
    }
}
exports.ChainIdMismatchError = ChainIdMismatchError;
class ChainMismatchError extends web3_error_base_js_1.InvalidValueError {
    constructor(value) {
        super(JSON.stringify(value), 'Chain doesnt match in tx.chain tx.common.basechain');
        this.code = error_codes_js_1.ERR_TX_CHAIN_MISMATCH;
    }
}
exports.ChainMismatchError = ChainMismatchError;
class HardforkMismatchError extends web3_error_base_js_1.InvalidValueError {
    constructor(value) {
        super(JSON.stringify(value), 'hardfork doesnt match in tx.hardfork tx.common.hardfork');
        this.code = error_codes_js_1.ERR_TX_HARDFORK_MISMATCH;
    }
}
exports.HardforkMismatchError = HardforkMismatchError;
class CommonOrChainAndHardforkError extends web3_error_base_js_1.InvalidValueError {
    constructor() {
        super('CommonOrChainAndHardforkError', 'Please provide the common object or the chain and hardfork property but not all together.');
        this.code = error_codes_js_1.ERR_TX_INVALID_CHAIN_INFO;
    }
}
exports.CommonOrChainAndHardforkError = CommonOrChainAndHardforkError;
class MissingChainOrHardforkError extends web3_error_base_js_1.InvalidValueError {
    constructor(value) {
        var _a, _b;
        super('MissingChainOrHardforkError', `When specifying chain and hardfork, both values must be defined. Received "chain": ${(_a = value.chain) !== null && _a !== void 0 ? _a : 'undefined'}, "hardfork": ${(_b = value.hardfork) !== null && _b !== void 0 ? _b : 'undefined'}`);
        this.code = error_codes_js_1.ERR_TX_MISSING_CHAIN_INFO;
    }
}
exports.MissingChainOrHardforkError = MissingChainOrHardforkError;
class MissingGasInnerError extends web3_error_base_js_1.BaseWeb3Error {
    constructor() {
        super('Missing properties in transaction, either define "gas" and "gasPrice" for type 0 transactions or "gas", "maxPriorityFeePerGas" and "maxFeePerGas" for type 2 transactions');
        this.code = error_codes_js_1.ERR_TX_MISSING_GAS_INNER_ERROR;
    }
}
exports.MissingGasInnerError = MissingGasInnerError;
class MissingGasError extends web3_error_base_js_1.InvalidValueError {
    constructor(value) {
        var _a, _b, _c, _d;
        super(`gas: ${(_a = value.gas) !== null && _a !== void 0 ? _a : 'undefined'}, gasPrice: ${(_b = value.gasPrice) !== null && _b !== void 0 ? _b : 'undefined'}, maxPriorityFeePerGas: ${(_c = value.maxPriorityFeePerGas) !== null && _c !== void 0 ? _c : 'undefined'}, maxFeePerGas: ${(_d = value.maxFeePerGas) !== null && _d !== void 0 ? _d : 'undefined'}`, '"gas" is missing');
        this.code = error_codes_js_1.ERR_TX_MISSING_GAS;
        this.cause = new MissingGasInnerError();
    }
}
exports.MissingGasError = MissingGasError;
class TransactionGasMismatchInnerError extends web3_error_base_js_1.BaseWeb3Error {
    constructor() {
        super('Missing properties in transaction, either define "gas" and "gasPrice" for type 0 transactions or "gas", "maxPriorityFeePerGas" and "maxFeePerGas" for type 2 transactions, not both');
        this.code = error_codes_js_1.ERR_TX_GAS_MISMATCH_INNER_ERROR;
    }
}
exports.TransactionGasMismatchInnerError = TransactionGasMismatchInnerError;
class TransactionGasMismatchError extends web3_error_base_js_1.InvalidValueError {
    constructor(value) {
        var _a, _b, _c, _d;
        super(`gas: ${(_a = value.gas) !== null && _a !== void 0 ? _a : 'undefined'}, gasPrice: ${(_b = value.gasPrice) !== null && _b !== void 0 ? _b : 'undefined'}, maxPriorityFeePerGas: ${(_c = value.maxPriorityFeePerGas) !== null && _c !== void 0 ? _c : 'undefined'}, maxFeePerGas: ${(_d = value.maxFeePerGas) !== null && _d !== void 0 ? _d : 'undefined'}`, 'transaction must specify legacy or fee market gas properties, not both');
        this.code = error_codes_js_1.ERR_TX_GAS_MISMATCH;
        this.cause = new TransactionGasMismatchInnerError();
    }
}
exports.TransactionGasMismatchError = TransactionGasMismatchError;
class InvalidGasOrGasPrice extends web3_error_base_js_1.InvalidValueError {
    constructor(value) {
        var _a, _b;
        super(`gas: ${(_a = value.gas) !== null && _a !== void 0 ? _a : 'undefined'}, gasPrice: ${(_b = value.gasPrice) !== null && _b !== void 0 ? _b : 'undefined'}`, 'Gas or gasPrice is lower than 0');
        this.code = error_codes_js_1.ERR_TX_INVALID_LEGACY_GAS;
    }
}
exports.InvalidGasOrGasPrice = InvalidGasOrGasPrice;
class InvalidMaxPriorityFeePerGasOrMaxFeePerGas extends web3_error_base_js_1.InvalidValueError {
    constructor(value) {
        var _a, _b;
        super(`maxPriorityFeePerGas: ${(_a = value.maxPriorityFeePerGas) !== null && _a !== void 0 ? _a : 'undefined'}, maxFeePerGas: ${(_b = value.maxFeePerGas) !== null && _b !== void 0 ? _b : 'undefined'}`, 'maxPriorityFeePerGas or maxFeePerGas is lower than 0');
        this.code = error_codes_js_1.ERR_TX_INVALID_FEE_MARKET_GAS;
    }
}
exports.InvalidMaxPriorityFeePerGasOrMaxFeePerGas = InvalidMaxPriorityFeePerGasOrMaxFeePerGas;
class Eip1559GasPriceError extends web3_error_base_js_1.InvalidValueError {
    constructor(value) {
        super(value, "eip-1559 transactions don't support gasPrice");
        this.code = error_codes_js_1.ERR_TX_INVALID_FEE_MARKET_GAS_PRICE;
    }
}
exports.Eip1559GasPriceError = Eip1559GasPriceError;
class UnsupportedFeeMarketError extends web3_error_base_js_1.InvalidValueError {
    constructor(value) {
        var _a, _b;
        super(`maxPriorityFeePerGas: ${(_a = value.maxPriorityFeePerGas) !== null && _a !== void 0 ? _a : 'undefined'}, maxFeePerGas: ${(_b = value.maxFeePerGas) !== null && _b !== void 0 ? _b : 'undefined'}`, "pre-eip-1559 transaction don't support maxFeePerGas/maxPriorityFeePerGas");
        this.code = error_codes_js_1.ERR_TX_INVALID_LEGACY_FEE_MARKET;
    }
}
exports.UnsupportedFeeMarketError = UnsupportedFeeMarketError;
class InvalidTransactionObjectError extends web3_error_base_js_1.InvalidValueError {
    constructor(value) {
        super(value, 'invalid transaction object');
        this.code = error_codes_js_1.ERR_TX_INVALID_OBJECT;
    }
}
exports.InvalidTransactionObjectError = InvalidTransactionObjectError;
class InvalidNonceOrChainIdError extends web3_error_base_js_1.InvalidValueError {
    constructor(value) {
        var _a, _b;
        super(`nonce: ${(_a = value.nonce) !== null && _a !== void 0 ? _a : 'undefined'}, chainId: ${(_b = value.chainId) !== null && _b !== void 0 ? _b : 'undefined'}`, 'Nonce or chainId is lower than 0');
        this.code = error_codes_js_1.ERR_TX_INVALID_NONCE_OR_CHAIN_ID;
    }
}
exports.InvalidNonceOrChainIdError = InvalidNonceOrChainIdError;
class UnableToPopulateNonceError extends web3_error_base_js_1.InvalidValueError {
    constructor() {
        super('UnableToPopulateNonceError', 'unable to populate nonce, no from address available');
        this.code = error_codes_js_1.ERR_TX_UNABLE_TO_POPULATE_NONCE;
    }
}
exports.UnableToPopulateNonceError = UnableToPopulateNonceError;
class Eip1559NotSupportedError extends web3_error_base_js_1.InvalidValueError {
    constructor() {
        super('Eip1559NotSupportedError', "Network doesn't support eip-1559");
        this.code = error_codes_js_1.ERR_TX_UNSUPPORTED_EIP_1559;
    }
}
exports.Eip1559NotSupportedError = Eip1559NotSupportedError;
class UnsupportedTransactionTypeError extends web3_error_base_js_1.InvalidValueError {
    constructor(value) {
        super(value, 'unsupported transaction type');
        this.code = error_codes_js_1.ERR_TX_UNSUPPORTED_TYPE;
    }
}
exports.UnsupportedTransactionTypeError = UnsupportedTransactionTypeError;
class TransactionDataAndInputError extends web3_error_base_js_1.InvalidValueError {
    constructor(value) {
        var _a, _b;
        super(`data: ${(_a = value.data) !== null && _a !== void 0 ? _a : 'undefined'}, input: ${(_b = value.input) !== null && _b !== void 0 ? _b : 'undefined'}`, 'You can\'t have "data" and "input" as properties of transactions at the same time, please use either "data" or "input" instead.');
        this.code = error_codes_js_1.ERR_TX_DATA_AND_INPUT;
    }
}
exports.TransactionDataAndInputError = TransactionDataAndInputError;
class TransactionSendTimeoutError extends web3_error_base_js_1.BaseWeb3Error {
    constructor(value) {
        super(`The connected Ethereum Node did not respond within ${value.numberOfSeconds} seconds, please make sure your transaction was properly sent and you are connected to a healthy Node. Be aware that transaction might still be pending or mined!\n\tTransaction Hash: ${value.transactionHash ? value.transactionHash.toString() : 'not available'}`);
        this.code = error_codes_js_1.ERR_TX_SEND_TIMEOUT;
    }
}
exports.TransactionSendTimeoutError = TransactionSendTimeoutError;
function transactionTimeoutHint(transactionHash) {
    return `Please make sure your transaction was properly sent and there are no previous pending transaction for the same account. However, be aware that it might still be mined!\n\tTransaction Hash: ${transactionHash ? transactionHash.toString() : 'not available'}`;
}
class TransactionPollingTimeoutError extends web3_error_base_js_1.BaseWeb3Error {
    constructor(value) {
        super(`Transaction was not mined within ${value.numberOfSeconds} seconds. ${transactionTimeoutHint(value.transactionHash)}`);
        this.code = error_codes_js_1.ERR_TX_POLLING_TIMEOUT;
    }
}
exports.TransactionPollingTimeoutError = TransactionPollingTimeoutError;
class TransactionBlockTimeoutError extends web3_error_base_js_1.BaseWeb3Error {
    constructor(value) {
        super(`Transaction started at ${value.starterBlockNumber} but was not mined within ${value.numberOfBlocks} blocks. ${transactionTimeoutHint(value.transactionHash)}`);
        this.code = error_codes_js_1.ERR_TX_BLOCK_TIMEOUT;
    }
}
exports.TransactionBlockTimeoutError = TransactionBlockTimeoutError;
class TransactionMissingReceiptOrBlockHashError extends web3_error_base_js_1.InvalidValueError {
    constructor(value) {
        var _a, _b;
        super(`receipt: ${JSON.stringify(value.receipt)}, blockHash: ${(_a = value.blockHash) === null || _a === void 0 ? void 0 : _a.toString()}, transactionHash: ${(_b = value.transactionHash) === null || _b === void 0 ? void 0 : _b.toString()}`, `Receipt missing or blockHash null`);
        this.code = error_codes_js_1.ERR_TX_RECEIPT_MISSING_OR_BLOCKHASH_NULL;
    }
}
exports.TransactionMissingReceiptOrBlockHashError = TransactionMissingReceiptOrBlockHashError;
class TransactionReceiptMissingBlockNumberError extends web3_error_base_js_1.InvalidValueError {
    constructor(value) {
        super(`receipt: ${JSON.stringify(value.receipt)}`, `Receipt missing block number`);
        this.code = error_codes_js_1.ERR_TX_RECEIPT_MISSING_BLOCK_NUMBER;
    }
}
exports.TransactionReceiptMissingBlockNumberError = TransactionReceiptMissingBlockNumberError;
class TransactionSigningError extends web3_error_base_js_1.BaseWeb3Error {
    constructor(errorDetails) {
        super(`Invalid signature. "${errorDetails}"`);
        this.code = error_codes_js_1.ERR_TX_SIGNING;
    }
}
exports.TransactionSigningError = TransactionSigningError;
class LocalWalletNotAvailableError extends web3_error_base_js_1.InvalidValueError {
    constructor() {
        super('LocalWalletNotAvailableError', `Attempted to index account in local wallet, but no wallet is available`);
        this.code = error_codes_js_1.ERR_TX_LOCAL_WALLET_NOT_AVAILABLE;
    }
}
exports.LocalWalletNotAvailableError = LocalWalletNotAvailableError;
class InvalidPropertiesForTransactionTypeError extends web3_error_base_js_1.BaseWeb3Error {
    constructor(validationError, txType) {
        const invalidPropertyNames = [];
        validationError.forEach(error => invalidPropertyNames.push(error.keyword));
        super(`The following properties are invalid for the transaction type ${txType}: ${invalidPropertyNames.join(', ')}`);
        this.code = error_codes_js_1.ERR_TX_INVALID_PROPERTIES_FOR_TYPE;
    }
}
exports.InvalidPropertiesForTransactionTypeError = InvalidPropertiesForTransactionTypeError;
//# sourceMappingURL=transaction_errors.js.map