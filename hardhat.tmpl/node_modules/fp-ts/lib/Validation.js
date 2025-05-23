"use strict";
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
var function_1 = require("./function");
var Eq_1 = require("./Eq");
exports.URI = 'Validation';
var Failure = /** @class */ (function () {
    function Failure(value) {
        this.value = value;
        this._tag = 'Failure';
    }
    /** @obsolete */
    Failure.prototype.map = function (f) {
        return this;
    };
    /** @obsolete */
    Failure.prototype.bimap = function (f, g) {
        return new Failure(f(this.value));
    };
    /** @obsolete */
    Failure.prototype.reduce = function (b, f) {
        return b;
    };
    /** @obsolete */
    Failure.prototype.fold = function (failure, success) {
        return failure(this.value);
    };
    /**
     * Returns the value from this `Success` or the given argument if this is a `Failure`
     * @obsolete
     */
    Failure.prototype.getOrElse = function (a) {
        return a;
    };
    /**
     * Returns the value from this `Success` or the result of given argument if this is a `Failure`
     * @obsolete
     */
    Failure.prototype.getOrElseL = function (f) {
        return f(this.value);
    };
    /** @obsolete */
    Failure.prototype.mapFailure = function (f) {
        return new Failure(f(this.value));
    };
    /** @obsolete */
    Failure.prototype.swap = function () {
        return new Success(this.value);
    };
    Failure.prototype.inspect = function () {
        return this.toString();
    };
    Failure.prototype.toString = function () {
        // tslint:disable-next-line: deprecation
        return "failure(" + function_1.toString(this.value) + ")";
    };
    /**
     * Returns `true` if the validation is an instance of `Failure`, `false` otherwise
     * @obsolete
     */
    Failure.prototype.isFailure = function () {
        return true;
    };
    /**
     * Returns `true` if the validation is an instance of `Success`, `false` otherwise
     * @obsolete
     */
    Failure.prototype.isSuccess = function () {
        return false;
    };
    return Failure;
}());
exports.Failure = Failure;
var Success = /** @class */ (function () {
    function Success(value) {
        this.value = value;
        this._tag = 'Success';
    }
    Success.prototype.map = function (f) {
        return new Success(f(this.value));
    };
    Success.prototype.bimap = function (f, g) {
        return new Success(g(this.value));
    };
    Success.prototype.reduce = function (b, f) {
        return f(b, this.value);
    };
    Success.prototype.fold = function (failure, success) {
        return success(this.value);
    };
    Success.prototype.getOrElse = function (a) {
        return this.value;
    };
    Success.prototype.getOrElseL = function (f) {
        return this.value;
    };
    Success.prototype.mapFailure = function (f) {
        return this;
    };
    Success.prototype.swap = function () {
        return new Failure(this.value);
    };
    Success.prototype.inspect = function () {
        return this.toString();
    };
    Success.prototype.toString = function () {
        // tslint:disable-next-line: deprecation
        return "success(" + function_1.toString(this.value) + ")";
    };
    Success.prototype.isFailure = function () {
        return false;
    };
    Success.prototype.isSuccess = function () {
        return true;
    };
    return Success;
}());
exports.Success = Success;
/**
 * @since 1.17.0
 */
exports.getShow = function (SL, SA) {
    return {
        show: function (e) { return e.fold(function (l) { return "failure(" + SL.show(l) + ")"; }, function (a) { return "success(" + SA.show(a) + ")"; }); }
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
        return x.isFailure() ? y.isFailure() && EL.equals(x.value, y.value) : y.isSuccess() && EA.equals(x.value, y.value);
    });
}
exports.getEq = getEq;
var map = function (fa, f) {
    return fa.map(f);
};
/**
 * @since 1.0.0
 */
