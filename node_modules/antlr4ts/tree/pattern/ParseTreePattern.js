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
exports.ParseTreePattern = void 0;
// CONVERSTION complete, Burt Harris 10/14/2016
const Decorators_1 = require("../../Decorators");
const XPath_1 = require("../xpath/XPath");
/**
 * A pattern like `<ID> = <expr>;` converted to a {@link ParseTree} by
 * {@link ParseTreePatternMatcher#compile(String, int)}.
 */
let ParseTreePattern = class ParseTreePattern {
    /**
     * Construct a new instance of the {@link ParseTreePattern} class.
     *
     * @param matcher The {@link ParseTreePatternMatcher} which created this
     * tree pattern.
     * @param pattern The tree pattern in concrete syntax form.
     * @param patternRuleIndex The parser rule which serves as the root of the
     * tree pattern.
     * @param patternTree The tree pattern in {@link ParseTree} form.
     */
    constructor(matcher, pattern, patternRuleIndex, patternTree) {
        this._matcher = matcher;
        this._patternRuleIndex = patternRuleIndex;
        this._pattern = pattern;
        this._patternTree = patternTree;
    }
    /**
     * Match a specific parse tree against this tree pattern.
     *
     * @param tree The parse tree to match against this tree pattern.
     * @returns A {@link ParseTreeMatch} object describing the result of the
     * match operation. The `ParseTreeMatch.succeeded` method can be
     * used to determine whether or not the match was successful.
     */
    match(tree) {
        return this._matcher.match(tree, this);
    }
    /**
     * Determine whether or not a parse tree matches this tree pattern.
     *
     * @param tree The parse tree to match against this tree pattern.
     * @returns `true` if `tree` is a match for the current tree
     * pattern; otherwise, `false`.
     */
    matches(tree) {
        return this._matcher.match(tree, this).succeeded;
    }
    /**
     * Find all nodes using XPath and then try to match those subtrees against
     * this tree pattern.
     *
     * @param tree The {@link ParseTree} to match against this pattern.
     * @param xpath An expression matching the nodes
     *
     * @returns A collection of {@link ParseTreeMatch} objects describing the
     * successful matches. Unsuccessful matches are omitted from the result,
     * regardless of the reason for the failure.
     */
    findAll(tree, xpath) {
        let subtrees = XPath_1.XPath.findAll(tree, xpath, this._matcher.parser);
        let matches = [];
        for (let t of subtrees) {
            let match = this.match(t);
            if (match.succeeded) {
                matches.push(match);
            }
        }
        return matches;
    }
    /**
     * Get the {@link ParseTreePatternMatcher} which created this tree pattern.
     *
     * @returns The {@link ParseTreePatternMatcher} which created this tree
     * pattern.
     */
    get matcher() {
        return this._matcher;
    }
    /**
     * Get the tree pattern in concrete syntax form.
     *
     * @returns The tree pattern in concrete syntax form.
     */
    get pattern() {
        return this._pattern;
    }
    /**
     * Get the parser rule which serves as the outermost rule for the tree
     * pattern.
     *
     * @returns The parser rule which serves as the outermost rule for the tree
     * pattern.
     */
    get patternRuleIndex() {
        return this._patternRuleIndex;
    }
    /**
     * Get the tree pattern as a {@link ParseTree}. The rule and token tags from
     * the pattern are present in the parse tree as terminal nodes with a symbol
     * of type {@link RuleTagToken} or {@link TokenTagToken}.
     *
     * @returns The tree pattern as a {@link ParseTree}.
     */
    get patternTree() {
        return this._patternTree;
    }
};
__decorate([
    Decorators_1.NotNull
], ParseTreePattern.prototype, "_pattern", void 0);
__decorate([
    Decorators_1.NotNull
], ParseTreePattern.prototype, "_patternTree", void 0);
__decorate([
    Decorators_1.NotNull
], ParseTreePattern.prototype, "_matcher", void 0);
__decorate([
    Decorators_1.NotNull,
    __param(0, Decorators_1.NotNull)
], ParseTreePattern.prototype, "match", null);
__decorate([
    __param(0, Decorators_1.NotNull)
], ParseTreePattern.prototype, "matches", null);
__decorate([
    Decorators_1.NotNull,
    __param(0, Decorators_1.NotNull), __param(1, Decorators_1.NotNull)
], ParseTreePattern.prototype, "findAll", null);
__decorate([
    Decorators_1.NotNull
], ParseTreePattern.prototype, "matcher", null);
__decorate([
    Decorators_1.NotNull
], ParseTreePattern.prototype, "pattern", null);
__decorate([
    Decorators_1.NotNull
], ParseTreePattern.prototype, "patternTree", null);
ParseTreePattern = __decorate([
    __param(0, Decorators_1.NotNull),
    __param(1, Decorators_1.NotNull),
    __param(3, Decorators_1.NotNull)
], ParseTreePattern);
exports.ParseTreePattern = ParseTreePattern;
//# sourceMappingURL=ParseTreePattern.js.map