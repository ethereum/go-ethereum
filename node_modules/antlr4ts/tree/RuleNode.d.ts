/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { RuleContext } from "../RuleContext";
import { ParseTree } from "./ParseTree";
import { ParseTreeVisitor } from "./ParseTreeVisitor";
import { Parser } from "../Parser";
import { Interval } from "../misc/Interval";
export declare abstract class RuleNode implements ParseTree {
    abstract readonly ruleContext: RuleContext;
    abstract readonly parent: RuleNode | undefined;
    abstract setParent(parent: RuleContext): void;
    abstract getChild(i: number): ParseTree;
    abstract accept<T>(visitor: ParseTreeVisitor<T>): T;
    abstract readonly text: string;
    abstract toStringTree(parser?: Parser | undefined): string;
    abstract readonly sourceInterval: Interval;
    abstract readonly payload: any;
    abstract readonly childCount: number;
}
