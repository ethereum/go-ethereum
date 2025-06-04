/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { DecisionEventInfo } from "./DecisionEventInfo";
import { SimulatorState } from "./SimulatorState";
import { TokenStream } from "../TokenStream";
/**
 * This class represents profiling event information for tracking the lookahead
 * depth required in order to make a prediction.
 *
 * @since 4.3
 */
export declare class LookaheadEventInfo extends DecisionEventInfo {
    /** The alternative chosen by adaptivePredict(), not necessarily
     *  the outermost alt shown for a rule; left-recursive rules have
     *  user-level alts that differ from the rewritten rule with a (...) block
     *  and a (..)* loop.
     */
    predictedAlt: number;
    /**
     * Constructs a new instance of the {@link LookaheadEventInfo} class with
     * the specified detailed lookahead information.
     *
     * @param decision The decision number
     * @param state The final simulator state containing the necessary
     * information to determine the result of a prediction, or `undefined` if
     * the final state is not available
     * @param input The input token stream
     * @param startIndex The start index for the current prediction
     * @param stopIndex The index at which the prediction was finally made
     * @param fullCtx `true` if the current lookahead is part of an LL
     * prediction; otherwise, `false` if the current lookahead is part of
     * an SLL prediction
     */
    constructor(decision: number, state: SimulatorState | undefined, predictedAlt: number, input: TokenStream, startIndex: number, stopIndex: number, fullCtx: boolean);
}
