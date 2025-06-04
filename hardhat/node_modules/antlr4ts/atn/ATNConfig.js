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
exports.ATNConfig = void 0;
// ConvertTo-TS run at 2016-10-04T11:26:25.2796692-07:00
const Array2DHashMap_1 = require("../misc/Array2DHashMap");
const DecisionState_1 = require("./DecisionState");
const MurmurHash_1 = require("../misc/MurmurHash");
const Decorators_1 = require("../Decorators");
const ObjectEqualityComparator_1 = require("../misc/ObjectEqualityComparator");
const PredictionContext_1 = require("./PredictionContext");
const SemanticContext_1 = require("./SemanticContext");
const assert = require("assert");
/**
 * This field stores the bit mask for implementing the
 * {@link #isPrecedenceFilterSuppressed} property as a bit within the
 * existing {@link #altAndOuterContextDepth} field.
 */
const SUPPRESS_PRECEDENCE_FILTER = 0x80000000;
/**
 * Represents a location with context in an ATN. The location is identified by the following values:
 *
 * * The current ATN state
 * * The predicted alternative
 * * The semantic context which must be true for this configuration to be enabled
 * * The syntactic context, which is represented as a graph-structured stack whose path(s) lead to the root of the rule
 *   invocations leading to this state
 *
 * In addition to these values, `ATNConfig` stores several properties about paths taken to get to the location which
 * were added over time to help with performance, correctness, and/or debugging.
 *
 * * `reachesIntoOuterContext`:: Used to ensure semantic predicates are not evaluated in the wrong context.
 * * `hasPassedThroughNonGreedyDecision`: Used for enabling first-match-wins instead of longest-match-wins after
 *   crossing a non-greedy decision.
 * * `lexerActionExecutor`: Used for tracking the lexer action(s) to execute should this instance be selected during
 *   lexing.
 * * `isPrecedenceFilterSuppressed`: A state variable for one of the dynamic disambiguation strategies employed by
 *   `ParserATNSimulator.applyPrecedenceFilter`.
 *
 * Due to the use of a graph-structured stack, a single `ATNConfig` is capable of representing many individual ATN
 * configurations which reached the same location in an ATN by following different paths.
 *
 * PERF: To conserve memory, `ATNConfig` is split into several different concrete types. `ATNConfig` itself stores the
 * minimum amount of information typically used to define an `ATNConfig` instance. Various derived types provide
 * additional storage space for cases where a non-default value is used for some of the object properties. The
 * `ATNConfig.create` and `ATNConfig.transform` methods automatically select the smallest concrete type capable of
 * representing the unique information for any given `ATNConfig`.
 */
