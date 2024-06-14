"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
function lift(contravariant) {
    return function (f) { return function (fa) { return contravariant.contramap(fa, f); }; };
}
exports.lift = lift;
