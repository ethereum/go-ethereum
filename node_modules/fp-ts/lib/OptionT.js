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
var Applicative_1 = require("./Applicative");
var Option_1 = require("./Option");
function fold(F) {
    return function (onNone, onSome, fa) { return F.map(fa, function (o) { return (o.isNone() ? onNone : onSome(o.value)); }); };
}
exports.fold = fold;
function getOptionT2v(M) {
    var applicativeComposition = Applicative_1.getApplicativeComposition(M, Option_1.option);
    return __assign({}, applicativeComposition, { chain: function (fa, f) { return M.chain(fa, function (o) { return (o.isNone() ? M.of(Option_1.none) : f(o.value)); }); } });
}
exports.getOptionT2v = getOptionT2v;
/** @deprecated */
// tslint:disable-next-line: deprecation
function chain(F) {
    return function (f, fa) { return F.chain(fa, function (o) { return (o.isNone() ? F.of(Option_1.none) : f(o.value)); }); };
}
exports.chain = chain;
// tslint:disable-next-line: deprecation
function getOptionT(M) {
    var applicativeComposition = Applicative_1.getApplicativeComposition(M, Option_1.option);
    return __assign({}, applicativeComposition, { 
        // tslint:disable-next-line: deprecation
        chain: chain(M) });
}
exports.getOptionT = getOptionT;
/** @deprecated */
function some(F) {
    return function (a) { return F.of(Option_1.some(a)); };
}
exports.some = some;
/** @deprecated */
function none(F) {
    return function () { return F.of(Option_1.none); };
}
exports.none = none;
/** @deprecated */
function fromOption(F) {
    return F.of;
}
exports.fromOption = fromOption;
/** @deprecated */
function liftF(F) {
    return function (fa) { return F.map(fa, Option_1.some); };
}
exports.liftF = liftF;
/** @deprecated */
function getOrElse(F) {
    return function (a) { return function (fa) { return F.map(fa, function (o) { return o.getOrElse(a); }); }; };
}
exports.getOrElse = getOrElse;
