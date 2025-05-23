import { CallExecutionState, DeploymentExecutionState, SendDataExecutionState, StaticCallExecutionState } from "../../types/execution-state";
import { NetworkInteractionRequestMessage, OnchainInteractionBumpFeesMessage, OnchainInteractionDroppedMessage, OnchainInteractionReplacedByUserMessage, OnchainInteractionTimeoutMessage, StaticCallCompleteMessage, TransactionConfirmMessage, TransactionPrepareSendMessage, TransactionSendMessage } from "../../types/messages";
/**
 * Add a new network interaction to the execution state.
 *
 * @param state - the execution state that will be added to
 * @param action - the request message that contains the network interaction
 * @returns a copy of the execution state with the addition network interaction
 */
export declare function appendNetworkInteraction<ExState extends DeploymentExecutionState | CallExecutionState | StaticCallExecutionState | SendDataExecutionState>(state: ExState, action: NetworkInteractionRequestMessage): ExState;
/**
 * Add a transaction to an onchain interaction within an execution state.
 *
 * If the onchain interaction didn't have a nonce yet, it will be set to
 * the nonce of the transaction.
 *
 * This function also sets the onchain interaction's `shouldBeResent` flag
 * to `false`.
 *
 * @param state - the execution state that will be added to
 * @param action - the request message that contains the transaction
 * @returns a copy of the execution state with the additional transaction
 */
export declare function appendTransactionToOnchainInteraction<ExState extends DeploymentExecutionState | CallExecutionState | StaticCallExecutionState | SendDataExecutionState>(state: ExState, action: TransactionSendMessage): ExState;
/**
 * Sets the nonce of the onchain interaction within an execution state.
 *
 * @param state - the execution state that will be added to
 * @param action - the request message that contains the transaction prepare message
 * @returns a copy of the execution state with the nonce set
 */
export declare function applyNonceToOnchainInteraction<ExState extends DeploymentExecutionState | CallExecutionState | StaticCallExecutionState | SendDataExecutionState>(state: ExState, action: TransactionPrepareSendMessage): ExState;
/**
 * Confirm a transaction for an onchain interaction within an execution state.
 *
 * @param state - the execution state that will be updated within
 * @param action - the request message that contains the transaction details
 * @returns a copy of the execution state with transaction confirmed
 */
export declare function confirmTransaction<ExState extends DeploymentExecutionState | CallExecutionState | SendDataExecutionState>(state: ExState, action: TransactionConfirmMessage): ExState;
/**
 * Complete the static call network interaction within an execution state.
 *
 * @param state - the execution state that will be updated
 * @param action - the request message that contains the static call result details
 * @returns a copy of the execution state with the static call confirmed
 */
export declare function completeStaticCall<ExState extends DeploymentExecutionState | CallExecutionState | SendDataExecutionState | StaticCallExecutionState>(state: ExState, action: StaticCallCompleteMessage): ExState;
/**
 * Sets the state `shouldBeResent` of an OnchainInteraction to `true`
 * so that a new transaction with higher fees is sent.
 *
 * @param state - the execution state that will be updated within
 * @param action - the request message that contains the onchain interaction details
 * @returns a copy of the execution state with transaction confirmed
 */
export declare function bumpOnchainInteractionFees<ExState extends DeploymentExecutionState | CallExecutionState | SendDataExecutionState>(state: ExState, action: OnchainInteractionBumpFeesMessage): ExState;
/**
 * Sets the state `shouldBeResent` of a dropped OnchainInteraction to `true`
 * so that a new transaction is sent.
 *
 * @param state - the execution state that will be updated within
 * @param action - the request message that contains the onchain interaction details
 * @returns a copy of the execution state with transaction confirmed
 */
export declare function resendDroppedOnchainInteraction<ExState extends DeploymentExecutionState | CallExecutionState | SendDataExecutionState>(state: ExState, action: OnchainInteractionDroppedMessage): ExState;
/**
 * Resets an OnchainInteraction's nonce, transactions and shouldBeResent
 * due to the user having invalidated the nonce that has been used.
 *
 * @param state - the execution state that will be updated within
 * @param action - the request message that contains the onchain interaction details
 * @returns a copy of the execution state with transaction confirmed
 */
export declare function resetOnchainInteractionReplacedByUser<ExState extends DeploymentExecutionState | CallExecutionState | SendDataExecutionState>(state: ExState, action: OnchainInteractionReplacedByUserMessage): ExState;
/**
 * Sets an execution state to `TIMEOUT` due to an onchain interaction
 * not being confirmed within the timeout period.
 */
export declare function onchainInteractionTimedOut<ExState extends DeploymentExecutionState | CallExecutionState | SendDataExecutionState>(state: ExState, _action: OnchainInteractionTimeoutMessage): ExState;
//# sourceMappingURL=network-interaction-helpers.d.ts.map