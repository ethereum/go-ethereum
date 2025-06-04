/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { ATNConfigSet } from "./atn/ATNConfigSet";
import { BitSet } from "./misc/BitSet";
import { DFA } from "./dfa/DFA";
import { Parser } from "./Parser";
import { ParserErrorListener } from "./ParserErrorListener";
import { RecognitionException } from "./RecognitionException";
import { Recognizer } from "./Recognizer";
import { SimulatorState } from "./atn/SimulatorState";
import { Token } from "./Token";
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
export declare class DiagnosticErrorListener implements ParserErrorListener {
    protected exactOnly: boolean;
    /**
     * Initializes a new instance of {@link DiagnosticErrorListener}, specifying
     * whether all ambiguities or only exact ambiguities are reported.
     *
     * @param exactOnly `true` to report only exact ambiguities, otherwise
     * `false` to report all ambiguities.  Defaults to true.
     */
    constructor(exactOnly?: boolean);
    syntaxError<T extends Token>(recognizer: Recognizer<T, any>, offendingSymbol: T | undefined, line: number, charPositionInLine: number, msg: string, e: RecognitionException | undefined): void;
    reportAmbiguity(recognizer: Parser, dfa: DFA, startIndex: number, stopIndex: number, exact: boolean, ambigAlts: BitSet | undefined, configs: ATNConfigSet): void;
    reportAttemptingFullContext(recognizer: Parser, dfa: DFA, startIndex: number, stopIndex: number, conflictingAlts: BitSet | undefined, conflictState: SimulatorState): void;
    reportContextSensitivity(recognizer: Parser, dfa: DFA, startIndex: number, stopIndex: number, prediction: number, acceptState: SimulatorState): void;
    protected getDecisionDescription(recognizer: Parser, dfa: DFA): string;
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
    protected getConflictingAlts(reportedAlts: BitSet | undefined, configs: ATNConfigSet): BitSet;
}
