var __assign = (this && this.__assign) || function () {
    __assign = Object.assign || function(t) {
        for (var s, i = 1, n = arguments.length; i < n; i++) {
            s = arguments[i];
            for (var p in s) if (Object.prototype.hasOwnProperty.call(s, p))
                t[p] = s[p];
        }
        return t;
    };
    return __assign.apply(this, arguments);
};
import { identity, toString } from './function';
import { getDualMonoid } from './Monoid';
import { fromCompare } from './Ord';
import { fromEquals } from './Eq';
import { pipeable } from './pipeable';
export var URI = 'Option';
var None = /** @class */ (function () {
    function None() {
        this._tag = 'None';
    }
    /**
     * Takes a function `f` and an `Option` of `A`. Maps `f` either on `None` or `Some`, Option's data constructors. If it
     * maps on `Some` then it will apply the `f` on `Some`'s value, if it maps on `None` it will return `None`.
     *
     * @example
     * import { some } from 'fp-ts/lib/Option'
     *
     * assert.deepStrictEqual(some(1).map(n => n * 2), some(2))
     * @obsolete
     */
    None.prototype.map = function (f) {
        return none;
    };
    /**
     * Maps `f` over this `Option`'s value. If the value returned from `f` is null or undefined, returns `None`
     *
     * @example
     * import { none, some } from 'fp-ts/lib/Option'
     *
     * interface Foo {
     *   bar?: {
     *     baz?: string
     *   }
     * }
     *
     * assert.deepStrictEqual(
     *   some<Foo>({ bar: { baz: 'quux' } })
     *     .mapNullable(foo => foo.bar)
     *     .mapNullable(bar => bar.baz),
     *   some('quux')
     * )
     * assert.deepStrictEqual(
     *   some<Foo>({ bar: {} })
     *     .mapNullable(foo => foo.bar)
     *     .mapNullable(bar => bar.baz),
     *   none
     * )
     * assert.deepStrictEqual(
     *   some<Foo>({})
     *     .mapNullable(foo => foo.bar)
     *     .mapNullable(bar => bar.baz),
     *   none
     * )
     * @obsolete
     */
    None.prototype.mapNullable = function (f) {
        return none;
    };
    /**
     * `ap`, some may also call it "apply". Takes a function `fab` that is in the context of `Option`, and applies that
     * function to this `Option`'s value. If the `Option` calling `ap` is `none` it will return `none`.
     *
     * @example
     * import { some, none } from 'fp-ts/lib/Option'
     *
     * assert.deepStrictEqual(some(2).ap(some((x: number) => x + 1)), some(3))
     * assert.deepStrictEqual(none.ap(some((x: number) => x + 1)), none)
     * @obsolete
     */
    None.prototype.ap = function (fab) {
        return none;
    };
    /**
     * Flipped version of `ap`
     *
     * @example
     * import { some, none } from 'fp-ts/lib/Option'
     *
     * assert.deepStrictEqual(some((x: number) => x + 1).ap_(some(2)), some(3))
     * assert.deepStrictEqual(none.ap_(some(2)), none)
     * @obsolete
     */
    None.prototype.ap_ = function (fb) {
        return fb.ap(this);
    };
    /**
     * Returns the result of applying f to this `Option`'s value if this `Option` is nonempty. Returns `None` if this
     * `Option` is empty. Slightly different from `map` in that `f` is expected to return an `Option` (which could be
     * `None`)
     * @obsolete
     */
    None.prototype.chain = function (f) {
        return none;
    };
    /** @obsolete */
    None.prototype.reduce = function (b, f) {
        return b;
    };
    /**
     * `alt` short for alternative, takes another `Option`. If this `Option` is a `Some` type then it will be returned, if
     * it is a `None` then it will return the next `Some` if it exist. If both are `None` then it will return `none`.
     *
     * @example
     * import { Option, some, none } from 'fp-ts/lib/Option'
     *
     * assert.deepStrictEqual(some(2).alt(some(4)), some(2))
     * const fa: Option<number> = none
     * assert.deepStrictEqual(fa.alt(some(4)), some(4))
     * @obsolete
     */
    None.prototype.alt = function (fa) {
        return fa;
    };
    /**
     * Lazy version of `alt`
     *
     * @example
     * import { some } from 'fp-ts/lib/Option'
     *
     * assert.deepStrictEqual(some(1).orElse(() => some(2)), some(1))
     *
     * @since 1.6.0
     * @obsolete
     */
    None.prototype.orElse = function (fa) {
        return fa();
    };
    /** @obsolete */
    None.prototype.extend = function (f) {
        return none;
    };
    /**
     * Applies a function to each case in the data structure
     *
     * @example
     * import { none, some } from 'fp-ts/lib/Option'
     *
     * assert.strictEqual(some(1).fold('none', a => `some: ${a}`), 'some: 1')
     * assert.strictEqual(none.fold('none', a => `some: ${a}`), 'none')
     * @obsolete
     */
    None.prototype.fold = function (b, onSome) {
        return b;
    };
    /**
     * Lazy version of `fold`
     * @obsolete
     */
    None.prototype.foldL = function (onNone, onSome) {
        return onNone();
    };
    /**
     * Returns the value from this `Some` or the given argument if this is a `None`
     *
     * @example
     * import { Option, none, some } from 'fp-ts/lib/Option'
     *
     * assert.strictEqual(some(1).getOrElse(0), 1)
     * const fa: Option<number> = none
     * assert.strictEqual(fa.getOrElse(0), 0)
     * @obsolete
     */
    None.prototype.getOrElse = function (a) {
        return a;
    };
    /**
     * Lazy version of `getOrElse`
     * @obsolete
     */
    None.prototype.getOrElseL = function (f) {
        return f();
    };
    /**
     * Returns the value from this `Some` or `null` if this is a `None`
     * @obsolete
     */
    None.prototype.toNullable = function () {
        return null;
    };
    /**
     * Returns the value from this `Some` or `undefined` if this is a `None`
     * @obsolete
     */
    None.prototype.toUndefined = function () {
        return undefined;
    };
    None.prototype.inspect = function () {
        return this.toString();
    };
    None.prototype.toString = function () {
        return 'none';
    };
    /**
     * Returns `true` if the option has an element that is equal (as determined by `S`) to `a`, `false` otherwise
     * @obsolete
     */
    None.prototype.contains = function (E, a) {
        return false;
    };
    /**
     * Returns `true` if the option is `None`, `false` otherwise
     * @obsolete
     */
    None.prototype.isNone = function () {
        return true;
    };
    /**
     * Returns `true` if the option is an instance of `Some`, `false` otherwise
     * @obsolete
     */
    None.prototype.isSome = function () {
        return false;
    };
    /**
     * Returns `true` if this option is non empty and the predicate `p` returns `true` when applied to this Option's value
     * @obsolete
     */
    None.prototype.exists = function (p) {
        return false;
    };
    None.prototype.filter = function (p) {
        return none;
    };
    /**
     * Use `filter` instead.
     * Returns this option refined as `Option<B>` if it is non empty and the `refinement` returns `true` when applied to
     * this Option's value. Otherwise returns `None`
     * @since 1.3.0
     * @deprecated
     */
    None.prototype.refine = function (refinement) {
        return none;
    };
    None.value = new None();
    return None;
}());
export { None };
/**
 * @since 1.0.0
 */
