/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { ParserRuleContext } from "./ParserRuleContext";
/** A handy class for use with
 *
 *  options {contextSuperClass=org.antlr.v4.runtime.RuleContextWithAltNum;}
 *
 *  that provides a backing field / impl for the outer alternative number
 *  matched for an internal parse tree node.
 *
 *  I'm only putting into Java runtime as I'm certain I'm the only one that
 *  will really every use this.
 */
export declare class RuleContextWithAltNum extends ParserRuleContext {
    private _altNumber;
    constructor();
    constructor(parent: ParserRuleContext | undefined, invokingStateNumber: number);
    get altNumber(): number;
    set altNumber(altNum: number);
}
