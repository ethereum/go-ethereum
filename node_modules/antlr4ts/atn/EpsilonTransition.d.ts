/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { ATNState } from "./ATNState";
import { Transition } from "./Transition";
import { TransitionType } from "./TransitionType";
export declare class EpsilonTransition extends Transition {
    private _outermostPrecedenceReturn;
    constructor(target: ATNState, outermostPrecedenceReturn?: number);
    /**
     * @returns the rule index of a precedence rule for which this transition is
     * returning from, where the precedence value is 0; otherwise, -1.
     *
     * @see ATNConfig.isPrecedenceFilterSuppressed
     * @see ParserATNSimulator#applyPrecedenceFilter(ATNConfigSet, ParserRuleContext, PredictionContextCache)
     * @since 4.4.1
     */
    get outermostPrecedenceReturn(): number;
    get serializationType(): TransitionType;
    get isEpsilon(): boolean;
    matches(symbol: number, minVocabSymbol: number, maxVocabSymbol: number): boolean;
    toString(): string;
}
