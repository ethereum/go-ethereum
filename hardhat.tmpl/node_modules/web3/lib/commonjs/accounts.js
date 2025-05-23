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
exports.initAccountsForContext = void 0;
const web3_types_1 = require("web3-types");
const web3_utils_1 = require("web3-utils");
const web3_eth_1 = require("web3-eth");
const web3_eth_accounts_1 = require("web3-eth-accounts");
/**
 * Initialize the accounts module for the given context.
 *
 * To avoid multiple package dependencies for `web3-eth-accounts` we are creating
 * this function in `web3` package. In future the actual `web3-eth-accounts` package
 * should be converted to context aware.
 */
const initAccountsForContext = (context) => {
    const signTransactionWithContext = (transaction, privateKey) => __awaiter(void 0, void 0, void 0, function* () {
        const tx = yield (0, web3_eth_1.prepareTransactionForSigning)(transaction, context);
        const privateKeyBytes = (0, web3_utils_1.format)({ format: 'bytes' }, privateKey, web3_types_1.ETH_DATA_FORMAT);
        return (0, web3_eth_accounts_1.signTransaction)(tx, privateKeyBytes);
    });
    const privateKeyToAccountWithContext = (privateKey) => {
        const account = (0, web3_eth_accounts_1.privateKeyToAccount)(privateKey);
        return Object.assign(Object.assign({}, account), { signTransaction: (transaction) => __awaiter(void 0, void 0, void 0, function* () { return signTransactionWithContext(transaction, account.privateKey); }) });
    };
    const decryptWithContext = (keystore, password, options) => __awaiter(void 0, void 0, void 0, function* () {
        var _a;
        const account = yield (0, web3_eth_accounts_1.decrypt)(keystore, password, (_a = options === null || options === void 0 ? void 0 : options.nonStrict) !== null && _a !== void 0 ? _a : true);
        return Object.assign(Object.assign({}, account), { signTransaction: (transaction) => __awaiter(void 0, void 0, void 0, function* () { return signTransactionWithContext(transaction, account.privateKey); }) });
    });
    const createWithContext = () => {
        const account = (0, web3_eth_accounts_1.create)();
        return Object.assign(Object.assign({}, account), { signTransaction: (transaction) => __awaiter(void 0, void 0, void 0, function* () { return signTransactionWithContext(transaction, account.privateKey); }) });
    };
    const wallet = new web3_eth_accounts_1.Wallet({
        create: createWithContext,
        privateKeyToAccount: privateKeyToAccountWithContext,
        decrypt: decryptWithContext,
    });
    return {
        signTransaction: signTransactionWithContext,
        create: createWithContext,
        privateKeyToAccount: privateKeyToAccountWithContext,
        decrypt: decryptWithContext,
        recoverTransaction: web3_eth_accounts_1.recoverTransaction,
        hashMessage: web3_eth_accounts_1.hashMessage,
        sign: web3_eth_accounts_1.sign,
        recover: web3_eth_accounts_1.recover,
        encrypt: web3_eth_accounts_1.encrypt,
        wallet,
        privateKeyToAddress: web3_eth_accounts_1.privateKeyToAddress,
        parseAndValidatePrivateKey: web3_eth_accounts_1.parseAndValidatePrivateKey,
        privateKeyToPublicKey: web3_eth_accounts_1.privateKeyToPublicKey,
    };
};
exports.initAccountsForContext = initAccountsForContext;
//# sourceMappingURL=accounts.js.map