export function fromReader(F) {
    return function (fa) { return function (e) { return F.of(fa.run(e)); }; };
}
export function getReaderT2v(M) {
    return {
        map: function (fa, f) { return function (e) { return M.map(fa(e), f); }; },
        of: function (a) { return function () { return M.of(a); }; },
        ap: function (fab, fa) { return function (e) { return M.ap(fab(e), fa(e)); }; },
        chain: function (fa, f) { return function (e) { return M.chain(fa(e), function (a) { return f(a)(e); }); }; }
    };
}
/** @deprecated */
export function map(F) {
    return function (f, fa) { return function (e) { return F.map(fa(e), f); }; };
}
/** @deprecated */
export function chain(F) {
    return function (f, fa) { return function (e) { return F.chain(fa(e), function (a) { return f(a)(e); }); }; };
}
/** @deprecated */
// tslint:disable-next-line: deprecation
export function getReaderT(M) {
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
/** @deprecated */
export function of(F) {
    return function (a) { return function () { return F.of(a); }; };
}
/** @deprecated */
export function ap(F) {
    return function (fab, fa) { return function (e) { return F.ap(fab(e), fa(e)); }; };
}
/** @deprecated */
export function ask(F) {
    return function () { return F.of; };
}
/** @deprecated */
export function asks(F) {
    return function (f) { return function (e) { return F.of(f(e)); }; };
}
