/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { ATNState } from "./ATNState";
import { ATNStateType } from "./ATNStateType";
import { RuleStopState } from "./RuleStopState";
export declare class RuleStartState extends ATNState {
    stopState: RuleStopState;
    isPrecedenceRule: boolean;
    leftFactored: boolean;
    get stateType(): ATNStateType;
}
