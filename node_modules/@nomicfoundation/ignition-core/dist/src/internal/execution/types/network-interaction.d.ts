import { RawStaticCallResult, Transaction } from "./jsonrpc";
/**
 * An interaction with an Ethereum network.
 *
 * It can be either an OnchainInteraction or a StaticCall.
 *
 * OnchainInteractions are interactions that need to be executed with a transaction, while
 * StaticCalls are interactions that can be resolved by your local node.
 */
export type NetworkInteraction = OnchainInteraction | StaticCall;
/**
 * The different types of network interactions.
 */
export declare enum NetworkInteractionType {
    ONCHAIN_INTERACTION = "ONCHAIN_INTERACTION",
    STATIC_CALL = "STATIC_CALL"
}
/**
 * This interface represents any kind of interaction between Ethereum accounts that
 * needs to be executed onchain.
 *
 * To execute this interaction, we need to send a transaction. As not every transaction
 * that we send gets confirmed, we may need to send multiple ones per OnchainInteraction.
 *
 * All the transactions of an OnchainInteraction are sent with the same nonce, so that
 * only one of them can be confirmed.
 *
 * The `nonce` field is only available if we have sent at least one transaction, and we
 * are tracking its progress.
 *
 * If the `nonce` is `undefined`, we either haven't sent any transaction for this
 * OnchainInteraction, or the ones we sent were replaced by transactions sent by the user
 * so we need to restart this OnchainInteraction's execution.
 *
 * The `shouldBeResent` field is `true` only in cases where we want to send a new
 * transaction for this `OnchainInteraction` using the same nonce. This can happen if
 * we need to bump the gas price, or if all of the transactions were dropped from the
 * mempool, yet the nonce is still valid.
 **/
export interface OnchainInteraction {
    id: number;
    type: NetworkInteractionType.ONCHAIN_INTERACTION;
    to: string | undefined;
    data: string;
    value: bigint;
    nonce?: number;
    transactions: Transaction[];
    shouldBeResent: boolean;
}
/**
 * This interface represents a static call to the Ethereum network.
 **/
export interface StaticCall {
    id: number;
    type: NetworkInteractionType.STATIC_CALL;
    to: string | undefined;
    data: string;
    value: bigint;
    from: string;
    result?: RawStaticCallResult;
}
//# sourceMappingURL=network-interaction.d.ts.map