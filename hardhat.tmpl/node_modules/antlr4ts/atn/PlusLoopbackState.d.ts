/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { ATNStateType } from "./ATNStateType";
import { DecisionState } from "./DecisionState";
/** Decision state for `A+` and `(A|B)+`.  It has two transitions:
 *  one to the loop back to start of the block and one to exit.
 */
export declare class PlusLoopbackState extends DecisionState {
    get stateType(): ATNStateType;
}
