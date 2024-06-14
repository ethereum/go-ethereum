"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
function get2v(F) {
    return function (s) { return F.of([s, s]); };
}
exports.get2v = get2v;
function put(F) {
    return function (s) { return function () { return F.of([undefined, s]); }; };
}
exports.put = put;
function modify(F) {
    return function (f) { return function (s) { return F.of([undefined, f(s)]); }; };
}
exports.modify = modify;
function gets(F) {
    return function (f) { return function (s) { return F.of([f(s), s]); }; };
}
exports.gets = gets;
function fromState(F) {
    return function (fa) { return function (s) { return F.of(fa.run(s)); }; };
}
exports.fromState = fromState;
function liftF(F) {
    return function (fa) { return function (s) { return F.map(fa, function (a) { return [a, s]; }); }; };
}
exports.liftF = liftF;
function getStateT2v(M) {
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
exports.getStateT2v = getStateT2v;
/** @deprecated */
function map(F) {
    return function (f, fa) { return function (s) { return F.map(fa(s), function (_a) {
        var a = _a[0], s1 = _a[1];
        return [f(a), s1];
    }); }; };
}
exports.map = map;
/** @deprecated */
function ap(F) {
    return function (fab, fa) { return function (s) { return F.chain(fab(s), function (_a) {
        var f = _a[0], s = _a[1];
        return F.map(fa(s), function (_a) {
            var a = _a[0], s = _a[1];
            return [f(a), s];
        });
    }); }; };
}
exports.ap = ap;
/** @deprecated */
function chain(F) {
    return function (f, fa) { return function (s) { return F.chain(fa(s), function (_a) {
        var a = _a[0], s1 = _a[1];
        return f(a)(s1);
    }); }; };
}
exports.chain = chain;
/** @deprecated */
// tslint:disable-next-line: deprecation
function getStateT(M) {
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
exports.getStateT = getStateT;
/** @deprecated */
function of(F) {
    return function (a) { return function (s) { return F.of([a, s]); }; };
}
exports.of = of;
/** @deprecated */
function get(F) {
    return function () { return function (s) { return F.of([s, s]); }; };
}
exports.get = get;