exports.success = function (a) {
    return new Success(a);
};
var of = exports.success;
/**
 * @example
 * import { Validation, success, failure, getApplicative } from 'fp-ts/lib/Validation'
 * import { getArraySemigroup } from 'fp-ts/lib/Semigroup'
 *
 * interface Person {
 *   name: string
 *   age: number
 * }
 *
 * const person = (name: string) => (age: number): Person => ({ name, age })
 *
 * const validateName = (name: string): Validation<string[], string> =>
 *   name.length === 0 ? failure(['invalid name']) : success(name)
 *
 * const validateAge = (age: number): Validation<string[], number> =>
 *   age > 0 && age % 1 === 0 ? success(age) : failure(['invalid age'])
 *
 * const A = getApplicative(getArraySemigroup<string>())
 *
 * const validatePerson = (name: string, age: number): Validation<string[], Person> =>
 *   A.ap(A.map(validateName(name), person), validateAge(age))
 *
 * assert.deepStrictEqual(validatePerson('Nicolas Bourbaki', 45), success({ "name": "Nicolas Bourbaki", "age": 45 }))
 * assert.deepStrictEqual(validatePerson('Nicolas Bourbaki', -1), failure(["invalid age"]))
 * assert.deepStrictEqual(validatePerson('', 0), failure(["invalid name", "invalid age"]))
 *
 * @since 1.0.0
 */
exports.getApplicative = function (S) {
    var ap = function (fab, fa) {
        return fab.isFailure()
            ? fa.isFailure()
                ? exports.failure(S.concat(fab.value, fa.value))
                : exports.failure(fab.value)
            : fa.isFailure()
                ? exports.failure(fa.value)
                : exports.success(fab.value(fa.value));
    };
    return {
        URI: exports.URI,
        _L: undefined,
        map: map,
        of: of,
        ap: ap
    };
};
/**
 * **Note**: This function is here just to avoid switching to / from `Either`
 *
 * @since 1.0.0
 */
exports.getMonad = function (S) {
    var chain = function (fa, f) {
        return fa.isFailure() ? exports.failure(fa.value) : f(fa.value);
    };
    return __assign({}, exports.getApplicative(S), { chain: chain });
};
var reduce = function (fa, b, f) {
    return fa.reduce(b, f);
};
var foldMap = function (M) { return function (fa, f) {
    return fa.isFailure() ? M.empty : f(fa.value);
}; };
var foldr = function (fa, b, f) {
    return fa.isFailure() ? b : f(fa.value, b);
};
var traverse = function (F) { return function (ta, f) {
    return ta.isFailure() ? F.of(exports.failure(ta.value)) : F.map(f(ta.value), of);
}; };
var sequence = function (F) { return function (ta) {
    return ta.isFailure() ? F.of(exports.failure(ta.value)) : F.map(ta.value, of);
}; };
var bimap = function (fla, f, g) {
    return fla.bimap(f, g);
};
/**
 * @since 1.0.0
 */
exports.failure = function (l) {
    return new Failure(l);
};
function fromPredicate(predicate, f) {
    return function (a) { return (predicate(a) ? exports.success(a) : exports.failure(f(a))); };
}
exports.fromPredicate = fromPredicate;
/**
 * @since 1.0.0
 */
exports.fromEither = function (e) {
    return e.isLeft() ? exports.failure(e.value) : exports.success(e.value);
};
/**
 * Constructs a new `Validation` from a function that might throw
 *
 * @example
 * import { Validation, failure, success, tryCatch } from 'fp-ts/lib/Validation'
 *
 * const unsafeHead = <A>(as: Array<A>): A => {
 *   if (as.length > 0) {
 *     return as[0]
 *   } else {
 *     throw new Error('empty array')
 *   }
 * }
 *
 * const head = <A>(as: Array<A>): Validation<Error, A> => {
 *   return tryCatch(() => unsafeHead(as), e => (e instanceof Error ? e : new Error('unknown error')))
 * }
 *
 * assert.deepStrictEqual(head([]), failure(new Error('empty array')))
 * assert.deepStrictEqual(head([1, 2, 3]), success(1))
 *
 * @since 1.16.0
 */
exports.tryCatch = function (f, onError) {
    try {
        return exports.success(f());
    }
    catch (e) {
        return exports.failure(onError(e));
    }
};
/**
 * @since 1.0.0
 */
exports.getSemigroup = function (SL, SA) {
    var concat = function (fx, fy) {
        return fx.isFailure()
            ? fy.isFailure()
                ? exports.failure(SL.concat(fx.value, fy.value))
                : exports.failure(fx.value)
            : fy.isFailure()
                ? exports.failure(fy.value)
                : exports.success(SA.concat(fx.value, fy.value));
    };
    return {
        concat: concat
    };
};
/**
 * @since 1.0.0
 */
exports.getMonoid = function (SL, SA) {
    return __assign({}, exports.getSemigroup(SL, SA), { empty: exports.success(SA.empty) });
};
/**
 * @since 1.0.0
 */
