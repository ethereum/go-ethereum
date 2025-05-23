"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.wasAnythingExecuted = void 0;
/**
 * Was anything executed during the deployment. We determine this based
 * on whether the batcher indicates that there was at least one batch.
 */
function wasAnythingExecuted({ batches, }) {
    return batches.length > 0;
}
exports.wasAnythingExecuted = wasAnythingExecuted;
//# sourceMappingURL=was-anything-executed.js.map