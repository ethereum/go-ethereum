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
var GenericResponse = require("http-response-object");
var FormData_1 = require("./FormData");
exports.FormData = FormData_1.FormData;
var init = require('sync-rpc');
var remote = init(require.resolve('./worker'));
function request(method, url, options) {
    var _a = options || { form: undefined }, form = _a.form, o = __rest(_a, ["form"]);
    var opts = o;
    if (form) {
        opts.form = FormData_1.getFormDataEntries(form);
    }
    var req = {
        m: method,
        u: url && typeof url === 'object' ? url.href : url,
        o: opts
    };
    var res = remote(req);
    return new GenericResponse(res.s, res.h, res.b, res.u);
}
exports["default"] = request;
module.exports = request;
module.exports["default"] = request;
module.exports.FormData = FormData_1.FormData;
