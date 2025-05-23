"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.calculateBatchDisplay = void 0;
const types_1 = require("../types");
function calculateBatchDisplay(state) {
    const batch = state.batches[state.currentBatch - 1];
    const height = batch.length + (state.ledgerMessageIsDisplayed ? 4 : 2);
    let text = `Batch #${state.currentBatch}\n`;
    text += batch
        .sort((a, b) => a.futureId.localeCompare(b.futureId))
        .map((v) => _futureStatus(v, state.gasBumps, state.maxFeeBumps))
        .join("\n");
    text += "\n";
    if (state.ledger) {
        text += `\n  Ledger: ${state.ledgerMessage}\n`;
    }
    return { text, height };
}
exports.calculateBatchDisplay = calculateBatchDisplay;
function _futureStatus(future, gasBumps, maxFeeBumps) {
    switch (future.status.type) {
        case types_1.UiFutureStatusType.UNSTARTED: {
            const gas = gasBumps[future.futureId];
            return `  Executing ${future.futureId}${gas !== undefined
                ? ` - bumping gas fee (${gas}/${maxFeeBumps})...`
                : "..."}`;
        }
        case types_1.UiFutureStatusType.SUCCESS: {
            return `  Executed ${future.futureId}`;
        }
        case types_1.UiFutureStatusType.TIMEDOUT: {
            return `  Timed out ${future.futureId}`;
        }
        case types_1.UiFutureStatusType.ERRORED: {
            return `  Failed ${future.futureId}`;
        }
        case types_1.UiFutureStatusType.HELD: {
            return `  Held ${future.futureId}`;
        }
    }
}
//# sourceMappingURL=calculate-batch-display.js.map