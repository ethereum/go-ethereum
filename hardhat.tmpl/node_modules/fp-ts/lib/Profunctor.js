"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
function lmap(profunctor) {
    return function (fbc, f) { return profunctor.promap(fbc, f, function (c) { return c; }); };
}
exports.lmap = lmap;
function rmap(profunctor) {
    return function (fbc, g) { return profunctor.promap(fbc, function (b) { return b; }, g); };
}
exports.rmap = rmap;
