/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { ATN } from "./ATN";
import { DFAState } from "../dfa/DFAState";
export declare abstract class ATNSimulator {
    /** Must distinguish between missing edge and edge we know leads nowhere */
    private static _ERROR;
    static get ERROR(): DFAState;
    atn: ATN;
    constructor(atn: ATN);
    abstract reset(): void;
    /**
     * Clear the DFA cache used by the current instance. Since the DFA cache may
     * be shared by multiple ATN simulators, this method may affect the
     * performance (but not accuracy) of other parsers which are being used
     * concurrently.
     *
     * @ if the current instance does not
     * support clearing the DFA.
     *
     * @since 4.3
     */
    clearDFA(): void;
}
export declare namespace ATNSimulator {
}
