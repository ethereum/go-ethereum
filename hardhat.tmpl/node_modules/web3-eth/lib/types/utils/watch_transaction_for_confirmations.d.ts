import { Bytes, EthExecutionAPI, TransactionReceipt } from 'web3-types';
import { Web3Context, Web3PromiEvent } from 'web3-core';
import { JsonSchema } from 'web3-validator';
import { DataFormat } from 'web3-types';
import { Web3PromiEventEventTypeBase } from './watch_transaction_by_polling.js';
export declare function watchTransactionForConfirmations<ReturnFormat extends DataFormat, Web3PromiEventEventType extends Web3PromiEventEventTypeBase<ReturnFormat>, ResolveType = TransactionReceipt>(web3Context: Web3Context<EthExecutionAPI>, transactionPromiEvent: Web3PromiEvent<ResolveType, Web3PromiEventEventType>, transactionReceipt: TransactionReceipt, transactionHash: Bytes, returnFormat: ReturnFormat, customTransactionReceiptSchema?: JsonSchema): void;
//# sourceMappingURL=watch_transaction_for_confirmations.d.ts.map