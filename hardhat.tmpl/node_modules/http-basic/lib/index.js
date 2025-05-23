"use strict";
var __assign = (this && this.__assign) || Object.assign || function(t) {
    for (var s, i = 1, n = arguments.length; i < n; i++) {
        s = arguments[i];
        for (var p in s) if (Object.prototype.hasOwnProperty.call(s, p))
            t[p] = s[p];
    }
    return t;
};
exports.__esModule = true;
var cacheUtils = require("./cache-utils");
var FileCache_1 = require("./FileCache");
var MemoryCache_1 = require("./MemoryCache");
var http_1 = require("http");
var zlib_1 = require("zlib");
var url_1 = require("url");
var stream_1 = require("stream");
var https_1 = require("https");
var Response = require("http-response-object");
exports.Response = Response;
var caseless = require('caseless');
var fileCache = new FileCache_1["default"](__dirname + '/cache');
var memoryCache = new MemoryCache_1["default"]();
function requestProtocol(protocol, options, callback) {
    if (protocol === 'http') {
        return http_1.request(options, callback);
    }
    else if (protocol === 'https') {
        return https_1.request(options, callback);
    }
    throw new Error('Unsupported protocol ' + protocol);
}
function request(method, url, options, callback) {
    if (typeof options === 'function') {
        callback = options;
        options = null;
    }
    if (options === null || options === undefined) {
        options = {};
    }
    if (typeof options !== 'object') {
        throw new TypeError('options must be an object (or null)');
    }
    if (typeof callback !== 'function') {
        throw new TypeError('callback must be a function');
    }
    return _request(method, ((url && typeof url === 'object') ? url.href : url), options, callback);
}
function _request(method, url, options, callback) {
    var start = Date.now();
    if (typeof method !== 'string') {
        throw new TypeError('The method must be a string.');
    }
    if (typeof url !== 'string') {
        throw new TypeError('The URL/path must be a string or a URL object.');
    }
    method = method.toUpperCase();
    var urlObject = url_1.parse(url);
    var protocol = (urlObject.protocol || '').replace(/\:$/, '');
    if (protocol !== 'http' && protocol !== 'https') {
        throw new TypeError('The protocol "' + protocol + '" is not supported, cannot load "' + url + '"');
    }
    var rawHeaders = options.headers || {};
    var headers = caseless(rawHeaders);
    if (urlObject.auth) {
        headers.set('Authorization', 'Basic ' + (Buffer.from(urlObject.auth)).toString('base64'));
    }
    var agent = 'agent' in options ? options.agent : false;
    var cache = options.cache;
    if (typeof cache === 'string') {
        if (cache === 'file') {
            cache = fileCache;
        }
        else if (cache === 'memory') {
            cache = memoryCache;
        }
    }
    if (cache && !(typeof cache === 'object' && typeof cache.getResponse === 'function' && typeof cache.setResponse === 'function' && typeof cache.invalidateResponse === 'function')) {
        throw new TypeError(cache + ' is not a valid cache, caches must have `getResponse`, `setResponse` and `invalidateResponse` methods.');
    }
    var ignoreFailedInvalidation = options.ignoreFailedInvalidation;
    if (options.duplex !== undefined && typeof options.duplex !== 'boolean') {
        throw new Error('expected options.duplex to be a boolean if provided');
    }
    var duplex = options.duplex !== undefined ? options.duplex : !(method === 'GET' || method === 'DELETE' || method === 'HEAD');
    var unsafe = !(method === 'GET' || method === 'OPTIONS' || method === 'HEAD');
    if (options.gzip) {
        headers.set('Accept-Encoding', headers.has('Accept-Encoding') ? headers.get('Accept-Encoding') + ',gzip,deflate' : 'gzip,deflate');
        return _request(method, url, {
            allowRedirectHeaders: options.allowRedirectHeaders,
            duplex: duplex,
            headers: rawHeaders,
            agent: agent,
            followRedirects: options.followRedirects,
            retry: options.retry,
            retryDelay: options.retryDelay,
            maxRetries: options.maxRetries,
            cache: cache,
            timeout: options.timeout
        }, function (err, res) {
            if (err)
                return callback(err);
            if (!res)
                return callback(new Error('Response should not be undefined if there is no error.'));
            var newHeaders = __assign({}, res.headers);
            var newBody = res.body;
            switch (newHeaders['content-encoding']) {
                case 'gzip':
                    delete newHeaders['content-encoding'];
                    newBody = res.body.pipe(zlib_1.createGunzip());
                    break;
                case 'deflate':
                    delete newHeaders['content-encoding'];
                    newBody = res.body.pipe(zlib_1.createInflate());
                    break;
            }
            return callback(err, new Response(res.statusCode, newHeaders, newBody, res.url));
        });
    }
    if (options.followRedirects) {
        return _request(method, url, {
            allowRedirectHeaders: options.allowRedirectHeaders,
            duplex: duplex,
            headers: rawHeaders,
            agent: agent,
            retry: options.retry,
            retryDelay: options.retryDelay,
            maxRetries: options.maxRetries,
            cache: cache,
            timeout: options.timeout
        }, function (err, res) {
            if (err)
                return callback(err);
            if (!res)
                return callback(new Error('Response should not be undefined if there is no error.'));
            if (options.followRedirects && isRedirect(res.statusCode)) {
                // prevent leakage of file handles
                res.body.resume();
                if (method === 'DELETE' && res.statusCode === 303) {
                    // 303 See Other should convert to GET for duplex
                    // requests and for DELETE
                    method = 'GET';
                }
                if (options.maxRedirects === 0) {
                    var err_1 = new Error('Maximum number of redirects exceeded');
                    err_1.res = res;
                    return callback(err_1, res);
                }
                options = __assign({}, options, { duplex: false, maxRedirects: options.maxRedirects && options.maxRedirects !== Infinity ? options.maxRedirects - 1 : options.maxRedirects });
                // don't maintain headers through redirects
                // This fixes a problem where a POST to http://example.com
                // might result in a GET to http://example.co.uk that includes "content-length"
                // as a header
                var headers_1 = caseless(options.headers);
                var redirectHeaders = {};
                if (options.allowRedirectHeaders) {
                    for (var i = 0; i < options.allowRedirectHeaders.length; i++) {
                        var headerName = options.allowRedirectHeaders[i];
                        var headerValue = headers_1.get(headerName);
                        if (headerValue) {
                            redirectHeaders[headerName] = headerValue;
                        }
                    }
                }
                options.headers = redirectHeaders;
                var location = res.headers.location;
                if (typeof location !== 'string') {
                    return callback(new Error('Cannot redirect to non string location: ' + location));
                }
                return request(duplex ? 'GET' : method, url_1.resolve(url, location), options, callback);
            }
            else {
                return callback(null, res);
            }
        });
    }
    if (cache && method === 'GET' && !duplex) {
        var timestamp_1 = Date.now();
        return cache.getResponse(url, function (err, cachedResponse) {
            if (err) {
                console.warn('Error reading from cache: ' + err.message);
            }
            var isMatch = !!(cachedResponse && cacheUtils.isMatch(rawHeaders, cachedResponse));
            if (cachedResponse && (options.isMatch ? options.isMatch(rawHeaders, cachedResponse, isMatch) : isMatch)) {
                var isExpired = cacheUtils.isExpired(cachedResponse);
                if (!(options.isExpired ? options.isExpired(cachedResponse, isExpired) : isExpired)) {
                    var res = new Response(cachedResponse.statusCode, cachedResponse.headers, cachedResponse.body, url);
                    res.fromCache = true;
                    res.fromNotModified = false;
                    return callback(null, res);
                }
                else {
                    if (cachedResponse.headers['etag']) {
                        headers.set('If-None-Match', cachedResponse.headers['etag']);
                    }
                    if (cachedResponse.headers['last-modified']) {
                        headers.set('If-Modified-Since', cachedResponse.headers['last-modified']);
                    }
                }
            }
            request('GET', url, {
                allowRedirectHeaders: options.allowRedirectHeaders,
                headers: rawHeaders,
                retry: options.retry,
                retryDelay: options.retryDelay,
                maxRetries: options.maxRetries,
                agent: agent,
                timeout: options.timeout
            }, function (err, res) {
                if (err)
                    return callback(err);
                if (!res)
                    return callback(new Error('Response should not be undefined if there is no error.'));
                if (res.statusCode === 304 && cachedResponse) { // Not Modified
                    // prevent leakage of file handles
                    res.body.resume();
                    var resultBody = cachedResponse.body;
                    var c = cache;
                    if (c.updateResponseHeaders) {
                        c.updateResponseHeaders(url, {
                            headers: res.headers,
                            requestTimestamp: timestamp_1
                        });
                    }
                    else {
                        var cachedResponseBody_1 = new stream_1.PassThrough();
                        var newResultBody_1 = new stream_1.PassThrough();
                        resultBody.on('data', function (data) {
                            cachedResponseBody_1.write(data);
                            newResultBody_1.write(data);
                        });
                        resultBody.on('end', function () {
                            cachedResponseBody_1.end();
                            newResultBody_1.end();
                        });
                        resultBody = newResultBody_1;
                        cache.setResponse(url, {
                            statusCode: cachedResponse.statusCode,
                            headers: res.headers,
                            body: cachedResponseBody_1,
                            requestHeaders: cachedResponse.requestHeaders,
                            requestTimestamp: timestamp_1
                        });
                    }
                    var response = new Response(cachedResponse.statusCode, cachedResponse.headers, resultBody, url);
                    response.fromCache = true;
                    response.fromNotModified = true;
                    return callback(null, response);
                }
                // prevent leakage of file handles
                cachedResponse && cachedResponse.body.resume();
                var canCache = cacheUtils.canCache(res);
                if (options.canCache ? options.canCache(res, canCache) : canCache) {
                    var cachedResponseBody_2 = new stream_1.PassThrough();
                    var resultResponseBody_1 = new stream_1.PassThrough();
                    res.body.on('data', function (data) {
                        cachedResponseBody_2.write(data);
                        resultResponseBody_1.write(data);
                    });
                    res.body.on('end', function () { cachedResponseBody_2.end(); resultResponseBody_1.end(); });
                    var resultResponse = new Response(res.statusCode, res.headers, resultResponseBody_1, url);
                    cache.setResponse(url, {
                        statusCode: res.statusCode,
                        headers: res.headers,
                        body: cachedResponseBody_2,
                        requestHeaders: rawHeaders,
                        requestTimestamp: timestamp_1
                    });
                    return callback(null, resultResponse);
                }
                else {
                    return callback(null, res);
                }
            });
        });
    }
    function attempt(n) {
        return _request(method, url, {
            allowRedirectHeaders: options.allowRedirectHeaders,
            headers: rawHeaders,
            agent: agent,
            timeout: options.timeout
        }, function (err, res) {
            var retry = err || !res || res.statusCode >= 400;
            if (typeof options.retry === 'function') {
                retry = options.retry(err, res, n + 1);
            }
            if (n >= (options.maxRetries || 5)) {
                retry = false;
            }
            if (retry) {
                var delay = options.retryDelay;
                if (typeof delay === 'function') {
                    delay = delay(err, res, n + 1);
                }
                delay = delay || 200;
                setTimeout(function () {
                    attempt(n + 1);
                }, delay);
            }
            else {
                callback(err, res);
            }
        });
    }
    if (options.retry && method === 'GET' && !duplex) {
        return attempt(0);
    }
    var responded = false;
    var timeout = null;
    var req = requestProtocol(protocol, {
        host: urlObject.hostname,
        port: urlObject.port == null ? undefined : +urlObject.port,
        path: urlObject.path,
        method: method,
        headers: rawHeaders,
        agent: agent
    }, function (res) {
        var end = Date.now();
        if (responded)
            return res.resume();
        responded = true;
        if (timeout !== null)
            clearTimeout(timeout);
        var result = new Response(res.statusCode || 0, res.headers, res, url);
        if (cache && unsafe && res.statusCode && res.statusCode < 400) {
            cache.invalidateResponse(url, function (err) {
                if (err && !ignoreFailedInvalidation) {
                    callback(new Error('Error invalidating the cache for' + url + ': ' + err.message), result);
                }
                else {
                    callback(null, result);
                }
            });
        }
        else {
            callback(null, result);
        }
    }).on('error', function (err) {
        if (responded)
            return;
        responded = true;
        if (timeout !== null)
            clearTimeout(timeout);
        callback(err);
    });
    function onTimeout() {
        if (responded)
            return;
        responded = true;
        if (timeout !== null)
            clearTimeout(timeout);
        req.abort();
        var duration = Date.now() - start;
        var err = new Error('Request timed out after ' + duration + 'ms');
        err.timeout = true;
        err.duration = duration;
        callback(err);
    }
    if (options.socketTimeout) {
        req.setTimeout(options.socketTimeout, onTimeout);
    }
    if (options.timeout) {
        timeout = setTimeout(onTimeout, options.timeout);
    }
    if (duplex) {
        return req;
    }
    else {
        req.end();
    }
    return undefined;
}
function isRedirect(statusCode) {
    return statusCode === 301 || statusCode === 302 || statusCode === 303 || statusCode === 307 || statusCode === 308;
}
exports["default"] = request;
module.exports = request;
module.exports["default"] = request;
module.exports.Response = Response;
