/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { ATNConfigSet } from "./atn/ATNConfigSet";
import { BitSet } from "./misc/BitSet";
import { DFA } from "./dfa/DFA";
import { Parser } from "./Parser";
import { ProxyErrorListener } from "./ProxyErrorListener";
import { ParserErrorListener } from "./ParserErrorListener";
import { SimulatorState } from "./atn/SimulatorState";
import { Token } from "./Token";
/**
 * @author Sam Harwell
 */
export declare class ProxyParserErrorListener extends ProxyErrorListener<Token, ParserErrorListener> implements ParserErrorListener {
    constructor(delegates: ParserErrorListener[]);
    reportAmbiguity(recognizer: Parser, dfa: DFA, startIndex: number, stopIndex: number, exact: boolean, ambigAlts: BitSet | undefined, configs: ATNConfigSet): void;
    reportAttemptingFullContext(recognizer: Parser, dfa: DFA, startIndex: number, stopIndex: number, conflictingAlts: BitSet | undefined, conflictState: SimulatorState): void;
    reportContextSensitivity(recognizer: Parser, dfa: DFA, startIndex: number, stopIndex: number, prediction: number, acceptState: SimulatorState): void;
}
