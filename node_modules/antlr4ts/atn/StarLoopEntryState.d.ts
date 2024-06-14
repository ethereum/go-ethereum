/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { ATNStateType } from "./ATNStateType";
import { BitSet } from "../misc/BitSet";
import { DecisionState } from "./DecisionState";
import { StarLoopbackState } from "./StarLoopbackState";
export declare class StarLoopEntryState extends DecisionState {
    loopBackState: StarLoopbackState;
    /**
     * Indicates whether this state can benefit from a precedence DFA during SLL
     * decision making.
     *
     * This is a computed property that is calculated during ATN deserialization
     * and stored for use in {@link ParserATNSimulator} and
     * {@link ParserInterpreter}.
     *
     * @see `DFA.isPrecedenceDfa`
     */
    precedenceRuleDecision: boolean;
    /**
     * For precedence decisions, this set marks states *S* which have all
     * of the following characteristics:
     *
     * * One or more invocation sites of the current rule returns to
     *   *S*.
     * * The closure from *S* includes the current decision without
     *   passing through any rule invocations or stepping out of the current
     *   rule.
     *
     * This field is not used when {@link #precedenceRuleDecision} is
     * `false`.
     */
    precedenceLoopbackStates: BitSet;
    get stateType(): ATNStateType;
}
