import { getGlobalObject, logger } from '@sentry/utils';
import { SpanStatus } from '../spanstatus';
import { getActiveTransaction } from '../utils';
var global = getGlobalObject();
/**
 * Add a listener that cancels and finishes a transaction when the global
 * document is hidden.
 */
export function registerBackgroundTabDetection() {
    if (global && global.document) {
        global.document.addEventListener('visibilitychange', function () {
            var activeTransaction = getActiveTransaction();
            if (global.document.hidden && activeTransaction) {
                logger.log("[Tracing] Transaction: " + SpanStatus.Cancelled + " -> since tab moved to the background, op: " + activeTransaction.op);
                // We should not set status if it is already set, this prevent important statuses like
                // error or data loss from being overwritten on transaction.
                if (!activeTransaction.status) {
                    activeTransaction.setStatus(SpanStatus.Cancelled);
                }
                activeTransaction.setTag('visibilitychange', 'document.hidden');
                activeTransaction.finish();
            }
        });
    }
    else {
        logger.warn('[Tracing] Could not set up background tab detection due to lack of global document');
    }
}
//# sourceMappingURL=backgroundtab.js.map