"use strict";
var __rest = (this && this.__rest) || function (s, e) {
    var t = {};
    for (var p in s) if (Object.prototype.hasOwnProperty.call(s, p) && e.indexOf(p) < 0)
        t[p] = s[p];
    if (s != null && typeof Object.getOwnPropertySymbols === "function")
        for (var i = 0, p = Object.getOwnPropertySymbols(s); i < p.length; i++) if (e.indexOf(p[i]) < 0)
            t[p[i]] = s[p[i]];
    return t;
};
exports.__esModule = true;
var then_request_1 = require("then-request");
function init() {
    return function (req) {
        // Note how even though we return a promise, the resulting rpc client will be synchronous
        var _a = req.o || { form: undefined }, form = _a.form, o = __rest(_a, ["form"]);
        var opts = o;
        if (form) {
            var fd_1 = new then_request_1.FormData();
            form.forEach(function (entry) {
                fd_1.append(entry.key, entry.value, entry.fileName);
            });
            opts.form = fd_1;
        }
        return then_request_1["default"](req.m, req.u, opts).then(function (response) { return ({
            s: response.statusCode,
            h: response.headers,
            b: response.body,
            u: response.url
        }); });
    };
}
module.exports = init;
