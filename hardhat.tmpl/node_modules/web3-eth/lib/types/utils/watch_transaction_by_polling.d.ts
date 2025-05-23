import { EthExecutionAPI, TransactionReceipt } from 'web3-types';
import { Web3Context, Web3PromiEvent } from 'web3-core';
import { DataFormat } from 'web3-types';
import { JsonSchema } from 'web3-validator';
import { SendSignedTransactionEvents, SendTransactionEvents } from '../types.js';
export type Web3PromiEventEventTypeBase<ReturnFormat extends DataFormat> = SendTransactionEvents<ReturnFormat> | SendSignedTransactionEvents<ReturnFormat>;
export type WaitProps<ReturnFormat extends DataFormat, ResolveType = TransactionReceipt> = {
    web3Context: Web3Context<EthExecutionAPI>;
    transactionReceipt: TransactionReceipt;
    customTransactionReceiptSchema?: JsonSchema;
    transactionPromiEvent: Web3PromiEvent<ResolveType, Web3PromiEventEventTypeBase<ReturnFormat>>;
    returnFormat: ReturnFormat;
};
/**
 * This function watches a Transaction by subscribing to new heads.
 * It is used by `watchTransactionForConfirmations`, in case the provider does not support subscription.
 * And it is also used by `watchTransactionBySubscription`, as a fallback, if the subscription failed for any reason.
 */
export declare const watchTransactionByPolling: <ReturnFormat extends DataFormat, ResolveType = TransactionReceipt>({ web3Context, transactionReceipt, transactionPromiEvent, customTransactionReceiptSchema, returnFormat, }: WaitProps<ReturnFormat, ResolveType>) => void;
//# sourceMappingURL=watch_transaction_by_polling.d.ts.map