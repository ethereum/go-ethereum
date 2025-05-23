import { Web3Context } from 'web3-core';
import { EthExecutionAPI, Bytes } from 'web3-types';
import { AsyncFunction } from 'web3-utils';
/**
 * An internal function to send a transaction or throws if sending did not finish during the timeout during the blocks-timeout.
 * @param web3Context - the context to read the configurations from
 * @param sendTransactionFunc - the function that will send the transaction (could be sendTransaction or sendRawTransaction)
 * @param transactionHash - to be used inside the exception message if there will be any exceptions.
 * @returns the Promise<string> returned by the `sendTransactionFunc`.
 */
export declare function trySendTransaction(web3Context: Web3Context<EthExecutionAPI>, sendTransactionFunc: AsyncFunction<string>, transactionHash?: Bytes): Promise<string>;
//# sourceMappingURL=try_send_transaction.d.ts.map