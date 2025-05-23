import { Web3Context } from 'web3-core';
import { EthExecutionAPI, Bytes, TransactionReceipt, DataFormat } from 'web3-types';
export declare function waitForTransactionReceipt<ReturnFormat extends DataFormat>(web3Context: Web3Context<EthExecutionAPI>, transactionHash: Bytes, returnFormat: ReturnFormat, customGetTransactionReceipt?: (web3Context: Web3Context<EthExecutionAPI>, transactionHash: Bytes, returnFormat: ReturnFormat) => Promise<TransactionReceipt>): Promise<TransactionReceipt>;
//# sourceMappingURL=wait_for_transaction_receipt.d.ts.map