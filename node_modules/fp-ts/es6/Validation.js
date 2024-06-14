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
import { toString } from './function';
import { fromEquals } from './Eq';
export var URI = 'Validation';
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
        return "failure(" + toString(this.value) + ")";
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
export { Failure };
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
        return "success(" + toString(this.value) + ")";
    };
    Success.prototype.isFailure = function () {
        return false;
    };
    Success.prototype.isSuccess = function () {
        return true;
    };
    return Success;
}());
export { Success };
/**
 * @since 1.17.0
 */
export var getShow = function (SL, SA) {
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
export var getSetoid = getEq;
/**
 * @since 1.19.0
 */
export function getEq(EL, EA) {
    return fromEquals(function (x, y) {
        return x.isFailure() ? y.isFailure() && EL.equals(x.value, y.value) : y.isSuccess() && EA.equals(x.value, y.value);
    });
}
var map = function (fa, f) {
    return fa.map(f);
};
/**
 * @since 1.0.0
 */
export var success = function (a) {
    return new Success(a);
};
var of = success;
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
export var getApplicative = function (S) {
    var ap = function (fab, fa) {
        return fab.isFailure()
            ? fa.isFailure()
                ? failure(S.concat(fab.value, fa.value))
                : failure(fab.value)
            : fa.isFailure()
                ? failure(fa.value)
                : success(fab.value(fa.value));
    };
    return {
        URI: URI,
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
export var getMonad = function (S) {
    var chain = function (fa, f) {
        return fa.isFailure() ? failure(fa.value) : f(fa.value);
    };
    return __assign({}, getApplicative(S), { chain: chain });
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
    return ta.isFailure() ? F.of(failure(ta.value)) : F.map(f(ta.value), of);
}; };
var sequence = function (F) { return function (ta) {
    return ta.isFailure() ? F.of(failure(ta.value)) : F.map(ta.value, of);
}; };
var bimap = function (fla, f, g) {
    return fla.bimap(f, g);
};
/**
 * @since 1.0.0
 */
export var failure = function (l) {
    return new Failure(l);
};
export function fromPredicate(predicate, f) {
    return function (a) { return (predicate(a) ? success(a) : failure(f(a))); };
}
/**
 * @since 1.0.0
 */
export var fromEither = function (e) {
    return e.isLeft() ? failure(e.value) : success(e.value);
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
export var tryCatch = function (f, onError) {
    try {
        return success(f());
    }
    catch (e) {
        return failure(onError(e));
    }
};
/**
 * @since 1.0.0
 */
export var getSemigroup = function (SL, SA) {
    var concat = function (fx, fy) {
        return fx.isFailure()
            ? fy.isFailure()
                ? failure(SL.concat(fx.value, fy.value))
                : failure(fx.value)
            : fy.isFailure()
                ? failure(fy.value)
                : success(SA.concat(fx.value, fy.value));
    };
    return {
        concat: concat
    };
};
/**
 * @since 1.0.0
 */
export var getMonoid = function (SL, SA) {
    return __assign({}, getSemigroup(SL, SA), { empty: success(SA.empty) });
};
/**
 * @since 1.0.0
 */
export var getAlt = function (S) {
    var alt = function (fx, fy) {
        return fx.isFailure() ? (fy.isFailure() ? failure(S.concat(fx.value, fy.value)) : fy) : fx;
    };
    return {
        URI: URI,
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
export var isFailure = function (fa) {
    return fa.isFailure();
};
/**
 * Returns `true` if the validation is an instance of `Success`, `false` otherwise
 *
 * @since 1.0.0
 */
export var isSuccess = function (fa) {
    return fa.isSuccess();
};
/**
 * Builds `Compactable` instance for `Validation` given `Monoid` for the failure side
 *
 * @since 1.7.0
 */
export function getCompactable(ML) {
    var compact = function (fa) {
        if (fa.isFailure()) {
            return fa;
        }
        if (fa.value.isNone()) {
            return failure(ML.empty);
        }
        return success(fa.value.value);
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
                left: success(fa.value.value),
                right: failure(ML.empty)
            };
        }
        return {
            left: failure(ML.empty),
            right: success(fa.value.value)
        };
    };
    return {
        URI: URI,
        _L: undefined,
        compact: compact,
        separate: separate
    };
}
/**
 * Builds `Filterable` instance for `Validation` given `Monoid` for the left side
 *
 * @since 1.7.0
 */
export function getFilterable(ML) {
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
                left: success(e.value),
                right: failure(ML.empty)
            };
        }
        return {
            left: failure(ML.empty),
            right: success(e.value)
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
                left: failure(ML.empty),
                right: success(fa.value)
            };
        }
        return {
            left: success(fa.value),
            right: failure(ML.empty)
        };
    };
    var filterMap = function (fa, f) {
        if (fa.isFailure()) {
            return fa;
        }
        var optionB = f(fa.value);
        if (optionB.isSome()) {
            return success(optionB.value);
        }
        return failure(ML.empty);
    };
    var filter = function (fa, p) {
        if (fa.isFailure()) {
            return fa;
        }
        var a = fa.value;
        if (p(a)) {
            return success(a);
        }
        return failure(ML.empty);
    };
    return __assign({}, C, { map: map,
        partitionMap: partitionMap,
        filterMap: filterMap,
        partition: partition,
        filter: filter });
}
/**
 * Builds `Witherable` instance for `Validation` given `Monoid` for the left side
 *
 * @since 1.7.0
 */
export function getWitherable(ML) {
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
var throwError = failure;
/**
 * @since 1.16.0
 */
export var getMonadThrow = function (S) {
    return __assign({}, getMonad(S), { throwError: throwError,
        fromEither: fromEither, fromOption: function (o, e) { return (o.isNone() ? throwError(e) : of(o.value)); } });
};
/**
 * @since 1.0.0
 */
export var validation = {
    URI: URI,
    map: map,
    bimap: bimap,
    reduce: reduce,
    foldMap: foldMap,
    foldr: foldr,
    traverse: traverse,
    sequence: sequence
};
