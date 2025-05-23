/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { ATNState } from "./ATNState";
import { ATNStateType } from "./ATNStateType";
import { BlockStartState } from "./BlockStartState";
/** Terminal node of a simple `(a|b|c)` block. */
export declare class BlockEndState extends ATNState {
    startState: BlockStartState;
    get stateType(): ATNStateType;
}
