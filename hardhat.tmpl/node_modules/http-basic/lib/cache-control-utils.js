"use strict";
exports.__esModule = true;
var parseCacheControl = require('parse-cache-control');
function parseCacheControlHeader(res) {
    var cacheControl = res.headers['cache-control'];
    var normalisedCacheControl = typeof cacheControl === 'string' ? cacheControl.trim() : ''; // must be normalised for parsing (e.g. parseCacheControl)
    if (!cacheControl) {
        return null;
    }
    return parseCacheControl(cacheControl);
}
// for the purposes of this library, we err on the side of caution and do not cache anything except public (or implicit public)
var nonCaching = ['private', 'no-cache', 'no-store', 'no-transform', 'must-revalidate', 'proxy-revalidate'];
function isCacheControlCacheable(parsedCacheControl) {
    if (!parsedCacheControl) {
        return false;
    }
    if (parsedCacheControl.public) {
        return true;
    }
    // note that the library does not currently support s-maxage
    if (parsedCacheControl["max-age"]) {
        // https://www.w3.org/Protocols/rfc2616/rfc2616-sec14.html#sec14.9.3
        // The max-age directive on a response implies that the response is cacheable (i.e., "public") unless some other, more restrictive cache directive is also present.
        for (var i = 0; i < nonCaching.length; i++) {
            if (parsedCacheControl[nonCaching[i]]) {
                return false;
            }
        }
        return true;
    }
    return false;
}
/**
 * returns true if this response is cacheable (according to cache-control headers)
 */
function isCacheable(res) {
    return isCacheControlCacheable(parseCacheControlHeader(res));
}
exports.isCacheable = isCacheable;
function buildPolicy(parsedCacheControl) {
    // note that the library does not currently support s-maxage
    return { maxage: parsedCacheControl['max-age'] || null };
}
/**
 * if the response is cacheable, returns an object detailing the maxage of the cache
 * otherwise returns null
 */
function cachePolicy(res) {
    var parsed = parseCacheControlHeader(res);
    return parsed && isCacheControlCacheable(parsed) ? buildPolicy(parsed) : null;
}
exports.cachePolicy = cachePolicy;
