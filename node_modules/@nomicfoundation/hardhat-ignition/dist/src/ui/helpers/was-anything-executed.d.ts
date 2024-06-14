import { UiState } from "../types";
/**
 * Was anything executed during the deployment. We determine this based
 * on whether the batcher indicates that there was at least one batch.
 */
export declare function wasAnythingExecuted({ batches, }: Pick<UiState, "batches">): boolean;
//# sourceMappingURL=was-anything-executed.d.ts.map