"use strict";
// Adapted from https://github.com/palantir/tslint/blob/master/src/rules/deprecationRule.ts
var __extends = (this && this.__extends) || (function () {
    var extendStatics = function (d, b) {
        extendStatics = Object.setPrototypeOf ||
            ({ __proto__: [] } instanceof Array && function (d, b) { d.__proto__ = b; }) ||
            function (d, b) { for (var p in b) if (b.hasOwnProperty(p)) d[p] = b[p]; };
        return extendStatics(d, b);
    };
    return function (d, b) {
        extendStatics(d, b);
        function __() { this.constructor = d; }
        d.prototype = b === null ? Object.create(b) : (__.prototype = b.prototype, new __());
    };
})();
var __makeTemplateObject = (this && this.__makeTemplateObject) || function (cooked, raw) {
    if (Object.defineProperty) { Object.defineProperty(cooked, "raw", { value: raw }); } else { cooked.raw = raw; }
    return cooked;
};
exports.__esModule = true;
var tsutils_1 = require("tsutils");
var ts = require("typescript");
var Lint = require("tslint");
var Rule = /** @class */ (function (_super) {
    __extends(Rule, _super);
    function Rule() {
        return _super !== null && _super.apply(this, arguments) || this;
    }
    /* tslint:enable:object-literal-sort-keys */
    Rule.FAILURE_STRING = function (name, message) {
        return name + " is obsolete" + (message === '' ? '.' : ": " + message.trim());
    };
    Rule.prototype.applyWithProgram = function (sourceFile, program) {
        return this.applyWithFunction(sourceFile, walk, undefined, program.getTypeChecker());
    };
    /* tslint:disable:object-literal-sort-keys */
    Rule.metadata = {
        ruleName: 'obsolete',
        description: 'Warns when obsolete APIs are used.',
        descriptionDetails: Lint.Utils.dedent(templateObject_1 || (templateObject_1 = __makeTemplateObject(["Any usage of an identifier\n          with the @obsolete JSDoc annotation will trigger a warning."], ["Any usage of an identifier\n          with the @obsolete JSDoc annotation will trigger a warning."]))),
        rationale: 'Obsolete APIs should be avoided, and usage updated.',
        optionsDescription: '',
        options: null,
        optionExamples: [true],
        type: 'maintainability',
        typescriptOnly: false,
        requiresTypeInfo: true
    };
    return Rule;
}(Lint.Rules.TypedRule));
exports.Rule = Rule;
function walk(ctx, tc) {
    return ts.forEachChild(ctx.sourceFile, function cb(node) {
        if (tsutils_1.isIdentifier(node)) {
            if (!isDeclaration(node)) {
                var obsolete = getObsolete(node, tc);
                if (obsolete !== undefined) {
                    ctx.addFailureAtNode(node, Rule.FAILURE_STRING(node.text, obsolete));
                }
            }
        }
        else {
            switch (node.kind) {
                case ts.SyntaxKind.ImportDeclaration:
                case ts.SyntaxKind.ImportEqualsDeclaration:
                case ts.SyntaxKind.ExportDeclaration:
                case ts.SyntaxKind.ExportAssignment:
                    return;
            }
            return ts.forEachChild(node, cb);
        }
    });
}
function isDeclaration(identifier) {
    var parent = identifier.parent;
    switch (parent.kind) {
        case ts.SyntaxKind.ClassDeclaration:
        case ts.SyntaxKind.ClassExpression:
        case ts.SyntaxKind.InterfaceDeclaration:
        case ts.SyntaxKind.TypeParameter:
        case ts.SyntaxKind.FunctionExpression:
        case ts.SyntaxKind.FunctionDeclaration:
        case ts.SyntaxKind.LabeledStatement:
        case ts.SyntaxKind.JsxAttribute:
        case ts.SyntaxKind.MethodDeclaration:
        case ts.SyntaxKind.MethodSignature:
        case ts.SyntaxKind.PropertySignature:
        case ts.SyntaxKind.TypeAliasDeclaration:
        case ts.SyntaxKind.GetAccessor:
        case ts.SyntaxKind.SetAccessor:
        case ts.SyntaxKind.EnumDeclaration:
        case ts.SyntaxKind.ModuleDeclaration:
            return true;
        case ts.SyntaxKind.VariableDeclaration:
        case ts.SyntaxKind.Parameter:
        case ts.SyntaxKind.PropertyDeclaration:
        case ts.SyntaxKind.EnumMember:
        case ts.SyntaxKind.ImportEqualsDeclaration:
            return parent.name === identifier;
        case ts.SyntaxKind.PropertyAssignment:
            return (parent.name === identifier &&
                !tsutils_1.isReassignmentTarget(identifier.parent.parent));
        case ts.SyntaxKind.BindingElement:
            // return true for `b` in `const {a: b} = obj"`
            return (parent.name === identifier && parent.propertyName !== undefined);
        default:
            return false;
    }
}
function getCallExpresion(node) {
    var parent = node.parent;
    if (tsutils_1.isPropertyAccessExpression(parent) && parent.name === node) {
        node = parent;
        parent = node.parent;
    }
    return tsutils_1.isTaggedTemplateExpression(parent) ||
        ((tsutils_1.isCallExpression(parent) || tsutils_1.isNewExpression(parent)) && parent.expression === node)
        ? parent
        : undefined;
}
function getObsolete(node, tc) {
    var callExpression = getCallExpresion(node);
    if (callExpression !== undefined) {
        var result = getSignatureObsolete(tc.getResolvedSignature(callExpression));
        if (result !== undefined) {
            return result;
        }
    }
    var symbol;
    var parent = node.parent;
    if (parent.kind === ts.SyntaxKind.BindingElement) {
        symbol = tc.getTypeAtLocation(parent.parent).getProperty(node.text);
    }
    else if ((tsutils_1.isPropertyAssignment(parent) && parent.name === node) ||
        (tsutils_1.isShorthandPropertyAssignment(parent) && parent.name === node && tsutils_1.isReassignmentTarget(node))) {
        symbol = tc.getPropertySymbolOfDestructuringAssignment(node);
    }
    else {
        symbol = tc.getSymbolAtLocation(node);
    }
    if (symbol !== undefined && tsutils_1.isSymbolFlagSet(symbol, ts.SymbolFlags.Alias)) {
        symbol = tc.getAliasedSymbol(symbol);
    }
    if (symbol === undefined ||
        // if this is a CallExpression and the declaration is a function or method,
        // stop here to avoid collecting JsDoc of all overload signatures
        (callExpression !== undefined && isFunctionOrMethod(symbol.declarations))) {
        return undefined;
    }
    return getSymbolObsolete(symbol);
}
function findObsoleteTag(tags) {
    for (var _i = 0, tags_1 = tags; _i < tags_1.length; _i++) {
        var tag = tags_1[_i];
        if (tag.name === 'obsolete') {
            return tag.text === undefined ? '' : tag.text;
        }
    }
    return undefined;
}
function getSymbolObsolete(symbol) {
    if (symbol.getJsDocTags !== undefined) {
        return findObsoleteTag(symbol.getJsDocTags());
    }
    // for compatibility with typescript@<2.3.0
    return getObsoleteFromDeclarations(symbol.declarations);
}
function getSignatureObsolete(signature) {
    if (signature === undefined) {
        return undefined;
    }
    if (signature.getJsDocTags !== undefined) {
        return findObsoleteTag(signature.getJsDocTags());
    }
    // for compatibility with typescript@<2.3.0
    return signature.declaration === undefined ? undefined : getObsoleteFromDeclaration(signature.declaration);
}
function getObsoleteFromDeclarations(declarations) {
    if (declarations === undefined) {
        return undefined;
    }
    var declaration;
    for (var _i = 0, declarations_1 = declarations; _i < declarations_1.length; _i++) {
        declaration = declarations_1[_i];
        if (tsutils_1.isBindingElement(declaration)) {
            declaration = tsutils_1.getDeclarationOfBindingElement(declaration);
        }
        if (tsutils_1.isVariableDeclaration(declaration)) {
            declaration = declaration.parent;
        }
        if (tsutils_1.isVariableDeclarationList(declaration)) {
            declaration = declaration.parent;
        }
        var result = getObsoleteFromDeclaration(declaration);
        if (result !== undefined) {
            return result;
        }
    }
    return undefined;
}
function getObsoleteFromDeclaration(declaration) {
    for (var _i = 0, _a = tsutils_1.getJsDoc(declaration); _i < _a.length; _i++) {
        var comment = _a[_i];
        if (comment.tags === undefined) {
            continue;
        }
        for (var _b = 0, _c = comment.tags; _b < _c.length; _b++) {
            var tag = _c[_b];
            if (tag.tagName.text === 'obsolete') {
                return tag.comment === undefined ? '' : tag.comment;
            }
        }
    }
    return undefined;
}
function isFunctionOrMethod(declarations) {
    if (declarations === undefined || declarations.length === 0) {
        return false;
    }
    switch (declarations[0].kind) {
        case ts.SyntaxKind.MethodDeclaration:
        case ts.SyntaxKind.FunctionDeclaration:
        case ts.SyntaxKind.FunctionExpression:
        case ts.SyntaxKind.MethodSignature:
            return true;
        default:
            return false;
    }
}
var templateObject_1;
