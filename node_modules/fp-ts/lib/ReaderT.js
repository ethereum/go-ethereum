"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
function fromReader(F) {
    return function (fa) { return function (e) { return F.of(fa.run(e)); }; };
}
exports.fromReader = fromReader;
function getReaderT2v(M) {
    return {
        map: function (fa, f) { return function (e) { return M.map(fa(e), f); }; },
        of: function (a) { return function () { return M.of(a); }; },
        ap: function (fab, fa) { return function (e) { return M.ap(fab(e), fa(e)); }; },
        chain: function (fa, f) { return function (e) { return M.chain(fa(e), function (a) { return f(a)(e); }); }; }
    };
}
exports.getReaderT2v = getReaderT2v;
/** @deprecated */
function map(F) {
    return function (f, fa) { return function (e) { return F.map(fa(e), f); }; };
}
exports.map = map;
/** @deprecated */
function chain(F) {
    return function (f, fa) { return function (e) { return F.chain(fa(e), function (a) { return f(a)(e); }); }; };
}
exports.chain = chain;
/** @deprecated */
// tslint:disable-next-line: deprecation
function getReaderT(M) {
    return {
        // tslint:disable-next-line: deprecation
        map: map(M),
        // tslint:disable-next-line: deprecation
        of: of(M),
        // tslint:disable-next-line: deprecation
        ap: ap(M),
        // tslint:disable-next-line: deprecation
        chain: chain(M)
    };
}
exports.getReaderT = getReaderT;
/** @deprecated */
function of(F) {
    return function (a) { return function () { return F.of(a); }; };
}
exports.of = of;
/** @deprecated */
function ap(F) {
    return function (fab, fa) { return function (e) { return F.ap(fab(e), fa(e)); }; };
}
exports.ap = ap;
/** @deprecated */
function ask(F) {
    return function () { return F.of; };
}
exports.ask = ask;
/** @deprecated */
function asks(F) {
    return function (f) { return function (e) { return F.of(f(e)); }; };
}
exports.asks = asks;
