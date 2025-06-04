/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { DFAState } from "../dfa/DFAState";
import { ParserRuleContext } from "../ParserRuleContext";
/**
 *
 * @author Sam Harwell
 */
export declare class SimulatorState {
    outerContext: ParserRuleContext;
    s0: DFAState;
    useContext: boolean;
    remainingOuterContext: ParserRuleContext | undefined;
    constructor(outerContext: ParserRuleContext, s0: DFAState, useContext: boolean, remainingOuterContext: ParserRuleContext | undefined);
}
