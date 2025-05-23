/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { ATNStateType } from "./ATNStateType";
import { BlockStartState } from "./BlockStartState";
/**
 *
 * @author Sam Harwell
 */
export declare class BasicBlockStartState extends BlockStartState {
    get stateType(): ATNStateType;
}
