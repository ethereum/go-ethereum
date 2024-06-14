"use strict";
/**
 * @file Represents a value of one of two possible types (a disjoint union).
 *
 * An instance of `Either` is either an instance of `Left` or `Right`.
 *
 * A common use of `Either` is as an alternative to `Option` for dealing with possible missing values. In this usage,
 * `None` is replaced with a `Left` which can contain useful information. `Right` takes the place of `Some`. Convention
 * dictates that `Left` is used for failure and `Right` is used for success.
 *
 * For example, you could use `Either<string, number>` to detect whether a received input is a `string` or a `number`.
 *
 * ```ts
 * const parse = (errorMessage: string) => (input: string): Either<string, number> => {
 *   const n = parseInt(input, 10)
 *   return isNaN(n) ? left(errorMessage) : right(n)
 * }
 * ```
 *
 * `Either` is right-biased, which means that `Right` is assumed to be the default case to operate on. If it is `Left`,
 * operations like `map`, `chain`, ... return the `Left` value unchanged:
 *
 * ```ts
 * right(12).map(double) // right(24)
 * left(23).map(double)  // left(23)
 * ```
 */
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
Object.defineProperty(exports, "__esModule", { value: true });
var ChainRec_1 = require("./ChainRec");
var function_1 = require("./function");
var Eq_1 = require("./Eq");
var pipeable_1 = require("./pipeable");
exports.URI = 'Either';
/**
 * Left side of `Either`
 */
var Left = /** @class */ (function () {
    function Left(value) {
        this.value = value;
        this._tag = 'Left';
    }
    /**
     * The given function is applied if this is a `Right`
     * @obsolete
     */
    Left.prototype.map = function (f) {
        return this;
    };
    /** @obsolete */
    Left.prototype.ap = function (fab) {
        return (fab.isLeft() ? fab : this);
    };
    /**
     * Flipped version of `ap`
     * @obsolete
     */
    Left.prototype.ap_ = function (fb) {
        return fb.ap(this);
    };
    /**
     * Binds the given function across `Right`
     * @obsolete
     */
    Left.prototype.chain = function (f) {
        return this;
    };
    /** @obsolete */
    Left.prototype.bimap = function (f, g) {
        return new Left(f(this.value));
    };
    /** @obsolete */
    Left.prototype.alt = function (fy) {
        return fy;
    };
    /**
     * Lazy version of `alt`
     *
     * @example
     * import { right } from 'fp-ts/lib/Either'
     *
     * assert.deepStrictEqual(right(1).orElse(() => right(2)), right(1))
     *
     * @since 1.6.0
     * @obsolete
     */
    Left.prototype.orElse = function (fy) {
        return fy(this.value);
    };
    /** @obsolete */
    Left.prototype.extend = function (f) {
        return this;
    };
    /** @obsolete */
    Left.prototype.reduce = function (b, f) {
        return b;
    };
    /**
     * Applies a function to each case in the data structure
     * @obsolete
     */
    Left.prototype.fold = function (onLeft, onRight) {
        return onLeft(this.value);
    };
    /**
     * Returns the value from this `Right` or the given argument if this is a `Left`
     * @obsolete
     */
    Left.prototype.getOrElse = function (a) {
        return a;
    };
    /**
     * Returns the value from this `Right` or the result of given argument if this is a `Left`
     * @obsolete
     */
    Left.prototype.getOrElseL = function (f) {
        return f(this.value);
    };
    /**
     * Maps the left side of the disjunction
     * @obsolete
     */
    Left.prototype.mapLeft = function (f) {
        return new Left(f(this.value));
    };
    Left.prototype.inspect = function () {
        return this.toString();
    };
    Left.prototype.toString = function () {
        // tslint:disable-next-line: deprecation
        return "left(" + function_1.toString(this.value) + ")";
    };
    /**
     * Returns `true` if the either is an instance of `Left`, `false` otherwise
     * @obsolete
     */
    Left.prototype.isLeft = function () {
        return true;
    };
    /**
     * Returns `true` if the either is an instance of `Right`, `false` otherwise
     * @obsolete
     */
    Left.prototype.isRight = function () {
        return false;
    };
    /**
     * Swaps the disjunction values
     * @obsolete
     */
    Left.prototype.swap = function () {
        return new Right(this.value);
    };
    Left.prototype.filterOrElse = function (_, zero) {
        return this;
    };
    Left.prototype.filterOrElseL = function (_, zero) {
        return this;
    };
    /**
     * Use `filterOrElse` instead
     * @since 1.6.0
     * @deprecated
     */
    Left.prototype.refineOrElse = function (p, zero) {
        return this;
    };
    /**
     * Lazy version of `refineOrElse`
     * Use `filterOrElseL` instead
     * @since 1.6.0
     * @deprecated
     */
    Left.prototype.refineOrElseL = function (p, zero) {
        return this;
    };
    return Left;
}());
exports.Left = Left;
/**
 * Right side of `Either`
 */
