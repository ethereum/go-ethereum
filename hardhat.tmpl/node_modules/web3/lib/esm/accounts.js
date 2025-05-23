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
import { ETH_DATA_FORMAT } from 'web3-types';
import { format } from 'web3-utils';
import { prepareTransactionForSigning } from 'web3-eth';
import { create, decrypt, encrypt, hashMessage, privateKeyToAccount, recover, recoverTransaction, signTransaction, sign, Wallet, privateKeyToAddress, parseAndValidatePrivateKey, privateKeyToPublicKey, } from 'web3-eth-accounts';
/**
 * Initialize the accounts module for the given context.
 *
 * To avoid multiple package dependencies for `web3-eth-accounts` we are creating
 * this function in `web3` package. In future the actual `web3-eth-accounts` package
 * should be converted to context aware.
 */
export const initAccountsForContext = (context) => {
    const signTransactionWithContext = (transaction, privateKey) => __awaiter(void 0, void 0, void 0, function* () {
        const tx = yield prepareTransactionForSigning(transaction, context);
        const privateKeyBytes = format({ format: 'bytes' }, privateKey, ETH_DATA_FORMAT);
        return signTransaction(tx, privateKeyBytes);
    });
    const privateKeyToAccountWithContext = (privateKey) => {
        const account = privateKeyToAccount(privateKey);
        return Object.assign(Object.assign({}, account), { signTransaction: (transaction) => __awaiter(void 0, void 0, void 0, function* () { return signTransactionWithContext(transaction, account.privateKey); }) });
    };
    const decryptWithContext = (keystore, password, options) => __awaiter(void 0, void 0, void 0, function* () {
        var _a;
        const account = yield decrypt(keystore, password, (_a = options === null || options === void 0 ? void 0 : options.nonStrict) !== null && _a !== void 0 ? _a : true);
        return Object.assign(Object.assign({}, account), { signTransaction: (transaction) => __awaiter(void 0, void 0, void 0, function* () { return signTransactionWithContext(transaction, account.privateKey); }) });
    });
    const createWithContext = () => {
        const account = create();
        return Object.assign(Object.assign({}, account), { signTransaction: (transaction) => __awaiter(void 0, void 0, void 0, function* () { return signTransactionWithContext(transaction, account.privateKey); }) });
    };
    const wallet = new Wallet({
        create: createWithContext,
        privateKeyToAccount: privateKeyToAccountWithContext,
        decrypt: decryptWithContext,
    });
    return {
        signTransaction: signTransactionWithContext,
        create: createWithContext,
        privateKeyToAccount: privateKeyToAccountWithContext,
        decrypt: decryptWithContext,
        recoverTransaction,
        hashMessage,
        sign,
        recover,
        encrypt,
        wallet,
        privateKeyToAddress,
        parseAndValidatePrivateKey,
        privateKeyToPublicKey,
    };
};
//# sourceMappingURL=accounts.js.map