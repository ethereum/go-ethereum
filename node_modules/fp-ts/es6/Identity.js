import { tailRec } from './ChainRec';
import { toString } from './function';
import { fromEquals } from './Eq';
import { pipeable } from './pipeable';
export var URI = 'Identity';
/**
 * @since 1.0.0
 */
var Identity = /** @class */ (function () {
    function Identity(value) {
        this.value = value;
    }
    /** @obsolete */
    Identity.prototype.map = function (f) {
        return new Identity(f(this.value));
    };
    /** @obsolete */
    Identity.prototype.ap = function (fab) {
        return this.map(fab.value);
    };
    /**
     * Flipped version of `ap`
     * @obsolete
     */
    Identity.prototype.ap_ = function (fb) {
        return fb.ap(this);
    };
    /** @obsolete */
    Identity.prototype.chain = function (f) {
        return f(this.value);
    };
    /** @obsolete */
    Identity.prototype.reduce = function (b, f) {
        return f(b, this.value);
    };
    /** @obsolete */
    Identity.prototype.alt = function (fx) {
        return this;
    };
    /**
     * Lazy version of `alt`
     *
     * @example
     * import { Identity } from 'fp-ts/lib/Identity'
     *
     * const a = new Identity(1)
     * assert.deepStrictEqual(a.orElse(() => new Identity(2)), a)
     *
     * @since 1.6.0
     * @obsolete
     */
    Identity.prototype.orElse = function (fx) {
        return this;
    };
    /** @obsolete */
    Identity.prototype.extract = function () {
        return this.value;
    };
    /** @obsolete */
    Identity.prototype.extend = function (f) {
        return identity.of(f(this));
    };
    /** @obsolete */
    Identity.prototype.fold = function (f) {
        return f(this.value);
    };
    Identity.prototype.inspect = function () {
        return this.toString();
    };
    Identity.prototype.toString = function () {
        // tslint:disable-next-line: deprecation
        return "new Identity(" + toString(this.value) + ")";
    };
    return Identity;
}());
export { Identity };
/**
 * @since 1.17.0
 */
export var getShow = function (S) {
    return {
        show: function (i) { return "new Identity(" + S.show(i.value) + ")"; }
    };
};
/**
 * Use `getEq`
 *
 * @since 1.0.0
 * @deprecated
 */
export var getSetoid = getEq;
/**
 * @since 1.19.0
 */
export function getEq(E) {
    return fromEquals(function (x, y) { return E.equals(x.value, y.value); });
}
/**
 * @since 1.0.0
 */
export var identity = {
    URI: URI,
    map: function (fa, f) { return fa.map(f); },
    of: function (a) { return new Identity(a); },
    ap: function (fab, fa) { return fa.ap(fab); },
    chain: function (fa, f) { return fa.chain(f); },
    reduce: function (fa, b, f) { return fa.reduce(b, f); },
    foldMap: function (_) { return function (fa, f) { return f(fa.value); }; },
    foldr: function (fa, b, f) { return f(fa.value, b); },
    traverse: function (F) { return function (ta, f) {
        return F.map(f(ta.value), identity.of);
    }; },
    sequence: function (F) { return function (ta) {
        return F.map(ta.value, identity.of);
    }; },
    alt: function (fx, fy) { return fx.alt(fy); },
    extract: function (wa) { return wa.extract(); },
    extend: function (wa, f) { return wa.extend(f); },
    chainRec: function (a, f) {
        return new Identity(tailRec(function (a) { return f(a).value; }, a));
    }
};
//
// backporting
//
var _a = pipeable(identity), alt = _a.alt, ap = _a.ap, apFirst = _a.apFirst, apSecond = _a.apSecond, chain = _a.chain, chainFirst = _a.chainFirst, duplicate = _a.duplicate, extend = _a.extend, flatten = _a.flatten, foldMap = _a.foldMap, map = _a.map, reduce = _a.reduce, reduceRight = _a.reduceRight;
export { alt, ap, apFirst, apSecond, chain, chainFirst, duplicate, extend, flatten, foldMap, map, reduce, reduceRight };
