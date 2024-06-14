import { FeeMarketEIP1559Transaction } from './eip1559Transaction.js';
import { AccessListEIP2930Transaction } from './eip2930Transaction.js';
import { BlobEIP4844Transaction } from './eip4844Transaction.js';
import { LegacyTransaction } from './legacyTransaction.js';
import { TransactionType } from './types.js';
import type { Transaction, TxData, TxOptions, TypedTxData } from './types.js';
import type { EthersProvider } from '@nomicfoundation/ethereumjs-util';
export declare class TransactionFactory {
    private constructor();
    /**
     * Create a transaction from a `txData` object
     *
     * @param txData - The transaction data. The `type` field will determine which transaction type is returned (if undefined, creates a legacy transaction)
     * @param txOptions - Options to pass on to the constructor of the transaction
     */
    static fromTxData<T extends TransactionType>(txData: TypedTxData, txOptions?: TxOptions): Transaction[T];
    /**
     * This method tries to decode serialized data.
     *
     * @param data - The data Uint8Array
     * @param txOptions - The transaction options
     */
    static fromSerializedData<T extends TransactionType>(data: Uint8Array, txOptions?: TxOptions): Transaction[T];
    /**
     * When decoding a BlockBody, in the transactions field, a field is either:
     * A Uint8Array (a TypedTransaction - encoded as TransactionType || rlp(TransactionPayload))
     * A Uint8Array[] (Legacy Transaction)
     * This method returns the right transaction.
     *
     * @param data - A Uint8Array or Uint8Array[]
     * @param txOptions - The transaction options
     */
    static fromBlockBodyData(data: Uint8Array | Uint8Array[], txOptions?: TxOptions): LegacyTransaction | AccessListEIP2930Transaction | FeeMarketEIP1559Transaction | BlobEIP4844Transaction;
    /**
     *  Method to retrieve a transaction from the provider
     * @param provider - a url string for a JSON-RPC provider or an Ethers JsonRPCProvider object
     * @param txHash - Transaction hash
     * @param txOptions - The transaction options
     * @returns the transaction specified by `txHash`
     */
    static fromJsonRpcProvider(provider: string | EthersProvider, txHash: string, txOptions?: TxOptions): Promise<LegacyTransaction | AccessListEIP2930Transaction | FeeMarketEIP1559Transaction | BlobEIP4844Transaction>;
    /**
     * Method to decode data retrieved from RPC, such as `eth_getTransactionByHash`
     * Note that this normalizes some of the parameters
     * @param txData The RPC-encoded data
     * @param txOptions The transaction options
     * @returns
     */
    static fromRPC<T extends TransactionType>(txData: TxData[T], txOptions?: TxOptions): Promise<Transaction[T]>;
}
//# sourceMappingURL=transactionFactory.d.ts.map