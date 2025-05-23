Object.defineProperty(exports, "__esModule", { value: true });
var utils_1 = require("@sentry/utils");
var spanstatus_1 = require("./spanstatus");
var utils_2 = require("./utils");
/**
 * Configures global error listeners
 */
function registerErrorInstrumentation() {
    utils_1.addInstrumentationHandler({
        callback: errorCallback,
        type: 'error',
    });
    utils_1.addInstrumentationHandler({
        callback: errorCallback,
        type: 'unhandledrejection',
    });
}
exports.registerErrorInstrumentation = registerErrorInstrumentation;
/**
 * If an error or unhandled promise occurs, we mark the active transaction as failed
 */
function errorCallback() {
    var activeTransaction = utils_2.getActiveTransaction();
    if (activeTransaction) {
        utils_1.logger.log("[Tracing] Transaction: " + spanstatus_1.SpanStatus.InternalError + " -> Global error occured");
        activeTransaction.setStatus(spanstatus_1.SpanStatus.InternalError);
    }
}
//# sourceMappingURL=errors.js.map