/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { ATN } from "../atn/ATN";
import { Vocabulary } from "../Vocabulary";
export declare namespace InterpreterDataReader {
    /**
     * The structure of the data file is very simple. Everything is line based with empty lines
     * separating the different parts. For lexers the layout is:
     * token literal names:
     * ...
     *
     * token symbolic names:
     * ...
     *
     * rule names:
     * ...
     *
     * channel names:
     * ...
     *
     * mode names:
     * ...
     *
     * atn:
     * <a single line with comma separated int values> enclosed in a pair of squared brackets.
     *
     * Data for a parser does not contain channel and mode names.
     */
    function parseFile(fileName: string): Promise<InterpreterDataReader.InterpreterData>;
    class InterpreterData {
        atn?: ATN;
        vocabulary: Vocabulary;
        ruleNames: string[];
        channels?: string[];
        modes?: string[];
    }
}
