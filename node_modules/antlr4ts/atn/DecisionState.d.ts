/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { ATNState } from "./ATNState";
export declare abstract class DecisionState extends ATNState {
    decision: number;
    nonGreedy: boolean;
    sll: boolean;
}
