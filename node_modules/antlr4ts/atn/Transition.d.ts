/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { ATNState } from "./ATNState";
import { IntervalSet } from "../misc/IntervalSet";
import { TransitionType } from "./TransitionType";
/** An ATN transition between any two ATN states.  Subclasses define
 *  atom, set, epsilon, action, predicate, rule transitions.
 *
 *  This is a one way link.  It emanates from a state (usually via a list of
 *  transitions) and has a target state.
 *
 *  Since we never have to change the ATN transitions once we construct it,
 *  we can fix these transitions as specific classes. The DFA transitions
 *  on the other hand need to update the labels as it adds transitions to
 *  the states. We'll use the term Edge for the DFA to distinguish them from
 *  ATN transitions.
 */
export declare abstract class Transition {
    static readonly serializationNames: string[];
    /** The target of this transition. */
    target: ATNState;
    constructor(target: ATNState);
    abstract readonly serializationType: TransitionType;
    /**
     * Determines if the transition is an "epsilon" transition.
     *
     * The default implementation returns `false`.
     *
     * @returns `true` if traversing this transition in the ATN does not
     * consume an input symbol; otherwise, `false` if traversing this
     * transition consumes (matches) an input symbol.
     */
    get isEpsilon(): boolean;
    get label(): IntervalSet | undefined;
    abstract matches(symbol: number, minVocabSymbol: number, maxVocabSymbol: number): boolean;
}
