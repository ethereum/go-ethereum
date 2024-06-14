/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { ATNState } from "./ATNState";
import { Transition } from "./Transition";
import { TransitionType } from "./TransitionType";
export declare class WildcardTransition extends Transition {
    constructor(target: ATNState);
    get serializationType(): TransitionType;
    matches(symbol: number, minVocabSymbol: number, maxVocabSymbol: number): boolean;
    toString(): string;
}
