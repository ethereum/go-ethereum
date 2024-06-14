import { JsonRpcClient } from "../jsonrpc-client";
/**
 * This interface is meant to be used to fetch new nonces for transactions.
 */
export interface NonceManager {
    /**
     * Returns the next nonce for a given sender, throwing if its not the one
     * expected by the network.
     *
     * If a nonce is returned by this method it must be immediately used to
     * send a transaction. If it can't be used, Ignition's execution must be
     * interrupted.
     */
    getNextNonce(sender: string): Promise<number>;
    /**
     * Reverts the last nonce allocation for a given sender.
     *
     * This method is used when a nonce has been allocated,
     * but the transaction fails during simulation and is not sent.
     */
    revertNonce(sender: string): void;
}
/**
 * An implementation of NonceManager that validates the nonces using
 * the _maxUsedNonce params and a JsonRpcClient.
 */
export declare class JsonRpcNonceManager implements NonceManager {
    private readonly _jsonRpcClient;
    private readonly _maxUsedNonce;
    constructor(_jsonRpcClient: JsonRpcClient, _maxUsedNonce: {
        [sender: string]: number;
    });
    getNextNonce(sender: string): Promise<number>;
    revertNonce(sender: string): void;
}
//# sourceMappingURL=json-rpc-nonce-manager.d.ts.map