/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { ATNState } from "./ATNState";
import { Transition } from "./Transition";
import { TransitionType } from "./TransitionType";
export declare class ActionTransition extends Transition {
    ruleIndex: number;
    actionIndex: number;
    isCtxDependent: boolean;
    constructor(target: ATNState, ruleIndex: number, actionIndex?: number, isCtxDependent?: boolean);
    get serializationType(): TransitionType;
    get isEpsilon(): boolean;
    matches(symbol: number, minVocabSymbol: number, maxVocabSymbol: number): boolean;
    toString(): string;
}
