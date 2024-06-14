"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
var IO_1 = require("./IO");
exports.URI = 'IxIO';
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
exports.IxIO = IxIO;
/**
 * @since 1.0.0
 */
exports.iof = function (a) {
    return new IxIO(IO_1.io.of(a));
};
var ichain = function (fa, f) {
    return fa.ichain(f);
};
var map = function (fa, f) {
    return fa.map(f);
};
var of = exports.iof;
var ap = function (fab, fa) {
    return fa.ap(fab);
};
var chain = function (fa, f) {
    return fa.chain(f);
};
/**
 * @since 1.0.0
 */
exports.getMonad = function () {
    return {
        URI: exports.URI,
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
exports.ixIO = {
    URI: exports.URI,
    iof: exports.iof,
    ichain: ichain
};
