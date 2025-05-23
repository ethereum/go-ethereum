"use strict";
var __assign = (this && this.__assign) || function () {
    __assign = Object.assign || function(t) {
        for (var s, i = 1, n = arguments.length; i < n; i++) {
            s = arguments[i];
            for (var p in s) if (Object.prototype.hasOwnProperty.call(s, p))
                t[p] = s[p];
        }
        return t;
    };
    return __assign.apply(this, arguments);
};
Object.defineProperty(exports, "__esModule", { value: true });
/**
 * @file The `Applicative` type class extends the `Apply` type class with a `of` function, which can be used to create values
 * of type `f a` from values of type `a`.
 *
 * Where `Apply` provides the ability to lift functions of two or more arguments to functions whose arguments are
 * wrapped using `f`, and `Functor` provides the ability to lift functions of one argument, `pure` can be seen as the
 * function which lifts functions of _zero_ arguments. That is, `Applicative` functors support a lifting operation for
 * any number of function arguments.
 *
 * Instances must satisfy the following laws in addition to the `Apply` laws:
 *
 * 1. Identity: `A.ap(A.of(a => a), fa) = fa`
 * 2. Homomorphism: `A.ap(A.of(ab), A.of(a)) = A.of(ab(a))`
 * 3. Interchange: `A.ap(fab, A.of(a)) = A.ap(A.of(ab => ab(a)), fab)`
 *
 * Note. `Functor`'s `map` can be derived: `A.map(x, f) = A.ap(A.of(f), x)`
 */
var Apply_1 = require("./Apply");
var Functor_1 = require("./Functor");
function when(F) {
    return function (condition, fu) { return (condition ? fu : F.of(undefined)); };
}
exports.when = when;
function getApplicativeComposition(F, G) {
    return __assign({}, Functor_1.getFunctorComposition(F, G), { of: function (a) { return F.of(G.of(a)); }, ap: function (fgab, fga) {
            return F.ap(F.map(fgab, function (h) { return function (ga) { return G.ap(h, ga); }; }), fga);
        } });
}
exports.getApplicativeComposition = getApplicativeComposition;
function getMonoid(F, M) {
    // tslint:disable-next-line: deprecation
    var S = Apply_1.getSemigroup(F, M)();
    return function () { return (__assign({}, S, { empty: F.of(M.empty) })); };
}
exports.getMonoid = getMonoid;