let ATNConfig = class ATNConfig {
    constructor(state, altOrConfig, context) {
        if (typeof altOrConfig === "number") {
            assert((altOrConfig & 0xFFFFFF) === altOrConfig);
            this._state = state;
            this.altAndOuterContextDepth = altOrConfig;
            this._context = context;
        }
        else {
            this._state = state;
            this.altAndOuterContextDepth = altOrConfig.altAndOuterContextDepth;
            this._context = context;
        }
    }
    static create(state, alt, context, semanticContext = SemanticContext_1.SemanticContext.NONE, lexerActionExecutor) {
        if (semanticContext !== SemanticContext_1.SemanticContext.NONE) {
            if (lexerActionExecutor != null) {
                return new ActionSemanticContextATNConfig(lexerActionExecutor, semanticContext, state, alt, context, false);
            }
            else {
                return new SemanticContextATNConfig(semanticContext, state, alt, context);
            }
        }
        else if (lexerActionExecutor != null) {
            return new ActionATNConfig(lexerActionExecutor, state, alt, context, false);
        }
        else {
            return new ATNConfig(state, alt, context);
        }
    }
    /** Gets the ATN state associated with this configuration */
    get state() {
        return this._state;
    }
    /** What alt (or lexer rule) is predicted by this configuration */
    get alt() {
        return this.altAndOuterContextDepth & 0x00FFFFFF;
    }
    get context() {
        return this._context;
    }
    set context(context) {
        this._context = context;
    }
    get reachesIntoOuterContext() {
        return this.outerContextDepth !== 0;
    }
    /**
     * We cannot execute predicates dependent upon local context unless
     * we know for sure we are in the correct context. Because there is
     * no way to do this efficiently, we simply cannot evaluate
     * dependent predicates unless we are in the rule that initially
     * invokes the ATN simulator.
     *
     * closure() tracks the depth of how far we dip into the outer context:
     * depth &gt; 0.  Note that it may not be totally accurate depth since I
     * don't ever decrement. TODO: make it a boolean then
     */
    get outerContextDepth() {
        return (this.altAndOuterContextDepth >>> 24) & 0x7F;
    }
    set outerContextDepth(outerContextDepth) {
        assert(outerContextDepth >= 0);
        // saturate at 0x7F - everything but zero/positive is only used for debug information anyway
        outerContextDepth = Math.min(outerContextDepth, 0x7F);
        this.altAndOuterContextDepth = ((outerContextDepth << 24) | (this.altAndOuterContextDepth & ~0x7F000000) >>> 0);
    }
    get lexerActionExecutor() {
        return undefined;
    }
    get semanticContext() {
        return SemanticContext_1.SemanticContext.NONE;
    }
    get hasPassedThroughNonGreedyDecision() {
        return false;
    }
    clone() {
        return this.transform(this.state, false);
    }
    transform(/*@NotNull*/ state, checkNonGreedy, arg2) {
        if (arg2 == null) {
            return this.transformImpl(state, this._context, this.semanticContext, checkNonGreedy, this.lexerActionExecutor);
        }
        else if (arg2 instanceof PredictionContext_1.PredictionContext) {
            return this.transformImpl(state, arg2, this.semanticContext, checkNonGreedy, this.lexerActionExecutor);
        }
        else if (arg2 instanceof SemanticContext_1.SemanticContext) {
            return this.transformImpl(state, this._context, arg2, checkNonGreedy, this.lexerActionExecutor);
        }
        else {
            return this.transformImpl(state, this._context, this.semanticContext, checkNonGreedy, arg2);
        }
    }
    transformImpl(state, context, semanticContext, checkNonGreedy, lexerActionExecutor) {
        let passedThroughNonGreedy = checkNonGreedy && ATNConfig.checkNonGreedyDecision(this, state);
        if (semanticContext !== SemanticContext_1.SemanticContext.NONE) {
            if (lexerActionExecutor != null || passedThroughNonGreedy) {
                return new ActionSemanticContextATNConfig(lexerActionExecutor, semanticContext, state, this, context, passedThroughNonGreedy);
            }
            else {
                return new SemanticContextATNConfig(semanticContext, state, this, context);
            }
        }
        else if (lexerActionExecutor != null || passedThroughNonGreedy) {
            return new ActionATNConfig(lexerActionExecutor, state, this, context, passedThroughNonGreedy);
        }
        else {
            return new ATNConfig(state, this, context);
        }
    }
    static checkNonGreedyDecision(source, target) {
        return source.hasPassedThroughNonGreedyDecision
            || target instanceof DecisionState_1.DecisionState && target.nonGreedy;
    }
    appendContext(context, contextCache) {
        if (typeof context === "number") {
            let appendedContext = this.context.appendSingleContext(context, contextCache);
            let result = this.transform(this.state, false, appendedContext);
            return result;
        }
        else {
            let appendedContext = this.context.appendContext(context, contextCache);
            let result = this.transform(this.state, false, appendedContext);
            return result;
        }
    }
    /**
     * Determines if this `ATNConfig` fully contains another `ATNConfig`.
     *
     * An ATN configuration represents a position (including context) in an ATN during parsing. Since `ATNConfig` stores
     * the context as a graph, a single `ATNConfig` instance is capable of representing many ATN configurations which
     * are all in the same "location" but have different contexts. These `ATNConfig` instances are again merged when
     * they are added to an `ATNConfigSet`. This method supports `ATNConfigSet.contains` by evaluating whether a
     * particular `ATNConfig` contains all of the ATN configurations represented by another `ATNConfig`.
     *
     * An `ATNConfig` _a_ contains another `ATNConfig` _b_ if all of the following conditions are met:
     *
     * * The configurations are in the same state (`state`)
     * * The configurations predict the same alternative (`alt`)
     * * The semantic context of _a_ implies the semantic context of _b_ (this method performs a weaker equality check)
     * * Joining the prediction contexts of _a_ and _b_ results in the prediction context of _a_
     *
     * This method implements a conservative approximation of containment. As a result, when this method returns `true`
     * it is known that parsing from `subconfig` can only recognize a subset of the inputs which can be recognized
     * starting at the current `ATNConfig`. However, due to the imprecise evaluation of implication for the semantic
     * contexts, no assumptions can be made about the relationship between the configurations when this method returns
     * `false`.
     *
     * @param subconfig The sub configuration.
     * @returns `true` if this configuration contains `subconfig`; otherwise, `false`.
     */
    contains(subconfig) {
        if (this.state.stateNumber !== subconfig.state.stateNumber
            || this.alt !== subconfig.alt
            || !this.semanticContext.equals(subconfig.semanticContext)) {
            return false;
        }
        let leftWorkList = [];
        let rightWorkList = [];
        leftWorkList.push(this.context);
        rightWorkList.push(subconfig.context);
        while (true) {
            let left = leftWorkList.pop();
            let right = rightWorkList.pop();
            if (!left || !right) {
                break;
            }
            if (left === right) {
                return true;
            }
            if (left.size < right.size) {
                return false;
            }
            if (right.isEmpty) {
                return left.hasEmpty;
            }
            else {
                for (let i = 0; i < right.size; i++) {
                    let index = left.findReturnState(right.getReturnState(i));
                    if (index < 0) {
                        // assumes invokingStates has no duplicate entries
                        return false;
                    }
                    leftWorkList.push(left.getParent(index));
                    rightWorkList.push(right.getParent(i));
                }
            }
        }
        return false;
    }
    get isPrecedenceFilterSuppressed() {
        return (this.altAndOuterContextDepth & SUPPRESS_PRECEDENCE_FILTER) !== 0;
    }
    set isPrecedenceFilterSuppressed(value) {
        if (value) {
            this.altAndOuterContextDepth |= SUPPRESS_PRECEDENCE_FILTER;
        }
        else {
            this.altAndOuterContextDepth &= ~SUPPRESS_PRECEDENCE_FILTER;
        }
    }
    /** An ATN configuration is equal to another if both have
     *  the same state, they predict the same alternative, and
     *  syntactic/semantic contexts are the same.
     */
    equals(o) {
        if (this === o) {
            return true;
        }
        else if (!(o instanceof ATNConfig)) {
            return false;
        }
        return this.state.stateNumber === o.state.stateNumber
            && this.alt === o.alt
            && this.reachesIntoOuterContext === o.reachesIntoOuterContext
            && this.context.equals(o.context)
            && this.semanticContext.equals(o.semanticContext)
            && this.isPrecedenceFilterSuppressed === o.isPrecedenceFilterSuppressed
            && this.hasPassedThroughNonGreedyDecision === o.hasPassedThroughNonGreedyDecision
            && ObjectEqualityComparator_1.ObjectEqualityComparator.INSTANCE.equals(this.lexerActionExecutor, o.lexerActionExecutor);
    }
    hashCode() {
        let hashCode = MurmurHash_1.MurmurHash.initialize(7);
        hashCode = MurmurHash_1.MurmurHash.update(hashCode, this.state.stateNumber);
        hashCode = MurmurHash_1.MurmurHash.update(hashCode, this.alt);
        hashCode = MurmurHash_1.MurmurHash.update(hashCode, this.reachesIntoOuterContext ? 1 : 0);
        hashCode = MurmurHash_1.MurmurHash.update(hashCode, this.context);
        hashCode = MurmurHash_1.MurmurHash.update(hashCode, this.semanticContext);
        hashCode = MurmurHash_1.MurmurHash.update(hashCode, this.hasPassedThroughNonGreedyDecision ? 1 : 0);
        hashCode = MurmurHash_1.MurmurHash.update(hashCode, this.lexerActionExecutor);
        hashCode = MurmurHash_1.MurmurHash.finish(hashCode, 7);
        return hashCode;
    }
    /**
     * Returns a graphical representation of the current `ATNConfig` in Graphviz format. The graph can be stored to a
     * **.dot** file and then rendered to an image using Graphviz.
     *
     * @returns A Graphviz graph representing the current `ATNConfig`.
     *
     * @see http://www.graphviz.org/
     */
    toDotString() {
        let builder = "";
        builder += ("digraph G {\n");
        builder += ("rankdir=LR;\n");
        let visited = new Array2DHashMap_1.Array2DHashMap(PredictionContext_1.PredictionContext.IdentityEqualityComparator.INSTANCE);
        let workList = [];
        function getOrAddContext(context) {
            let newNumber = visited.size;
            let result = visited.putIfAbsent(context, newNumber);
            if (result != null) {
                // Already saw this context
                return result;
            }
            workList.push(context);
            return newNumber;
        }
        workList.push(this.context);
        visited.put(this.context, 0);
        while (true) {
            let current = workList.pop();
            if (!current) {
                break;
            }
            for (let i = 0; i < current.size; i++) {
                builder += ("  s") + (getOrAddContext(current));
                builder += ("->");
                builder += ("s") + (getOrAddContext(current.getParent(i)));
                builder += ("[label=\"") + (current.getReturnState(i)) + ("\"];\n");
            }
        }
        builder += ("}\n");
        return builder.toString();
    }
    toString(recog, showAlt, showContext) {
        // Must check showContext before showAlt to preserve original overload behavior
        if (showContext == null) {
            showContext = showAlt != null;
        }
        if (showAlt == null) {
            showAlt = true;
        }
        let buf = "";
        // if (this.state.ruleIndex >= 0) {
        // 	if (recog != null) {
        // 		buf += (recog.ruleNames[this.state.ruleIndex] + ":");
        // 	} else {
        // 		buf += (this.state.ruleIndex + ":");
        // 	}
        // }
        let contexts;
        if (showContext) {
            contexts = this.context.toStrings(recog, this.state.stateNumber);
        }
        else {
            contexts = ["?"];
        }
        let first = true;
        for (let contextDesc of contexts) {
            if (first) {
                first = false;
            }
            else {
                buf += (", ");
            }
            buf += ("(");
            buf += (this.state);
            if (showAlt) {
                buf += (",");
                buf += (this.alt);
            }
            if (this.context) {
                buf += (",");
                buf += (contextDesc);
            }
            if (this.semanticContext !== SemanticContext_1.SemanticContext.NONE) {
                buf += (",");
                buf += (this.semanticContext);
            }
            if (this.reachesIntoOuterContext) {
                buf += (",up=") + (this.outerContextDepth);
            }
            buf += (")");
        }
        return buf.toString();
    }
};
__decorate([
    Decorators_1.NotNull
], ATNConfig.prototype, "_state", void 0);
__decorate([
    Decorators_1.NotNull
], ATNConfig.prototype, "_context", void 0);
__decorate([
    Decorators_1.NotNull
], ATNConfig.prototype, "state", null);
__decorate([
    Decorators_1.NotNull,
    __param(0, Decorators_1.NotNull)
], ATNConfig.prototype, "context", null);
__decorate([
    Decorators_1.NotNull
], ATNConfig.prototype, "semanticContext", null);
__decorate([
    Decorators_1.Override
], ATNConfig.prototype, "clone", null);
__decorate([
    __param(0, Decorators_1.NotNull), __param(2, Decorators_1.NotNull)
], ATNConfig.prototype, "transformImpl", null);
__decorate([
    Decorators_1.Override
], ATNConfig.prototype, "equals", null);
__decorate([
    Decorators_1.Override
], ATNConfig.prototype, "hashCode", null);
__decorate([
    __param(0, Decorators_1.NotNull), __param(3, Decorators_1.NotNull)
], ATNConfig, "create", null);
ATNConfig = __decorate([
    __param(0, Decorators_1.NotNull), __param(2, Decorators_1.NotNull)
], ATNConfig);
exports.ATNConfig = ATNConfig;
/**
 * This class was derived from `ATNConfig` purely as a memory optimization. It allows for the creation of an `ATNConfig`
 * with a non-default semantic context.
 *
 * See the `ATNConfig` documentation for more information about conserving memory through the use of several concrete
 * types.
 */
