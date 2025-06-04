import { toString } from './function';
export var URI = 'Free';
var Pure = /** @class */ (function () {
    function Pure(value) {
        this.value = value;
        this._tag = 'Pure';
    }
    Pure.prototype.map = function (f) {
        return new Pure(f(this.value));
    };
    Pure.prototype.ap = function (fab) {
        var _this = this;
        return fab.chain(function (f) { return _this.map(f); }); // <- derived
    };
    /**
     * Flipped version of `ap`
     */
    Pure.prototype.ap_ = function (fb) {
        return fb.ap(this);
    };
    Pure.prototype.chain = function (f) {
        return f(this.value);
    };
    Pure.prototype.inspect = function () {
        return this.toString();
    };
    Pure.prototype.toString = function () {
        // tslint:disable-next-line: deprecation
        return "new Pure(" + toString(this.value) + ")";
    };
    Pure.prototype.isPure = function () {
        return true;
    };
    Pure.prototype.isImpure = function () {
        return false;
    };
    return Pure;
}());
export { Pure };
var Impure = /** @class */ (function () {
    function Impure(fx, f) {
        this.fx = fx;
        this.f = f;
        this._tag = 'Impure';
    }
    Impure.prototype.map = function (f) {
        var _this = this;
        return new Impure(this.fx, function (x) { return _this.f(x).map(f); });
    };
    Impure.prototype.ap = function (fab) {
        var _this = this;
        return fab.chain(function (f) { return _this.map(f); }); // <- derived
    };
    Impure.prototype.ap_ = function (fb) {
        return fb.ap(this);
    };
    Impure.prototype.chain = function (f) {
        var _this = this;
        return new Impure(this.fx, function (x) { return _this.f(x).chain(f); });
    };
    Impure.prototype.inspect = function () {
        return this.toString();
    };
    Impure.prototype.toString = function () {
        // tslint:disable-next-line: deprecation
        return "new Impure(" + (toString(this.fx), toString(this.f)) + ")";
    };
    Impure.prototype.isPure = function () {
        return false;
    };
    Impure.prototype.isImpure = function () {
        return true;
    };
    return Impure;
}());
export { Impure };
/**
 * @since 1.0.0
 */
export var of = function (a) {
    return new Pure(a);
};
/**
 * Lift an impure value described by the generating type constructor `F` into the free monad
 *
 * @since 1.0.0
 */
export var liftF = function (fa) {
    return new Impure(fa, function (a) { return of(a); });
};
var substFree = function (f) {
    function go(fa) {
        switch (fa._tag) {
            case 'Pure':
                return of(fa.value);
            case 'Impure':
                return f(fa.fx).chain(function (x) { return go(fa.f(x)); });
        }
    }
    return go;
};
export function hoistFree(nt) {
    return substFree(function (fa) { return liftF(nt(fa)); });
}
export function foldFree(M) {
    return function (nt, fa) {
        if (fa.isPure()) {
            return M.of(fa.value);
        }
        else {
            return M.chain(nt(fa.fx), function (x) { return foldFree(M)(nt, fa.f(x)); });
        }
    };
}
