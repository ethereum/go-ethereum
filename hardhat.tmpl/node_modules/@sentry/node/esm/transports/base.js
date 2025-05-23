import { __assign, __awaiter, __generator } from "tslib";
import { API, eventToSentryRequest } from '@sentry/core';
import { Status } from '@sentry/types';
import { logger, parseRetryAfterHeader, PromiseBuffer, SentryError } from '@sentry/utils';
import * as fs from 'fs';
import * as url from 'url';
import { SDK_NAME, SDK_VERSION } from '../version';
/** Base Transport class implementation */
var BaseTransport = /** @class */ (function () {
    /** Create instance and set this.dsn */
    function BaseTransport(options) {
        this.options = options;
        /** A simple buffer holding all requests. */
        this._buffer = new PromiseBuffer(30);
        /** Locks transport after receiving 429 response */
        this._disabledUntil = new Date(Date.now());
        this._api = new API(options.dsn);
    }
    /**
     * @inheritDoc
     */
    BaseTransport.prototype.sendEvent = function (_) {
        throw new SentryError('Transport Class has to implement `sendEvent` method.');
    };
    /**
     * @inheritDoc
     */
    BaseTransport.prototype.close = function (timeout) {
        return this._buffer.drain(timeout);
    };
    /** Returns a build request option object used by request */
    BaseTransport.prototype._getRequestOptions = function (uri) {
        var headers = __assign(__assign({}, this._api.getRequestHeaders(SDK_NAME, SDK_VERSION)), this.options.headers);
        var hostname = uri.hostname, pathname = uri.pathname, port = uri.port, protocol = uri.protocol;
        // See https://github.com/nodejs/node/blob/38146e717fed2fabe3aacb6540d839475e0ce1c6/lib/internal/url.js#L1268-L1290
        // We ignore the query string on purpose
        var path = "" + pathname;
        return __assign({ agent: this.client, headers: headers,
            hostname: hostname, method: 'POST', path: path,
            port: port,
            protocol: protocol }, (this.options.caCerts && {
            ca: fs.readFileSync(this.options.caCerts),
        }));
    };
    /** JSDoc */
    BaseTransport.prototype._sendWithModule = function (httpModule, event) {
        return __awaiter(this, void 0, void 0, function () {
            var _this = this;
            return __generator(this, function (_a) {
                if (new Date(Date.now()) < this._disabledUntil) {
                    return [2 /*return*/, Promise.reject(new SentryError("Transport locked till " + this._disabledUntil + " due to too many requests."))];
                }
                if (!this._buffer.isReady()) {
                    return [2 /*return*/, Promise.reject(new SentryError('Not adding Promise due to buffer limit reached.'))];
                }
                return [2 /*return*/, this._buffer.add(new Promise(function (resolve, reject) {
                        var sentryReq = eventToSentryRequest(event, _this._api);
                        var options = _this._getRequestOptions(new url.URL(sentryReq.url));
                        var req = httpModule.request(options, function (res) {
                            var statusCode = res.statusCode || 500;
                            var status = Status.fromHttpCode(statusCode);
                            res.setEncoding('utf8');
                            if (status === Status.Success) {
                                resolve({ status: status });
                            }
                            else {
                                if (status === Status.RateLimit) {
                                    var now = Date.now();
                                    /**
                                     * "Key-value pairs of header names and values. Header names are lower-cased."
                                     * https://nodejs.org/api/http.html#http_message_headers
                                     */
                                    var retryAfterHeader = res.headers ? res.headers['retry-after'] : '';
                                    retryAfterHeader = (Array.isArray(retryAfterHeader) ? retryAfterHeader[0] : retryAfterHeader);
                                    _this._disabledUntil = new Date(now + parseRetryAfterHeader(now, retryAfterHeader));
                                    logger.warn("Too many requests, backing off till: " + _this._disabledUntil);
                                }
                                var rejectionMessage = "HTTP Error (" + statusCode + ")";
                                if (res.headers && res.headers['x-sentry-error']) {
                                    rejectionMessage += ": " + res.headers['x-sentry-error'];
                                }
                                reject(new SentryError(rejectionMessage));
                            }
                            // Force the socket to drain
                            res.on('data', function () {
                                // Drain
                            });
                            res.on('end', function () {
                                // Drain
                            });
                        });
                        req.on('error', reject);
                        req.end(sentryReq.body);
                    }))];
            });
        });
    };
    return BaseTransport;
}());
export { BaseTransport };
//# sourceMappingURL=base.js.map