# http-basic

Simple wrapper arround http.request/https.request

[![Build Status](https://img.shields.io/travis/ForbesLindesay/http-basic/master.svg)](https://travis-ci.org/ForbesLindesay/http-basic)
[![Dependency Status](https://img.shields.io/david/ForbesLindesay/http-basic.svg)](https://david-dm.org/ForbesLindesay/http-basic)
[![NPM version](https://img.shields.io/npm/v/http-basic.svg)](https://www.npmjs.org/package/http-basic)

## Installation

    npm install http-basic

## Usage

```js
var request = require('http-basic');

var options = {followRedirects: true, gzip: true, cache: 'memory'};

var req = request('GET', 'http://example.com', options, function (err, res) {
  if (err) throw err;
  console.dir(res.statusCode);
  res.body.resume();
});
req.end();
```

**method:**

The http method (e.g. `GET`, `POST`, `PUT`, `DELETE` etc.)

**url:**

The url as a string (e.g. `http://example.com`).  It must be fully qualified and either http or https.

**options:**

 - `headers` - (default `{}`) http headers
 - `agent` - (default: `false`) controlls keep-alive (see http://nodejs.org/api/http.html#http_http_request_options_callback)
 - `duplex` - (default: `true` except for `GET`, `OPTIONS` and `HEAD` requests) allows you to explicitly set a body on a request that uses a method that normally would not have a body
 - `followRedirects` - (default: `false`) - if true, redirects are followed (note that this only affects the result in the callback)
 - `maxRedirects` - (default: `Infinity`) - limit the number of redirects allowed.
 - `allowRedirectHeaders` (default: `null`) - an array of headers allowed for redirects (none if `null`).
 - `gzip` (default: `false`) - automatically accept gzip and deflate encodings.  This is kept completely transparent to the user.
 - `cache` - (default: `null`) - `'memory'` or `'file'` to use the default built in caches or you can pass your own cache implementation.
 - `timeout` (default: `false`) - times out if no response is returned within the given number of milliseconds.
 - `socketTimeout` (default: `false`) - calls `req.setTimeout` internally which causes the request to timeout if no new data is seen for the given number of milliseconds.
 - `retry` (default: `false`) - retry GET requests.  Set this to `true` to retry when the request errors or returns a status code greater than or equal to 400 (can also be a function that takes `(err, req, attemptNo) => shouldRetry`)
 - `retryDelay` (default: `200`) - the delay between retries (can also be set to a function that takes `(err, res, attemptNo) => delay`)
 - `maxRetries` (default: `5`) - the number of times to retry before giving up.
 - `ignoreFailedInvalidation` (default: `false`) - whether the cache should swallow errors if there is a problem removing a cached response. Note that enabling this setting may result in incorrect, cached data being returned to the user.
 - `isMatch` - `(requestHeaders: Headers, cachedResponse: CachedResponse, defaultValue: boolean) => boolean` - override the default behaviour for testing whether a cached response matches a request.
 - `isExpired` - `(cachedResponse: CachedResponse, defaultValue: boolean) => boolean` - override the default behaviour for testing whether a cached response has expired
 - `canCache` - `(res: Response<NodeJS.ReadableStream>, defaultValue: boolean) => boolean` - override the default behaviour for testing whether a response can be cached

**callback:**

The callback is called with `err` as the first argument and `res` as the second argument. `res` is an [http-response-object](https://github.com/ForbesLindesay/http-response-object).  It has the following properties:

 - `statusCode` - a number representing the HTTP Status Code
 - `headers` - an object representing the HTTP headers
 - `body` - a readable stream respresenting the request body.
 - `url` - the URL that was requested (in the case of redirects, this is the final url that was requested)

**returns:**

If the method is `GET`, `DELETE` or `HEAD`, it returns `undefined`.

Otherwise, it returns a writable stream for the body of the request.

## Implementing a Cache

A `Cache` is an object with three methods:

 - `getResponse(url, callback)` - retrieve a cached response object
 - `setResponse(url, response)` - cache a response object
 - `invalidateResponse(url, callback)` - remove a response which is no longer valid

A cached response object is an object with the following properties:

 - `statusCode` - Number
 - `headers` - Object (key value pairs of strings)
 - `body` - Stream (a stream of binary data)
 - `requestHeaders` - Object (key value pairs of strings)
 - `requestTimestamp` - Number

`getResponse` should call the callback with an optional error and either `null` or a cached response object, depending on whether the url can be found in the cache.  Only `GET`s are cached.

`setResponse` should just swallow any errors it has (or resport them using `console.warn`).

`invalidateResponse` should call the callback with an optional error if it is unable to invalidate a response.

A cache may also define any of the methods from `lib/cache-utils.js` to override behaviour for what gets cached.  It is currently still only possible to cache "get" requests, although this could be changed.

## License

  MIT