var Right = /** @class */ (function () {
    function Right(value) {
        this.value = value;
        this._tag = 'Right';
    }
    Right.prototype.map = function (f) {
        return new Right(f(this.value));
    };
    Right.prototype.ap = function (fab) {
        return fab.isRight() ? this.map(fab.value) : exports.left(fab.value);
    };
    Right.prototype.ap_ = function (fb) {
        return fb.ap(this);
    };
    Right.prototype.chain = function (f) {
        return f(this.value);
    };
    Right.prototype.bimap = function (f, g) {
        return new Right(g(this.value));
    };
    Right.prototype.alt = function (fy) {
        return this;
    };
    Right.prototype.orElse = function (fy) {
        return this;
    };
    Right.prototype.extend = function (f) {
        return new Right(f(this));
    };
    Right.prototype.reduce = function (b, f) {
        return f(b, this.value);
    };
    Right.prototype.fold = function (onLeft, onRight) {
        return onRight(this.value);
    };
    Right.prototype.getOrElse = function (a) {
        return this.value;
    };
    Right.prototype.getOrElseL = function (f) {
        return this.value;
    };
    Right.prototype.mapLeft = function (f) {
        return new Right(this.value);
    };
    Right.prototype.inspect = function () {
        return this.toString();
    };
    Right.prototype.toString = function () {
        // tslint:disable-next-line: deprecation
        return "right(" + function_1.toString(this.value) + ")";
    };
    Right.prototype.isLeft = function () {
        return false;
    };
    Right.prototype.isRight = function () {
        return true;
    };
    Right.prototype.swap = function () {
        return new Left(this.value);
    };
    Right.prototype.filterOrElse = function (p, zero) {
        return p(this.value) ? this : exports.left(zero);
    };
    Right.prototype.filterOrElseL = function (p, zero) {
        return p(this.value) ? this : exports.left(zero(this.value));
    };
    Right.prototype.refineOrElse = function (p, zero) {
        return p(this.value) ? this : exports.left(zero);
    };
    Right.prototype.refineOrElseL = function (p, zero) {
        return p(this.value) ? this : exports.left(zero(this.value));
    };
    return Right;
}());
exports.Right = Right;
/**
 * @since 1.17.0
 */
exports.getShow = function (SL, SA) {
    return {
        show: function (e) { return e.fold(function (l) { return "left(" + SL.show(l) + ")"; }, function (a) { return "right(" + SA.show(a) + ")"; }); }
    };
};
/**
 * Use `getEq`
 *
 * @since 1.0.0
 * @deprecated
 */
exports.getSetoid = getEq;
/**
 * @since 1.19.0
 */
function getEq(EL, EA) {
    return Eq_1.fromEquals(function (x, y) {
        return x.isLeft() ? y.isLeft() && EL.equals(x.value, y.value) : y.isRight() && EA.equals(x.value, y.value);
    });
}
exports.getEq = getEq;
/**
 * Semigroup returning the left-most non-`Left` value. If both operands are `Right`s then the inner values are
 * appended using the provided `Semigroup`
 *
 * @example
 * import { getSemigroup, left, right } from 'fp-ts/lib/Either'
 * import { semigroupSum } from 'fp-ts/lib/Semigroup'
 *
 * const S = getSemigroup<string, number>(semigroupSum)
 * assert.deepStrictEqual(S.concat(left('a'), left('b')), left('a'))
 * assert.deepStrictEqual(S.concat(left('a'), right(2)), right(2))
 * assert.deepStrictEqual(S.concat(right(1), left('b')), right(1))
 * assert.deepStrictEqual(S.concat(right(1), right(2)), right(3))
 *
 *
 * @since 1.7.0
 */
exports.getSemigroup = function (S) {
    return {
        concat: function (x, y) { return (y.isLeft() ? x : x.isLeft() ? y : exports.right(S.concat(x.value, y.value))); }
    };
};
/**
 * `Apply` semigroup
 *
 * @example
 * import { getApplySemigroup, left, right } from 'fp-ts/lib/Either'
 * import { semigroupSum } from 'fp-ts/lib/Semigroup'
 *
 * const S = getApplySemigroup<string, number>(semigroupSum)
 * assert.deepStrictEqual(S.concat(left('a'), left('b')), left('a'))
 * assert.deepStrictEqual(S.concat(left('a'), right(2)), left('a'))
 * assert.deepStrictEqual(S.concat(right(1), left('b')), left('b'))
 * assert.deepStrictEqual(S.concat(right(1), right(2)), right(3))
 *
 *
 * @since 1.7.0
 */
