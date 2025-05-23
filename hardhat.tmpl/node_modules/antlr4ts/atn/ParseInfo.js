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
exports.ParseInfo = void 0;
const Decorators_1 = require("../Decorators");
/**
 * This class provides access to specific and aggregate statistics gathered
 * during profiling of a parser.
 *
 * @since 4.3
 */
let ParseInfo = class ParseInfo {
    constructor(atnSimulator) {
        this.atnSimulator = atnSimulator;
    }
    /**
     * Gets an array of {@link DecisionInfo} instances containing the profiling
     * information gathered for each decision in the ATN.
     *
     * @returns An array of {@link DecisionInfo} instances, indexed by decision
     * number.
     */
    getDecisionInfo() {
        return this.atnSimulator.getDecisionInfo();
    }
    /**
     * Gets the decision numbers for decisions that required one or more
     * full-context predictions during parsing. These are decisions for which
     * {@link DecisionInfo#LL_Fallback} is non-zero.
     *
     * @returns A list of decision numbers which required one or more
     * full-context predictions during parsing.
     */
    getLLDecisions() {
        let decisions = this.atnSimulator.getDecisionInfo();
        let LL = [];
        for (let i = 0; i < decisions.length; i++) {
            let fallBack = decisions[i].LL_Fallback;
            if (fallBack > 0) {
                LL.push(i);
            }
        }
        return LL;
    }
    /**
     * Gets the total time spent during prediction across all decisions made
     * during parsing. This value is the sum of
     * {@link DecisionInfo#timeInPrediction} for all decisions.
     */
    getTotalTimeInPrediction() {
        let decisions = this.atnSimulator.getDecisionInfo();
        let t = 0;
        for (let decision of decisions) {
            t += decision.timeInPrediction;
        }
        return t;
    }
    /**
     * Gets the total number of SLL lookahead operations across all decisions
     * made during parsing. This value is the sum of
     * {@link DecisionInfo#SLL_TotalLook} for all decisions.
     */
    getTotalSLLLookaheadOps() {
        let decisions = this.atnSimulator.getDecisionInfo();
        let k = 0;
        for (let decision of decisions) {
            k += decision.SLL_TotalLook;
        }
        return k;
    }
    /**
     * Gets the total number of LL lookahead operations across all decisions
     * made during parsing. This value is the sum of
     * {@link DecisionInfo#LL_TotalLook} for all decisions.
     */
    getTotalLLLookaheadOps() {
        let decisions = this.atnSimulator.getDecisionInfo();
        let k = 0;
        for (let decision of decisions) {
            k += decision.LL_TotalLook;
        }
        return k;
    }
    /**
     * Gets the total number of ATN lookahead operations for SLL prediction
     * across all decisions made during parsing.
     */
    getTotalSLLATNLookaheadOps() {
        let decisions = this.atnSimulator.getDecisionInfo();
        let k = 0;
        for (let decision of decisions) {
            k += decision.SLL_ATNTransitions;
        }
        return k;
    }
    /**
     * Gets the total number of ATN lookahead operations for LL prediction
     * across all decisions made during parsing.
     */
    getTotalLLATNLookaheadOps() {
        let decisions = this.atnSimulator.getDecisionInfo();
        let k = 0;
        for (let decision of decisions) {
            k += decision.LL_ATNTransitions;
        }
        return k;
    }
    /**
     * Gets the total number of ATN lookahead operations for SLL and LL
     * prediction across all decisions made during parsing.
     *
     * This value is the sum of {@link #getTotalSLLATNLookaheadOps} and
     * {@link #getTotalLLATNLookaheadOps}.
     */
    getTotalATNLookaheadOps() {
        let decisions = this.atnSimulator.getDecisionInfo();
        let k = 0;
        for (let decision of decisions) {
            k += decision.SLL_ATNTransitions;
            k += decision.LL_ATNTransitions;
        }
        return k;
    }
    getDFASize(decision) {
        if (decision) {
            let decisionToDFA = this.atnSimulator.atn.decisionToDFA[decision];
            return decisionToDFA.states.size;
        }
        else {
            let n = 0;
            let decisionToDFA = this.atnSimulator.atn.decisionToDFA;
            for (let i = 0; i < decisionToDFA.length; i++) {
                n += this.getDFASize(i);
            }
            return n;
        }
    }
};
__decorate([
    Decorators_1.NotNull
], ParseInfo.prototype, "getDecisionInfo", null);
__decorate([
    Decorators_1.NotNull
], ParseInfo.prototype, "getLLDecisions", null);
ParseInfo = __decorate([
    __param(0, Decorators_1.NotNull)
], ParseInfo);
exports.ParseInfo = ParseInfo;
//# sourceMappingURL=ParseInfo.js.map