exports.getAlt = function (S) {
    var alt = function (fx, fy) {
        return fx.isFailure() ? (fy.isFailure() ? exports.failure(S.concat(fx.value, fy.value)) : fy) : fx;
    };
    return {
        URI: exports.URI,
        _L: undefined,
        map: map,
        alt: alt
    };
};
/**
 * Returns `true` if the validation is an instance of `Failure`, `false` otherwise
 *
 * @since 1.0.0
 */
exports.isFailure = function (fa) {
    return fa.isFailure();
};
/**
 * Returns `true` if the validation is an instance of `Success`, `false` otherwise
 *
 * @since 1.0.0
 */
exports.isSuccess = function (fa) {
    return fa.isSuccess();
};
/**
 * Builds `Compactable` instance for `Validation` given `Monoid` for the failure side
 *
 * @since 1.7.0
 */
function getCompactable(ML) {
    var compact = function (fa) {
        if (fa.isFailure()) {
            return fa;
        }
        if (fa.value.isNone()) {
            return exports.failure(ML.empty);
        }
        return exports.success(fa.value.value);
    };
    var separate = function (fa) {
        if (fa.isFailure()) {
            return {
                left: fa,
                right: fa
            };
        }
        if (fa.value.isLeft()) {
            return {
                left: exports.success(fa.value.value),
                right: exports.failure(ML.empty)
            };
        }
        return {
            left: exports.failure(ML.empty),
            right: exports.success(fa.value.value)
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
 * Builds `Filterable` instance for `Validation` given `Monoid` for the left side
 *
 * @since 1.7.0
 */
function getFilterable(ML) {
    var C = getCompactable(ML);
    var partitionMap = function (fa, f) {
        if (fa.isFailure()) {
            return {
                left: fa,
                right: fa
            };
        }
        var e = f(fa.value);
        if (e.isLeft()) {
            return {
                left: exports.success(e.value),
                right: exports.failure(ML.empty)
            };
        }
        return {
            left: exports.failure(ML.empty),
            right: exports.success(e.value)
        };
    };
    var partition = function (fa, p) {
        if (fa.isFailure()) {
            return {
                left: fa,
                right: fa
            };
        }
        if (p(fa.value)) {
            return {
                left: exports.failure(ML.empty),
                right: exports.success(fa.value)
            };
        }
        return {
            left: exports.success(fa.value),
            right: exports.failure(ML.empty)
        };
    };
    var filterMap = function (fa, f) {
        if (fa.isFailure()) {
            return fa;
        }
        var optionB = f(fa.value);
        if (optionB.isSome()) {
            return exports.success(optionB.value);
        }
        return exports.failure(ML.empty);
    };
    var filter = function (fa, p) {
        if (fa.isFailure()) {
            return fa;
        }
        var a = fa.value;
        if (p(a)) {
            return exports.success(a);
        }
        return exports.failure(ML.empty);
    };
    return __assign({}, C, { map: map,
        partitionMap: partitionMap,
        filterMap: filterMap,
        partition: partition,
        filter: filter });
}
exports.getFilterable = getFilterable;
/**
 * Builds `Witherable` instance for `Validation` given `Monoid` for the left side
 *
 * @since 1.7.0
 */
function getWitherable(ML) {
    var filterableValidation = getFilterable(ML);
    var wither = function (F) {
        var traverseF = traverse(F);
        return function (wa, f) { return F.map(traverseF(wa, f), filterableValidation.compact); };
    };
    var wilt = function (F) {
        var traverseF = traverse(F);
        return function (wa, f) { return F.map(traverseF(wa, f), filterableValidation.separate); };
    };
    return __assign({}, filterableValidation, { traverse: traverse,
        reduce: reduce,
        wither: wither,
        wilt: wilt });
}
exports.getWitherable = getWitherable;
var throwError = exports.failure;
/**
 * @since 1.16.0
 */
exports.getMonadThrow = function (S) {
    return __assign({}, exports.getMonad(S), { throwError: throwError,
        fromEither: exports.fromEither, fromOption: function (o, e) { return (o.isNone() ? throwError(e) : of(o.value)); } });
};
/**
 * @since 1.0.0
 */
exports.validation = {
    URI: exports.URI,
    map: map,
    bimap: bimap,
    reduce: reduce,
    foldMap: foldMap,
    foldr: foldr,
    traverse: traverse,
    sequence: sequence
};
