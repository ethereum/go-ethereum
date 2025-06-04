/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { ATNState } from "./ATNState";
import { IntervalSet } from "../misc/IntervalSet";
import { Transition } from "./Transition";
import { TransitionType } from "./TransitionType";
/** TODO: make all transitions sets? no, should remove set edges */
export declare class AtomTransition extends Transition {
    /** The token type or character value; or, signifies special label. */
    _label: number;
    constructor(target: ATNState, label: number);
    get serializationType(): TransitionType;
    get label(): IntervalSet;
    matches(symbol: number, minVocabSymbol: number, maxVocabSymbol: number): boolean;
    toString(): string;
}
