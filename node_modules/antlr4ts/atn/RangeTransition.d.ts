/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { ATNState } from "./ATNState";
import { IntervalSet } from "../misc/IntervalSet";
import { Transition } from "./Transition";
import { TransitionType } from "./TransitionType";
export declare class RangeTransition extends Transition {
    from: number;
    to: number;
    constructor(target: ATNState, from: number, to: number);
    get serializationType(): TransitionType;
    get label(): IntervalSet;
    matches(symbol: number, minVocabSymbol: number, maxVocabSymbol: number): boolean;
    toString(): string;
}
