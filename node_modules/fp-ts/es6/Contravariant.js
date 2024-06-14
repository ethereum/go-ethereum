export function lift(contravariant) {
    return function (f) { return function (fa) { return contravariant.contramap(fa, f); }; };
}
