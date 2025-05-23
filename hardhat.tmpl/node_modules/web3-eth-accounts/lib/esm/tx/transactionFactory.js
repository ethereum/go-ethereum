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
import { isUint8Array } from 'web3-utils';
import { toUint8Array, uint8ArrayToBigInt } from '../common/utils.js';
import { FeeMarketEIP1559Transaction } from './eip1559Transaction.js';
import { AccessListEIP2930Transaction } from './eip2930Transaction.js';
import { Transaction } from './legacyTransaction.js';
const extraTxTypes = new Map();
// eslint-disable-next-line @typescript-eslint/no-extraneous-class
export class TransactionFactory {
    // It is not possible to instantiate a TransactionFactory object.
    // eslint-disable-next-line no-useless-constructor, @typescript-eslint/no-empty-function
    constructor() { }
    static typeToInt(txType) {
        return Number(uint8ArrayToBigInt(toUint8Array(txType)));
    }
    static registerTransactionType(type, txClass) {
        const txType = TransactionFactory.typeToInt(type);
        extraTxTypes.set(txType, txClass);
    }
    /**
     * Create a transaction from a `txData` object
     *
     * @param txData - The transaction data. The `type` field will determine which transaction type is returned (if undefined, creates a legacy transaction)
     * @param txOptions - Options to pass on to the constructor of the transaction
     */
    static fromTxData(txData, txOptions = {}) {
        if (!('type' in txData) || txData.type === undefined) {
            // Assume legacy transaction
            return Transaction.fromTxData(txData, txOptions);
        }
        const txType = TransactionFactory.typeToInt(txData.type);
        if (txType === 0) {
            return Transaction.fromTxData(txData, txOptions);
        }
        if (txType === 1) {
            // eslint-disable-next-line @typescript-eslint/consistent-type-assertions
            return AccessListEIP2930Transaction.fromTxData(
            // eslint-disable-next-line @typescript-eslint/consistent-type-assertions
            txData, txOptions);
        }
        if (txType === 2) {
            return FeeMarketEIP1559Transaction.fromTxData(
            // eslint-disable-next-line @typescript-eslint/consistent-type-assertions
            txData, txOptions);
        }
        const ExtraTransaction = extraTxTypes.get(txType);
        if (ExtraTransaction === null || ExtraTransaction === void 0 ? void 0 : ExtraTransaction.fromTxData) {
            return ExtraTransaction.fromTxData(txData, txOptions);
        }
        throw new Error(`Tx instantiation with type ${txType} not supported`);
    }
    /**
     * This method tries to decode serialized data.
     *
     * @param data - The data Uint8Array
     * @param txOptions - The transaction options
     */
    static fromSerializedData(data, txOptions = {}) {
        if (data[0] <= 0x7f) {
            // Determine the type.
            switch (data[0]) {
                case 1:
                    return AccessListEIP2930Transaction.fromSerializedTx(data, txOptions);
                case 2:
                    return FeeMarketEIP1559Transaction.fromSerializedTx(data, txOptions);
                default: {
                    const ExtraTransaction = extraTxTypes.get(Number(data[0]));
                    if (ExtraTransaction === null || ExtraTransaction === void 0 ? void 0 : ExtraTransaction.fromSerializedTx) {
                        return ExtraTransaction.fromSerializedTx(data, txOptions);
                    }
                    throw new Error(`TypedTransaction with ID ${data[0]} unknown`);
                }
            }
        }
        else {
            return Transaction.fromSerializedTx(data, txOptions);
        }
    }
    /**
     * When decoding a BlockBody, in the transactions field, a field is either:
     * A Uint8Array (a TypedTransaction - encoded as TransactionType || rlp(TransactionPayload))
     * A Uint8Array[] (Legacy Transaction)
     * This method returns the right transaction.
     *
     * @param data - A Uint8Array or Uint8Array[]
     * @param txOptions - The transaction options
     */
    static fromBlockBodyData(data, txOptions = {}) {
        if (isUint8Array(data)) {
            return this.fromSerializedData(data, txOptions);
        }
        if (Array.isArray(data)) {
            // It is a legacy transaction
            return Transaction.fromValuesArray(data, txOptions);
        }
        throw new Error('Cannot decode transaction: unknown type input');
    }
}
//# sourceMappingURL=transactionFactory.js.map