exports.getApplySemigroup = function (S) {
    return {
        concat: function (x, y) { return (x.isLeft() ? x : y.isLeft() ? y : exports.right(S.concat(x.value, y.value))); }
    };
};
/**
 * @since 1.7.0
 */
exports.getApplyMonoid = function (M) {
    return __assign({}, exports.getApplySemigroup(M), { empty: exports.right(M.empty) });
};
/**
 * Constructs a new `Either` holding a `Left` value. This usually represents a failure, due to the right-bias of this
 * structure
 *
 * @since 1.0.0
 */
exports.left = function (l) {
    return new Left(l);
};
/**
 * Constructs a new `Either` holding a `Right` value. This usually represents a successful value due to the right bias
 * of this structure
 *
 * @since 1.0.0
 */
exports.right = function (a) {
    return new Right(a);
};
/**
 * Use `fromPredicate` instead
 *
 * @since 1.6.0
 * @deprecated
 */
exports.fromRefinement = function (refinement, onFalse) { return function (a) {
    return refinement(a) ? exports.right(a) : exports.left(onFalse(a));
}; };
/**
 * Takes a default and a `Option` value, if the value is a `Some`, turn it into a `Right`, if the value is a `None` use
 * the provided default as a `Left`
 *
 * @since 1.0.0
 */
exports.fromOption = function (onNone) { return function (fa) {
    return fa.isNone() ? exports.left(onNone) : exports.right(fa.value);
}; };
/**
 * Takes a default and a nullable value, if the value is not nully, turn it into a `Right`, if the value is nully use
 * the provided default as a `Left`
 *
 * @since 1.0.0
 */
exports.fromNullable = function (defaultValue) { return function (a) {
    return a == null ? exports.left(defaultValue) : exports.right(a);
}; };
/**
 * Default value for the optional `onerror` argument of `tryCatch`
 *
 * @since 1.0.0
 */
exports.toError = function (e) {
    if (e instanceof Error) {
        return e;
    }
    else {
        return new Error(String(e));
    }
};
/**
 * Use `tryCatch2v` instead
 *
 * @since 1.0.0
 * @deprecated
 */
exports.tryCatch = function (f, onerror) {
    if (onerror === void 0) { onerror = exports.toError; }
    return exports.tryCatch2v(f, onerror);
};
/**
 * Constructs a new `Either` from a function that might throw
 *
 * @example
 * import { Either, left, right, tryCatch2v } from 'fp-ts/lib/Either'
 *
 * const unsafeHead = <A>(as: Array<A>): A => {
 *   if (as.length > 0) {
 *     return as[0]
 *   } else {
 *     throw new Error('empty array')
 *   }
 * }
 *
 * const head = <A>(as: Array<A>): Either<Error, A> => {
 *   return tryCatch2v(() => unsafeHead(as), e => (e instanceof Error ? e : new Error('unknown error')))
 * }
 *
 * assert.deepStrictEqual(head([]), left(new Error('empty array')))
 * assert.deepStrictEqual(head([1, 2, 3]), right(1))
 *
 * @since 1.11.0
 */
exports.tryCatch2v = function (f, onerror) {
    try {
        return exports.right(f());
    }
    catch (e) {
        return exports.left(onerror(e));
    }
};
/**
 * @since 1.0.0
 */
exports.fromValidation = function (fa) {
    return fa.isFailure() ? exports.left(fa.value) : exports.right(fa.value);
};
/**
 * Returns `true` if the either is an instance of `Left`, `false` otherwise
 *
 * @since 1.0.0
 */
exports.isLeft = function (fa) {
    return fa.isLeft();
};
/**
 * Returns `true` if the either is an instance of `Right`, `false` otherwise
 *
 * @since 1.0.0
 */
exports.isRight = function (fa) {
    return fa.isRight();
};
/**
 * Use `getWitherable`
 *
 * @since 1.7.0
 * @deprecated
 */
