"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
var function_1 = require("./function");
function iapplyFirst(ixmonad) {
    return function (fa, fb) { return ixmonad.ichain(fa, function (a) { return ixmonad.ichain(fb, function () { return ixmonad.iof(a); }); }); };
}
exports.iapplyFirst = iapplyFirst;
function iapplySecond(ixmonad) {
    return function (fa, fb) { return ixmonad.ichain(fa, function_1.constant(fb)); };
}
exports.iapplySecond = iapplySecond;
