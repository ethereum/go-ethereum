Object.defineProperty(exports, "__esModule", { value: true });
var tslib_1 = require("tslib");
var utils_1 = require("@sentry/utils");
var hubextensions_1 = require("../hubextensions");
var idletransaction_1 = require("../idletransaction");
var spanstatus_1 = require("../spanstatus");
var utils_2 = require("../utils");
var backgroundtab_1 = require("./backgroundtab");
var metrics_1 = require("./metrics");
var request_1 = require("./request");
var router_1 = require("./router");
exports.DEFAULT_MAX_TRANSACTION_DURATION_SECONDS = 600;
var DEFAULT_BROWSER_TRACING_OPTIONS = tslib_1.__assign({ idleTimeout: idletransaction_1.DEFAULT_IDLE_TIMEOUT, markBackgroundTransactions: true, maxTransactionDuration: exports.DEFAULT_MAX_TRANSACTION_DURATION_SECONDS, routingInstrumentation: router_1.defaultRoutingInstrumentation, startTransactionOnLocationChange: true, startTransactionOnPageLoad: true }, request_1.defaultRequestInstrumentationOptions);
/**
 * The Browser Tracing integration automatically instruments browser pageload/navigation
 * actions as transactions, and captures requests, metrics and errors as spans.
 *
 * The integration can be configured with a variety of options, and can be extended to use
 * any routing library. This integration uses {@see IdleTransaction} to create transactions.
 */
var BrowserTracing = /** @class */ (function () {
    function BrowserTracing(_options) {
        /**
         * @inheritDoc
         */
        this.name = BrowserTracing.id;
        this._metrics = new metrics_1.MetricsInstrumentation();
        this._emitOptionsWarning = false;
        var tracingOrigins = request_1.defaultRequestInstrumentationOptions.tracingOrigins;
        // NOTE: Logger doesn't work in constructors, as it's initialized after integrations instances
        if (_options &&
            _options.tracingOrigins &&
            Array.isArray(_options.tracingOrigins) &&
            _options.tracingOrigins.length !== 0) {
            tracingOrigins = _options.tracingOrigins;
        }
        else {
            this._emitOptionsWarning = true;
        }
        this.options = tslib_1.__assign(tslib_1.__assign(tslib_1.__assign({}, DEFAULT_BROWSER_TRACING_OPTIONS), _options), { tracingOrigins: tracingOrigins });
    }
    /**
     * @inheritDoc
     */
    BrowserTracing.prototype.setupOnce = function (_, getCurrentHub) {
        var _this = this;
        this._getCurrentHub = getCurrentHub;
        if (this._emitOptionsWarning) {
            utils_1.logger.warn('[Tracing] You need to define `tracingOrigins` in the options. Set an array of urls or patterns to trace.');
            utils_1.logger.warn("[Tracing] We added a reasonable default for you: " + request_1.defaultRequestInstrumentationOptions.tracingOrigins);
        }
        // eslint-disable-next-line @typescript-eslint/unbound-method
        var _a = this.options, routingInstrumentation = _a.routingInstrumentation, startTransactionOnLocationChange = _a.startTransactionOnLocationChange, startTransactionOnPageLoad = _a.startTransactionOnPageLoad, markBackgroundTransactions = _a.markBackgroundTransactions, traceFetch = _a.traceFetch, traceXHR = _a.traceXHR, tracingOrigins = _a.tracingOrigins, shouldCreateSpanForRequest = _a.shouldCreateSpanForRequest;
        routingInstrumentation(function (context) { return _this._createRouteTransaction(context); }, startTransactionOnPageLoad, startTransactionOnLocationChange);
        if (markBackgroundTransactions) {
            backgroundtab_1.registerBackgroundTabDetection();
        }
        request_1.registerRequestInstrumentation({ traceFetch: traceFetch, traceXHR: traceXHR, tracingOrigins: tracingOrigins, shouldCreateSpanForRequest: shouldCreateSpanForRequest });
    };
    /** Create routing idle transaction. */
    BrowserTracing.prototype._createRouteTransaction = function (context) {
        var _this = this;
        if (!this._getCurrentHub) {
            utils_1.logger.warn("[Tracing] Did not create " + context.op + " transaction because _getCurrentHub is invalid.");
            return undefined;
        }
        // eslint-disable-next-line @typescript-eslint/unbound-method
        var _a = this.options, beforeNavigate = _a.beforeNavigate, idleTimeout = _a.idleTimeout, maxTransactionDuration = _a.maxTransactionDuration;
        var parentContextFromHeader = context.op === 'pageload' ? getHeaderContext() : undefined;
        var expandedContext = tslib_1.__assign(tslib_1.__assign(tslib_1.__assign({}, context), parentContextFromHeader), { trimEnd: true });
        var modifiedContext = typeof beforeNavigate === 'function' ? beforeNavigate(expandedContext) : expandedContext;
        // For backwards compatibility reasons, beforeNavigate can return undefined to "drop" the transaction (prevent it
        // from being sent to Sentry).
        var finalContext = modifiedContext === undefined ? tslib_1.__assign(tslib_1.__assign({}, expandedContext), { sampled: false }) : modifiedContext;
        if (finalContext.sampled === false) {
            utils_1.logger.log("[Tracing] Will not send " + finalContext.op + " transaction because of beforeNavigate.");
        }
        var hub = this._getCurrentHub();
        var idleTransaction = hubextensions_1.startIdleTransaction(hub, finalContext, idleTimeout, true);
        utils_1.logger.log("[Tracing] Starting " + finalContext.op + " transaction on scope");
        idleTransaction.registerBeforeFinishCallback(function (transaction, endTimestamp) {
            _this._metrics.addPerformanceEntries(transaction);
            adjustTransactionDuration(utils_2.secToMs(maxTransactionDuration), transaction, endTimestamp);
        });
        return idleTransaction;
    };
    /**
     * @inheritDoc
     */
    BrowserTracing.id = 'BrowserTracing';
    return BrowserTracing;
}());
exports.BrowserTracing = BrowserTracing;
/**
 * Gets transaction context from a sentry-trace meta.
 *
 * @returns Transaction context data from the header or undefined if there's no header or the header is malformed
 */
function getHeaderContext() {
    var header = getMetaContent('sentry-trace');
    if (header) {
        return utils_2.extractTraceparentData(header);
    }
    return undefined;
}
exports.getHeaderContext = getHeaderContext;
/** Returns the value of a meta tag */
function getMetaContent(metaName) {
    var el = document.querySelector("meta[name=" + metaName + "]");
    return el ? el.getAttribute('content') : null;
}
exports.getMetaContent = getMetaContent;
/** Adjusts transaction value based on max transaction duration */
function adjustTransactionDuration(maxDuration, transaction, endTimestamp) {
    var diff = endTimestamp - transaction.startTimestamp;
    var isOutdatedTransaction = endTimestamp && (diff > maxDuration || diff < 0);
    if (isOutdatedTransaction) {
        transaction.setStatus(spanstatus_1.SpanStatus.DeadlineExceeded);
        transaction.setTag('maxTransactionDurationExceeded', 'true');
    }
}
//# sourceMappingURL=browsertracing.js.map