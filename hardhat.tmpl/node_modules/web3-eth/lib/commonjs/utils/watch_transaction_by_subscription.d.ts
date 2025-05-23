import { TransactionReceipt } from 'web3-types';
import { DataFormat } from 'web3-types';
import { WaitProps } from './watch_transaction_by_polling.js';
/**
 * This function watches a Transaction by subscribing to new heads.
 * It is used by `watchTransactionForConfirmations`, in case the provider supports subscription.
 */
export declare const watchTransactionBySubscription: <ReturnFormat extends DataFormat, ResolveType = TransactionReceipt>({ web3Context, transactionReceipt, transactionPromiEvent, customTransactionReceiptSchema, returnFormat, }: WaitProps<ReturnFormat, ResolveType>) => void;
