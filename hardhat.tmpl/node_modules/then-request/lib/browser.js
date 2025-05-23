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
var ResponsePromise_1 = require("./ResponsePromise");
exports.ResponsePromise = ResponsePromise_1.ResponsePromise;
var handle_qs_1 = require("./handle-qs");
function request(method, url, options) {
    return ResponsePromise_1["default"](new Promise(function (resolve, reject) {
        var xhr = new XMLHttpRequest();
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
        function attempt(n, options) {
            request(method, url, {
                qs: options.qs,
                headers: options.headers,
                timeout: options.timeout
            }).nodeify(function (err, res) {
                var retry = !!(err || res.statusCode >= 400);
                if (typeof options.retry === 'function') {
                    retry = options.retry(err, res, n + 1);
                }
                if (n >= (options.maxRetries || 5)) {
                    retry = false;
                }
                if (retry) {
                    var delay = options.retryDelay;
                    if (typeof options.retryDelay === 'function') {
                        delay = options.retryDelay(err, res, n + 1);
                    }
                    delay = delay || 200;
                    setTimeout(function () {
                        attempt(n + 1, options);
                    }, delay);
                }
                else {
                    if (err)
                        reject(err);
                    else
                        resolve(res);
                }
            });
        }
        if (options.retry && method === 'GET') {
            return attempt(0, options);
        }
        var headers = options.headers || {};
        // handle cross domain
        var match;
        var crossDomain = !!((match = /^([\w-]+:)?\/\/([^\/]+)/.exec(url)) && (match[2] != location.host));
        if (!crossDomain) {
            headers = __assign({}, headers, { 'X-Requested-With': 'XMLHttpRequest' });
        }
        // handle query string
        if (options.qs) {
            url = handle_qs_1["default"](url, options.qs);
        }
        // handle json body
        if (options.json) {
            options.body = JSON.stringify(options.json);
            headers = __assign({}, headers, { 'Content-Type': 'application/json' });
        }
        if (options.form) {
            options.body = options.form;
        }
        if (options.timeout) {
            xhr.timeout = options.timeout;
            var start_1 = Date.now();
            xhr.ontimeout = function () {
                var duration = Date.now() - start_1;
                var err = new Error('Request timed out after ' + duration + 'ms');
                err.timeout = true;
                err.duration = duration;
                reject(err);
            };
        }
        xhr.onreadystatechange = function () {
            if (xhr.readyState === 4) {
                var headers = {};
                xhr.getAllResponseHeaders().split('\r\n').forEach(function (header) {
                    var h = header.split(':');
                    if (h.length > 1) {
                        headers[h[0].toLowerCase()] = h.slice(1).join(':').trim();
                    }
                });
                var res = new GenericResponse(xhr.status, headers, xhr.responseText, url);
                resolve(res);
            }
        };
        // method, url, async
        xhr.open(method, url, true);
        for (var name in headers) {
            xhr.setRequestHeader(name, headers[name]);
        }
        // avoid sending empty string (#319)
        xhr.send(options.body ? options.body : null);
    }));
}
var fd = FormData;
exports.FormData = fd;
exports["default"] = request;
module.exports = request;
module.exports["default"] = request;
module.exports.FormData = fd;
