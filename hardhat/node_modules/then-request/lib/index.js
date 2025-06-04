'use strict';
var __assign = (this && this.__assign) || Object.assign || function(t) {
    for (var s, i = 1, n = arguments.length; i < n; i++) {
        s = arguments[i];
        for (var p in s) if (Object.prototype.hasOwnProperty.call(s, p))
            t[p] = s[p];
    }
    return t;
};
exports.__esModule = true;
var GenericResponse = require("http-response-object");
var Promise = require("promise");
var concat = require("concat-stream");
var ResponsePromise_1 = require("./ResponsePromise");
exports.ResponsePromise = ResponsePromise_1.ResponsePromise;
var handle_qs_1 = require("./handle-qs");
var http_basic_1 = require("http-basic");
var FormData = require("form-data");
exports.FormData = FormData;
var caseless = require('caseless');
var basicRequest = http_basic_1["default"];
var BufferBody = /** @class */ (function () {
    function BufferBody(body, extraHeaders) {
        this._body = body;
        this._headers = extraHeaders;
    }
    BufferBody.prototype.getHeaders = function () {
        return Promise.resolve(__assign({ 'content-length': '' + this._body.length }, this._headers));
    };
    BufferBody.prototype.pipe = function (stream) {
        stream.end(this._body);
    };
    return BufferBody;
}());
var FormBody = /** @class */ (function () {
    function FormBody(body) {
        this._body = body;
    }
    FormBody.prototype.getHeaders = function () {
        var _this = this;
        var headers = this._body.getHeaders();
        return new Promise(function (resolve, reject) {
            var gotLength = false;
            _this._body.getLength(function (err, length) {
                if (gotLength)
                    return;
                gotLength = true;
                if (err) {
                    return reject(typeof err == 'string'
                        ? new Error(err)
                        : err);
                }
                headers['content-length'] = '' + length;
                resolve(headers);
            });
        });
    };
    FormBody.prototype.pipe = function (stream) {
        this._body.pipe(stream);
    };
    return FormBody;
}());
var StreamBody = /** @class */ (function () {
    function StreamBody(body) {
        this._body = body;
    }
    StreamBody.prototype.getHeaders = function () {
        return Promise.resolve({});
    };
    StreamBody.prototype.pipe = function (stream) {
        this._body.pipe(stream);
    };
    return StreamBody;
}());
function handleBody(options) {
    if (options.form) {
        return new FormBody(options.form);
    }
    var extraHeaders = {};
    var body = options.body;
    if (options.json) {
        extraHeaders['content-type'] = 'application/json';
        body = JSON.stringify(options.json);
    }
    if (typeof body === 'string') {
        body = Buffer.from(body);
    }
    if (!body) {
        body = Buffer.alloc(0);
    }
    if (!Buffer.isBuffer(body)) {
        if (typeof body.pipe === 'function') {
            return new StreamBody(body);
        }
        throw new TypeError('body should be a Buffer or a String');
    }
    return new BufferBody(body, extraHeaders);
}
function request(method, url, options) {
    if (options === void 0) { options = {}; }
    return ResponsePromise_1["default"](new Promise(function (resolve, reject) {
        // check types of arguments
        if (typeof method !== 'string') {
            throw new TypeError('The method must be a string.');
        }
        if (typeof url !== 'string') {
            throw new TypeError('The URL/path must be a string.');
        }
        if (options == null) {
            options = {};
        }
        if (typeof options !== 'object') {
            throw new TypeError('Options must be an object (or null).');
        }
        method = method.toUpperCase();
        options.headers = options.headers || {};
        var headers = caseless(options.headers);
        // handle query string
        if (options.qs) {
            url = handle_qs_1["default"](url, options.qs);
        }
        var duplex = !(method === 'GET' || method === 'DELETE' || method === 'HEAD');
        if (duplex) {
            var body_1 = handleBody(options);
            body_1.getHeaders().then(function (bodyHeaders) {
                Object.keys(bodyHeaders).forEach(function (key) {
                    if (!headers.has(key)) {
                        headers.set(key, bodyHeaders[key]);
                    }
                });
                ready(body_1);
            })["catch"](reject);
        }
        else if (options.body) {
            throw new Error('You cannot pass a body to a ' + method + ' request.');
        }
        else {
            ready();
        }
        function ready(body) {
            var req = basicRequest(method, url, {
                allowRedirectHeaders: options.allowRedirectHeaders,
                headers: options.headers,
                followRedirects: options.followRedirects !== false,
                maxRedirects: options.maxRedirects,
                gzip: options.gzip !== false,
                cache: options.cache,
                agent: options.agent,
                timeout: options.timeout,
                socketTimeout: options.socketTimeout,
                retry: options.retry,
                retryDelay: options.retryDelay,
                maxRetries: options.maxRetries,
                isMatch: options.isMatch,
                isExpired: options.isExpired,
                canCache: options.canCache
            }, function (err, res) {
                if (err)
                    return reject(err);
                if (!res)
                    return reject(new Error('No request was received'));
                res.body.on('error', reject);
                res.body.pipe(concat(function (body) {
                    resolve(new GenericResponse(res.statusCode, res.headers, Array.isArray(body) ? Buffer.alloc(0) : body, res.url));
                }));
            });
            if (req && body) {
                body.pipe(req);
            }
        }
    }));
}
exports["default"] = request;
module.exports = request;
module.exports["default"] = request;
module.exports.FormData = FormData;