export var none = None.value;
var Some = /** @class */ (function () {
    function Some(value) {
        this.value = value;
        this._tag = 'Some';
    }
    Some.prototype.map = function (f) {
        return new Some(f(this.value));
    };
    Some.prototype.mapNullable = function (f) {
        return fromNullable(f(this.value));
    };
    Some.prototype.ap = function (fab) {
        return fab.isNone() ? none : new Some(fab.value(this.value));
    };
    Some.prototype.ap_ = function (fb) {
        return fb.ap(this);
    };
    Some.prototype.chain = function (f) {
        return f(this.value);
    };
    Some.prototype.reduce = function (b, f) {
        return f(b, this.value);
    };
    Some.prototype.alt = function (fa) {
        return this;
    };
    Some.prototype.orElse = function (fa) {
        return this;
    };
    Some.prototype.extend = function (f) {
        return new Some(f(this));
    };
    Some.prototype.fold = function (b, onSome) {
        return onSome(this.value);
    };
    Some.prototype.foldL = function (onNone, onSome) {
        return onSome(this.value);
    };
    Some.prototype.getOrElse = function (a) {
        return this.value;
    };
    Some.prototype.getOrElseL = function (f) {
        return this.value;
    };
    Some.prototype.toNullable = function () {
        return this.value;
    };
    Some.prototype.toUndefined = function () {
        return this.value;
    };
    Some.prototype.inspect = function () {
        return this.toString();
    };
    Some.prototype.toString = function () {
        // tslint:disable-next-line: deprecation
        return "some(" + toString(this.value) + ")";
    };
    Some.prototype.contains = function (E, a) {
        return E.equals(this.value, a);
    };
    Some.prototype.isNone = function () {
        return false;
    };
    Some.prototype.isSome = function () {
        return true;
    };
    Some.prototype.exists = function (p) {
        return p(this.value);
    };
    Some.prototype.filter = function (p) {
        return this.exists(p) ? this : none;
    };
    Some.prototype.refine = function (refinement) {
        return this.filter(refinement);
    };
    return Some;
}());
export { Some };
/**
 * @since 1.17.0
 */
