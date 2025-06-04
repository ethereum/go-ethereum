/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { ATNState } from "./ATNState";
import { ATNStateType } from "./ATNStateType";
import { StarLoopEntryState } from "./StarLoopEntryState";
export declare class StarLoopbackState extends ATNState {
    get loopEntryState(): StarLoopEntryState;
    get stateType(): ATNStateType;
}