function getCompactable(ML) {
    var compact = function (fa) {
        if (fa.isLeft()) {
            return fa;
        }
        if (fa.value.isNone()) {
            return exports.left(ML.empty);
        }
        return exports.right(fa.value.value);
    };
    var separate = function (fa) {
        if (fa.isLeft()) {
            return {
                left: fa,
                right: fa
            };
        }
        if (fa.value.isLeft()) {
            return {
                left: exports.right(fa.value.value),
                right: exports.left(ML.empty)
            };
        }
        return {
            left: exports.left(ML.empty),
            right: exports.right(fa.value.value)
        };
    };
    return {
        URI: exports.URI,
        _L: undefined,
        compact: compact,
        separate: separate
    };
}
exports.getCompactable = getCompactable;
/**
 * Use `getWitherable`
 *
 * @since 1.7.0
 * @deprecated
 */
function getFilterable(ML) {
    // tslint:disable-next-line: deprecation
    var C = getCompactable(ML);
    var partitionMap = function (fa, f) {
        if (fa.isLeft()) {
            return {
                left: fa,
                right: fa
            };
        }
        var e = f(fa.value);
        if (e.isLeft()) {
            return {
                left: exports.right(e.value),
                right: exports.left(ML.empty)
            };
        }
        return {
            left: exports.left(ML.empty),
            right: exports.right(e.value)
        };
    };
    var partition = function (fa, p) {
        if (fa.isLeft()) {
            return {
                left: fa,
                right: fa
            };
        }
        if (p(fa.value)) {
            return {
                left: exports.left(ML.empty),
                right: exports.right(fa.value)
            };
        }
        return {
            left: exports.right(fa.value),
            right: exports.left(ML.empty)
        };
    };
    var filterMap = function (fa, f) {
        if (fa.isLeft()) {
            return fa;
        }
        var optionB = f(fa.value);
        if (optionB.isSome()) {
            return exports.right(optionB.value);
        }
        return exports.left(ML.empty);
    };
    var filter = function (fa, p) { return fa.filterOrElse(p, ML.empty); };
    return __assign({}, C, { map: exports.either.map, partitionMap: partitionMap,
        filterMap: filterMap,
        partition: partition,
        filter: filter });
}
exports.getFilterable = getFilterable;
/**
 * Builds `Witherable` instance for `Either` given `Monoid` for the left side
 *
 * @since 1.7.0
 */
function getWitherable(ML) {
    // tslint:disable-next-line: deprecation
    var filterableEither = getFilterable(ML);
    var wither = function (F) {
        var traverseF = exports.either.traverse(F);
        return function (wa, f) { return F.map(traverseF(wa, f), filterableEither.compact); };
    };
    var wilt = function (F) {
        var traverseF = exports.either.traverse(F);
        return function (wa, f) { return F.map(traverseF(wa, f), filterableEither.separate); };
    };
    return __assign({}, filterableEither, { traverse: exports.either.traverse, reduce: exports.either.reduce, wither: wither,
        wilt: wilt });
}
exports.getWitherable = getWitherable;
/**
 * Converts a JavaScript Object Notation (JSON) string into an object.
 *
 * @example
 * import { parseJSON, toError } from 'fp-ts/lib/Either'
 *
 * assert.deepStrictEqual(parseJSON('{"a":1}', toError).value, { a: 1 })
 * assert.deepStrictEqual(parseJSON('{"a":}', toError).value, new SyntaxError('Unexpected token } in JSON at position 5'))
 *
 * @since 1.16.0
 */
exports.parseJSON = function (s, onError) {
    return exports.tryCatch2v(function () { return JSON.parse(s); }, onError);
};
/**
 * Converts a JavaScript value to a JavaScript Object Notation (JSON) string.
 *
 * @example
 * import { stringifyJSON, toError } from 'fp-ts/lib/Either'
 *
 * assert.deepStrictEqual(stringifyJSON({ a: 1 }, toError).value, '{"a":1}')
 * const circular: any = { ref: null }
 * circular.ref = circular
 * assert.deepStrictEqual(stringifyJSON(circular, toError).value, new TypeError('Converting circular structure to JSON'))
 *
 * @since 1.16.0
 */
exports.stringifyJSON = function (u, onError) {
    return exports.tryCatch2v(function () { return JSON.stringify(u); }, onError);
};
var throwError = exports.left;
var fromEither = function_1.identity;
/**
 * @since 1.0.0
 */
