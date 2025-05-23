var __awaiter = (this && this.__awaiter) || function (thisArg, _arguments, P, generator) {
    function adopt(value) { return value instanceof P ? value : new P(function (resolve) { resolve(value); }); }
    return new (P || (P = Promise))(function (resolve, reject) {
        function fulfilled(value) { try { step(generator.next(value)); } catch (e) { reject(e); } }
        function rejected(value) { try { step(generator["throw"](value)); } catch (e) { reject(e); } }
        function step(result) { result.done ? resolve(result.value) : adopt(result.value).then(fulfilled, rejected); }
        step((generator = generator.apply(thisArg, _arguments || [])).next());
    });
};
import { toChecksumAddress, utf8ToHex } from 'web3-utils';
import { formatTransaction } from 'web3-eth';
import { ETH_DATA_FORMAT } from 'web3-types';
import { validator, isHexStrict } from 'web3-validator';
import { personalRpcMethods } from 'web3-rpc-methods';
export const getAccounts = (requestManager) => __awaiter(void 0, void 0, void 0, function* () {
    const result = yield personalRpcMethods.getAccounts(requestManager);
    return result.map(toChecksumAddress);
});
export const newAccount = (requestManager, password) => __awaiter(void 0, void 0, void 0, function* () {
    validator.validate(['string'], [password]);
    const result = yield personalRpcMethods.newAccount(requestManager, password);
    return toChecksumAddress(result);
});
export const unlockAccount = (requestManager, address, password, unlockDuration) => __awaiter(void 0, void 0, void 0, function* () {
    validator.validate(['address', 'string', 'uint'], [address, password, unlockDuration]);
    return personalRpcMethods.unlockAccount(requestManager, address, password, unlockDuration);
});
export const lockAccount = (requestManager, address) => __awaiter(void 0, void 0, void 0, function* () {
    validator.validate(['address'], [address]);
    return personalRpcMethods.lockAccount(requestManager, address);
});
export const importRawKey = (requestManager, keyData, passphrase) => __awaiter(void 0, void 0, void 0, function* () {
    validator.validate(['string', 'string'], [keyData, passphrase]);
    return personalRpcMethods.importRawKey(requestManager, keyData, passphrase);
});
export const sendTransaction = (requestManager, tx, passphrase, config) => __awaiter(void 0, void 0, void 0, function* () {
    const formattedTx = formatTransaction(tx, ETH_DATA_FORMAT, {
        transactionSchema: config === null || config === void 0 ? void 0 : config.customTransactionSchema,
    });
    return personalRpcMethods.sendTransaction(requestManager, formattedTx, passphrase);
});
export const signTransaction = (requestManager, tx, passphrase, config) => __awaiter(void 0, void 0, void 0, function* () {
    const formattedTx = formatTransaction(tx, ETH_DATA_FORMAT, {
        transactionSchema: config === null || config === void 0 ? void 0 : config.customTransactionSchema,
    });
    return personalRpcMethods.signTransaction(requestManager, formattedTx, passphrase);
});
export const sign = (requestManager, data, address, passphrase) => __awaiter(void 0, void 0, void 0, function* () {
    validator.validate(['string', 'address', 'string'], [data, address, passphrase]);
    const dataToSign = isHexStrict(data) ? data : utf8ToHex(data);
    return personalRpcMethods.sign(requestManager, dataToSign, address, passphrase);
});
export const ecRecover = (requestManager, signedData, signature) => __awaiter(void 0, void 0, void 0, function* () {
    validator.validate(['string', 'string'], [signedData, signature]);
    const signedDataString = isHexStrict(signedData) ? signedData : utf8ToHex(signedData);
    return personalRpcMethods.ecRecover(requestManager, signedDataString, signature);
});
//# sourceMappingURL=rpc_method_wrappers.js.map