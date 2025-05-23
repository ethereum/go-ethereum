/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { ANTLRErrorListener } from "./ANTLRErrorListener";
import { ATNConfigSet } from "./atn/ATNConfigSet";
import { BitSet } from "./misc/BitSet";
import { DFA } from "./dfa/DFA";
import { Parser } from "./Parser";
import { SimulatorState } from "./atn/SimulatorState";
import { Token } from "./Token";
/** How to emit recognition errors for parsers.
 */
export interface ParserErrorListener extends ANTLRErrorListener<Token> {
    /**
     * This method is called by the parser when a full-context prediction
     * results in an ambiguity.
     *
     * Each full-context prediction which does not result in a syntax error
     * will call either {@link #reportContextSensitivity} or
     * {@link #reportAmbiguity}.
     *
     * When `ambigAlts` is not `undefined`, it contains the set of potentially
     * viable alternatives identified by the prediction algorithm. When
     * `ambigAlts` is `undefined`, use
     * {@link ATNConfigSet#getRepresentedAlternatives} to obtain the represented
     * alternatives from the `configs` argument.
     *
     * When `exact` is `true`, *all* of the potentially
     * viable alternatives are truly viable, i.e. this is reporting an exact
     * ambiguity. When `exact` is `false`, *at least two* of
     * the potentially viable alternatives are viable for the current input, but
     * the prediction algorithm terminated as soon as it determined that at
     * least the *minimum* potentially viable alternative is truly
     * viable.
     *
     * When the {@link PredictionMode#LL_EXACT_AMBIG_DETECTION} prediction
     * mode is used, the parser is required to identify exact ambiguities so
     * `exact` will always be `true`.
     *
     * @param recognizer the parser instance
     * @param dfa the DFA for the current decision
     * @param startIndex the input index where the decision started
     * @param stopIndex the input input where the ambiguity was identified
     * @param exact `true` if the ambiguity is exactly known, otherwise
     * `false`. This is always `true` when
     * {@link PredictionMode#LL_EXACT_AMBIG_DETECTION} is used.
     * @param ambigAlts the potentially ambiguous alternatives, or `undefined`
     * to indicate that the potentially ambiguous alternatives are the complete
     * set of represented alternatives in `configs`
     * @param configs the ATN configuration set where the ambiguity was
     * identified
     */
    reportAmbiguity?: (recognizer: Parser, dfa: DFA, startIndex: number, stopIndex: number, exact: boolean, ambigAlts: BitSet | undefined, configs: ATNConfigSet) => void;
    /**
     * This method is called when an SLL conflict occurs and the parser is about
     * to use the full context information to make an LL decision.
     *
     * If one or more configurations in `configs` contains a semantic
     * predicate, the predicates are evaluated before this method is called. The
     * subset of alternatives which are still viable after predicates are
     * evaluated is reported in `conflictingAlts`.
     *
     * @param recognizer the parser instance
     * @param dfa the DFA for the current decision
     * @param startIndex the input index where the decision started
     * @param stopIndex the input index where the SLL conflict occurred
     * @param conflictingAlts The specific conflicting alternatives. If this is
     * `undefined`, the conflicting alternatives are all alternatives
     * represented in `configs`.
     * @param conflictState the simulator state when the SLL conflict was
     * detected
     */
    reportAttemptingFullContext?: (recognizer: Parser, dfa: DFA, startIndex: number, stopIndex: number, conflictingAlts: BitSet | undefined, conflictState: SimulatorState) => void;
    /**
     * This method is called by the parser when a full-context prediction has a
     * unique result.
     *
     * Each full-context prediction which does not result in a syntax error
     * will call either {@link #reportContextSensitivity} or
     * {@link #reportAmbiguity}.
     *
     * For prediction implementations that only evaluate full-context
     * predictions when an SLL conflict is found (including the default
     * {@link ParserATNSimulator} implementation), this method reports cases
     * where SLL conflicts were resolved to unique full-context predictions,
     * i.e. the decision was context-sensitive. This report does not necessarily
     * indicate a problem, and it may appear even in completely unambiguous
     * grammars.
     *
     * `configs` may have more than one represented alternative if the
     * full-context prediction algorithm does not evaluate predicates before
     * beginning the full-context prediction. In all cases, the final prediction
     * is passed as the `prediction` argument.
     *
     * Note that the definition of "context sensitivity" in this method
     * differs from the concept in {@link DecisionInfo#contextSensitivities}.
     * This method reports all instances where an SLL conflict occurred but LL
     * parsing produced a unique result, whether or not that unique result
     * matches the minimum alternative in the SLL conflicting set.
     *
     * @param recognizer the parser instance
     * @param dfa the DFA for the current decision
     * @param startIndex the input index where the decision started
     * @param stopIndex the input index where the context sensitivity was
     * finally determined
     * @param prediction the unambiguous result of the full-context prediction
     * @param acceptState the simulator state when the unambiguous prediction
     * was determined
     */
    reportContextSensitivity?: (recognizer: Parser, dfa: DFA, startIndex: number, stopIndex: number, prediction: number, acceptState: SimulatorState) => void;
}
