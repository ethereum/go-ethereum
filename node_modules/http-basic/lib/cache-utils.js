"use strict";
exports.__esModule = true;
var cache_control_utils_1 = require("./cache-control-utils");
function isMatch(requestHeaders, cachedResponse) {
    var vary = cachedResponse.headers['vary'];
    if (vary && cachedResponse.requestHeaders) {
        vary = '' + vary;
        return vary.split(',').map(function (header) { return header.trim().toLowerCase(); }).every(function (header) {
            return requestHeaders[header] === cachedResponse.requestHeaders[header];
        });
    }
    else {
        return true;
    }
}
exports.isMatch = isMatch;
;
function isExpired(cachedResponse) {
    var policy = cache_control_utils_1.cachePolicy(cachedResponse);
    if (policy) {
        var time = (Date.now() - cachedResponse.requestTimestamp) / 1000;
        if (policy.maxage !== null && policy.maxage > time) {
            return false;
        }
    }
    if (cachedResponse.statusCode === 301 || cachedResponse.statusCode === 308)
        return false;
    return true;
}
exports.isExpired = isExpired;
;
function canCache(res) {
    if (res.headers['etag'])
        return true;
    if (res.headers['last-modified'])
        return true;
    if (cache_control_utils_1.isCacheable(res))
        return true;
    if (res.statusCode === 301 || res.statusCode === 308)
        return true;
    return false;
}
exports.canCache = canCache;
;