export var getShow = function (S) {
    return {
        show: function (oa) { return oa.fold('none', function (a) { return "some(" + S.show(a) + ")"; }); }
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
 * @example
 * import { none, some, getEq } from 'fp-ts/lib/Option'
 * import { eqNumber } from 'fp-ts/lib/Eq'
 *
 * const S = getEq(eqNumber)
 * assert.strictEqual(S.equals(none, none), true)
 * assert.strictEqual(S.equals(none, some(1)), false)
 * assert.strictEqual(S.equals(some(1), none), false)
 * assert.strictEqual(S.equals(some(1), some(2)), false)
 * assert.strictEqual(S.equals(some(1), some(1)), true)
 *
 * @since 1.19.0
 */
export function getEq(E) {
    return fromEquals(function (x, y) { return (x.isNone() ? y.isNone() : y.isNone() ? false : E.equals(x.value, y.value)); });
}
/**
 * The `Ord` instance allows `Option` values to be compared with
 * `compare`, whenever there is an `Ord` instance for
 * the type the `Option` contains.
 *
 * `None` is considered to be less than any `Some` value.
 *
 *
 * @example
 * import { none, some, getOrd } from 'fp-ts/lib/Option'
 * import { ordNumber } from 'fp-ts/lib/Ord'
 *
 * const O = getOrd(ordNumber)
 * assert.strictEqual(O.compare(none, none), 0)
 * assert.strictEqual(O.compare(none, some(1)), -1)
 * assert.strictEqual(O.compare(some(1), none), 1)
 * assert.strictEqual(O.compare(some(1), some(2)), -1)
 * assert.strictEqual(O.compare(some(1), some(1)), 0)
 *
 * @since 1.2.0
 */
export var getOrd = function (O) {
    return fromCompare(function (x, y) { return (x.isSome() ? (y.isSome() ? O.compare(x.value, y.value) : 1) : -1); });
};
/**
 * @since 1.0.0
 */
export var some = function (a) {
    return new Some(a);
};
/**
 * `Apply` semigroup
 *
 * | x       | y       | concat(x, y)       |
 * | ------- | ------- | ------------------ |
 * | none    | none    | none               |
 * | some(a) | none    | none               |
 * | none    | some(a) | none               |
 * | some(a) | some(b) | some(concat(a, b)) |
 *
 * @example
 * import { getApplySemigroup, some, none } from 'fp-ts/lib/Option'
 * import { semigroupSum } from 'fp-ts/lib/Semigroup'
 *
 * const S = getApplySemigroup(semigroupSum)
 * assert.deepStrictEqual(S.concat(none, none), none)
 * assert.deepStrictEqual(S.concat(some(1), none), none)
 * assert.deepStrictEqual(S.concat(none, some(1)), none)
 * assert.deepStrictEqual(S.concat(some(1), some(2)), some(3))
 *
 * @since 1.7.0
 */
export var getApplySemigroup = function (S) {
    return {
        concat: function (x, y) { return (x.isSome() && y.isSome() ? some(S.concat(x.value, y.value)) : none); }
    };
};
/**
 * @since 1.7.0
 */
export var getApplyMonoid = function (M) {
    return __assign({}, getApplySemigroup(M), { empty: some(M.empty) });
};
/**
 * Monoid returning the left-most non-`None` value
 *
 * | x       | y       | concat(x, y) |
 * | ------- | ------- | ------------ |
 * | none    | none    | none         |
 * | some(a) | none    | some(a)      |
 * | none    | some(a) | some(a)      |
 * | some(a) | some(b) | some(a)      |
 *
 * @example
 * import { getFirstMonoid, some, none } from 'fp-ts/lib/Option'
 *
 * const M = getFirstMonoid<number>()
 * assert.deepStrictEqual(M.concat(none, none), none)
 * assert.deepStrictEqual(M.concat(some(1), none), some(1))
 * assert.deepStrictEqual(M.concat(none, some(1)), some(1))
 * assert.deepStrictEqual(M.concat(some(1), some(2)), some(1))
 *
 * @since 1.0.0
 */
export var getFirstMonoid = function () {
    return {
        concat: option.alt,
        empty: none
    };
};
/**
 * Monoid returning the right-most non-`None` value
 *
 * | x       | y       | concat(x, y) |
 * | ------- | ------- | ------------ |
 * | none    | none    | none         |
 * | some(a) | none    | some(a)      |
 * | none    | some(a) | some(a)      |
 * | some(a) | some(b) | some(b)      |
 *
 * @example
 * import { getLastMonoid, some, none } from 'fp-ts/lib/Option'
 *
 * const M = getLastMonoid<number>()
 * assert.deepStrictEqual(M.concat(none, none), none)
 * assert.deepStrictEqual(M.concat(some(1), none), some(1))
 * assert.deepStrictEqual(M.concat(none, some(1)), some(1))
 * assert.deepStrictEqual(M.concat(some(1), some(2)), some(2))
 *
 * @since 1.0.0
 */
export var getLastMonoid = function () {
    return getDualMonoid(getFirstMonoid());
};
/**
 * Monoid returning the left-most non-`None` value. If both operands are `Some`s then the inner values are
 * appended using the provided `Semigroup`
 *
 * | x       | y       | concat(x, y)       |
 * | ------- | ------- | ------------------ |
 * | none    | none    | none               |
 * | some(a) | none    | some(a)            |
 * | none    | some(a) | some(a)            |
 * | some(a) | some(b) | some(concat(a, b)) |
 *
 * @example
 * import { getMonoid, some, none } from 'fp-ts/lib/Option'
 * import { semigroupSum } from 'fp-ts/lib/Semigroup'
 *
 * const M = getMonoid(semigroupSum)
 * assert.deepStrictEqual(M.concat(none, none), none)
 * assert.deepStrictEqual(M.concat(some(1), none), some(1))
 * assert.deepStrictEqual(M.concat(none, some(1)), some(1))
 * assert.deepStrictEqual(M.concat(some(1), some(2)), some(3))
 *
 * @since 1.0.0
 */
export var getMonoid = function (S) {
    return {
        concat: function (x, y) { return (x.isNone() ? y : y.isNone() ? x : some(S.concat(x.value, y.value))); },
        empty: none
    };
};
/**
 * Constructs a new `Option` from a nullable type. If the value is `null` or `undefined`, returns `None`, otherwise
 * returns the value wrapped in a `Some`
 *
 * @example
 * import { none, some, fromNullable } from 'fp-ts/lib/Option'
 *
 * assert.deepStrictEqual(fromNullable(undefined), none)
 * assert.deepStrictEqual(fromNullable(null), none)
 * assert.deepStrictEqual(fromNullable(1), some(1))
 *
 * @since 1.0.0
 */
export var fromNullable = function (a) {
    return a == null ? none : new Some(a);
};
export function fromPredicate(predicate) {
    return function (a) { return (predicate(a) ? some(a) : none); };
}
/**
 * Transforms an exception into an `Option`. If `f` throws, returns `None`, otherwise returns the output wrapped in
 * `Some`
 *
 * @example
 * import { none, some, tryCatch } from 'fp-ts/lib/Option'
 *
 * assert.deepStrictEqual(
 *   tryCatch(() => {
 *     throw new Error()
 *   }),
 *   none
 * )
 * assert.deepStrictEqual(tryCatch(() => 1), some(1))
 *
 * @since 1.0.0
 */
export var tryCatch = function (f) {
    try {
        return some(f());
    }
    catch (e) {
        return none;
    }
};
/**
 * Constructs a new `Option` from a `Either`. If the value is a `Left`, returns `None`, otherwise returns the inner
 * value wrapped in a `Some`
 *
 * @example
 * import { none, some, fromEither } from 'fp-ts/lib/Option'
 * import { left, right } from 'fp-ts/lib/Either'
 *
 * assert.deepStrictEqual(fromEither(left(1)), none)
 * assert.deepStrictEqual(fromEither(right(1)), some(1))
 *
 * @since 1.0.0
 */
export var fromEither = function (fa) {
    return fa.isLeft() ? none : some(fa.value);
};
/**
 * Returns `true` if the option is an instance of `Some`, `false` otherwise
 *
 * @since 1.0.0
 */
export var isSome = function (fa) {
    return fa.isSome();
};
/**
 * Returns `true` if the option is `None`, `false` otherwise
 *
 * @since 1.0.0
 */
export var isNone = function (fa) {
    return fa.isNone();
};
/**
 * Use `fromPredicate` instead.
 * Refinement version of `fromPredicate`
 *
 * @since 1.3.0
 * @deprecated
 */
export var fromRefinement = function (refinement) { return function (a) {
    return refinement(a) ? some(a) : none;
}; };
/**
 * Returns a refinement from a prism.
 * This function ensures that a custom type guard definition is type-safe.
 *
 * ```ts
 * import { some, none, getRefinement } from 'fp-ts/lib/Option'
 *
 * type A = { type: 'A' }
 * type B = { type: 'B' }
 * type C = A | B
 *
 * const isA = (c: C): c is A => c.type === 'B' // <= typo but typescript doesn't complain
 * const isA = getRefinement<C, A>(c => (c.type === 'B' ? some(c) : none)) // static error: Type '"B"' is not assignable to type '"A"'
 * ```
 *
 * @since 1.7.0
 */
export var getRefinement = function (getOption) {
    return function (a) { return getOption(a).isSome(); };
};
var defaultSeparate = { left: none, right: none };
/**
 * @since 1.0.0
 */
export var option = {
    URI: URI,
    map: function (ma, f) { return (isNone(ma) ? none : some(f(ma.value))); },
    of: some,
    ap: function (mab, ma) { return (isNone(mab) ? none : isNone(ma) ? none : some(mab.value(ma.value))); },
    chain: function (ma, f) { return (isNone(ma) ? none : f(ma.value)); },
    reduce: function (fa, b, f) { return (isNone(fa) ? b : f(b, fa.value)); },
    foldMap: function (M) { return function (fa, f) { return (isNone(fa) ? M.empty : f(fa.value)); }; },
    foldr: function (fa, b, f) { return (isNone(fa) ? b : f(fa.value, b)); },
    traverse: function (F) { return function (ta, f) {
        return isNone(ta) ? F.of(none) : F.map(f(ta.value), some);
    }; },
    sequence: function (F) { return function (ta) {
        return isNone(ta) ? F.of(none) : F.map(ta.value, some);
    }; },
    zero: function () { return none; },
    alt: function (mx, my) { return (isNone(mx) ? my : mx); },
    extend: function (wa, f) { return (isNone(wa) ? none : some(f(wa))); },
    compact: function (ma) { return option.chain(ma, identity); },
    separate: function (ma) {
        var o = option.map(ma, function (e) { return ({
            left: getLeft(e),
            right: getRight(e)
        }); });
        return isNone(o) ? defaultSeparate : o.value;
    },
    filter: function (fa, predicate) {
        return isNone(fa) ? none : predicate(fa.value) ? fa : none;
    },
    filterMap: function (ma, f) { return (isNone(ma) ? none : f(ma.value)); },
    partition: function (fa, predicate) {
        return {
            left: option.filter(fa, function (a) { return !predicate(a); }),
            right: option.filter(fa, predicate)
        };
    },
    partitionMap: function (fa, f) { return option.separate(option.map(fa, f)); },
    wither: function (F) { return function (fa, f) {
        return isNone(fa) ? F.of(none) : f(fa.value);
    }; },
    wilt: function (F) { return function (fa, f) {
        var o = option.map(fa, function (a) {
            return F.map(f(a), function (e) { return ({
                left: getLeft(e),
                right: getRight(e)
            }); });
        });
        return isNone(o)
            ? F.of({
                left: none,
                right: none
            })
            : o.value;
    }; },
    throwError: function () { return none; },
    fromEither: fromEither,
    fromOption: identity
};
//
// backporting
//
/**
 * Returns an `L` value if possible
 *
 * @since 1.19.0
 */
export function getLeft(ma) {
    return ma._tag === 'Right' ? none : some(ma.value);
}
/**
 * Returns an `A` value if possible
 *
 * @since 1.19.0
 */
export function getRight(ma) {
    return ma._tag === 'Left' ? none : some(ma.value);
}
/**
 * @since 1.19.0
 */
export function fold(onNone, onSome) {
    return function (ma) { return ma.foldL(onNone, onSome); };
}
/**
 * @since 1.19.0
 */
export function toNullable(ma) {
    return ma.toNullable();
}
/**
 * @since 1.19.0
 */
export function toUndefined(ma) {
    return ma.toUndefined();
}
/**
 * @since 1.19.0
 */
export function getOrElse(f) {
    return function (ma) { return ma.getOrElseL(f); };
}
/**
 * @since 1.19.0
 */
export function elem(E) {
    return function (a) { return function (ma) { return ma.contains(E, a); }; };
}
/**
 * @since 1.19.0
 */
export function exists(predicate) {
    return function (ma) { return ma.exists(predicate); };
}
/**
 * @since 1.19.0
 */
export function mapNullable(f) {
    return function (ma) { return ma.mapNullable(f); };
}
var _a = pipeable(option), alt = _a.alt, ap = _a.ap, apFirst = _a.apFirst, apSecond = _a.apSecond, chain = _a.chain, chainFirst = _a.chainFirst, duplicate = _a.duplicate, extend = _a.extend, filter = _a.filter, filterMap = _a.filterMap, flatten = _a.flatten, foldMap = _a.foldMap, map = _a.map, partition = _a.partition, partitionMap = _a.partitionMap, reduce = _a.reduce, reduceRight = _a.reduceRight, compact = _a.compact, separate = _a.separate;
export { alt, ap, apFirst, apSecond, chain, chainFirst, duplicate, extend, filter, filterMap, flatten, foldMap, map, partition, partitionMap, reduce, reduceRight, compact, separate };
