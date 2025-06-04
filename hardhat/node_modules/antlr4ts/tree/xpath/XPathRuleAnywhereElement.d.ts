/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { ParseTree } from "../ParseTree";
import { XPathElement } from "./XPathElement";
/**
 * Either `ID` at start of path or `...//ID` in middle of path.
 */
export declare class XPathRuleAnywhereElement extends XPathElement {
    protected ruleIndex: number;
    constructor(ruleName: string, ruleIndex: number);
    evaluate(t: ParseTree): ParseTree[];
}
