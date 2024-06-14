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
Object.defineProperty(exports, "__esModule", { value: true });
exports.XPathRuleAnywhereElement = void 0;
const Decorators_1 = require("../../Decorators");
const Trees_1 = require("../Trees");
const XPathElement_1 = require("./XPathElement");
/**
 * Either `ID` at start of path or `...//ID` in middle of path.
 */
class XPathRuleAnywhereElement extends XPathElement_1.XPathElement {
    constructor(ruleName, ruleIndex) {
        super(ruleName);
        this.ruleIndex = ruleIndex;
    }
    evaluate(t) {
        return Trees_1.Trees.findAllRuleNodes(t, this.ruleIndex);
    }
}
__decorate([
    Decorators_1.Override
], XPathRuleAnywhereElement.prototype, "evaluate", null);
exports.XPathRuleAnywhereElement = XPathRuleAnywhereElement;
//# sourceMappingURL=XPathRuleAnywhereElement.js.map