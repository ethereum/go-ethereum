"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
/**
 * @file Mutable references in the `IO` monad
 */
var IO_1 = require("./IO");
/**
 * @example
 * import { newIORef } from 'fp-ts/lib/IORef'
 *
 * assert.strictEqual(
 *   newIORef(1)
 *     .chain(ref => ref.write(2).chain(() => ref.read))
 *     .run(),
 *   2
 * )
 * @since 1.8.0
 */
var IORef = /** @class */ (function () {
    function IORef(value) {
        var _this = this;
        this.value = value;
        this.read = new IO_1.IO(function () { return _this.value; });
    }
    /**
     * @since 1.8.0
     */
    IORef.prototype.write = function (a) {
        var _this = this;
        return new IO_1.IO(function () {
            _this.value = a;
        });
    };
    /**
     * @since 1.8.0
     */
    IORef.prototype.modify = function (f) {
        var _this = this;
        return new IO_1.IO(function () {
            _this.value = f(_this.value);
        });
    };
    return IORef;
}());
exports.IORef = IORef;
/**
 * @since 1.8.0
 */
exports.newIORef = function (a) {
    return new IO_1.IO(function () { return new IORef(a); });
};
