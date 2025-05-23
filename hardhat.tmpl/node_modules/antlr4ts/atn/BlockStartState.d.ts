/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { BlockEndState } from "./BlockEndState";
import { DecisionState } from "./DecisionState";
/**  The start of a regular `(...)` block. */
export declare abstract class BlockStartState extends DecisionState {
    endState: BlockEndState;
}
