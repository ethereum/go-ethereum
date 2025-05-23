"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
/**
 * @file A `Functor` is a type constructor which supports a mapping operation `map`.
 *
 * `map` can be used to turn functions `a -> b` into functions `f a -> f b` whose argument and return types use the type
 * constructor `f` to represent some computational context.
 *
 * Instances must satisfy the following laws:
 *
 * 1. Identity: `F.map(fa, a => a) = fa`
 * 2. Composition: `F.map(fa, a => bc(ab(a))) = F.map(F.map(fa, ab), bc)`
 */
var function_1 = require("./function");
function lift(F) {
    return function (f) { return function (fa) { return F.map(fa, f); }; };
}
exports.lift = lift;
function voidRight(F) {
    return function (a, fb) { return F.map(fb, function_1.constant(a)); };
}
exports.voidRight = voidRight;
function voidLeft(F) {
    return function (fa, b) { return F.map(fa, function_1.constant(b)); };
}
exports.voidLeft = voidLeft;
function flap(functor) {
    return function (a, ff) { return functor.map(ff, function (f) { return f(a); }); };
}
exports.flap = flap;
function getFunctorComposition(F, G) {
    return {
        map: function (fa, f) { return F.map(fa, function (ga) { return G.map(ga, f); }); }
    };
}
exports.getFunctorComposition = getFunctorComposition;
