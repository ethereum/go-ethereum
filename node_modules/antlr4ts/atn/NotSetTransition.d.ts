/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { ATNState } from "./ATNState";
import { IntervalSet } from "../misc/IntervalSet";
import { SetTransition } from "./SetTransition";
import { TransitionType } from "./TransitionType";
export declare class NotSetTransition extends SetTransition {
    constructor(target: ATNState, set: IntervalSet);
    get serializationType(): TransitionType;
    matches(symbol: number, minVocabSymbol: number, maxVocabSymbol: number): boolean;
    toString(): string;
}
