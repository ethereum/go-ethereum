# sync-request

Make synchronous web requests with cross-platform support.

> Requires at least node 8

# **N.B.** You should **not** be using this in a production application. In a node.js application you will find that you are completely unable to scale your server. In a client application you will find that sync-request causes the app to hang/freeze. Synchronous web requests are the number one cause of browser crashes. For production apps, you should use [then-request](https://github.com/then/then-request), which is exactly the same except that it is asynchronous.

[![Build Status](https://img.shields.io/travis/ForbesLindesay/sync-request/master.svg)](https://travis-ci.org/ForbesLindesay/sync-request)
[![Dependency Status](https://img.shields.io/david/ForbesLindesay/sync-request.svg)](https://david-dm.org/ForbesLindesay/sync-request)
[![NPM version](https://img.shields.io/npm/v/sync-request.svg)](https://www.npmjs.org/package/sync-request)

## Installation

    npm install sync-request

## Usage

```js
request(method, url, options);
```

e.g.

* GET request without options

```js
var request = require('sync-request');
var res = request('GET', 'http://example.com');
console.log(res.getBody());
```

* GET request with options

```js
var request = require('sync-request');
var res = request('GET', 'https://example.com', {
  headers: {
    'user-agent': 'example-user-agent',
  },
});
console.log(res.getBody());
```

* POST request to a JSON endpoint

```js
var request = require('sync-request');
var res = request('POST', 'https://example.com/create-user', {
  json: {username: 'ForbesLindesay'},
});
var user = JSON.parse(res.getBody('utf8'));
```

**Method:**

An HTTP method (e.g. `GET`, `POST`, `PUT`, `DELETE` or `HEAD`). It is not case sensitive.

**URL:**

A url as a string (e.g. `http://example.com`). Relative URLs are allowed in the browser.

**Options:**

* `qs` - an object containing querystring values to be appended to the uri
* `headers` - http headers (default: `{}`)
* `body` - body for PATCH, POST and PUT requests. Must be a `Buffer` or `String` (only strings are accepted client side)
* `json` - sets `body` but to JSON representation of value and adds `Content-type: application/json`. Does not have any affect on how the response is treated.
* `cache` - Set this to `'file'` to enable a local cache of content. A separate process is still spawned even for cache requests. This option is only used if running in node.js
* `followRedirects` - defaults to `true` but can be explicitly set to `false` on node.js to prevent then-request following redirects automatically.
* `maxRedirects` - sets the maximum number of redirects to follow before erroring on node.js (default: `Infinity`)
* `allowRedirectHeaders` (default: `null`) - an array of headers allowed for redirects (none if `null`).
* `gzip` - defaults to `true` but can be explicitly set to `false` on node.js to prevent then-request automatically supporting the gzip encoding on responses.
* `timeout` (default: `false`) - times out if no response is returned within the given number of milliseconds.
* `socketTimeout` (default: `false`) - calls `req.setTimeout` internally which causes the request to timeout if no new data is seen for the given number of milliseconds. This option is ignored in the browser.
* `retry` (default: `false`) - retry GET requests. Set this to `true` to retry when the request errors or returns a status code greater than or equal to 400
* `retryDelay` (default: `200`) - the delay between retries in milliseconds
* `maxRetries` (default: `5`) - the number of times to retry before giving up.

These options are passed through to [then-request](https://github.com/then/then-request), so any options that work for then-request should work for sync-request (with the exception of custom and memory caching strategies, and passing functions for handling retries).

**Returns:**

A `Response` object.

Note that even for status codes that represent an error, the request function will still return a response. You can call `getBody` if you want to error on invalid status codes. The response has the following properties:

* `statusCode` - a number representing the HTTP status code
* `headers` - http response headers
* `body` - a string if in the browser or a buffer if on the server

It also has a method `res.getBody(encoding?)` which looks like:

```js
function getBody(encoding) {
  if (this.statusCode >= 300) {
    var err = new Error(
      'Server responded with status code ' +
        this.statusCode +
        ':\n' +
        this.body.toString(encoding)
    );
    err.statusCode = this.statusCode;
    err.headers = this.headers;
    err.body = this.body;
    throw err;
  }
  return encoding ? this.body.toString(encoding) : this.body;
}
```

## Common Problems

### Could not use "nc", falling back to slower node.js method for sync requests.

If you are running on windows, or some unix systems, you may see the message above. It will not cause any problems, but will add an overhead of ~100ms to each request you make. If you want to speed up your requests, you will need to install an implementation of the `nc` unix utility. This usually done via something like:

```
apt-get install netcat
```

## How is this possible?

Internally, this uses a separate worker process that is run using [childProcess.spawnSync](http://nodejs.org/docs/v0.11.13/api/child_process.html#child_process_child_process_spawnsync_command_args_options).

The worker then makes the actual request using [then-request](https://www.npmjs.org/package/then-request) so this has almost exactly the same API as that.

This can also be used in a web browser via browserify because xhr has built in support for synchronous execution. Note that this is not recommended as it will be blocking.

## License

MIT
