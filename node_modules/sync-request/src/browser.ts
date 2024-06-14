import {URL} from 'url';
import {HttpVerb, Response} from 'then-request';
import handleQs from 'then-request/lib/handle-qs.js';
import {Options} from './Options';
import GenericResponse = require('http-response-object');

const fd = FormData as any;
export {fd as FormData};
export default function doRequest(
  method: HttpVerb,
  url: string | URL,
  options?: Options
): Response {
  var xhr = new XMLHttpRequest();

  // check types of arguments

  if (typeof method !== 'string') {
    throw new TypeError('The method must be a string.');
  }
  if (url && typeof url === 'object') {
    url = url.href;
  }
  if (typeof url !== 'string') {
    throw new TypeError('The URL/path must be a string.');
  }
  if (options === null || options === undefined) {
    options = {};
  }
  if (typeof options !== 'object') {
    throw new TypeError('Options must be an object (or null).');
  }

  method = method.toUpperCase() as any;
  options.headers = options.headers || {};

  // handle cross domain

  var match;
  var crossDomain = !!(
    (match = /^([\w-]+:)?\/\/([^\/]+)/.exec(url)) && match[2] != location.host
  );
  if (!crossDomain) options.headers['X-Requested-With'] = 'XMLHttpRequest';

  // handle query string
  if (options.qs) {
    url = handleQs(url, options.qs);
  }

  // handle json body
  if (options.json) {
    options.body = JSON.stringify(options.json);
    options.headers['content-type'] = 'application/json';
  }
  if (options.form) {
    options.body = options.form as any;
  }

  // method, url, async
  xhr.open(method, url, false);

  for (var name in options.headers) {
    xhr.setRequestHeader(name.toLowerCase(), '' + options.headers[name]);
  }

  // avoid sending empty string (#319)
  xhr.send(options.body ? options.body : null);

  var headers = {};
  xhr
    .getAllResponseHeaders()
    .split('\r\n')
    .forEach(function(header) {
      var h = header.split(':');
      if (h.length > 1) {
        (headers as any)[h[0].toLowerCase()] = h
          .slice(1)
          .join(':')
          .trim();
      }
    });
  return new GenericResponse<string>(
    xhr.status,
    headers,
    xhr.responseText,
    url
  );
}
module.exports = doRequest;
module.exports.default = doRequest;
module.exports.FormData = fd;
