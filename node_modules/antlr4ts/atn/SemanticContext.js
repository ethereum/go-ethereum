"use strict";
/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
var __decorate = (this && this.__decorate) || function (decorators, target, key, desc) {
    var c = arguments.length, r = c < 3 ? target : desc === null ? desc = Object.getOwnPropertyDescriptor(target, key) : desc, d;
    if (typeof Reflect === "object" && typeof Reflect.decorate === "function") r = Reflect.decorate(decorators, target, key, desc);
    else for (var i = decorators.length - 1; i >= 0; i--) if (d = decorators[i]) r = (c < 3 ? d(r) : c > 3 ? d(target, key, r) : d(target, key)) || r;
    return c > 3 && r && Object.defineProperty(target, key, r), r;
};
var __param = (this && this.__param) || function (paramIndex, decorator) {
    return function (target, key) { decorator(target, key, paramIndex); }
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.SemanticContext = void 0;
// ConvertTo-TS run at 2016-10-04T11:26:36.9521478-07:00
const Array2DHashSet_1 = require("../misc/Array2DHashSet");
const ArrayEqualityComparator_1 = require("../misc/ArrayEqualityComparator");
const MurmurHash_1 = require("../misc/MurmurHash");
const Decorators_1 = require("../Decorators");
const ObjectEqualityComparator_1 = require("../misc/ObjectEqualityComparator");
const Utils = require("../misc/Utils");
function max(items) {
    let result;
    for (let current of items) {
        if (result === undefined) {
            result = current;
            continue;
        }
        let comparison = result.compareTo(current);
        if (comparison < 0) {
            result = current;
        }
    }
    return result;
}
function min(items) {
    let result;
    for (let current of items) {
        if (result === undefined) {
            result = current;
            continue;
        }
        let comparison = result.compareTo(current);
        if (comparison > 0) {
            result = current;
        }
    }
    return result;
}
/** A tree structure used to record the semantic context in which
 *  an ATN configuration is valid.  It's either a single predicate,
 *  a conjunction `p1&&p2`, or a sum of products `p1||p2`.
 *
 *  I have scoped the {@link AND}, {@link OR}, and {@link Predicate} subclasses of
 *  {@link SemanticContext} within the scope of this outer class.
 */
class SemanticContext {
    /**
     * The default {@link SemanticContext}, which is semantically equivalent to
     * a predicate of the form `{true}?`.
     */
    static get NONE() {
        if (SemanticContext._NONE === undefined) {
            SemanticContext._NONE = new SemanticContext.Predicate();
        }
        return SemanticContext._NONE;
    }
    /**
     * Evaluate the precedence predicates for the context and reduce the result.
     *
     * @param parser The parser instance.
     * @param parserCallStack
     * @returns The simplified semantic context after precedence predicates are
     * evaluated, which will be one of the following values.
     *
     * * {@link #NONE}: if the predicate simplifies to `true` after
     *   precedence predicates are evaluated.
     * * `undefined`: if the predicate simplifies to `false` after
     *   precedence predicates are evaluated.
     * * `this`: if the semantic context is not changed as a result of
     *   precedence predicate evaluation.
     * * A non-`undefined` {@link SemanticContext}: the new simplified
     *   semantic context after precedence predicates are evaluated.
     */
    evalPrecedence(parser, parserCallStack) {
        return this;
    }
    static and(a, b) {
        if (!a || a === SemanticContext.NONE) {
            return b;
        }
        if (b === SemanticContext.NONE) {
            return a;
        }
        let result = new SemanticContext.AND(a, b);
        if (result.opnds.length === 1) {
            return result.opnds[0];
        }
        return result;
    }
    /**
     *
     *  @see ParserATNSimulator#getPredsForAmbigAlts
     */
    static or(a, b) {
        if (!a) {
            return b;
        }
        if (a === SemanticContext.NONE || b === SemanticContext.NONE) {
            return SemanticContext.NONE;
        }
        let result = new SemanticContext.OR(a, b);
        if (result.opnds.length === 1) {
            return result.opnds[0];
        }
        return result;
    }
}
exports.SemanticContext = SemanticContext;
(function (SemanticContext) {
    /**
     * This random 30-bit prime represents the value of `AND.class.hashCode()`.
     */
    const AND_HASHCODE = 40363613;
    /**
     * This random 30-bit prime represents the value of `OR.class.hashCode()`.
     */
    const OR_HASHCODE = 486279973;
    function filterPrecedencePredicates(collection) {
        let result = [];
        for (let i = 0; i < collection.length; i++) {
            let context = collection[i];
            if (context instanceof SemanticContext.PrecedencePredicate) {
                result.push(context);
                // Remove the item from 'collection' and move i back so we look at the same index again
                collection.splice(i, 1);
                i--;
            }
        }
        return result;
    }
    class Predicate extends SemanticContext {
        constructor(ruleIndex = -1, predIndex = -1, isCtxDependent = false) {
            super();
            this.ruleIndex = ruleIndex;
            this.predIndex = predIndex;
            this.isCtxDependent = isCtxDependent;
        }
        eval(parser, parserCallStack) {
            let localctx = this.isCtxDependent ? parserCallStack : undefined;
            return parser.sempred(localctx, this.ruleIndex, this.predIndex);
        }
        hashCode() {
            let hashCode = MurmurHash_1.MurmurHash.initialize();
            hashCode = MurmurHash_1.MurmurHash.update(hashCode, this.ruleIndex);
            hashCode = MurmurHash_1.MurmurHash.update(hashCode, this.predIndex);
            hashCode = MurmurHash_1.MurmurHash.update(hashCode, this.isCtxDependent ? 1 : 0);
            hashCode = MurmurHash_1.MurmurHash.finish(hashCode, 3);
            return hashCode;
        }
        equals(obj) {
            if (!(obj instanceof Predicate)) {
                return false;
            }
            if (this === obj) {
                return true;
            }
            return this.ruleIndex === obj.ruleIndex &&
                this.predIndex === obj.predIndex &&
                this.isCtxDependent === obj.isCtxDependent;
        }
        toString() {
            return "{" + this.ruleIndex + ":" + this.predIndex + "}?";
        }
    }
    __decorate([
        Decorators_1.Override
    ], Predicate.prototype, "eval", null);
    __decorate([
        Decorators_1.Override
    ], Predicate.prototype, "hashCode", null);
    __decorate([
        Decorators_1.Override
    ], Predicate.prototype, "equals", null);
    __decorate([
        Decorators_1.Override
    ], Predicate.prototype, "toString", null);
    SemanticContext.Predicate = Predicate;
    class PrecedencePredicate extends SemanticContext {
        constructor(precedence) {
            super();
            this.precedence = precedence;
        }
        eval(parser, parserCallStack) {
            return parser.precpred(parserCallStack, this.precedence);
        }
        evalPrecedence(parser, parserCallStack) {
            if (parser.precpred(parserCallStack, this.precedence)) {
                return SemanticContext.NONE;
            }
            else {
                return undefined;
            }
        }
        compareTo(o) {
            return this.precedence - o.precedence;
        }
        hashCode() {
            let hashCode = 1;
            hashCode = 31 * hashCode + this.precedence;
            return hashCode;
        }
        equals(obj) {
            if (!(obj instanceof PrecedencePredicate)) {
                return false;
            }
            if (this === obj) {
                return true;
            }
            return this.precedence === obj.precedence;
        }
        toString() {
            return "{" + this.precedence + ">=prec}?";
        }
    }
    __decorate([
        Decorators_1.Override
    ], PrecedencePredicate.prototype, "eval", null);
    __decorate([
        Decorators_1.Override
    ], PrecedencePredicate.prototype, "evalPrecedence", null);
    __decorate([
        Decorators_1.Override
    ], PrecedencePredicate.prototype, "compareTo", null);
    __decorate([
        Decorators_1.Override
    ], PrecedencePredicate.prototype, "hashCode", null);
    __decorate([
        Decorators_1.Override
    ], PrecedencePredicate.prototype, "equals", null);
    __decorate([
        Decorators_1.Override
    ], PrecedencePredicate.prototype, "toString", null);
    SemanticContext.PrecedencePredicate = PrecedencePredicate;
    /**
     * This is the base class for semantic context "operators", which operate on
     * a collection of semantic context "operands".
     *
     * @since 4.3
     */
    class Operator extends SemanticContext {
    }
    SemanticContext.Operator = Operator;
    /**
     * A semantic context which is true whenever none of the contained contexts
     * is false.
     */
    let AND = class AND extends Operator {
        constructor(a, b) {
            super();
            let operands = new Array2DHashSet_1.Array2DHashSet(ObjectEqualityComparator_1.ObjectEqualityComparator.INSTANCE);
            if (a instanceof AND) {
                operands.addAll(a.opnds);
            }
            else {
                operands.add(a);
            }
            if (b instanceof AND) {
                operands.addAll(b.opnds);
            }
            else {
                operands.add(b);
            }
            this.opnds = operands.toArray();
            let precedencePredicates = filterPrecedencePredicates(this.opnds);
            // interested in the transition with the lowest precedence
            let reduced = min(precedencePredicates);
            if (reduced) {
                this.opnds.push(reduced);
            }
        }
        get operands() {
            return this.opnds;
        }
        equals(obj) {
            if (this === obj) {
                return true;
            }
            if (!(obj instanceof AND)) {
                return false;
            }
            return ArrayEqualityComparator_1.ArrayEqualityComparator.INSTANCE.equals(this.opnds, obj.opnds);
        }
        hashCode() {
            return MurmurHash_1.MurmurHash.hashCode(this.opnds, AND_HASHCODE);
        }
        /**
         * {@inheritDoc}
         *
         * The evaluation of predicates by this context is short-circuiting, but
         * unordered.
         */
        eval(parser, parserCallStack) {
            for (let opnd of this.opnds) {
                if (!opnd.eval(parser, parserCallStack)) {
                    return false;
                }
            }
            return true;
        }
        evalPrecedence(parser, parserCallStack) {
            let differs = false;
            let operands = [];
            for (let context of this.opnds) {
                let evaluated = context.evalPrecedence(parser, parserCallStack);
                differs = differs || (evaluated !== context);
                if (evaluated == null) {
                    // The AND context is false if any element is false
                    return undefined;
                }
                else if (evaluated !== SemanticContext.NONE) {
                    // Reduce the result by skipping true elements
                    operands.push(evaluated);
                }
            }
            if (!differs) {
                return this;
            }
            if (operands.length === 0) {
                // all elements were true, so the AND context is true
                return SemanticContext.NONE;
            }
            let result = operands[0];
            for (let i = 1; i < operands.length; i++) {
                result = SemanticContext.and(result, operands[i]);
            }
            return result;
        }
        toString() {
            return Utils.join(this.opnds, "&&");
        }
    };
    __decorate([
        Decorators_1.Override
    ], AND.prototype, "operands", null);
    __decorate([
        Decorators_1.Override
    ], AND.prototype, "equals", null);
    __decorate([
        Decorators_1.Override
    ], AND.prototype, "hashCode", null);
    __decorate([
        Decorators_1.Override
    ], AND.prototype, "eval", null);
    __decorate([
        Decorators_1.Override
    ], AND.prototype, "evalPrecedence", null);
    __decorate([
        Decorators_1.Override
    ], AND.prototype, "toString", null);
    AND = __decorate([
        __param(0, Decorators_1.NotNull), __param(1, Decorators_1.NotNull)
    ], AND);
    SemanticContext.AND = AND;
    /**
     * A semantic context which is true whenever at least one of the contained
     * contexts is true.
     */
    let OR = class OR extends Operator {
        constructor(a, b) {
            super();
            let operands = new Array2DHashSet_1.Array2DHashSet(ObjectEqualityComparator_1.ObjectEqualityComparator.INSTANCE);
            if (a instanceof OR) {
                operands.addAll(a.opnds);
            }
            else {
                operands.add(a);
            }
            if (b instanceof OR) {
                operands.addAll(b.opnds);
            }
            else {
                operands.add(b);
            }
            this.opnds = operands.toArray();
            let precedencePredicates = filterPrecedencePredicates(this.opnds);
            // interested in the transition with the highest precedence
            let reduced = max(precedencePredicates);
            if (reduced) {
                this.opnds.push(reduced);
            }
        }
        get operands() {
            return this.opnds;
        }
        equals(obj) {
            if (this === obj) {
                return true;
            }
            if (!(obj instanceof OR)) {
                return false;
            }
            return ArrayEqualityComparator_1.ArrayEqualityComparator.INSTANCE.equals(this.opnds, obj.opnds);
        }
        hashCode() {
            return MurmurHash_1.MurmurHash.hashCode(this.opnds, OR_HASHCODE);
        }
        /**
         * {@inheritDoc}
         *
         * The evaluation of predicates by this context is short-circuiting, but
         * unordered.
         */
        eval(parser, parserCallStack) {
            for (let opnd of this.opnds) {
                if (opnd.eval(parser, parserCallStack)) {
                    return true;
                }
            }
            return false;
        }
        evalPrecedence(parser, parserCallStack) {
            let differs = false;
            let operands = [];
            for (let context of this.opnds) {
                let evaluated = context.evalPrecedence(parser, parserCallStack);
                differs = differs || (evaluated !== context);
                if (evaluated === SemanticContext.NONE) {
                    // The OR context is true if any element is true
                    return SemanticContext.NONE;
                }
                else if (evaluated) {
                    // Reduce the result by skipping false elements
                    operands.push(evaluated);
                }
            }
            if (!differs) {
                return this;
            }
            if (operands.length === 0) {
                // all elements were false, so the OR context is false
                return undefined;
            }
            let result = operands[0];
            for (let i = 1; i < operands.length; i++) {
                result = SemanticContext.or(result, operands[i]);
            }
            return result;
        }
        toString() {
            return Utils.join(this.opnds, "||");
        }
    };
    __decorate([
        Decorators_1.Override
    ], OR.prototype, "operands", null);
    __decorate([
        Decorators_1.Override
    ], OR.prototype, "equals", null);
    __decorate([
        Decorators_1.Override
    ], OR.prototype, "hashCode", null);
    __decorate([
        Decorators_1.Override
    ], OR.prototype, "eval", null);
    __decorate([
        Decorators_1.Override
    ], OR.prototype, "evalPrecedence", null);
    __decorate([
        Decorators_1.Override
    ], OR.prototype, "toString", null);
    OR = __decorate([
        __param(0, Decorators_1.NotNull), __param(1, Decorators_1.NotNull)
    ], OR);
    SemanticContext.OR = OR;
})(SemanticContext = exports.SemanticContext || (exports.SemanticContext = {}));
//# sourceMappingURL=SemanticContext.js.map