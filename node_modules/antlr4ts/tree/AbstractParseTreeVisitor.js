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
exports.AbstractParseTreeVisitor = void 0;
const Decorators_1 = require("../Decorators");
class AbstractParseTreeVisitor {
    /**
     * {@inheritDoc}
     *
     * The default implementation calls {@link ParseTree#accept} on the
     * specified tree.
     */
    visit(tree) {
        return tree.accept(this);
    }
    /**
     * {@inheritDoc}
     *
     * The default implementation initializes the aggregate result to
     * {@link #defaultResult defaultResult()}. Before visiting each child, it
     * calls {@link #shouldVisitNextChild shouldVisitNextChild}; if the result
     * is `false` no more children are visited and the current aggregate
     * result is returned. After visiting a child, the aggregate result is
     * updated by calling {@link #aggregateResult aggregateResult} with the
     * previous aggregate result and the result of visiting the child.
     *
     * The default implementation is not safe for use in visitors that modify
     * the tree structure. Visitors that modify the tree should override this
     * method to behave properly in respect to the specific algorithm in use.
     */
    visitChildren(node) {
        let result = this.defaultResult();
        let n = node.childCount;
        for (let i = 0; i < n; i++) {
            if (!this.shouldVisitNextChild(node, result)) {
                break;
            }
            let c = node.getChild(i);
            let childResult = c.accept(this);
            result = this.aggregateResult(result, childResult);
        }
        return result;
    }
    /**
     * {@inheritDoc}
     *
     * The default implementation returns the result of
     * {@link #defaultResult defaultResult}.
     */
    visitTerminal(node) {
        return this.defaultResult();
    }
    /**
     * {@inheritDoc}
     *
     * The default implementation returns the result of
     * {@link #defaultResult defaultResult}.
     */
    visitErrorNode(node) {
        return this.defaultResult();
    }
    /**
     * Aggregates the results of visiting multiple children of a node. After
     * either all children are visited or {@link #shouldVisitNextChild} returns
     * `false`, the aggregate value is returned as the result of
     * {@link #visitChildren}.
     *
     * The default implementation returns `nextResult`, meaning
     * {@link #visitChildren} will return the result of the last child visited
     * (or return the initial value if the node has no children).
     *
     * @param aggregate The previous aggregate value. In the default
     * implementation, the aggregate value is initialized to
     * {@link #defaultResult}, which is passed as the `aggregate` argument
     * to this method after the first child node is visited.
     * @param nextResult The result of the immediately preceeding call to visit
     * a child node.
     *
     * @returns The updated aggregate result.
     */
    aggregateResult(aggregate, nextResult) {
        return nextResult;
    }
    /**
     * This method is called after visiting each child in
     * {@link #visitChildren}. This method is first called before the first
     * child is visited; at that point `currentResult` will be the initial
     * value (in the default implementation, the initial value is returned by a
     * call to {@link #defaultResult}. This method is not called after the last
     * child is visited.
     *
     * The default implementation always returns `true`, indicating that
     * `visitChildren` should only return after all children are visited.
     * One reason to override this method is to provide a "short circuit"
     * evaluation option for situations where the result of visiting a single
     * child has the potential to determine the result of the visit operation as
     * a whole.
     *
     * @param node The {@link RuleNode} whose children are currently being
     * visited.
     * @param currentResult The current aggregate result of the children visited
     * to the current point.
     *
     * @returns `true` to continue visiting children. Otherwise return
     * `false` to stop visiting children and immediately return the
     * current aggregate result from {@link #visitChildren}.
     */
    shouldVisitNextChild(node, currentResult) {
        return true;
    }
}
__decorate([
    Decorators_1.Override,
    __param(0, Decorators_1.NotNull)
], AbstractParseTreeVisitor.prototype, "visit", null);
__decorate([
    Decorators_1.Override,
    __param(0, Decorators_1.NotNull)
], AbstractParseTreeVisitor.prototype, "visitChildren", null);
__decorate([
    Decorators_1.Override,
    __param(0, Decorators_1.NotNull)
], AbstractParseTreeVisitor.prototype, "visitTerminal", null);
__decorate([
    Decorators_1.Override,
    __param(0, Decorators_1.NotNull)
], AbstractParseTreeVisitor.prototype, "visitErrorNode", null);
__decorate([
    __param(0, Decorators_1.NotNull)
], AbstractParseTreeVisitor.prototype, "shouldVisitNextChild", null);
exports.AbstractParseTreeVisitor = AbstractParseTreeVisitor;
//# sourceMappingURL=AbstractParseTreeVisitor.js.map