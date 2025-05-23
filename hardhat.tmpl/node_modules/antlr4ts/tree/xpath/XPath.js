"use strict";
/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
Object.defineProperty(exports, "__esModule", { value: true });
exports.XPath = void 0;
// ConvertTo-TS run at 2016-10-04T11:26:46.4373888-07:00
const CharStreams_1 = require("../../CharStreams");
const CommonTokenStream_1 = require("../../CommonTokenStream");
const LexerNoViableAltException_1 = require("../../LexerNoViableAltException");
const ParserRuleContext_1 = require("../../ParserRuleContext");
const Token_1 = require("../../Token");
const XPathLexer_1 = require("./XPathLexer");
const XPathLexerErrorListener_1 = require("./XPathLexerErrorListener");
const XPathRuleAnywhereElement_1 = require("./XPathRuleAnywhereElement");
const XPathRuleElement_1 = require("./XPathRuleElement");
const XPathTokenAnywhereElement_1 = require("./XPathTokenAnywhereElement");
const XPathTokenElement_1 = require("./XPathTokenElement");
const XPathWildcardAnywhereElement_1 = require("./XPathWildcardAnywhereElement");
const XPathWildcardElement_1 = require("./XPathWildcardElement");
/**
 * Represent a subset of XPath XML path syntax for use in identifying nodes in
 * parse trees.
 *
 * Split path into words and separators `/` and `//` via ANTLR
 * itself then walk path elements from left to right. At each separator-word
 * pair, find set of nodes. Next stage uses those as work list.
 *
 * The basic interface is
 * {@link XPath#findAll ParseTree.findAll}`(tree, pathString, parser)`.
 * But that is just shorthand for:
 *
 * ```
 * let p = new XPath(parser, pathString);
 * return p.evaluate(tree);
 * ```
 *
 * See `TestXPath` for descriptions. In short, this
 * allows operators:
 *
 * | | |
 * | --- | --- |
 * | `/` | root |
 * | `//` | anywhere |
 * | `!` | invert; this much appear directly after root or anywhere operator |
 *
 * and path elements:
 *
 * | | |
 * | --- | --- |
 * | `ID` | token name |
 * | `'string'` | any string literal token from the grammar |
 * | `expr` | rule name |
 * | `*` | wildcard matching any node |
 *
 * Whitespace is not allowed.
 */
class XPath {
    constructor(parser, path) {
        this.parser = parser;
        this.path = path;
        this.elements = this.split(path);
        // console.log(this.elements.toString());
    }
    // TODO: check for invalid token/rule names, bad syntax
    split(path) {
        let lexer = new XPathLexer_1.XPathLexer(CharStreams_1.CharStreams.fromString(path));
        lexer.recover = (e) => { throw e; };
        lexer.removeErrorListeners();
        lexer.addErrorListener(new XPathLexerErrorListener_1.XPathLexerErrorListener());
        let tokenStream = new CommonTokenStream_1.CommonTokenStream(lexer);
        try {
            tokenStream.fill();
        }
        catch (e) {
            if (e instanceof LexerNoViableAltException_1.LexerNoViableAltException) {
                let pos = lexer.charPositionInLine;
                let msg = "Invalid tokens or characters at index " + pos + " in path '" + path + "' -- " + e.message;
                throw new RangeError(msg);
            }
            throw e;
        }
        let tokens = tokenStream.getTokens();
        // console.log("path=" + path + "=>" + tokens);
        let elements = [];
        let n = tokens.length;
        let i = 0;
        loop: while (i < n) {
            let el = tokens[i];
            let next;
            switch (el.type) {
                case XPathLexer_1.XPathLexer.ROOT:
                case XPathLexer_1.XPathLexer.ANYWHERE:
                    let anywhere = el.type === XPathLexer_1.XPathLexer.ANYWHERE;
                    i++;
                    next = tokens[i];
                    let invert = next.type === XPathLexer_1.XPathLexer.BANG;
                    if (invert) {
                        i++;
                        next = tokens[i];
                    }
                    let pathElement = this.getXPathElement(next, anywhere);
                    pathElement.invert = invert;
                    elements.push(pathElement);
                    i++;
                    break;
                case XPathLexer_1.XPathLexer.TOKEN_REF:
                case XPathLexer_1.XPathLexer.RULE_REF:
                case XPathLexer_1.XPathLexer.WILDCARD:
                    elements.push(this.getXPathElement(el, false));
                    i++;
                    break;
                case Token_1.Token.EOF:
                    break loop;
                default:
                    throw new Error("Unknowth path element " + el);
            }
        }
        return elements;
    }
    /**
     * Convert word like `*` or `ID` or `expr` to a path
     * element. `anywhere` is `true` if `//` precedes the
     * word.
     */
    getXPathElement(wordToken, anywhere) {
        if (wordToken.type === Token_1.Token.EOF) {
            throw new Error("Missing path element at end of path");
        }
        let word = wordToken.text;
        if (word == null) {
            throw new Error("Expected wordToken to have text content.");
        }
        let ttype = this.parser.getTokenType(word);
        let ruleIndex = this.parser.getRuleIndex(word);
        switch (wordToken.type) {
            case XPathLexer_1.XPathLexer.WILDCARD:
                return anywhere ?
                    new XPathWildcardAnywhereElement_1.XPathWildcardAnywhereElement() :
                    new XPathWildcardElement_1.XPathWildcardElement();
            case XPathLexer_1.XPathLexer.TOKEN_REF:
            case XPathLexer_1.XPathLexer.STRING:
                if (ttype === Token_1.Token.INVALID_TYPE) {
                    throw new Error(word + " at index " +
                        wordToken.startIndex +
                        " isn't a valid token name");
                }
                return anywhere ?
                    new XPathTokenAnywhereElement_1.XPathTokenAnywhereElement(word, ttype) :
                    new XPathTokenElement_1.XPathTokenElement(word, ttype);
            default:
                if (ruleIndex === -1) {
                    throw new Error(word + " at index " +
                        wordToken.startIndex +
                        " isn't a valid rule name");
                }
                return anywhere ?
                    new XPathRuleAnywhereElement_1.XPathRuleAnywhereElement(word, ruleIndex) :
                    new XPathRuleElement_1.XPathRuleElement(word, ruleIndex);
        }
    }
    static findAll(tree, xpath, parser) {
        let p = new XPath(parser, xpath);
        return p.evaluate(tree);
    }
    /**
     * Return a list of all nodes starting at `t` as root that satisfy the
     * path. The root `/` is relative to the node passed to {@link evaluate}.
     */
    evaluate(t) {
        let dummyRoot = new ParserRuleContext_1.ParserRuleContext();
        dummyRoot.addChild(t);
        let work = new Set([dummyRoot]);
        let i = 0;
        while (i < this.elements.length) {
            let next = new Set();
            for (let node of work) {
                if (node.childCount > 0) {
                    // only try to match next element if it has children
                    // e.g., //func/*/stat might have a token node for which
                    // we can't go looking for stat nodes.
                    let matching = this.elements[i].evaluate(node);
                    matching.forEach(next.add, next);
                }
            }
            i++;
            work = next;
        }
        return work;
    }
}
exports.XPath = XPath;
XPath.WILDCARD = "*"; // word not operator/separator
XPath.NOT = "!"; // word for invert operator
//# sourceMappingURL=XPath.js.map