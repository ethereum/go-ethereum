import { io } from './IO';
export var URI = 'IxIO';
/**
 * @since 1.0.0
 */
var IxIO = /** @class */ (function () {
    function IxIO(value) {
        this.value = value;
    }
    IxIO.prototype.run = function () {
        return this.value.run();
    };
    IxIO.prototype.ichain = function (f) {
        return new IxIO(this.value.chain(function (a) { return f(a).value; }));
    };
    IxIO.prototype.map = function (f) {
        return new IxIO(this.value.map(f));
    };
    IxIO.prototype.ap = function (fab) {
        return new IxIO(this.value.ap(fab.value));
    };
    IxIO.prototype.chain = function (f) {
        return new IxIO(this.value.chain(function (a) { return f(a).value; }));
    };
    return IxIO;
}());
export { IxIO };
/**
 * @since 1.0.0
 */
export var iof = function (a) {
    return new IxIO(io.of(a));
};
var ichain = function (fa, f) {
    return fa.ichain(f);
};
var map = function (fa, f) {
    return fa.map(f);
};
var of = iof;
var ap = function (fab, fa) {
    return fa.ap(fab);
};
var chain = function (fa, f) {
    return fa.chain(f);
};
/**
 * @since 1.0.0
 */
export var getMonad = function () {
    return {
        URI: URI,
        _L: undefined,
        _U: undefined,
        map: map,
        of: of,
        ap: ap,
        chain: chain
    };
};
/**
 * @since 1.0.0
 */
export var ixIO = {
    URI: URI,
    iof: iof,
    ichain: ichain
};
