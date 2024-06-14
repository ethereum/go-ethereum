import { JsonRpcClient } from "../../jsonrpc-client";
import { TransactionTrackingTimer } from "../../transaction-tracking-timer";
import { CallExecutionState, DeploymentExecutionState, SendDataExecutionState } from "../../types/execution-state";
import { OnchainInteractionBumpFeesMessage, OnchainInteractionTimeoutMessage, TransactionConfirmMessage } from "../../types/messages";
export interface GetTransactionRetryConfig {
    maxRetries: number;
    retryInterval: number;
}
/**
 * Checks the transactions of the latest network interaction of the execution state,
 * and returns a message, or undefined if we need to wait for more confirmations.
 *
 * This method can return messages indicating that a transaction has enough confirmations,
 * that we need to bump the fees, or that the execution of this onchain interaction has
 * timed out.
 *
 * If all of the transactions of the latest network interaction have been dropped, this
 * method throws an IgnitionError.
 *
 * SIDE EFFECTS: This function doesn't have any side effects.
 *
 * @param exState The execution state that requires the transactions to be checked.
 * @param jsonRpcClient The JSON RPC client to use for accessing the network.
 * @param transactionTrackingTimer The TransactionTrackingTimer to use for checking the
 *  if a transaction has been pending for too long.
 * @param requiredConfirmations The number of confirmations required for a transaction
 *  to be considered confirmed.
 * @param millisecondBeforeBumpingFees The number of milliseconds before bumping the fees
 *  of a transaction.
 * @param maxFeeBumps The maximum number of times we can bump the fees of a transaction
 *  before considering the onchain interaction timed out.
 * @returns A message indicating the result of checking the transactions of the latest
 *  network interaction.
 */
export declare function monitorOnchainInteraction(exState: DeploymentExecutionState | CallExecutionState | SendDataExecutionState, jsonRpcClient: JsonRpcClient, transactionTrackingTimer: TransactionTrackingTimer, requiredConfirmations: number, millisecondBeforeBumpingFees: number, maxFeeBumps: number, getTransactionRetryConfig?: GetTransactionRetryConfig): Promise<TransactionConfirmMessage | OnchainInteractionBumpFeesMessage | OnchainInteractionTimeoutMessage | undefined>;
//# sourceMappingURL=monitor-onchain-interaction.d.ts.map