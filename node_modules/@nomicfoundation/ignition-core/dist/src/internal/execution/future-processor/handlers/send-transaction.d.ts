import { JsonRpcClient } from "../../jsonrpc-client";
import { NonceManager } from "../../nonce-management/json-rpc-nonce-manager";
import { TransactionTrackingTimer } from "../../transaction-tracking-timer";
import { CallExecutionState, DeploymentExecutionState, SendDataExecutionState } from "../../types/execution-state";
import { ExecutionStrategy } from "../../types/execution-strategy";
import { CallExecutionStateCompleteMessage, DeploymentExecutionStateCompleteMessage, SendDataExecutionStateCompleteMessage, TransactionSendMessage } from "../../types/messages";
/**
 * Sends a transaction for the execution state's latest NetworkInteraction
 * and returns a TransactionSendMessage, or an execution state complete message
 * in case of an error.
 *
 * This function can send the first transaction of an OnchainInteraction, as well
 * as new transactions to bump fees and recovering from dropped transactions.
 *
 * SIDE EFFECTS: This function has side effects, as it sends a transaction. These
 *  include: sending the transaction to the network, allocating a nonce in the
 *  NonceManager if needed, and adding the transaction to the TransactionTrackingTimer.
 *
 * @param exState The execution state that requires a transaction to be sent.
 * @param executionStrategy The execution strategy to use for simulations.
 * @param jsonRpcClient The JSON RPC client to use for the transaction.
 * @param nonceManager The NonceManager to allocate nonces if needed.
 * @param transactionTrackingTimer The TransactionTrackingTimer to add the transaction to.
 * @returns A message indicating the result of trying to send the transaction.
 */
export declare function sendTransaction(exState: DeploymentExecutionState | CallExecutionState | SendDataExecutionState, executionStrategy: ExecutionStrategy, jsonRpcClient: JsonRpcClient, nonceManager: NonceManager, transactionTrackingTimer: TransactionTrackingTimer): Promise<TransactionSendMessage | DeploymentExecutionStateCompleteMessage | CallExecutionStateCompleteMessage | SendDataExecutionStateCompleteMessage>;
//# sourceMappingURL=send-transaction.d.ts.map