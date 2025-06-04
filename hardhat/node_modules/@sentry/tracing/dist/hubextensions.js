Object.defineProperty(exports, "__esModule", { value: true });
var tslib_1 = require("tslib");
var hub_1 = require("@sentry/hub");
var types_1 = require("@sentry/types");
var utils_1 = require("@sentry/utils");
var errors_1 = require("./errors");
var idletransaction_1 = require("./idletransaction");
var transaction_1 = require("./transaction");
var utils_2 = require("./utils");
/** Returns all trace headers that are currently on the top scope. */
function traceHeaders() {
    var scope = this.getScope();
    if (scope) {
        var span = scope.getSpan();
        if (span) {
            return {
                'sentry-trace': span.toTraceparent(),
            };
        }
    }
    return {};
}
/**
 * Makes a sampling decision for the given transaction and stores it on the transaction.
 *
 * Called every time a transaction is created. Only transactions which emerge with a `sampled` value of `true` will be
 * sent to Sentry.
 *
 * @param hub: The hub off of which to read config options
 * @param transaction: The transaction needing a sampling decision
 * @param samplingContext: Default and user-provided data which may be used to help make the decision
 *
 * @returns The given transaction with its `sampled` value set
 */
function sample(hub, transaction, samplingContext) {
    var _a;
    var client = hub.getClient();
    var options = (client && client.getOptions()) || {};
    // nothing to do if there's no client or if tracing is disabled
    if (!client || !utils_2.hasTracingEnabled(options)) {
        transaction.sampled = false;
        return transaction;
    }
    // if the user has forced a sampling decision by passing a `sampled` value in their transaction context, go with that
    if (transaction.sampled !== undefined) {
        transaction.tags = tslib_1.__assign(tslib_1.__assign({}, transaction.tags), { __sentry_samplingMethod: types_1.TransactionSamplingMethod.Explicit });
        return transaction;
    }
    // we would have bailed already if neither `tracesSampler` nor `tracesSampleRate` were defined, so one of these should
    // work; prefer the hook if so
    var sampleRate;
    if (typeof options.tracesSampler === 'function') {
        sampleRate = options.tracesSampler(samplingContext);
        // cast the rate to a number first in case it's a boolean
        transaction.tags = tslib_1.__assign(tslib_1.__assign({}, transaction.tags), { __sentry_samplingMethod: types_1.TransactionSamplingMethod.Sampler, 
            // TODO kmclb - once tag types are loosened, don't need to cast to string here
            __sentry_sampleRate: String(Number(sampleRate)) });
    }
    else if (samplingContext.parentSampled !== undefined) {
        sampleRate = samplingContext.parentSampled;
        transaction.tags = tslib_1.__assign(tslib_1.__assign({}, transaction.tags), { __sentry_samplingMethod: types_1.TransactionSamplingMethod.Inheritance });
    }
    else {
        sampleRate = options.tracesSampleRate;
        // cast the rate to a number first in case it's a boolean
        transaction.tags = tslib_1.__assign(tslib_1.__assign({}, transaction.tags), { __sentry_samplingMethod: types_1.TransactionSamplingMethod.Rate, 
            // TODO kmclb - once tag types are loosened, don't need to cast to string here
            __sentry_sampleRate: String(Number(sampleRate)) });
    }
    // Since this is coming from the user (or from a function provided by the user), who knows what we might get. (The
    // only valid values are booleans or numbers between 0 and 1.)
    if (!isValidSampleRate(sampleRate)) {
        utils_1.logger.warn("[Tracing] Discarding transaction because of invalid sample rate.");
        transaction.sampled = false;
        return transaction;
    }
    // if the function returned 0 (or false), or if `tracesSampleRate` is 0, it's a sign the transaction should be dropped
    if (!sampleRate) {
        utils_1.logger.log("[Tracing] Discarding transaction because " + (typeof options.tracesSampler === 'function'
            ? 'tracesSampler returned 0 or false'
            : 'a negative sampling decision was inherited or tracesSampleRate is set to 0'));
        transaction.sampled = false;
        return transaction;
    }
    // Now we roll the dice. Math.random is inclusive of 0, but not of 1, so strict < is safe here. In case sampleRate is
    // a boolean, the < comparison will cause it to be automatically cast to 1 if it's true and 0 if it's false.
    transaction.sampled = Math.random() < sampleRate;
    // if we're not going to keep it, we're done
    if (!transaction.sampled) {
        utils_1.logger.log("[Tracing] Discarding transaction because it's not included in the random sample (sampling rate = " + Number(sampleRate) + ")");
        return transaction;
    }
    // at this point we know we're keeping the transaction, whether because of an inherited decision or because it got
    // lucky with the dice roll
    transaction.initSpanRecorder((_a = options._experiments) === null || _a === void 0 ? void 0 : _a.maxSpans);
    utils_1.logger.log("[Tracing] starting " + transaction.op + " transaction - " + transaction.name);
    return transaction;
}
/**
 * Gets the correct context to pass to the tracesSampler, based on the environment (i.e., which SDK is being used)
 *
 * @returns The default sample context
 */
