import { IgnitionModule, IgnitionModuleResult } from "../../../types/module";
import { JsonRpcClient } from "../jsonrpc-client";
import { DeploymentState } from "../types/deployment-state";
import { OnchainInteractionDroppedMessage, OnchainInteractionReplacedByUserMessage } from "../types/messages";
/**
 * This function is meant to be used to sync the local state's nonces
 * with those of the network.
 *
 * This function has three goals:
 *  - Ensure that we never proceed with Ignition if there are transactions
 *    sent by the user that haven't got enough confirmations yet.
 *  - Detect if the user has repaced a transaction sent by Ignition.
 *  - Distinguish if a transaction not being present in the mempool was
 *    dropped or replaced by the user.
 *
 * The way this function works means that there's one case we don't handle:
 *  - If the user replaces a transaction sent by Ignition with one of theirs
 *    we'll allocate a new nonce for our transaction.
 *  - If the user's transaction gets dropped, we won't reallocate the original
 *    nonce for any of our transactions, and Ignition will eventually fail,
 *    setting one or more ExecutionState as TIMEOUT.
 *  - This is intentional, as reusing user nonces can lead to unexpected
 *    results.
 *  - To understand this better, please consider that a transaction being
 *    dropped by your node doesn't mean that the entire network forgot about it.
 *
 * @param jsonRpcClient The client used to interact with the network.
 * @param deploymentState The current deployment state, which we want to sync.
 * @param requiredConfirmations The amount of confirmations that a transaction
 *  must have before we consider it confirmed.
 * @returns The messages that should be applied to the state.
 */
export declare function getNonceSyncMessages(jsonRpcClient: JsonRpcClient, deploymentState: DeploymentState, ignitionModule: IgnitionModule<string, string, IgnitionModuleResult<string>>, accounts: string[], defaultSender: string, requiredConfirmations: number): Promise<Array<OnchainInteractionReplacedByUserMessage | OnchainInteractionDroppedMessage>>;
//# sourceMappingURL=get-nonce-sync-messages.d.ts.map