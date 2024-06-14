import request, {Options, FormData} from 'then-request';
import {Req, Res} from './messages';

function init() {
  return (req: Req): Promise<Res> => {
    // Note how even though we return a promise, the resulting rpc client will be synchronous
    const {form, ...o} = req.o || {form: undefined};
    const opts: Options = o;
    if (form) {
      const fd = new FormData();
      form.forEach(entry => {
        fd.append(entry.key, entry.value, entry.fileName);
      });
      opts.form = fd;
    }
    return request(req.m, req.u, opts).then(response => ({
      s: response.statusCode,
      h: response.headers,
      b: response.body,
      u: response.url,
    }));
  };
}
module.exports = init;
