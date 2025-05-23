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
var Foldable_1 = require("./Foldable");
var Functor_1 = require("./Functor");
function traverse(F, 
// tslint:disable-next-line: deprecation
T) {
    return T.traverse(F);
}
exports.traverse = traverse;
// tslint:disable-next-line: deprecation
function sequence(F, T) {
    return function (tfa) { return T.traverse(F)(tfa, function (fa) { return fa; }); };
}
exports.sequence = sequence;
// tslint:disable-next-line: deprecation
function getTraversableComposition(F, G) {
    return __assign({}, Functor_1.getFunctorComposition(F, G), Foldable_1.getFoldableComposition(F, G), { traverse: function (H) {
            var traverseF = F.traverse(H);
            var traverseG = G.traverse(H);
            return function (fga, f) { return traverseF(fga, function (ga) { return traverseG(ga, f); }); };
        } });
}
exports.getTraversableComposition = getTraversableComposition;
