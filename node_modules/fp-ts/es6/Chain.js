export function flatten(chain) {
    return function (mma) { return chain.chain(mma, function (ma) { return ma; }); };
}
