"use strict";
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
exports.ecRecover = exports.sign = exports.signTransaction = exports.sendTransaction = exports.importRawKey = exports.lockAccount = exports.unlockAccount = exports.newAccount = exports.getAccounts = void 0;
const web3_utils_1 = require("web3-utils");
const web3_eth_1 = require("web3-eth");
const web3_types_1 = require("web3-types");
const web3_validator_1 = require("web3-validator");
const web3_rpc_methods_1 = require("web3-rpc-methods");
const getAccounts = (requestManager) => __awaiter(void 0, void 0, void 0, function* () {
    const result = yield web3_rpc_methods_1.personalRpcMethods.getAccounts(requestManager);
    return result.map(web3_utils_1.toChecksumAddress);
});
exports.getAccounts = getAccounts;
const newAccount = (requestManager, password) => __awaiter(void 0, void 0, void 0, function* () {
    web3_validator_1.validator.validate(['string'], [password]);
    const result = yield web3_rpc_methods_1.personalRpcMethods.newAccount(requestManager, password);
    return (0, web3_utils_1.toChecksumAddress)(result);
});
exports.newAccount = newAccount;
const unlockAccount = (requestManager, address, password, unlockDuration) => __awaiter(void 0, void 0, void 0, function* () {
    web3_validator_1.validator.validate(['address', 'string', 'uint'], [address, password, unlockDuration]);
    return web3_rpc_methods_1.personalRpcMethods.unlockAccount(requestManager, address, password, unlockDuration);
});
exports.unlockAccount = unlockAccount;
const lockAccount = (requestManager, address) => __awaiter(void 0, void 0, void 0, function* () {
    web3_validator_1.validator.validate(['address'], [address]);
    return web3_rpc_methods_1.personalRpcMethods.lockAccount(requestManager, address);
});
exports.lockAccount = lockAccount;
const importRawKey = (requestManager, keyData, passphrase) => __awaiter(void 0, void 0, void 0, function* () {
    web3_validator_1.validator.validate(['string', 'string'], [keyData, passphrase]);
    return web3_rpc_methods_1.personalRpcMethods.importRawKey(requestManager, keyData, passphrase);
});
exports.importRawKey = importRawKey;
const sendTransaction = (requestManager, tx, passphrase, config) => __awaiter(void 0, void 0, void 0, function* () {
    const formattedTx = (0, web3_eth_1.formatTransaction)(tx, web3_types_1.ETH_DATA_FORMAT, {
        transactionSchema: config === null || config === void 0 ? void 0 : config.customTransactionSchema,
    });
    return web3_rpc_methods_1.personalRpcMethods.sendTransaction(requestManager, formattedTx, passphrase);
});
exports.sendTransaction = sendTransaction;
const signTransaction = (requestManager, tx, passphrase, config) => __awaiter(void 0, void 0, void 0, function* () {
    const formattedTx = (0, web3_eth_1.formatTransaction)(tx, web3_types_1.ETH_DATA_FORMAT, {
        transactionSchema: config === null || config === void 0 ? void 0 : config.customTransactionSchema,
    });
    return web3_rpc_methods_1.personalRpcMethods.signTransaction(requestManager, formattedTx, passphrase);
});
exports.signTransaction = signTransaction;
const sign = (requestManager, data, address, passphrase) => __awaiter(void 0, void 0, void 0, function* () {
    web3_validator_1.validator.validate(['string', 'address', 'string'], [data, address, passphrase]);
    const dataToSign = (0, web3_validator_1.isHexStrict)(data) ? data : (0, web3_utils_1.utf8ToHex)(data);
    return web3_rpc_methods_1.personalRpcMethods.sign(requestManager, dataToSign, address, passphrase);
});
exports.sign = sign;
const ecRecover = (requestManager, signedData, signature) => __awaiter(void 0, void 0, void 0, function* () {
    web3_validator_1.validator.validate(['string', 'string'], [signedData, signature]);
    const signedDataString = (0, web3_validator_1.isHexStrict)(signedData) ? signedData : (0, web3_utils_1.utf8ToHex)(signedData);
    return web3_rpc_methods_1.personalRpcMethods.ecRecover(requestManager, signedDataString, signature);
});
exports.ecRecover = ecRecover;
//# sourceMappingURL=rpc_method_wrappers.js.map