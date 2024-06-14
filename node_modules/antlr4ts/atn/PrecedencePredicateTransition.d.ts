/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { AbstractPredicateTransition } from "./AbstractPredicateTransition";
import { ATNState } from "./ATNState";
import { SemanticContext } from "./SemanticContext";
import { TransitionType } from "./TransitionType";
/**
 *
 * @author Sam Harwell
 */
export declare class PrecedencePredicateTransition extends AbstractPredicateTransition {
    precedence: number;
    constructor(target: ATNState, precedence: number);
    get serializationType(): TransitionType;
    get isEpsilon(): boolean;
    matches(symbol: number, minVocabSymbol: number, maxVocabSymbol: number): boolean;
    get predicate(): SemanticContext.PrecedencePredicate;
    toString(): string;
}
