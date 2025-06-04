import { EIP1193Provider } from "./types/provider";
/**
 * Tracks a transaction associated with a given deployment.
 *
 * @param deploymentDir - the directory of the deployment the transaction belongs to
 * @param txHash - the hash of the transaction to track
 * @param provider - a JSON RPC provider to retrieve transaction information from
 * @param requiredConfirmations - the number of confirmations required for the transaction to be considered confirmed
 * @param applyNewMessageFn - only used for ease of testing this function and should not be used otherwise
 *
 * @beta
 */
export declare function trackTransaction(deploymentDir: string, txHash: string, provider: EIP1193Provider, requiredConfirmations?: number, applyNewMessageFn?: (message: any, _a: any, _b: any) => Promise<any>): Promise<string | void>;
//# sourceMappingURL=track-transaction.d.ts.map