/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { AbstractPredicateTransition } from "./AbstractPredicateTransition";
import { ATNState } from "./ATNState";
import { SemanticContext } from "./SemanticContext";
import { TransitionType } from "./TransitionType";
/** TODO: this is old comment:
 *  A tree of semantic predicates from the grammar AST if label==SEMPRED.
 *  In the ATN, labels will always be exactly one predicate, but the DFA
 *  may have to combine a bunch of them as it collects predicates from
 *  multiple ATN configurations into a single DFA state.
 */
export declare class PredicateTransition extends AbstractPredicateTransition {
    ruleIndex: number;
    predIndex: number;
    isCtxDependent: boolean;
    constructor(target: ATNState, ruleIndex: number, predIndex: number, isCtxDependent: boolean);
    get serializationType(): TransitionType;
    get isEpsilon(): boolean;
    matches(symbol: number, minVocabSymbol: number, maxVocabSymbol: number): boolean;
    get predicate(): SemanticContext.Predicate;
    toString(): string;
}
