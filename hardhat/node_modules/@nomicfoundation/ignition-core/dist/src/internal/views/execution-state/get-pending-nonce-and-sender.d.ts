import { ExecutionState } from "../../execution/types/execution-state";
/**
 * Returns the nonce and sender of a pending transaction of the execution state,
 * if any.
 *
 * @param exState The execution state to check.
 * @returns Returns the nonce and sender of the last (and only) pending tx
 *  of the execution state, if any.
 */
export declare function getPendingNonceAndSender(exState: ExecutionState): {
    nonce: number;
    sender: string;
} | undefined;
//# sourceMappingURL=get-pending-nonce-and-sender.d.ts.map