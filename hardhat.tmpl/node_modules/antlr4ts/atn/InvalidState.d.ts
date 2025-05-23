/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { ATNStateType } from "./ATNStateType";
import { BasicState } from "./BasicState";
/**
 *
 * @author Sam Harwell
 */
export declare class InvalidState extends BasicState {
    get stateType(): ATNStateType;
}
