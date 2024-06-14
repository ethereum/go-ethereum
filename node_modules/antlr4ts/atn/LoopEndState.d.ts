/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { ATNState } from "./ATNState";
import { ATNStateType } from "./ATNStateType";
/** Mark the end of a * or + loop. */
export declare class LoopEndState extends ATNState {
    loopBackState: ATNState;
    get stateType(): ATNStateType;
}
