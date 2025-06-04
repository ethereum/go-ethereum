"use strict";
/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
var __decorate = (this && this.__decorate) || function (decorators, target, key, desc) {
    var c = arguments.length, r = c < 3 ? target : desc === null ? desc = Object.getOwnPropertyDescriptor(target, key) : desc, d;
    if (typeof Reflect === "object" && typeof Reflect.decorate === "function") r = Reflect.decorate(decorators, target, key, desc);
    else for (var i = decorators.length - 1; i >= 0; i--) if (d = decorators[i]) r = (c < 3 ? d(r) : c > 3 ? d(target, key, r) : d(target, key)) || r;
    return c > 3 && r && Object.defineProperty(target, key, r), r;
};
var __param = (this && this.__param) || function (paramIndex, decorator) {
    return function (target, key) { decorator(target, key, paramIndex); }
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.Trees = void 0;
const ATN_1 = require("../atn/ATN");
const CommonToken_1 = require("../CommonToken");
const ErrorNode_1 = require("./ErrorNode");
const Decorators_1 = require("../Decorators");
const Parser_1 = require("../Parser");
const ParserRuleContext_1 = require("../ParserRuleContext");
const RuleNode_1 = require("./RuleNode");
const TerminalNode_1 = require("./TerminalNode");
const Token_1 = require("../Token");
const Utils = require("../misc/Utils");
/** A set of utility routines useful for all kinds of ANTLR trees. */
class Trees {
    static toStringTree(t, arg2) {
        let ruleNames;
        if (arg2 instanceof Parser_1.Parser) {
            ruleNames = arg2.ruleNames;
        }
        else {
            ruleNames = arg2;
        }
        let s = Utils.escapeWhitespace(this.getNodeText(t, ruleNames), false);
        if (t.childCount === 0) {
            return s;
        }
        let buf = "";
        buf += ("(");
        s = Utils.escapeWhitespace(this.getNodeText(t, ruleNames), false);
        buf += (s);
        buf += (" ");
        for (let i = 0; i < t.childCount; i++) {
            if (i > 0) {
                buf += (" ");
            }
            buf += (this.toStringTree(t.getChild(i), ruleNames));
        }
        buf += (")");
        return buf;
    }
    static getNodeText(t, arg2) {
        let ruleNames;
        if (arg2 instanceof Parser_1.Parser) {
            ruleNames = arg2.ruleNames;
        }
        else if (arg2) {
            ruleNames = arg2;
        }
        else {
            // no recog or rule names
            let payload = t.payload;
            if (typeof payload.text === "string") {
                return payload.text;
            }
            return t.payload.toString();
        }
        if (t instanceof RuleNode_1.RuleNode) {
            let ruleContext = t.ruleContext;
            let ruleIndex = ruleContext.ruleIndex;
            let ruleName = ruleNames[ruleIndex];
            let altNumber = ruleContext.altNumber;
            if (altNumber !== ATN_1.ATN.INVALID_ALT_NUMBER) {
                return ruleName + ":" + altNumber;
            }
            return ruleName;
        }
        else if (t instanceof ErrorNode_1.ErrorNode) {
            return t.toString();
        }
        else if (t instanceof TerminalNode_1.TerminalNode) {
            let symbol = t.symbol;
            return symbol.text || "";
        }
        throw new TypeError("Unexpected node type");
    }
    static getChildren(t) {
        let kids = [];
        for (let i = 0; i < t.childCount; i++) {
            kids.push(t.getChild(i));
        }
        return kids;
    }
    static getAncestors(t) {
        let ancestors = [];
        let p = t.parent;
        while (p) {
            ancestors.unshift(p); // insert at start
            p = p.parent;
        }
        return ancestors;
    }
    /** Return true if t is u's parent or a node on path to root from u.
     *  Use === not equals().
     *
     *  @since 4.5.1
     */
    static isAncestorOf(t, u) {
        if (!t || !u || !t.parent) {
            return false;
        }
        let p = u.parent;
        while (p) {
            if (t === p) {
                return true;
            }
            p = p.parent;
        }
        return false;
    }
    static findAllTokenNodes(t, ttype) {
        return Trees.findAllNodes(t, ttype, true);
    }
    static findAllRuleNodes(t, ruleIndex) {
        return Trees.findAllNodes(t, ruleIndex, false);
    }
    static findAllNodes(t, index, findTokens) {
        let nodes = [];
        Trees._findAllNodes(t, index, findTokens, nodes);
        return nodes;
    }
    static _findAllNodes(t, index, findTokens, nodes) {
        // check this node (the root) first
        if (findTokens && t instanceof TerminalNode_1.TerminalNode) {
            if (t.symbol.type === index) {
                nodes.push(t);
            }
        }
        else if (!findTokens && t instanceof ParserRuleContext_1.ParserRuleContext) {
            if (t.ruleIndex === index) {
                nodes.push(t);
            }
        }
        // check children
        for (let i = 0; i < t.childCount; i++) {
            Trees._findAllNodes(t.getChild(i), index, findTokens, nodes);
        }
    }
    /** Get all descendents; includes t itself.
     *
     * @since 4.5.1
     */
    static getDescendants(t) {
        let nodes = [];
        function recurse(e) {
            nodes.push(e);
            const n = e.childCount;
            for (let i = 0; i < n; i++) {
                recurse(e.getChild(i));
            }
        }
        recurse(t);
        return nodes;
    }
    /** Find smallest subtree of t enclosing range startTokenIndex..stopTokenIndex
     *  inclusively using postorder traversal.  Recursive depth-first-search.
     *
     *  @since 4.5
     */
    static getRootOfSubtreeEnclosingRegion(t, startTokenIndex, // inclusive
    stopTokenIndex) {
        let n = t.childCount;
        for (let i = 0; i < n; i++) {
            let child = t.getChild(i);
            let r = Trees.getRootOfSubtreeEnclosingRegion(child, startTokenIndex, stopTokenIndex);
            if (r) {
                return r;
            }
        }
        if (t instanceof ParserRuleContext_1.ParserRuleContext) {
            let stopToken = t.stop;
            if (startTokenIndex >= t.start.tokenIndex && // is range fully contained in t?
                (stopToken == null || stopTokenIndex <= stopToken.tokenIndex)) {
                // note: r.stop==null likely implies that we bailed out of parser and there's nothing to the right
                return t;
            }
        }
        return undefined;
    }
    /** Replace any subtree siblings of root that are completely to left
     *  or right of lookahead range with a CommonToken(Token.INVALID_TYPE,"...")
     *  node. The source interval for t is not altered to suit smaller range!
     *
     *  WARNING: destructive to t.
     *
     *  @since 4.5.1
     */
    static stripChildrenOutOfRange(t, root, startIndex, stopIndex) {
        if (!t) {
            return;
        }
        let count = t.childCount;
        for (let i = 0; i < count; i++) {
            let child = t.getChild(i);
            let range = child.sourceInterval;
            if (child instanceof ParserRuleContext_1.ParserRuleContext && (range.b < startIndex || range.a > stopIndex)) {
                if (Trees.isAncestorOf(child, root)) { // replace only if subtree doesn't have displayed root
                    let abbrev = new CommonToken_1.CommonToken(Token_1.Token.INVALID_TYPE, "...");
                    t.children[i] = new TerminalNode_1.TerminalNode(abbrev); // HACK access to private
                }
            }
        }
    }
    static findNodeSuchThat(t, pred) {
        // No type check needed as long as users only use one of the available overloads
        if (pred(t)) {
            return t;
        }
        let n = t.childCount;
        for (let i = 0; i < n; i++) {
            let u = Trees.findNodeSuchThat(t.getChild(i), pred);
            if (u !== undefined) {
                return u;
            }
        }
        return undefined;
    }
}
__decorate([
    __param(0, Decorators_1.NotNull)
], Trees, "toStringTree", null);
__decorate([
    Decorators_1.NotNull,
    __param(0, Decorators_1.NotNull)
], Trees, "getAncestors", null);
__decorate([
    __param(0, Decorators_1.NotNull)
], Trees, "getRootOfSubtreeEnclosingRegion", null);
exports.Trees = Trees;
//# sourceMappingURL=Trees.js.map