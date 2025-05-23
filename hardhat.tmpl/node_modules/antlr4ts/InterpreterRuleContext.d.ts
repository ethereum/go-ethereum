/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { ParserRuleContext } from "./ParserRuleContext";
/**
 * This class extends {@link ParserRuleContext} by allowing the value of
 * {@link #getRuleIndex} to be explicitly set for the context.
 *
 * {@link ParserRuleContext} does not include field storage for the rule index
 * since the context classes created by the code generator override the
 * {@link #getRuleIndex} method to return the correct value for that context.
 * Since the parser interpreter does not use the context classes generated for a
 * parser, this class (with slightly more memory overhead per node) is used to
 * provide equivalent functionality.
 */
export declare class InterpreterRuleContext extends ParserRuleContext {
    /**
     * This is the backing field for {@link #getRuleIndex}.
     */
    private _ruleIndex;
    constructor(ruleIndex: number);
    /**
     * Constructs a new {@link InterpreterRuleContext} with the specified
     * parent, invoking state, and rule index.
     *
     * @param ruleIndex The rule index for the current context.
     * @param parent The parent context.
     * @param invokingStateNumber The invoking state number.
     */
    constructor(ruleIndex: number, parent: ParserRuleContext | undefined, invokingStateNumber: number);
    get ruleIndex(): number;
}