function getDefaultSamplingContext(transactionContext) {
    // promote parent sampling decision (if any) for easy access
    var parentSampled = transactionContext.parentSampled;
    var defaultSamplingContext = { transactionContext: transactionContext, parentSampled: parentSampled };
    if (utils_1.isNodeEnv()) {
        var domain = hub_1.getActiveDomain();
        if (domain) {
            // for all node servers that we currently support, we store the incoming request object (which is an instance of
            // http.IncomingMessage) on the domain
            // the domain members are stored as an array, so our only way to find the request is to iterate through the array
            // and compare types
            var nodeHttpModule = utils_1.dynamicRequire(module, 'http');
            // eslint-disable-next-line @typescript-eslint/no-unsafe-member-access
            var requestType_1 = nodeHttpModule.IncomingMessage;
            var request = domain.members.find(function (member) { return utils_1.isInstanceOf(member, requestType_1); });
            if (request) {
                defaultSamplingContext.request = utils_1.extractNodeRequestData(request);
            }
        }
    }
    // we must be in browser-js (or some derivative thereof)
    else {
        // we use `getGlobalObject()` rather than `window` since service workers also have a `location` property on `self`
        var globalObject = utils_1.getGlobalObject();
        if ('location' in globalObject) {
            // eslint-disable-next-line @typescript-eslint/no-unsafe-member-access, @typescript-eslint/no-explicit-any
            defaultSamplingContext.location = tslib_1.__assign({}, globalObject.location);
        }
    }
    return defaultSamplingContext;
}
/**
 * Checks the given sample rate to make sure it is valid type and value (a boolean, or a number between 0 and 1).
 */
function isValidSampleRate(rate) {
    // we need to check NaN explicitly because it's of type 'number' and therefore wouldn't get caught by this typecheck
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    if (isNaN(rate) || !(typeof rate === 'number' || typeof rate === 'boolean')) {
        utils_1.logger.warn("[Tracing] Given sample rate is invalid. Sample rate must be a boolean or a number between 0 and 1. Got " + JSON.stringify(rate) + " of type " + JSON.stringify(typeof rate) + ".");
        return false;
    }
    // in case sampleRate is a boolean, it will get automatically cast to 1 if it's true and 0 if it's false
    if (rate < 0 || rate > 1) {
        utils_1.logger.warn("[Tracing] Given sample rate is invalid. Sample rate must be between 0 and 1. Got " + rate + ".");
        return false;
    }
    return true;
}
/**
 * Creates a new transaction and adds a sampling decision if it doesn't yet have one.
 *
 * The Hub.startTransaction method delegates to this method to do its work, passing the Hub instance in as `this`, as if
 * it had been called on the hub directly. Exists as a separate function so that it can be injected into the class as an
 * "extension method."
 *
 * @param this: The Hub starting the transaction
 * @param transactionContext: Data used to configure the transaction
 * @param CustomSamplingContext: Optional data to be provided to the `tracesSampler` function (if any)
 *
 * @returns The new transaction
 *
 * @see {@link Hub.startTransaction}
 */
function _startTransaction(transactionContext, customSamplingContext) {
    var transaction = new transaction_1.Transaction(transactionContext, this);
    return sample(this, transaction, tslib_1.__assign(tslib_1.__assign({}, getDefaultSamplingContext(transactionContext)), customSamplingContext));
}
/**
 * Create new idle transaction.
 */
function startIdleTransaction(hub, transactionContext, idleTimeout, onScope) {
    var transaction = new idletransaction_1.IdleTransaction(transactionContext, hub, idleTimeout, onScope);
    return sample(hub, transaction, getDefaultSamplingContext(transactionContext));
}
exports.startIdleTransaction = startIdleTransaction;
/**
 * @private
 */
function _addTracingExtensions() {
    var carrier = hub_1.getMainCarrier();
    if (carrier.__SENTRY__) {
        carrier.__SENTRY__.extensions = carrier.__SENTRY__.extensions || {};
        if (!carrier.__SENTRY__.extensions.startTransaction) {
            carrier.__SENTRY__.extensions.startTransaction = _startTransaction;
        }
        if (!carrier.__SENTRY__.extensions.traceHeaders) {
            carrier.__SENTRY__.extensions.traceHeaders = traceHeaders;
        }
    }
}
exports._addTracingExtensions = _addTracingExtensions;
/**
 * This patches the global object and injects the Tracing extensions methods
 */
function addExtensionMethods() {
    _addTracingExtensions();
    // If an error happens globally, we should make sure transaction status is set to error.
    errors_1.registerErrorInstrumentation();
}
exports.addExtensionMethods = addExtensionMethods;
//# sourceMappingURL=hubextensions.js.map