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
exports.validateTransactionForSigning = exports.validateGas = exports.validateFeeMarketGas = exports.validateLegacyGas = exports.validateHardfork = exports.validateBaseChain = exports.validateChainInfo = exports.validateCustomChainInfo = void 0;
exports.isBaseTransaction = isBaseTransaction;
exports.isAccessListEntry = isAccessListEntry;
exports.isAccessList = isAccessList;
exports.isTransaction1559Unsigned = isTransaction1559Unsigned;
exports.isTransaction2930Unsigned = isTransaction2930Unsigned;
exports.isTransactionLegacyUnsigned = isTransactionLegacyUnsigned;
exports.isTransactionWithSender = isTransactionWithSender;
exports.validateTransactionWithSender = validateTransactionWithSender;
exports.isTransactionCall = isTransactionCall;
exports.validateTransactionCall = validateTransactionCall;
const web3_types_1 = require("web3-types");
const web3_validator_1 = require("web3-validator");
const web3_errors_1 = require("web3-errors");
const format_transaction_js_1 = require("./utils/format_transaction.js");
function isBaseTransaction(value) {
    if (!(0, web3_validator_1.isNullish)(value.to) && !(0, web3_validator_1.isAddress)(value.to))
        return false;
    if (!(0, web3_validator_1.isHexStrict)(value.type) && !(0, web3_validator_1.isNullish)(value.type) && value.type.length !== 2)
        return false;
    if (!(0, web3_validator_1.isHexStrict)(value.nonce))
        return false;
    if (!(0, web3_validator_1.isHexStrict)(value.gas))
        return false;
    if (!(0, web3_validator_1.isHexStrict)(value.value))
        return false;
    if (!(0, web3_validator_1.isHexStrict)(value.input))
        return false;
    if (value.chainId && !(0, web3_validator_1.isHexStrict)(value.chainId))
        return false;
    return true;
}
function isAccessListEntry(value) {
    if (!(0, web3_validator_1.isNullish)(value.address) && !(0, web3_validator_1.isAddress)(value.address))
        return false;
    if (!(0, web3_validator_1.isNullish)(value.storageKeys) &&
        !value.storageKeys.every(storageKey => (0, web3_validator_1.isHexString32Bytes)(storageKey)))
        return false;
    return true;
}
function isAccessList(value) {
    if (!Array.isArray(value) ||
        !value.every(accessListEntry => isAccessListEntry(accessListEntry)))
        return false;
    return true;
}
function isTransaction1559Unsigned(value) {
    if (!isBaseTransaction(value))
        return false;
    if (!(0, web3_validator_1.isHexStrict)(value.maxFeePerGas))
        return false;
    if (!(0, web3_validator_1.isHexStrict)(value.maxPriorityFeePerGas))
        return false;
    if (!isAccessList(value.accessList))
        return false;
    return true;
}
function isTransaction2930Unsigned(value) {
    if (!isBaseTransaction(value))
        return false;
    if (!(0, web3_validator_1.isHexStrict)(value.gasPrice))
        return false;
    if (!isAccessList(value.accessList))
        return false;
    return true;
}
function isTransactionLegacyUnsigned(value) {
    if (!isBaseTransaction(value))
        return false;
    if (!(0, web3_validator_1.isHexStrict)(value.gasPrice))
        return false;
    return true;
}
function isTransactionWithSender(value) {
    if (!(0, web3_validator_1.isAddress)(value.from))
        return false;
    if (!isBaseTransaction(value))
        return false;
    if (!isTransaction1559Unsigned(value) &&
        !isTransaction2930Unsigned(value) &&
        !isTransactionLegacyUnsigned(value))
        return false;
    return true;
}
function validateTransactionWithSender(value) {
    if (!isTransactionWithSender(value))
        throw new web3_errors_1.InvalidTransactionWithSender(value);
}
function isTransactionCall(value) {
    if (!(0, web3_validator_1.isNullish)(value.from) && !(0, web3_validator_1.isAddress)(value.from))
        return false;
    if (!(0, web3_validator_1.isAddress)(value.to))
        return false;
    if (!(0, web3_validator_1.isNullish)(value.gas) && !(0, web3_validator_1.isHexStrict)(value.gas))
        return false;
    if (!(0, web3_validator_1.isNullish)(value.gasPrice) && !(0, web3_validator_1.isHexStrict)(value.gasPrice))
        return false;
    if (!(0, web3_validator_1.isNullish)(value.value) && !(0, web3_validator_1.isHexStrict)(value.value))
        return false;
    if (!(0, web3_validator_1.isNullish)(value.data) && !(0, web3_validator_1.isHexStrict)(value.data))
        return false;
    if (!(0, web3_validator_1.isNullish)(value.input) && !(0, web3_validator_1.isHexStrict)(value.input))
        return false;
    if (!(0, web3_validator_1.isNullish)(value.type))
        return false;
    if (isTransaction1559Unsigned(value))
        return false;
    if (isTransaction2930Unsigned(value))
        return false;
    return true;
}
function validateTransactionCall(value) {
    if (!isTransactionCall(value))
        throw new web3_errors_1.InvalidTransactionCall(value);
}
const validateCustomChainInfo = (transaction) => {
    if (!(0, web3_validator_1.isNullish)(transaction.common)) {
        if ((0, web3_validator_1.isNullish)(transaction.common.customChain))
            throw new web3_errors_1.MissingCustomChainError();
        if ((0, web3_validator_1.isNullish)(transaction.common.customChain.chainId))
            throw new web3_errors_1.MissingCustomChainIdError();
        if (!(0, web3_validator_1.isNullish)(transaction.chainId) &&
            transaction.chainId !== transaction.common.customChain.chainId)
            throw new web3_errors_1.ChainIdMismatchError({
                txChainId: transaction.chainId,
                customChainId: transaction.common.customChain.chainId,
            });
    }
};
exports.validateCustomChainInfo = validateCustomChainInfo;
const validateChainInfo = (transaction) => {
    if (!(0, web3_validator_1.isNullish)(transaction.common) &&
        !(0, web3_validator_1.isNullish)(transaction.chain) &&
        !(0, web3_validator_1.isNullish)(transaction.hardfork)) {
        throw new web3_errors_1.CommonOrChainAndHardforkError();
    }
    if ((!(0, web3_validator_1.isNullish)(transaction.chain) && (0, web3_validator_1.isNullish)(transaction.hardfork)) ||
        (!(0, web3_validator_1.isNullish)(transaction.hardfork) && (0, web3_validator_1.isNullish)(transaction.chain)))
        throw new web3_errors_1.MissingChainOrHardforkError({
            chain: transaction.chain,
            hardfork: transaction.hardfork,
        });
};
exports.validateChainInfo = validateChainInfo;
const validateBaseChain = (transaction) => {
    if (!(0, web3_validator_1.isNullish)(transaction.common))
        if (!(0, web3_validator_1.isNullish)(transaction.common.baseChain))
            if (!(0, web3_validator_1.isNullish)(transaction.chain) &&
                transaction.chain !== transaction.common.baseChain) {
                throw new web3_errors_1.ChainMismatchError({
                    txChain: transaction.chain,
                    baseChain: transaction.common.baseChain,
                });
            }
};
exports.validateBaseChain = validateBaseChain;
const validateHardfork = (transaction) => {
    if (!(0, web3_validator_1.isNullish)(transaction.common))
        if (!(0, web3_validator_1.isNullish)(transaction.common.hardfork))
            if (!(0, web3_validator_1.isNullish)(transaction.hardfork) &&
                transaction.hardfork !== transaction.common.hardfork) {
                throw new web3_errors_1.HardforkMismatchError({
                    txHardfork: transaction.hardfork,
                    commonHardfork: transaction.common.hardfork,
                });
            }
};
exports.validateHardfork = validateHardfork;
const validateLegacyGas = (transaction) => {
    if (
    // This check is verifying gas and gasPrice aren't less than 0.
    (0, web3_validator_1.isNullish)(transaction.gas) ||
        !(0, web3_validator_1.isUInt)(transaction.gas) ||
        (0, web3_validator_1.isNullish)(transaction.gasPrice) ||
        !(0, web3_validator_1.isUInt)(transaction.gasPrice))
        throw new web3_errors_1.InvalidGasOrGasPrice({
            gas: transaction.gas,
            gasPrice: transaction.gasPrice,
        });
    if (!(0, web3_validator_1.isNullish)(transaction.maxFeePerGas) || !(0, web3_validator_1.isNullish)(transaction.maxPriorityFeePerGas))
        throw new web3_errors_1.UnsupportedFeeMarketError({
            maxFeePerGas: transaction.maxFeePerGas,
            maxPriorityFeePerGas: transaction.maxPriorityFeePerGas,
        });
};
exports.validateLegacyGas = validateLegacyGas;
const validateFeeMarketGas = (transaction) => {
    // These errors come from 1.x, so they must be checked before
    // InvalidMaxPriorityFeePerGasOrMaxFeePerGas to throw the same error
    // for the same code executing in 1.x
    if (!(0, web3_validator_1.isNullish)(transaction.gasPrice) && transaction.type === '0x2')
        throw new web3_errors_1.Eip1559GasPriceError(transaction.gasPrice);
    if (transaction.type === '0x0' || transaction.type === '0x1')
        throw new web3_errors_1.UnsupportedFeeMarketError({
            maxFeePerGas: transaction.maxFeePerGas,
            maxPriorityFeePerGas: transaction.maxPriorityFeePerGas,
        });
    if ((0, web3_validator_1.isNullish)(transaction.maxFeePerGas) ||
        !(0, web3_validator_1.isUInt)(transaction.maxFeePerGas) ||
        (0, web3_validator_1.isNullish)(transaction.maxPriorityFeePerGas) ||
        !(0, web3_validator_1.isUInt)(transaction.maxPriorityFeePerGas))
        throw new web3_errors_1.InvalidMaxPriorityFeePerGasOrMaxFeePerGas({
            maxPriorityFeePerGas: transaction.maxPriorityFeePerGas,
            maxFeePerGas: transaction.maxFeePerGas,
        });
};
exports.validateFeeMarketGas = validateFeeMarketGas;
/**
 * This method checks if all required gas properties are present for either
 * legacy gas (type 0x0 and 0x1) OR fee market transactions (0x2)
 */
