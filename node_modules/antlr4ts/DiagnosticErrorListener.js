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
exports.DiagnosticErrorListener = void 0;
const BitSet_1 = require("./misc/BitSet");
const Decorators_1 = require("./Decorators");
const Interval_1 = require("./misc/Interval");
/**
 * This implementation of {@link ANTLRErrorListener} can be used to identify
 * certain potential correctness and performance problems in grammars. "Reports"
 * are made by calling {@link Parser#notifyErrorListeners} with the appropriate
 * message.
 *
 * * **Ambiguities**: These are cases where more than one path through the
 *   grammar can match the input.
 * * **Weak context sensitivity**: These are cases where full-context
 *   prediction resolved an SLL conflict to a unique alternative which equaled the
 *   minimum alternative of the SLL conflict.
 * * **Strong (forced) context sensitivity**: These are cases where the
 *   full-context prediction resolved an SLL conflict to a unique alternative,
 *   *and* the minimum alternative of the SLL conflict was found to not be
 *   a truly viable alternative. Two-stage parsing cannot be used for inputs where
 *   this situation occurs.
 *
 * @author Sam Harwell
 */
class DiagnosticErrorListener {
    /**
     * Initializes a new instance of {@link DiagnosticErrorListener}, specifying
     * whether all ambiguities or only exact ambiguities are reported.
     *
     * @param exactOnly `true` to report only exact ambiguities, otherwise
     * `false` to report all ambiguities.  Defaults to true.
     */
    constructor(exactOnly = true) {
        this.exactOnly = exactOnly;
        this.exactOnly = exactOnly;
    }
    syntaxError(
    /*@NotNull*/
    recognizer, offendingSymbol, line, charPositionInLine, 
    /*@NotNull*/
    msg, e) {
        // intentionally empty
    }
    reportAmbiguity(recognizer, dfa, startIndex, stopIndex, exact, ambigAlts, configs) {
        if (this.exactOnly && !exact) {
            return;
        }
        let decision = this.getDecisionDescription(recognizer, dfa);
        let conflictingAlts = this.getConflictingAlts(ambigAlts, configs);
        let text = recognizer.inputStream.getText(Interval_1.Interval.of(startIndex, stopIndex));
        let message = `reportAmbiguity d=${decision}: ambigAlts=${conflictingAlts}, input='${text}'`;
        recognizer.notifyErrorListeners(message);
    }
    reportAttemptingFullContext(recognizer, dfa, startIndex, stopIndex, conflictingAlts, conflictState) {
        let format = "reportAttemptingFullContext d=%s, input='%s'";
        let decision = this.getDecisionDescription(recognizer, dfa);
        let text = recognizer.inputStream.getText(Interval_1.Interval.of(startIndex, stopIndex));
        let message = `reportAttemptingFullContext d=${decision}, input='${text}'`;
        recognizer.notifyErrorListeners(message);
    }
    reportContextSensitivity(recognizer, dfa, startIndex, stopIndex, prediction, acceptState) {
        let format = "reportContextSensitivity d=%s, input='%s'";
        let decision = this.getDecisionDescription(recognizer, dfa);
        let text = recognizer.inputStream.getText(Interval_1.Interval.of(startIndex, stopIndex));
        let message = `reportContextSensitivity d=${decision}, input='${text}'`;
        recognizer.notifyErrorListeners(message);
    }
    getDecisionDescription(recognizer, dfa) {
        let decision = dfa.decision;
        let ruleIndex = dfa.atnStartState.ruleIndex;
        let ruleNames = recognizer.ruleNames;
        if (ruleIndex < 0 || ruleIndex >= ruleNames.length) {
            return decision.toString();
        }
        let ruleName = ruleNames[ruleIndex];
        if (!ruleName) {
            return decision.toString();
        }
        return `${decision} (${ruleName})`;
    }
    /**
     * Computes the set of conflicting or ambiguous alternatives from a
     * configuration set, if that information was not already provided by the
     * parser.
     *
     * @param reportedAlts The set of conflicting or ambiguous alternatives, as
     * reported by the parser.
     * @param configs The conflicting or ambiguous configuration set.
     * @returns Returns `reportedAlts` if it is not `undefined`, otherwise
     * returns the set of alternatives represented in `configs`.
     */
    getConflictingAlts(reportedAlts, configs) {
        if (reportedAlts != null) {
            return reportedAlts;
        }
        let result = new BitSet_1.BitSet();
        for (let config of configs) {
            result.set(config.alt);
        }
        return result;
    }
}
__decorate([
    Decorators_1.Override
], DiagnosticErrorListener.prototype, "syntaxError", null);
__decorate([
    Decorators_1.Override,
    __param(0, Decorators_1.NotNull),
    __param(1, Decorators_1.NotNull),
    __param(6, Decorators_1.NotNull)
], DiagnosticErrorListener.prototype, "reportAmbiguity", null);
__decorate([
    Decorators_1.Override,
    __param(0, Decorators_1.NotNull),
    __param(1, Decorators_1.NotNull),
    __param(5, Decorators_1.NotNull)
], DiagnosticErrorListener.prototype, "reportAttemptingFullContext", null);
__decorate([
    Decorators_1.Override,
    __param(0, Decorators_1.NotNull),
    __param(1, Decorators_1.NotNull),
    __param(5, Decorators_1.NotNull)
], DiagnosticErrorListener.prototype, "reportContextSensitivity", null);
__decorate([
    __param(0, Decorators_1.NotNull),
    __param(1, Decorators_1.NotNull)
], DiagnosticErrorListener.prototype, "getDecisionDescription", null);
__decorate([
    Decorators_1.NotNull,
    __param(1, Decorators_1.NotNull)
], DiagnosticErrorListener.prototype, "getConflictingAlts", null);
exports.DiagnosticErrorListener = DiagnosticErrorListener;
//# sourceMappingURL=DiagnosticErrorListener.js.map