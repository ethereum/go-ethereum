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
Object.defineProperty(exports, "__esModule", { value: true });
exports.DecisionInfo = void 0;
const Decorators_1 = require("../Decorators");
/**
 * This class contains profiling gathered for a particular decision.
 *
 * Parsing performance in ANTLR 4 is heavily influenced by both static factors
 * (e.g. the form of the rules in the grammar) and dynamic factors (e.g. the
 * choice of input and the state of the DFA cache at the time profiling
 * operations are started). For best results, gather and use aggregate
 * statistics from a large sample of inputs representing the inputs expected in
 * production before using the results to make changes in the grammar.
 *
 * @since 4.3
 */
class DecisionInfo {
    /**
     * Constructs a new instance of the {@link DecisionInfo} class to contain
     * statistics for a particular decision.
     *
     * @param decision The decision number
     */
    constructor(decision) {
        /**
         * The total number of times {@link ParserATNSimulator#adaptivePredict} was
         * invoked for this decision.
         */
        this.invocations = 0;
        /**
         * The total time spent in {@link ParserATNSimulator#adaptivePredict} for
         * this decision, in nanoseconds.
         *
         * The value of this field contains the sum of differential results obtained
         * by {@link System#nanoTime()}, and is not adjusted to compensate for JIT
         * and/or garbage collection overhead. For best accuracy, use a modern JVM
         * implementation that provides precise results from
         * {@link System#nanoTime()}, and perform profiling in a separate process
         * which is warmed up by parsing the input prior to profiling. If desired,
         * call {@link ATNSimulator#clearDFA} to reset the DFA cache to its initial
         * state before starting the profiling measurement pass.
         */
        this.timeInPrediction = 0;
        /**
         * The sum of the lookahead required for SLL prediction for this decision.
         * Note that SLL prediction is used before LL prediction for performance
         * reasons even when {@link PredictionMode#LL} or
         * {@link PredictionMode#LL_EXACT_AMBIG_DETECTION} is used.
         */
        this.SLL_TotalLook = 0;
        /**
         * Gets the minimum lookahead required for any single SLL prediction to
         * complete for this decision, by reaching a unique prediction, reaching an
         * SLL conflict state, or encountering a syntax error.
         */
        this.SLL_MinLook = 0;
        /**
         * Gets the maximum lookahead required for any single SLL prediction to
         * complete for this decision, by reaching a unique prediction, reaching an
         * SLL conflict state, or encountering a syntax error.
         */
        this.SLL_MaxLook = 0;
        /**
         * The sum of the lookahead required for LL prediction for this decision.
         * Note that LL prediction is only used when SLL prediction reaches a
         * conflict state.
         */
        this.LL_TotalLook = 0;
        /**
         * Gets the minimum lookahead required for any single LL prediction to
         * complete for this decision. An LL prediction completes when the algorithm
         * reaches a unique prediction, a conflict state (for
         * {@link PredictionMode#LL}, an ambiguity state (for
         * {@link PredictionMode#LL_EXACT_AMBIG_DETECTION}, or a syntax error.
         */
        this.LL_MinLook = 0;
        /**
         * Gets the maximum lookahead required for any single LL prediction to
         * complete for this decision. An LL prediction completes when the algorithm
         * reaches a unique prediction, a conflict state (for
         * {@link PredictionMode#LL}, an ambiguity state (for
         * {@link PredictionMode#LL_EXACT_AMBIG_DETECTION}, or a syntax error.
         */
        this.LL_MaxLook = 0;
        /**
         * A collection of {@link ContextSensitivityInfo} instances describing the
         * context sensitivities encountered during LL prediction for this decision.
         *
         * @see ContextSensitivityInfo
         */
        this.contextSensitivities = [];
        /**
         * A collection of {@link ErrorInfo} instances describing the parse errors
         * identified during calls to {@link ParserATNSimulator#adaptivePredict} for
         * this decision.
         *
         * @see ErrorInfo
         */
        this.errors = [];
        /**
         * A collection of {@link AmbiguityInfo} instances describing the
         * ambiguities encountered during LL prediction for this decision.
         *
         * @see AmbiguityInfo
         */
        this.ambiguities = [];
        /**
         * A collection of {@link PredicateEvalInfo} instances describing the
         * results of evaluating individual predicates during prediction for this
         * decision.
         *
         * @see PredicateEvalInfo
         */
        this.predicateEvals = [];
        /**
         * The total number of ATN transitions required during SLL prediction for
         * this decision. An ATN transition is determined by the number of times the
         * DFA does not contain an edge that is required for prediction, resulting
         * in on-the-fly computation of that edge.
         *
         * If DFA caching of SLL transitions is employed by the implementation, ATN
         * computation may cache the computed edge for efficient lookup during
         * future parsing of this decision. Otherwise, the SLL parsing algorithm
         * will use ATN transitions exclusively.
         *
         * @see #SLL_ATNTransitions
         * @see ParserATNSimulator#computeTargetState
         * @see LexerATNSimulator#computeTargetState
         */
        this.SLL_ATNTransitions = 0;
        /**
         * The total number of DFA transitions required during SLL prediction for
         * this decision.
         *
         * If the ATN simulator implementation does not use DFA caching for SLL
         * transitions, this value will be 0.
         *
         * @see ParserATNSimulator#getExistingTargetState
         * @see LexerATNSimulator#getExistingTargetState
         */
        this.SLL_DFATransitions = 0;
        /**
         * Gets the total number of times SLL prediction completed in a conflict
         * state, resulting in fallback to LL prediction.
         *
         * Note that this value is not related to whether or not
         * {@link PredictionMode#SLL} may be used successfully with a particular
         * grammar. If the ambiguity resolution algorithm applied to the SLL
         * conflicts for this decision produce the same result as LL prediction for
         * this decision, {@link PredictionMode#SLL} would produce the same overall
         * parsing result as {@link PredictionMode#LL}.
         */
        this.LL_Fallback = 0;
        /**
         * The total number of ATN transitions required during LL prediction for
         * this decision. An ATN transition is determined by the number of times the
         * DFA does not contain an edge that is required for prediction, resulting
         * in on-the-fly computation of that edge.
         *
         * If DFA caching of LL transitions is employed by the implementation, ATN
         * computation may cache the computed edge for efficient lookup during
         * future parsing of this decision. Otherwise, the LL parsing algorithm will
         * use ATN transitions exclusively.
         *
         * @see #LL_DFATransitions
         * @see ParserATNSimulator#computeTargetState
         * @see LexerATNSimulator#computeTargetState
         */
        this.LL_ATNTransitions = 0;
        /**
         * The total number of DFA transitions required during LL prediction for
         * this decision.
         *
         * If the ATN simulator implementation does not use DFA caching for LL
         * transitions, this value will be 0.
         *
         * @see ParserATNSimulator#getExistingTargetState
         * @see LexerATNSimulator#getExistingTargetState
         */
        this.LL_DFATransitions = 0;
        this.decision = decision;
    }
    toString() {
        return "{" +
            "decision=" + this.decision +
            ", contextSensitivities=" + this.contextSensitivities.length +
            ", errors=" + this.errors.length +
            ", ambiguities=" + this.ambiguities.length +
            ", SLL_lookahead=" + this.SLL_TotalLook +
            ", SLL_ATNTransitions=" + this.SLL_ATNTransitions +
            ", SLL_DFATransitions=" + this.SLL_DFATransitions +
            ", LL_Fallback=" + this.LL_Fallback +
            ", LL_lookahead=" + this.LL_TotalLook +
            ", LL_ATNTransitions=" + this.LL_ATNTransitions +
            "}";
    }
}
__decorate([
    Decorators_1.Override
], DecisionInfo.prototype, "toString", null);
exports.DecisionInfo = DecisionInfo;
//# sourceMappingURL=DecisionInfo.js.map