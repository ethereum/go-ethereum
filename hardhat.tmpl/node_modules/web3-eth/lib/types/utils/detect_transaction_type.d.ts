import { TransactionTypeParser, Web3Context } from 'web3-core';
import { EthExecutionAPI } from 'web3-types';
import { InternalTransaction } from '../types.js';
export declare const defaultTransactionTypeParser: TransactionTypeParser;
export declare const detectTransactionType: (transaction: InternalTransaction, web3Context?: Web3Context<EthExecutionAPI>) => string | undefined;
export declare const detectRawTransactionType: (transaction: Uint8Array) => string;
//# sourceMappingURL=detect_transaction_type.d.ts.map