exports.either = {
    URI: exports.URI,
    map: function (ma, f) { return ma.map(f); },
    of: exports.right,
    ap: function (mab, ma) { return ma.ap(mab); },
    chain: function (ma, f) { return ma.chain(f); },
    reduce: function (fa, b, f) { return fa.reduce(b, f); },
    foldMap: function (M) { return function (fa, f) { return (fa.isLeft() ? M.empty : f(fa.value)); }; },
    foldr: function (fa, b, f) { return (fa.isLeft() ? b : f(fa.value, b)); },
    traverse: function (F) { return function (ta, f) {
        return ta.isLeft() ? F.of(exports.left(ta.value)) : F.map(f(ta.value), exports.right);
    }; },
    sequence: function (F) { return function (ta) {
        return ta.isLeft() ? F.of(exports.left(ta.value)) : F.map(ta.value, exports.right);
    }; },
    bimap: function (fla, f, g) { return fla.bimap(f, g); },
    alt: function (mx, my) { return mx.alt(my); },
    extend: function (wa, f) { return wa.extend(f); },
    chainRec: function (a, f) {
        return ChainRec_1.tailRec(function (e) {
            if (e.isLeft()) {
                return exports.right(exports.left(e.value));
            }
            else {
                var r = e.value;
                return r.isLeft() ? exports.left(f(r.value)) : exports.right(exports.right(r.value));
            }
        }, f(a));
    },
    throwError: throwError,
    fromEither: fromEither,
    fromOption: function (o, e) { return (o.isNone() ? throwError(e) : exports.right(o.value)); }
};
//
// backporting
//
/**
 * @since 1.19.0
 */
function fold(onLeft, onRight) {
    return function (ma) { return ma.fold(onLeft, onRight); };
}
exports.fold = fold;
/**
 * @since 1.19.0
 */
function orElse(f) {
    return function (ma) { return ma.orElse(f); };
}
exports.orElse = orElse;
/**
 * @since 1.19.0
 */
function getOrElse(f) {
    return function (ma) { return ma.getOrElseL(f); };
}
exports.getOrElse = getOrElse;
/**
 * @since 1.19.0
 */
function elem(E) {
    return function (a) { return function (ma) { return (exports.isLeft(ma) ? false : E.equals(a, ma.value)); }; };
}
exports.elem = elem;
/**
 * @since 1.19.0
 */
function getValidation(S) {
    return {
        URI: exports.URI,
        _L: undefined,
        map: exports.either.map,
        of: exports.either.of,
        ap: function (mab, ma) {
            return exports.isLeft(mab)
                ? exports.isLeft(ma)
                    ? exports.left(S.concat(mab.value, ma.value))
                    : mab
                : exports.isLeft(ma)
                    ? ma
                    : exports.right(mab.value(ma.value));
        },
        chain: exports.either.chain,
        alt: function (fx, fy) {
            if (exports.isRight(fx)) {
                return fx;
            }
            return exports.isLeft(fy) ? exports.left(S.concat(fx.value, fy.value)) : fy;
        }
    };
}
exports.getValidation = getValidation;
/**
 * @since 1.19.0
 */
function getValidationSemigroup(SE, SA) {
    return {
        concat: function (fx, fy) {
            return exports.isLeft(fx)
                ? exports.isLeft(fy)
                    ? exports.left(SE.concat(fx.value, fy.value))
                    : fx
                : exports.isLeft(fy)
                    ? fy
                    : exports.right(SA.concat(fx.value, fy.value));
        }
    };
}
exports.getValidationSemigroup = getValidationSemigroup;
/**
 * @since 1.19.0
 */
function getValidationMonoid(SE, SA) {
    return {
        concat: getValidationSemigroup(SE, SA).concat,
        empty: exports.right(SA.empty)
    };
}
exports.getValidationMonoid = getValidationMonoid;
var _a = pipeable_1.pipeable(exports.either), alt = _a.alt, ap = _a.ap, apFirst = _a.apFirst, apSecond = _a.apSecond, bimap = _a.bimap, chain = _a.chain, chainFirst = _a.chainFirst, duplicate = _a.duplicate, extend = _a.extend, flatten = _a.flatten, foldMap = _a.foldMap, map = _a.map, mapLeft = _a.mapLeft, reduce = _a.reduce, reduceRight = _a.reduceRight, fromPredicate = _a.fromPredicate, filterOrElse = _a.filterOrElse, pipeableFromOption = _a.fromOption;
exports.alt = alt;
exports.ap = ap;
exports.apFirst = apFirst;
exports.apSecond = apSecond;
exports.bimap = bimap;
exports.chain = chain;
exports.chainFirst = chainFirst;
exports.duplicate = duplicate;
exports.extend = extend;
exports.flatten = flatten;
exports.foldMap = foldMap;
exports.map = map;
exports.mapLeft = mapLeft;
exports.reduce = reduce;
exports.reduceRight = reduceRight;
exports.fromPredicate = fromPredicate;
exports.filterOrElse = filterOrElse;
/**
 * Lazy version of `fromOption`
 *
 * @since 1.3.0
 */
exports.fromOptionL = pipeableFromOption;
