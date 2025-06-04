export function lmap(profunctor) {
    return function (fbc, f) { return profunctor.promap(fbc, f, function (c) { return c; }); };
}
export function rmap(profunctor) {
    return function (fbc, g) { return profunctor.promap(fbc, function (b) { return b; }, g); };
}
