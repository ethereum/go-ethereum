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
var Functor_1 = require("./Functor");
var Option_1 = require("./Option");
function getCompactableComposition(F, G) {
    var FC = Functor_1.getFunctorComposition(F, G);
    var CC = __assign({}, FC, { compact: function (fga) { return F.map(fga, G.compact); }, separate: function (fge) {
            var left = CC.compact(FC.map(fge, function (e) { return e.fold(Option_1.some, function () { return Option_1.none; }); }));
            var right = CC.compact(FC.map(fge, Option_1.fromEither));
            return { left: left, right: right };
        } });
    return CC;
}
exports.getCompactableComposition = getCompactableComposition;
