import {HttpVerb, Response} from 'then-request';
import GenericResponse = require('http-response-object');
import {URL} from 'url';
import {Req, Res} from './messages';
import {FormData, getFormDataEntries} from './FormData';
import {Options, MessageOptions} from './Options';
const init = require('sync-rpc');
const remote = init(require.resolve('./worker'));

export {HttpVerb, Response, Options};
export {FormData};
export default function request(
  method: HttpVerb,
  url: string | URL,
  options?: Options
): Response {
  const {form, ...o} = options || {form: undefined};
  const opts: MessageOptions = o;
  if (form) {
    opts.form = getFormDataEntries(form);
  }
  const req: Req = {
    m: method,
    u: url && typeof url === 'object' ? url.href : (url as string),
    o: opts,
  };
  const res: Res = remote(req);
  return new GenericResponse(res.s, res.h, res.b, res.u);
}
module.exports = request;
module.exports.default = request;
module.exports.FormData = FormData;
