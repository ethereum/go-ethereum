import { __assign } from "tslib";
import { getCurrentHub } from '@sentry/core';
import { fill, logger, parseSemver } from '@sentry/utils';
import { cleanSpanDescription, extractUrl, isSentryRequest, normalizeRequestArgs, } from './utils/http';
var NODE_VERSION = parseSemver(process.versions.node);
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
        fill(httpModule, 'get', wrappedHandlerMaker);
        fill(httpModule, 'request', wrappedHandlerMaker);
        // NOTE: Prior to Node 9, `https` used internals of `http` module, thus we don't patch it.
        // If we do, we'd get double breadcrumbs and double spans for `https` calls.
        // It has been changed in Node 9, so for all versions equal and above, we patch `https` separately.
        if (NODE_VERSION.major && NODE_VERSION.major > 8) {
            var httpsModule = require('https');
            fill(httpsModule, 'get', wrappedHandlerMaker);
            fill(httpsModule, 'request', wrappedHandlerMaker);
        }
    };
    /**
     * @inheritDoc
     */
    Http.id = 'Http';
    return Http;
}());
export { Http };
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
            var requestArgs = normalizeRequestArgs(args);
            var requestOptions = requestArgs[0];
            var requestUrl = extractUrl(requestOptions);
            // we don't want to record requests to Sentry as either breadcrumbs or spans, so just use the original method
            if (isSentryRequest(requestUrl)) {
                return originalRequestMethod.apply(httpModule, requestArgs);
            }
            var span;
            var parentSpan;
            var scope = getCurrentHub().getScope();
            if (scope && tracingEnabled) {
                parentSpan = scope.getSpan();
                if (parentSpan) {
                    span = parentSpan.startChild({
                        description: (requestOptions.method || 'GET') + " " + requestUrl,
                        op: 'request',
                    });
                    var sentryTraceHeader = span.toTraceparent();
                    logger.log("[Tracing] Adding sentry-trace header to outgoing request: " + sentryTraceHeader);
                    requestOptions.headers = __assign(__assign({}, requestOptions.headers), { 'sentry-trace': sentryTraceHeader });
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
                    span.description = cleanSpanDescription(span.description, requestOptions, req);
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
                    span.description = cleanSpanDescription(span.description, requestOptions, req);
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
    if (!getCurrentHub().getIntegration(Http)) {
        return;
    }
    getCurrentHub().addBreadcrumb({
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