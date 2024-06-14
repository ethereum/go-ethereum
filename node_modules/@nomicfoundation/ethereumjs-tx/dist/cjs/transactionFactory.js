"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.TransactionFactory = void 0;
const ethereumjs_util_1 = require("@nomicfoundation/ethereumjs-util");
const eip1559Transaction_js_1 = require("./eip1559Transaction.js");
const eip2930Transaction_js_1 = require("./eip2930Transaction.js");
const eip4844Transaction_js_1 = require("./eip4844Transaction.js");
const fromRpc_js_1 = require("./fromRpc.js");
const legacyTransaction_js_1 = require("./legacyTransaction.js");
const types_js_1 = require("./types.js");
class TransactionFactory {
    // It is not possible to instantiate a TransactionFactory object.
    constructor() { }
    /**
     * Create a transaction from a `txData` object
     *
     * @param txData - The transaction data. The `type` field will determine which transaction type is returned (if undefined, creates a legacy transaction)
     * @param txOptions - Options to pass on to the constructor of the transaction
     */
    static fromTxData(txData, txOptions = {}) {
        if (!('type' in txData) || txData.type === undefined) {
            // Assume legacy transaction
            return legacyTransaction_js_1.LegacyTransaction.fromTxData(txData, txOptions);
        }
        else {
            if ((0, types_js_1.isLegacyTxData)(txData)) {
                return legacyTransaction_js_1.LegacyTransaction.fromTxData(txData, txOptions);
            }
            else if ((0, types_js_1.isAccessListEIP2930TxData)(txData)) {
                return eip2930Transaction_js_1.AccessListEIP2930Transaction.fromTxData(txData, txOptions);
            }
            else if ((0, types_js_1.isFeeMarketEIP1559TxData)(txData)) {
                return eip1559Transaction_js_1.FeeMarketEIP1559Transaction.fromTxData(txData, txOptions);
            }
            else if ((0, types_js_1.isBlobEIP4844TxData)(txData)) {
                return eip4844Transaction_js_1.BlobEIP4844Transaction.fromTxData(txData, txOptions);
            }
            else {
                throw new Error(`Tx instantiation with type ${txData?.type} not supported`);
            }
        }
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
                case types_js_1.TransactionType.AccessListEIP2930:
                    return eip2930Transaction_js_1.AccessListEIP2930Transaction.fromSerializedTx(data, txOptions);
                case types_js_1.TransactionType.FeeMarketEIP1559:
                    return eip1559Transaction_js_1.FeeMarketEIP1559Transaction.fromSerializedTx(data, txOptions);
                case types_js_1.TransactionType.BlobEIP4844:
                    return eip4844Transaction_js_1.BlobEIP4844Transaction.fromSerializedTx(data, txOptions);
                default:
                    throw new Error(`TypedTransaction with ID ${data[0]} unknown`);
            }
        }
        else {
            return legacyTransaction_js_1.LegacyTransaction.fromSerializedTx(data, txOptions);
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
        if (data instanceof Uint8Array) {
            return this.fromSerializedData(data, txOptions);
        }
        else if (Array.isArray(data)) {
            // It is a legacy transaction
            return legacyTransaction_js_1.LegacyTransaction.fromValuesArray(data, txOptions);
        }
        else {
            throw new Error('Cannot decode transaction: unknown type input');
        }
    }
    /**
     *  Method to retrieve a transaction from the provider
     * @param provider - a url string for a JSON-RPC provider or an Ethers JsonRPCProvider object
     * @param txHash - Transaction hash
     * @param txOptions - The transaction options
     * @returns the transaction specified by `txHash`
     */
    static async fromJsonRpcProvider(provider, txHash, txOptions) {
        const prov = (0, ethereumjs_util_1.getProvider)(provider);
        const txData = await (0, ethereumjs_util_1.fetchFromProvider)(prov, {
            method: 'eth_getTransactionByHash',
            params: [txHash],
        });
        if (txData === null) {
            throw new Error('No data returned from provider');
        }
        return TransactionFactory.fromRPC(txData, txOptions);
    }
    /**
     * Method to decode data retrieved from RPC, such as `eth_getTransactionByHash`
     * Note that this normalizes some of the parameters
     * @param txData The RPC-encoded data
     * @param txOptions The transaction options
     * @returns
     */
    static async fromRPC(txData, txOptions = {}) {
        return TransactionFactory.fromTxData((0, fromRpc_js_1.normalizeTxParams)(txData), txOptions);
    }
}
exports.TransactionFactory = TransactionFactory;
//# sourceMappingURL=transactionFactory.js.map