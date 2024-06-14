Object.defineProperty(exports, "__esModule", { value: true });
var tslib_1 = require("tslib");
var core_1 = require("@sentry/core");
var utils_1 = require("@sentry/utils");
var http_1 = require("./utils/http");
var NODE_VERSION = utils_1.parseSemver(process.versions.node);
/** http module integration */
var Http = /** @class */ (function () {
    /**
     * @inheritDoc
     */
    function Http(options) {
        if (options === void 0) { options = {}; }
        /**
         * @inheritDoc
         */
        this.name = Http.id;
        this._breadcrumbs = typeof options.breadcrumbs === 'undefined' ? true : options.breadcrumbs;
        this._tracing = typeof options.tracing === 'undefined' ? false : options.tracing;
    }
    /**
     * @inheritDoc
     */
    Http.prototype.setupOnce = function () {
        // No need to instrument if we don't want to track anything
        if (!this._breadcrumbs && !this._tracing) {
            return;
        }
        var wrappedHandlerMaker = _createWrappedRequestMethodFactory(this._breadcrumbs, this._tracing);
        var httpModule = require('http');
        utils_1.fill(httpModule, 'get', wrappedHandlerMaker);
        utils_1.fill(httpModule, 'request', wrappedHandlerMaker);
        // NOTE: Prior to Node 9, `https` used internals of `http` module, thus we don't patch it.
        // If we do, we'd get double breadcrumbs and double spans for `https` calls.
        // It has been changed in Node 9, so for all versions equal and above, we patch `https` separately.
        if (NODE_VERSION.major && NODE_VERSION.major > 8) {
            var httpsModule = require('https');
            utils_1.fill(httpsModule, 'get', wrappedHandlerMaker);
            utils_1.fill(httpsModule, 'request', wrappedHandlerMaker);
        }
    };
    /**
     * @inheritDoc
     */
    Http.id = 'Http';
    return Http;
}());
exports.Http = Http;
/**
 * Function which creates a function which creates wrapped versions of internal `request` and `get` calls within `http`
 * and `https` modules. (NB: Not a typo - this is a creator^2!)
 *
 * @param breadcrumbsEnabled Whether or not to record outgoing requests as breadcrumbs
 * @param tracingEnabled Whether or not to record outgoing requests as tracing spans
 *
 * @returns A function which accepts the exiting handler and returns a wrapped handler
 */
function _createWrappedRequestMethodFactory(breadcrumbsEnabled, tracingEnabled) {
    return function wrappedRequestMethodFactory(originalRequestMethod) {
        return function wrappedMethod() {
            var args = [];
            for (var _i = 0; _i < arguments.length; _i++) {
                args[_i] = arguments[_i];
            }
            // eslint-disable-next-line @typescript-eslint/no-this-alias
            var httpModule = this;
            var requestArgs = http_1.normalizeRequestArgs(args);
            var requestOptions = requestArgs[0];
            var requestUrl = http_1.extractUrl(requestOptions);
            // we don't want to record requests to Sentry as either breadcrumbs or spans, so just use the original method
            if (http_1.isSentryRequest(requestUrl)) {
                return originalRequestMethod.apply(httpModule, requestArgs);
            }
            var span;
            var parentSpan;
            var scope = core_1.getCurrentHub().getScope();
            if (scope && tracingEnabled) {
                parentSpan = scope.getSpan();
                if (parentSpan) {
                    span = parentSpan.startChild({
                        description: (requestOptions.method || 'GET') + " " + requestUrl,
                        op: 'request',
                    });
                    var sentryTraceHeader = span.toTraceparent();
                    utils_1.logger.log("[Tracing] Adding sentry-trace header to outgoing request: " + sentryTraceHeader);
                    requestOptions.headers = tslib_1.__assign(tslib_1.__assign({}, requestOptions.headers), { 'sentry-trace': sentryTraceHeader });
                }
            }
            // eslint-disable-next-line @typescript-eslint/no-unsafe-member-access
            return originalRequestMethod
                .apply(httpModule, requestArgs)
                .once('response', function (res) {
                // eslint-disable-next-line @typescript-eslint/no-this-alias
                var req = this;
                if (breadcrumbsEnabled) {
                    addRequestBreadcrumb('response', requestUrl, req, res);
                }
                if (tracingEnabled && span) {
                    if (res.statusCode) {
                        span.setHttpStatus(res.statusCode);
                    }
                    span.description = http_1.cleanSpanDescription(span.description, requestOptions, req);
                    span.finish();
                }
            })
                .once('error', function () {
                // eslint-disable-next-line @typescript-eslint/no-this-alias
                var req = this;
                if (breadcrumbsEnabled) {
                    addRequestBreadcrumb('error', requestUrl, req);
                }
                if (tracingEnabled && span) {
                    span.setHttpStatus(500);
                    span.description = http_1.cleanSpanDescription(span.description, requestOptions, req);
                    span.finish();
                }
            });
        };
    };
}
/**
 * Captures Breadcrumb based on provided request/response pair
 */
function addRequestBreadcrumb(event, url, req, res) {
    if (!core_1.getCurrentHub().getIntegration(Http)) {
        return;
    }
    core_1.getCurrentHub().addBreadcrumb({
        category: 'http',
        data: {
            method: req.method,
            status_code: res && res.statusCode,
            url: url,
        },
        type: 'http',
    }, {
        event: event,
        request: req,
        response: res,
    });
}
//# sourceMappingURL=http.js.map