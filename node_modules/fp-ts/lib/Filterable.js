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
 * @file `Filterable` represents data structures which can be _partitioned_/_filtered_.
 *
 * Adapted from https://github.com/LiamGoodacre/purescript-filterable/blob/master/src/Data/Filterable.purs
 */
var Compactable_1 = require("./Compactable");
var Option_1 = require("./Option");
function getFilterableComposition(F, G) {
    var FC = __assign({}, Compactable_1.getCompactableComposition(F, G), { partitionMap: function (fga, f) {
            var left = FC.filterMap(fga, function (a) { return f(a).fold(Option_1.some, function () { return Option_1.none; }); });
            var right = FC.filterMap(fga, function (a) { return f(a).fold(function () { return Option_1.none; }, Option_1.some); });
            return { left: left, right: right };
        }, partition: function (fga, p) {
            var left = FC.filter(fga, function (a) { return !p(a); });
            var right = FC.filter(fga, p);
            return { left: left, right: right };
        }, filterMap: function (fga, f) { return F.map(fga, function (ga) { return G.filterMap(ga, f); }); }, filter: function (fga, f) { return F.map(fga, function (ga) { return G.filter(ga, f); }); } });
    return FC;
}
exports.getFilterableComposition = getFilterableComposition;