const validateGas = (transaction) => {
    const gasPresent = !(0, web3_validator_1.isNullish)(transaction.gas) || !(0, web3_validator_1.isNullish)(transaction.gasLimit);
    const legacyGasPresent = gasPresent && !(0, web3_validator_1.isNullish)(transaction.gasPrice);
    const feeMarketGasPresent = gasPresent &&
        !(0, web3_validator_1.isNullish)(transaction.maxPriorityFeePerGas) &&
        !(0, web3_validator_1.isNullish)(transaction.maxFeePerGas);
    if (!legacyGasPresent && !feeMarketGasPresent)
        throw new web3_errors_1.MissingGasError({
            gas: transaction.gas,
            gasPrice: transaction.gasPrice,
            maxPriorityFeePerGas: transaction.maxPriorityFeePerGas,
            maxFeePerGas: transaction.maxFeePerGas,
        });
    if (legacyGasPresent && feeMarketGasPresent)
        throw new web3_errors_1.TransactionGasMismatchError({
            gas: transaction.gas,
            gasPrice: transaction.gasPrice,
            maxPriorityFeePerGas: transaction.maxPriorityFeePerGas,
            maxFeePerGas: transaction.maxFeePerGas,
        });
    (legacyGasPresent ? exports.validateLegacyGas : exports.validateFeeMarketGas)(transaction);
    (!(0, web3_validator_1.isNullish)(transaction.type) && transaction.type > '0x1'
        ? exports.validateFeeMarketGas
        : exports.validateLegacyGas)(transaction);
};
exports.validateGas = validateGas;
const validateTransactionForSigning = (transaction, overrideMethod, options = { transactionSchema: undefined }) => {
    if (!(0, web3_validator_1.isNullish)(overrideMethod)) {
        overrideMethod(transaction);
        return;
    }
    if (typeof transaction !== 'object' || (0, web3_validator_1.isNullish)(transaction))
        throw new web3_errors_1.InvalidTransactionObjectError(transaction);
    (0, exports.validateCustomChainInfo)(transaction);
    (0, exports.validateChainInfo)(transaction);
    (0, exports.validateBaseChain)(transaction);
    (0, exports.validateHardfork)(transaction);
    const formattedTransaction = (0, format_transaction_js_1.formatTransaction)(transaction, web3_types_1.ETH_DATA_FORMAT, {
        transactionSchema: options.transactionSchema,
    });
    (0, exports.validateGas)(formattedTransaction);
    if ((0, web3_validator_1.isNullish)(formattedTransaction.nonce) ||
        (0, web3_validator_1.isNullish)(formattedTransaction.chainId) ||
        formattedTransaction.nonce.startsWith('-') ||
        formattedTransaction.chainId.startsWith('-'))
        throw new web3_errors_1.InvalidNonceOrChainIdError({
            nonce: transaction.nonce,
            chainId: transaction.chainId,
        });
};
exports.validateTransactionForSigning = validateTransactionForSigning;
//# sourceMappingURL=validation.js.map