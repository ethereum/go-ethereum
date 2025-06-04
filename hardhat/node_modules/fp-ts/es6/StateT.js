export function get2v(F) {
    return function (s) { return F.of([s, s]); };
}
export function put(F) {
    return function (s) { return function () { return F.of([undefined, s]); }; };
}
export function modify(F) {
    return function (f) { return function (s) { return F.of([undefined, f(s)]); }; };
}
export function gets(F) {
    return function (f) { return function (s) { return F.of([f(s), s]); }; };
}
export function fromState(F) {
    return function (fa) { return function (s) { return F.of(fa.run(s)); }; };
}
export function liftF(F) {
    return function (fa) { return function (s) { return F.map(fa, function (a) { return [a, s]; }); }; };
}
export function getStateT2v(M) {
    return {
        map: function (fa, f) { return function (s) { return M.map(fa(s), function (_a) {
            var a = _a[0], s1 = _a[1];
            return [f(a), s1];
        }); }; },
        of: function (a) { return function (s) { return M.of([a, s]); }; },
        ap: function (fab, fa) { return function (s) { return M.chain(fab(s), function (_a) {
            var f = _a[0], s = _a[1];
            return M.map(fa(s), function (_a) {
                var a = _a[0], s = _a[1];
                return [f(a), s];
            });
        }); }; },
        chain: function (fa, f) { return function (s) { return M.chain(fa(s), function (_a) {
            var a = _a[0], s1 = _a[1];
            return f(a)(s1);
        }); }; }
    };
}
/** @deprecated */
export function map(F) {
    return function (f, fa) { return function (s) { return F.map(fa(s), function (_a) {
        var a = _a[0], s1 = _a[1];
        return [f(a), s1];
    }); }; };
}
/** @deprecated */
export function ap(F) {
    return function (fab, fa) { return function (s) { return F.chain(fab(s), function (_a) {
        var f = _a[0], s = _a[1];
        return F.map(fa(s), function (_a) {
            var a = _a[0], s = _a[1];
            return [f(a), s];
        });
    }); }; };
}
/** @deprecated */
export function chain(F) {
    return function (f, fa) { return function (s) { return F.chain(fa(s), function (_a) {
        var a = _a[0], s1 = _a[1];
        return f(a)(s1);
    }); }; };
}
/** @deprecated */
// tslint:disable-next-line: deprecation
export function getStateT(M) {
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
    return function (a) { return function (s) { return F.of([a, s]); }; };
}
/** @deprecated */
export function get(F) {
    return function () { return function (s) { return F.of([s, s]); }; };
}
