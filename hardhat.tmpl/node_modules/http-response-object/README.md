# http-response-object

A simple object to represent an http response (with flow and typescript types)

[![Build Status](https://img.shields.io/travis/ForbesLindesay/http-response-object/master.svg)](https://travis-ci.org/ForbesLindesay/http-response-object)
[![Dependency Status](https://img.shields.io/david/ForbesLindesay/http-response-object.svg)](https://david-dm.org/ForbesLindesay/http-response-object)
[![NPM version](https://img.shields.io/npm/v/http-response-object.svg)](https://www.npmjs.org/package/http-response-object)


## Installation

    npm install http-response-object

## Usage

```js
var Response = require('http-response-object');
var res = new Response(200, {}, new Buffer('A ok'), 'http://example.com');
//res.statusCode === 200
//res.headers === {}
//res.body === new Buffer('A ok')
//res.url === 'http://example.com'
res.getBody();
// => new Buffer('A ok')

var res = new Response(404, {'Header': 'value'}, new Buffer('Wheres this page'), 'http://example.com');
//res.statusCode === 404
//res.headers === {header: 'value'}
//res.body === new Buffer('Wheres this page')
//res.url === 'http://example.com'
res.getBody();
// => throws error with `statusCode`, `headers`, `body` and `url` properties copied from the response
```

## Properties

 - `statusCode`: Number - the status code of the response
 - `headers`: Object - the headers of the response.  The keys are automatically made lower case.
 - `body`: Buffer | String - the body of the response. Should be a buffer on the server side, but may be a simple string for lighter weight clients.
 - `url`: String - the url that was requested.  If there were redirects, this should be the last url to get requested.

## License

  MIT
