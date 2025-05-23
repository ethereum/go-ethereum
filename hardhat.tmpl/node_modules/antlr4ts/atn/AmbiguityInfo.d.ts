/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { BitSet } from "../misc/BitSet";
import { DecisionEventInfo } from "./DecisionEventInfo";
import { SimulatorState } from "./SimulatorState";
import { TokenStream } from "../TokenStream";
/**
 * This class represents profiling event information for an ambiguity.
 * Ambiguities are decisions where a particular input resulted in an SLL
 * conflict, followed by LL prediction also reaching a conflict state
 * (indicating a true ambiguity in the grammar).
 *
 * This event may be reported during SLL prediction in cases where the
 * conflicting SLL configuration set provides sufficient information to
 * determine that the SLL conflict is truly an ambiguity. For example, if none
 * of the ATN configurations in the conflicting SLL configuration set have
 * traversed a global follow transition (i.e.
 * {@link ATNConfig#getReachesIntoOuterContext} is `false` for all
 * configurations), then the result of SLL prediction for that input is known to
 * be equivalent to the result of LL prediction for that input.
 *
 * In some cases, the minimum represented alternative in the conflicting LL
 * configuration set is not equal to the minimum represented alternative in the
 * conflicting SLL configuration set. Grammars and inputs which result in this
 * scenario are unable to use {@link PredictionMode#SLL}, which in turn means
 * they cannot use the two-stage parsing strategy to improve parsing performance
 * for that input.
 *
 * @see ParserATNSimulator#reportAmbiguity
 * @see ParserErrorListener#reportAmbiguity
 *
 * @since 4.3
 */
export declare class AmbiguityInfo extends DecisionEventInfo {
    /** The set of alternative numbers for this decision event that lead to a valid parse. */
    private ambigAlts;
    /**
     * Constructs a new instance of the {@link AmbiguityInfo} class with the
     * specified detailed ambiguity information.
     *
     * @param decision The decision number
     * @param state The final simulator state identifying the ambiguous
     * alternatives for the current input
     * @param ambigAlts The set of alternatives in the decision that lead to a valid parse.
     *                  The predicted alt is the min(ambigAlts)
     * @param input The input token stream
     * @param startIndex The start index for the current prediction
     * @param stopIndex The index at which the ambiguity was identified during
     * prediction
     */
    constructor(decision: number, state: SimulatorState, ambigAlts: BitSet, input: TokenStream, startIndex: number, stopIndex: number);
    /**
     * Gets the set of alternatives in the decision that lead to a valid parse.
     *
     * @since 4.5
     */
    get ambiguousAlternatives(): BitSet;
}
