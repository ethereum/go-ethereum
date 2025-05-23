import { addInstrumentationHandler, logger } from '@sentry/utils';
import { SpanStatus } from './spanstatus';
import { getActiveTransaction } from './utils';
/**
 * Configures global error listeners
 */
export function registerErrorInstrumentation() {
    addInstrumentationHandler({
        callback: errorCallback,
        type: 'error',
    });
    addInstrumentationHandler({
        callback: errorCallback,
        type: 'unhandledrejection',
    });
}
/**
 * If an error or unhandled promise occurs, we mark the active transaction as failed
 */
function errorCallback() {
    var activeTransaction = getActiveTransaction();
    if (activeTransaction) {
        logger.log("[Tracing] Transaction: " + SpanStatus.InternalError + " -> Global error occured");
        activeTransaction.setStatus(SpanStatus.InternalError);
    }
}
//# sourceMappingURL=errors.js.map