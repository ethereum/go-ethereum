# then-request

A request library that returns promises and supports both browsers and node.js

[![Build Status](https://img.shields.io/travis/then/then-request/master.svg)](https://travis-ci.org/then/then-request)
[![Dependency Status](https://img.shields.io/david/then/then-request.svg)](https://david-dm.org/then/then-request)
[![NPM version](https://img.shields.io/npm/v/then-request.svg)](https://www.npmjs.org/package/then-request)

<a target='_blank' rel='nofollow' href='https://app.codesponsor.io/link/gg9sZwctSLxyov1sJwW6pfyS/then/then-request'>
  <img alt='Sponsor' width='888' height='68' src='https://app.codesponsor.io/embed/gg9sZwctSLxyov1sJwW6pfyS/then/then-request.svg' />
</a>

## Installation

    npm install then-request

## Usage

`request(method, url, options, callback?)`

The following examples all work on both client and server.

```js
var request = require('then-request');

request('GET', 'http://example.com').done(function (res) {
  console.log(res.getBody());
});

request('POST', 'http://example.com/json-api', {json: {some: 'values'}}).getBody('utf8').then(JSON.parse).done(function (res) {
  console.log(res);
});

var FormData = request.FormData;
var data = new FormData();

data.append('some', 'values');

request('POST', 'http://example.com/form-api', {form: data}).done(function (res) {
  console.log(res.getBody());
});
```

Or with ES6

```js
import request, {FormData} from 'then-request';

request('GET', 'http://example.com').done((res) => {
  console.log(res.getBody());
});

request('POST', 'http://example.com/json-api', {json: {some: 'values'}}).getBody('utf8').then(JSON.parse).done((res) => {
  console.log(res);
});

var FormData = request.FormData;
var data = new FormData();

data.append('some', 'values');

request('POST', 'http://example.com/form-api', {form: data}).done((res) => {
  console.log(res.getBody());
});
```

**Method:**

An HTTP method (e.g. `GET`, `POST`, `PUT`, `DELETE` or `HEAD`). It is not case sensitive.

**URL:**

A url as a string (e.g. `http://example.com`). Relative URLs are allowed in the browser.

**Options:**

 - `qs` - an object containing querystring values to be appended to the uri
 - `headers` - http headers (default: `{}`)
 - `body` - body for PATCH, POST and PUT requests.  Must be a `Buffer`, `ReadableStream` or `String` (only strings are accepted client side)
 - `json` - sets `body` but to JSON representation of value and adds `Content-type: application/json`.  Does not have any affect on how the response is treated.
 - `form` - You can pass a `FormData` instance to the `form` option, this will manage all the appropriate headers for you.  Does not have any affect on how the response is treated.
 - `cache` - only used in node.js (browsers already have their own caches) Can be `'memory'`, `'file'` or your own custom implementaton (see https://github.com/ForbesLindesay/http-basic#implementing-a-cache).
 - `followRedirects` - defaults to `true` but can be explicitly set to `false` on node.js to prevent then-request following redirects automatically.
 - `maxRedirects` - sets the maximum number of redirects to follow before erroring on node.js (default: `Infinity`)
 - `allowRedirectHeaders` (default: `null`) - an array of headers allowed for redirects (none if `null`).
 - `gzip` - defaults to `true` but can be explicitly set to `false` on node.js to prevent then-request automatically supporting the gzip encoding on responses.
 - `agent` - (default: `false`) - An `Agent` to controll keep-alive. When set to `false` use an `Agent` with default values.
 - `timeout` (default: `false`) - times out if no response is returned within the given number of milliseconds.
 - `socketTimeout` (default: `false`) - calls `req.setTimeout` internally which causes the request to timeout if no new data is seen for the given number of milliseconds.  This option is ignored in the browser.
 - `retry` (default: `false`) - retry GET requests.  Set this to `true` to retry when the request errors or returns a status code greater than or equal to 400 (can also be a function that takes `(err, req, attemptNo) => shouldRetry`)
 - `retryDelay` (default: `200`) - the delay between retries (can also be set to a function that takes `(err, res, attemptNo) => delay`)
 - `maxRetries` (default: `5`) - the number of times to retry before giving up.


**Returns:**

A [Promise](https://www.promisejs.org/) is returned that eventually resolves to the `Response`.  The resulting Promise also has an additional `.getBody(encoding?)` method that is equivallent to calling `.then(function (res) { return res.getBody(encoding?); })`.

### Response

Note that even for status codes that represent an error, the promise will be resolved as the request succeeded.  You can call `getBody` if you want to error on invalid status codes.  The response has the following properties:

 - `statusCode` - a number representing the HTTP status code
 - `headers` - http response headers
 - `body` - a string if in the browser or a buffer if on the server
 - `url` - the URL that was requested (in the case of redirects on the server, this is the final url that was requested)

It also has a method `getBody(encoding?)` which looks like:

```js
function getBody(encoding) {
  if (this.statusCode >= 300) {
    var err = new Error('Server responded with status code ' + this.statusCode + ':\n' + this.body.toString(encoding));
    err.statusCode = this.statusCode;
    err.headers = this.headers;
    err.body = this.body;
    throw err;
  }
  return encoding ? this.body.toString(encoding) : this.body;
}
```

### FormData

```js
var FormData = require('then-request').FormData;
```

Form data either exposes the node.js module, [form-data](https://www.npmjs.com/package/form-data), or the builtin browser object [FormData](https://developer.mozilla.org/en/docs/Web/API/FormData), as appropriate.

They have broadly the same API, with the exception that form-data handles node.js streams and Buffers, while FormData handles the browser's `File` Objects.

## License

  MIT