let SemanticContextATNConfig = class SemanticContextATNConfig extends ATNConfig {
    constructor(semanticContext, state, altOrConfig, context) {
        if (typeof altOrConfig === "number") {
            super(state, altOrConfig, context);
        }
        else {
            super(state, altOrConfig, context);
        }
        this._semanticContext = semanticContext;
    }
    get semanticContext() {
        return this._semanticContext;
    }
};
__decorate([
    Decorators_1.NotNull
], SemanticContextATNConfig.prototype, "_semanticContext", void 0);
__decorate([
    Decorators_1.Override
], SemanticContextATNConfig.prototype, "semanticContext", null);
SemanticContextATNConfig = __decorate([
    __param(1, Decorators_1.NotNull), __param(2, Decorators_1.NotNull)
], SemanticContextATNConfig);
/**
 * This class was derived from `ATNConfig` purely as a memory optimization. It allows for the creation of an `ATNConfig`
 * with a lexer action.
 *
 * See the `ATNConfig` documentation for more information about conserving memory through the use of several concrete
 * types.
 */
let ActionATNConfig = class ActionATNConfig extends ATNConfig {
    constructor(lexerActionExecutor, state, altOrConfig, context, passedThroughNonGreedyDecision) {
        if (typeof altOrConfig === "number") {
            super(state, altOrConfig, context);
        }
        else {
            super(state, altOrConfig, context);
            if (altOrConfig.semanticContext !== SemanticContext_1.SemanticContext.NONE) {
                throw new Error("Not supported");
            }
        }
        this._lexerActionExecutor = lexerActionExecutor;
        this.passedThroughNonGreedyDecision = passedThroughNonGreedyDecision;
    }
    get lexerActionExecutor() {
        return this._lexerActionExecutor;
    }
    get hasPassedThroughNonGreedyDecision() {
        return this.passedThroughNonGreedyDecision;
    }
};
__decorate([
    Decorators_1.Override
], ActionATNConfig.prototype, "lexerActionExecutor", null);
__decorate([
    Decorators_1.Override
], ActionATNConfig.prototype, "hasPassedThroughNonGreedyDecision", null);
ActionATNConfig = __decorate([
    __param(1, Decorators_1.NotNull), __param(2, Decorators_1.NotNull)
], ActionATNConfig);
/**
 * This class was derived from `SemanticContextATNConfig` purely as a memory optimization. It allows for the creation of
 * an `ATNConfig` with both a lexer action and a non-default semantic context.
 *
 * See the `ATNConfig` documentation for more information about conserving memory through the use of several concrete
 * types.
 */
let ActionSemanticContextATNConfig = class ActionSemanticContextATNConfig extends SemanticContextATNConfig {
    constructor(lexerActionExecutor, semanticContext, state, altOrConfig, context, passedThroughNonGreedyDecision) {
        if (typeof altOrConfig === "number") {
            super(semanticContext, state, altOrConfig, context);
        }
        else {
            super(semanticContext, state, altOrConfig, context);
        }
        this._lexerActionExecutor = lexerActionExecutor;
        this.passedThroughNonGreedyDecision = passedThroughNonGreedyDecision;
    }
    get lexerActionExecutor() {
        return this._lexerActionExecutor;
    }
    get hasPassedThroughNonGreedyDecision() {
        return this.passedThroughNonGreedyDecision;
    }
};
__decorate([
    Decorators_1.Override
], ActionSemanticContextATNConfig.prototype, "lexerActionExecutor", null);
__decorate([
    Decorators_1.Override
], ActionSemanticContextATNConfig.prototype, "hasPassedThroughNonGreedyDecision", null);
ActionSemanticContextATNConfig = __decorate([
    __param(1, Decorators_1.NotNull), __param(2, Decorators_1.NotNull)
], ActionSemanticContextATNConfig);
//# sourceMappingURL=ATNConfig.js.map