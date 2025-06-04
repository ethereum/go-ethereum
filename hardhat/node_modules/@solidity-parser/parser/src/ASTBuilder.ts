import { ParserRuleContext } from 'antlr4ts'
import { AbstractParseTreeVisitor } from 'antlr4ts/tree/AbstractParseTreeVisitor'
import { ParseTree } from 'antlr4ts/tree/ParseTree'
import * as SP from './antlr/SolidityParser'

import { SolidityVisitor } from './antlr/SolidityVisitor'
import { ParseOptions } from './types'
import * as AST from './ast-types'
import { ErrorNode } from 'antlr4ts/tree/ErrorNode'

interface SourceLocation {
  start: {
    line: number
    column: number
  }
  end: {
    line: number
    column: number
  }
}

interface WithMeta {
  __withMeta: never
}

type ASTBuilderNode = AST.ASTNode & WithMeta

export class ASTBuilder
  extends AbstractParseTreeVisitor<ASTBuilderNode>
  implements SolidityVisitor<ASTBuilderNode | ASTBuilderNode[]> {
  public result: AST.SourceUnit | null = null
  private _currentContract?: string

  constructor(public options: ParseOptions) {
    super()
  }

  defaultResult(): AST.ASTNode & WithMeta {
    throw new Error('Unknown node')
  }

  aggregateResult() {
    return ({ type: '' } as unknown) as AST.ASTNode & WithMeta
  }

  public visitSourceUnit(ctx: SP.SourceUnitContext): AST.SourceUnit & WithMeta {
    const children = (ctx.children ?? []).filter(
      (x) => !(x instanceof ErrorNode)
    )

    const node: AST.SourceUnit = {
      type: 'SourceUnit',
      children: children.slice(0, -1).map((child) => this.visit(child)),
    }
    const result = this._addMeta(node, ctx)
    this.result = result

    return result
  }

  public visitContractPart(ctx: SP.ContractPartContext) {
    return this.visit(ctx.getChild(0))
  }

  public visitContractDefinition(
    ctx: SP.ContractDefinitionContext
  ): AST.ContractDefinition & WithMeta {
    const name = this._toText(ctx.identifier())
    const kind = this._toText(ctx.getChild(0))

    this._currentContract = name

    const node: AST.ContractDefinition = {
      type: 'ContractDefinition',
      name,
      baseContracts: ctx
        .inheritanceSpecifier()
        .map((x) => this.visitInheritanceSpecifier(x)),
      subNodes: ctx.contractPart().map((x) => this.visit(x)),
      kind,
    }

    return this._addMeta(node, ctx)
  }

  public visitStateVariableDeclaration(
    ctx: SP.StateVariableDeclarationContext
  ) {
    const type = this.visitTypeName(ctx.typeName())
    const iden = ctx.identifier()
    const name = this._toText(iden)

    let expression: AST.Expression | null = null
    const ctxExpression = ctx.expression()
    if (ctxExpression) {
      expression = this.visitExpression(ctxExpression)
    }

    let visibility: AST.VariableDeclaration['visibility'] = 'default'
    if (ctx.InternalKeyword().length > 0) {
      visibility = 'internal'
    } else if (ctx.PublicKeyword().length > 0) {
      visibility = 'public'
    } else if (ctx.PrivateKeyword().length > 0) {
      visibility = 'private'
    }

    let isDeclaredConst = false
    if (ctx.ConstantKeyword().length > 0) {
      isDeclaredConst = true
    }

    let override
    const overrideSpecifier = ctx.overrideSpecifier()
    if (overrideSpecifier.length === 0) {
      override = null
    } else {
      override = overrideSpecifier[0]
        .userDefinedTypeName()
        .map((x) => this.visitUserDefinedTypeName(x))
    }

    let isImmutable = false
    if (ctx.ImmutableKeyword().length > 0) {
      isImmutable = true
    }

    const decl: AST.StateVariableDeclarationVariable = {
      type: 'VariableDeclaration',
      typeName: type,
      name,
      identifier: this.visitIdentifier(iden),
      expression,
      visibility,
      isStateVar: true,
      isDeclaredConst,
      isIndexed: false,
      isImmutable,
      override,
      storageLocation: null,
    }

    const node: AST.StateVariableDeclaration = {
      type: 'StateVariableDeclaration',
      variables: [this._addMeta(decl, ctx)],
      initialValue: expression,
    }

    return this._addMeta(node, ctx)
  }

  public visitVariableDeclaration(
    ctx: SP.VariableDeclarationContext
  ): AST.VariableDeclaration & WithMeta {
    let storageLocation: string | null = null
    const ctxStorageLocation = ctx.storageLocation()
    if (ctxStorageLocation) {
      storageLocation = this._toText(ctxStorageLocation)
    }

    const identifierCtx = ctx.identifier()

    const node: AST.VariableDeclaration = {
      type: 'VariableDeclaration',
      typeName: this.visitTypeName(ctx.typeName()),
      name: this._toText(identifierCtx),
      identifier: this.visitIdentifier(identifierCtx),
      storageLocation,
      isStateVar: false,
      isIndexed: false,
      expression: null,
    }

    return this._addMeta(node, ctx)
  }

  public visitVariableDeclarationStatement(
    ctx: SP.VariableDeclarationStatementContext
  ): AST.VariableDeclarationStatement & WithMeta {
    let variables: Array<AST.BaseASTNode | null> = []
    const ctxVariableDeclaration = ctx.variableDeclaration()
    const ctxIdentifierList = ctx.identifierList()
    const ctxVariableDeclarationList = ctx.variableDeclarationList()
    if (ctxVariableDeclaration !== undefined) {
      variables = [this.visitVariableDeclaration(ctxVariableDeclaration)]
    } else if (ctxIdentifierList !== undefined) {
      variables = this.buildIdentifierList(ctxIdentifierList)
    } else if (ctxVariableDeclarationList) {
      variables = this.buildVariableDeclarationList(ctxVariableDeclarationList)
    }

    let initialValue: AST.Expression | null = null
    const ctxExpression = ctx.expression()
    if (ctxExpression) {
      initialValue = this.visitExpression(ctxExpression)
    }

    const node: AST.VariableDeclarationStatement = {
      type: 'VariableDeclarationStatement',
      variables,
      initialValue,
    }

    return this._addMeta(node, ctx)
  }

  public visitStatement(ctx: SP.StatementContext) {
    return this.visit(ctx.getChild(0)) as AST.Statement & WithMeta
  }

  public visitSimpleStatement(ctx: SP.SimpleStatementContext) {
    return this.visit(ctx.getChild(0)) as AST.SimpleStatement & WithMeta
  }

  public visitEventDefinition(ctx: SP.EventDefinitionContext) {
    const parameters = ctx
      .eventParameterList()
      .eventParameter()
      .map((paramCtx) => {
        const type = this.visitTypeName(paramCtx.typeName())
        let name: string | null = null
        const paramCtxIdentifier = paramCtx.identifier()
        if (paramCtxIdentifier) {
          name = this._toText(paramCtxIdentifier)
        }

        const node: AST.VariableDeclaration = {
          type: 'VariableDeclaration',
          typeName: type,
          name,
          identifier:
            paramCtxIdentifier !== undefined
              ? this.visitIdentifier(paramCtxIdentifier)
              : null,
          isStateVar: false,
          isIndexed: paramCtx.IndexedKeyword() !== undefined,
          storageLocation: null,
          expression: null,
        }
        return this._addMeta(node, paramCtx)
      })

    const node: AST.EventDefinition = {
      type: 'EventDefinition',
      name: this._toText(ctx.identifier()),
      parameters,
      isAnonymous: ctx.AnonymousKeyword() !== undefined,
    }

    return this._addMeta(node, ctx)
  }

  public visitBlock(ctx: SP.BlockContext): AST.Block & WithMeta {
    const node: AST.Block = {
      type: 'Block',
      statements: ctx.statement().map((x) => this.visitStatement(x)),
    }

    return this._addMeta(node, ctx)
  }

  public visitParameter(ctx: SP.ParameterContext) {
    let storageLocation: string | null = null
    const ctxStorageLocation = ctx.storageLocation()
    if (ctxStorageLocation !== undefined) {
      storageLocation = this._toText(ctxStorageLocation)
    }

    let name: string | null = null
    const ctxIdentifier = ctx.identifier()
    if (ctxIdentifier !== undefined) {
      name = this._toText(ctxIdentifier)
    }

    const node: AST.VariableDeclaration = {
      type: 'VariableDeclaration',
      typeName: this.visitTypeName(ctx.typeName()),
      name,
      identifier:
        ctxIdentifier !== undefined
          ? this.visitIdentifier(ctxIdentifier)
          : null,
      storageLocation,
      isStateVar: false,
      isIndexed: false,
      expression: null,
    }

    return this._addMeta(node, ctx)
  }

  public visitFunctionDefinition(
    ctx: SP.FunctionDefinitionContext
  ): AST.FunctionDefinition & WithMeta {
    let isConstructor = false
    let isFallback = false
    let isReceiveEther = false
    let isVirtual = false
    let name: string | null = null
    let parameters: any = []
    let returnParameters: AST.VariableDeclaration[] | null = null
    let visibility: AST.FunctionDefinition['visibility'] = 'default'

    let block: AST.Block | null = null
    const ctxBlock = ctx.block()
    if (ctxBlock !== undefined) {
      block = this.visitBlock(ctxBlock)
    }

    const modifiers = ctx
      .modifierList()
      .modifierInvocation()
      .map((mod) => this.visitModifierInvocation(mod))

    let stateMutability = null
    if (ctx.modifierList().stateMutability().length > 0) {
      stateMutability = this._stateMutabilityToText(
        ctx.modifierList().stateMutability(0)
      )
    }

    // see what type of function we're dealing with
    const ctxReturnParameters = ctx.returnParameters()
    switch (this._toText(ctx.functionDescriptor().getChild(0))) {
      case 'constructor':
        parameters = ctx
          .parameterList()
          .parameter()
          .map((x) => this.visit(x))

        // error out on incorrect function visibility
        if (ctx.modifierList().InternalKeyword().length > 0) {
          visibility = 'internal'
        } else if (ctx.modifierList().PublicKeyword().length > 0) {
          visibility = 'public'
        } else {
          visibility = 'default'
        }

        isConstructor = true
        break
      case 'fallback':
        parameters = ctx
          .parameterList()
          .parameter()
          .map((x) => this.visit(x))
        returnParameters =
          ctxReturnParameters !== undefined
            ? this.visitReturnParameters(ctxReturnParameters)
            : null

        visibility = 'external'
        isFallback = true
        break
      case 'receive':
        visibility = 'external'
        isReceiveEther = true
        break
      case 'function': {
        const identifier = ctx.functionDescriptor().identifier()
        name = identifier !== undefined ? this._toText(identifier) : ''

        parameters = ctx
          .parameterList()
          .parameter()
          .map((x) => this.visit(x))
        returnParameters =
          ctxReturnParameters !== undefined
            ? this.visitReturnParameters(ctxReturnParameters)
            : null

        // parse function visibility
        if (ctx.modifierList().ExternalKeyword().length > 0) {
          visibility = 'external'
        } else if (ctx.modifierList().InternalKeyword().length > 0) {
          visibility = 'internal'
        } else if (ctx.modifierList().PublicKeyword().length > 0) {
          visibility = 'public'
        } else if (ctx.modifierList().PrivateKeyword().length > 0) {
          visibility = 'private'
        }

        isConstructor = name === this._currentContract
        isFallback = name === ''
        break
      }
    }

    // check if function is virtual
    if (ctx.modifierList().VirtualKeyword().length > 0) {
      isVirtual = true
    }

    let override: AST.UserDefinedTypeName[] | null
    const overrideSpecifier = ctx.modifierList().overrideSpecifier()
    if (overrideSpecifier.length === 0) {
      override = null
    } else {
      override = overrideSpecifier[0]
        .userDefinedTypeName()
        .map((x) => this.visitUserDefinedTypeName(x))
    }

    const node: AST.FunctionDefinition = {
      type: 'FunctionDefinition',
      name,
      parameters,
      returnParameters,
      body: block,
      visibility,
      modifiers,
      override,
      isConstructor,
      isReceiveEther,
      isFallback,
      isVirtual,
      stateMutability,
    }

    return this._addMeta(node, ctx)
  }

  public visitEnumDefinition(
    ctx: SP.EnumDefinitionContext
  ): AST.EnumDefinition & WithMeta {
    const node: AST.EnumDefinition = {
      type: 'EnumDefinition',
      name: this._toText(ctx.identifier()),
      members: ctx.enumValue().map((x) => this.visitEnumValue(x)),
    }

    return this._addMeta(node, ctx)
  }

  public visitEnumValue(ctx: SP.EnumValueContext): AST.EnumValue & WithMeta {
    const node: AST.EnumValue = {
      type: 'EnumValue',
      name: this._toText(ctx.identifier()),
    }
    return this._addMeta(node, ctx)
  }

  public visitElementaryTypeName(
    ctx: SP.ElementaryTypeNameContext
  ): AST.ElementaryTypeName & WithMeta {
    const node: AST.ElementaryTypeName = {
      type: 'ElementaryTypeName',
      name: this._toText(ctx),
      stateMutability: null,
    }

    return this._addMeta(node, ctx)
  }

  public visitIdentifier(ctx: SP.IdentifierContext): AST.Identifier & WithMeta {
    const node: AST.Identifier = {
      type: 'Identifier',
      name: this._toText(ctx),
    }
    return this._addMeta(node, ctx)
  }

  public visitTypeName(ctx: SP.TypeNameContext): AST.TypeName & WithMeta {
    if (ctx.children !== undefined && ctx.children.length > 2) {
      let length = null
      if (ctx.children.length === 4) {
        const expression = ctx.expression()
        if (expression === undefined) {
          throw new Error(
            'Assertion error: a typeName with 4 children should have an expression'
          )
        }
        length = this.visitExpression(expression)
      }

      const ctxTypeName = ctx.typeName()

      const node: AST.ArrayTypeName = {
        type: 'ArrayTypeName',
        baseTypeName: this.visitTypeName(ctxTypeName!),
        length,
      }

      return this._addMeta(node, ctx)
    }

    if (ctx.children?.length === 2) {
      const node: AST.ElementaryTypeName = {
        type: 'ElementaryTypeName',
        name: this._toText(ctx.getChild(0)),
        stateMutability: this._toText(ctx.getChild(1)),
      }

      return this._addMeta(node, ctx)
    }

    if (ctx.elementaryTypeName() !== undefined) {
      return this.visitElementaryTypeName(ctx.elementaryTypeName()!)
    }

    if (ctx.userDefinedTypeName() !== undefined) {
      return this.visitUserDefinedTypeName(ctx.userDefinedTypeName()!)
    }

    if (ctx.mapping() !== undefined) {
      return this.visitMapping(ctx.mapping()!)
    }

    if (ctx.functionTypeName() !== undefined) {
      return this.visitFunctionTypeName(ctx.functionTypeName()!)
    }

    throw new Error('Assertion error: unhandled type name case')
  }

  public visitUserDefinedTypeName(
    ctx: SP.UserDefinedTypeNameContext
  ): AST.UserDefinedTypeName & WithMeta {
    const node: AST.UserDefinedTypeName = {
      type: 'UserDefinedTypeName',
      namePath: this._toText(ctx),
    }

    return this._addMeta(node, ctx)
  }

  public visitUsingForDeclaration(
    ctx: SP.UsingForDeclarationContext
  ): AST.UsingForDeclaration & WithMeta {
    let typeName = null
    const ctxTypeName = ctx.typeName()
    if (ctxTypeName !== undefined) {
      typeName = this.visitTypeName(ctxTypeName)
    }

    const isGlobal = ctx.GlobalKeyword() !== undefined;

    // the object of the `usingForDeclaration` can be a single identifier
    // (the library name) or a group of functions:
    //   using Lib for uint;
    //   using { f } for uint;
    let node: AST.UsingForDeclaration
    const usingForObject = ctx.usingForObject()
    const firstChild = this._toText(usingForObject.getChild(0))
    if (firstChild === '{') {
      node = {
        type: 'UsingForDeclaration',
        isGlobal,
        typeName,
        libraryName: null,
        functions: usingForObject
          .userDefinedTypeName()
          .map((x) => this._toText(x)),
      }
    } else {
      node = {
        type: 'UsingForDeclaration',
        isGlobal,
        typeName,
        libraryName: this._toText(usingForObject.userDefinedTypeName(0)),
        functions: [],
      }
    }

    return this._addMeta(node, ctx)
  }

  public visitPragmaDirective(
    ctx: SP.PragmaDirectiveContext
  ): AST.PragmaDirective & WithMeta {
    // this converts something like >= 0.5.0  <0.7.0
    // in >=0.5.0 <0.7.0
    const versionContext = ctx.pragmaValue().version()

    let value = this._toText(ctx.pragmaValue())
    if (versionContext?.children !== undefined) {
      value = versionContext.children.map((x) => this._toText(x)).join(' ')
    }

    const node: AST.PragmaDirective = {
      type: 'PragmaDirective',
      name: this._toText(ctx.pragmaName()),
      value,
    }

    return this._addMeta(node, ctx)
  }

  public visitInheritanceSpecifier(
    ctx: SP.InheritanceSpecifierContext
  ): AST.InheritanceSpecifier & WithMeta {
    const exprList = ctx.expressionList()
    const args =
      exprList !== undefined
        ? exprList.expression().map((x) => this.visitExpression(x))
        : []

    const node: AST.InheritanceSpecifier = {
      type: 'InheritanceSpecifier',
      baseName: this.visitUserDefinedTypeName(ctx.userDefinedTypeName()),
      arguments: args,
    }

    return this._addMeta(node, ctx)
  }

  public visitModifierInvocation(
    ctx: SP.ModifierInvocationContext
  ): AST.ModifierInvocation & WithMeta {
    const exprList = ctx.expressionList()

    let args
    if (exprList != null) {
      args = exprList.expression().map((x) => this.visit(x))
    } else if (ctx.children !== undefined && ctx.children.length > 1) {
      args = []
    } else {
      args = null
    }

    const node: AST.ModifierInvocation = {
      type: 'ModifierInvocation',
      name: this._toText(ctx.identifier()),
      arguments: args,
    }
    return this._addMeta(node, ctx)
  }

  public visitFunctionTypeName(
    ctx: SP.FunctionTypeNameContext
  ): AST.FunctionTypeName & WithMeta {
    const parameterTypes = ctx
      .functionTypeParameterList(0)
      .functionTypeParameter()
      .map((typeCtx) => this.visitFunctionTypeParameter(typeCtx))

    let returnTypes: AST.VariableDeclaration[] = []
    if (ctx.functionTypeParameterList().length > 1) {
      returnTypes = ctx
        .functionTypeParameterList(1)
        .functionTypeParameter()
        .map((typeCtx) => this.visitFunctionTypeParameter(typeCtx))
    }

    let visibility = 'default'
    if (ctx.InternalKeyword().length > 0) {
      visibility = 'internal'
    } else if (ctx.ExternalKeyword().length > 0) {
      visibility = 'external'
    }

    let stateMutability = null
    if (ctx.stateMutability().length > 0) {
      stateMutability = this._toText(ctx.stateMutability(0))
    }

    const node: AST.FunctionTypeName = {
      type: 'FunctionTypeName',
      parameterTypes,
      returnTypes,
      visibility,
      stateMutability,
    }

    return this._addMeta(node, ctx)
  }

  public visitFunctionTypeParameter(
    ctx: SP.FunctionTypeParameterContext
  ): AST.VariableDeclaration & WithMeta {
    let storageLocation = null
    if (ctx.storageLocation()) {
      storageLocation = this._toText(ctx.storageLocation()!)
    }

    const node: AST.VariableDeclaration = {
      type: 'VariableDeclaration',
      typeName: this.visitTypeName(ctx.typeName()),
      name: null,
      identifier: null,
      storageLocation,
      isStateVar: false,
      isIndexed: false,
      expression: null,
    }

    return this._addMeta(node, ctx)
  }

  public visitThrowStatement(
    ctx: SP.ThrowStatementContext
  ): AST.ThrowStatement & WithMeta {
    const node: AST.ThrowStatement = {
      type: 'ThrowStatement',
    }

    return this._addMeta(node, ctx)
  }

  public visitReturnStatement(
    ctx: SP.ReturnStatementContext
  ): AST.ReturnStatement & WithMeta {
    let expression = null
    const ctxExpression = ctx.expression()
    if (ctxExpression) {
      expression = this.visitExpression(ctxExpression)
    }

    const node: AST.ReturnStatement = {
      type: 'ReturnStatement',
      expression,
    }

    return this._addMeta(node, ctx)
  }

  public visitEmitStatement(
    ctx: SP.EmitStatementContext
  ): AST.EmitStatement & WithMeta {
    const node: AST.EmitStatement = {
      type: 'EmitStatement',
      eventCall: this.visitFunctionCall(ctx.functionCall()),
    }

    return this._addMeta(node, ctx)
  }

  public visitCustomErrorDefinition(
    ctx: SP.CustomErrorDefinitionContext
  ): AST.CustomErrorDefinition & WithMeta {
    const node: AST.CustomErrorDefinition = {
      type: 'CustomErrorDefinition',
      name: this._toText(ctx.identifier()),
      parameters: this.visitParameterList(ctx.parameterList()),
    }

    return this._addMeta(node, ctx)
  }

  public visitTypeDefinition(
    ctx: SP.TypeDefinitionContext
  ): AST.TypeDefinition & WithMeta {
    const node: AST.TypeDefinition = {
      type: 'TypeDefinition',
      name: this._toText(ctx.identifier()),
      definition: this.visitElementaryTypeName(ctx.elementaryTypeName()),
    }

    return this._addMeta(node, ctx)
  }

  public visitRevertStatement(
    ctx: SP.RevertStatementContext
  ): AST.RevertStatement & WithMeta {
    const node: AST.RevertStatement = {
      type: 'RevertStatement',
      revertCall: this.visitFunctionCall(ctx.functionCall()),
    }

    return this._addMeta(node, ctx)
  }

  public visitFunctionCall(
    ctx: SP.FunctionCallContext
  ): AST.FunctionCall & WithMeta {
    let args: AST.Expression[] = []
    const names = []
    const identifiers = []

    const ctxArgs = ctx.functionCallArguments()
    const ctxArgsExpressionList = ctxArgs.expressionList()
    const ctxArgsNameValueList = ctxArgs.nameValueList()
    if (ctxArgsExpressionList) {
      args = ctxArgsExpressionList
        .expression()
        .map((exprCtx) => this.visitExpression(exprCtx))
    } else if (ctxArgsNameValueList) {
      for (const nameValue of ctxArgsNameValueList.nameValue()) {
        args.push(this.visitExpression(nameValue.expression()))
        names.push(this._toText(nameValue.identifier()))
        identifiers.push(this.visitIdentifier(nameValue.identifier()))
      }
    }

    const node: AST.FunctionCall = {
      type: 'FunctionCall',
      expression: this.visitExpression(ctx.expression()),
      arguments: args,
      names,
      identifiers,
    }

    return this._addMeta(node, ctx)
  }

  public visitStructDefinition(
    ctx: SP.StructDefinitionContext
  ): AST.StructDefinition & WithMeta {
    const node: AST.StructDefinition = {
      type: 'StructDefinition',
      name: this._toText(ctx.identifier()),
      members: ctx
        .variableDeclaration()
        .map((x) => this.visitVariableDeclaration(x)),
    }

    return this._addMeta(node, ctx)
  }

  public visitWhileStatement(
    ctx: SP.WhileStatementContext
  ): AST.WhileStatement & WithMeta {
    const node: AST.WhileStatement = {
      type: 'WhileStatement',
      condition: this.visitExpression(ctx.expression()),
      body: this.visitStatement(ctx.statement()),
    }

    return this._addMeta(node, ctx)
  }

  public visitDoWhileStatement(
    ctx: SP.DoWhileStatementContext
  ): AST.DoWhileStatement & WithMeta {
    const node: AST.DoWhileStatement = {
      type: 'DoWhileStatement',
      condition: this.visitExpression(ctx.expression()),
      body: this.visitStatement(ctx.statement()),
    }

    return this._addMeta(node, ctx)
  }

  public visitIfStatement(
    ctx: SP.IfStatementContext
  ): AST.IfStatement & WithMeta {
    const trueBody = this.visitStatement(ctx.statement(0))

    let falseBody = null
    if (ctx.statement().length > 1) {
      falseBody = this.visitStatement(ctx.statement(1))
    }

    const node: AST.IfStatement = {
      type: 'IfStatement',
      condition: this.visitExpression(ctx.expression()),
      trueBody,
      falseBody,
    }

    return this._addMeta(node, ctx)
  }

  public visitTryStatement(
    ctx: SP.TryStatementContext
  ): AST.TryStatement & WithMeta {
    let returnParameters = null
    const ctxReturnParameters = ctx.returnParameters()
    if (ctxReturnParameters !== undefined) {
      returnParameters = this.visitReturnParameters(ctxReturnParameters)
    }

    const catchClauses = ctx
      .catchClause()
      .map((exprCtx) => this.visitCatchClause(exprCtx))

    const node: AST.TryStatement = {
      type: 'TryStatement',
      expression: this.visitExpression(ctx.expression()),
      returnParameters,
      body: this.visitBlock(ctx.block()),
      catchClauses,
    }

    return this._addMeta(node, ctx)
  }

  public visitCatchClause(
    ctx: SP.CatchClauseContext
  ): AST.CatchClause & WithMeta {
    let parameters = null
    if (ctx.parameterList()) {
      parameters = this.visitParameterList(ctx.parameterList()!)
    }

    if (
      ctx.identifier() &&
      this._toText(ctx.identifier()!) !== 'Error' &&
      this._toText(ctx.identifier()!) !== 'Panic'
    ) {
      throw new Error('Expected "Error" or "Panic" identifier in catch clause')
    }

    let kind = null
    const ctxIdentifier = ctx.identifier()
    if (ctxIdentifier !== undefined) {
      kind = this._toText(ctxIdentifier)
    }

    const node: AST.CatchClause = {
      type: 'CatchClause',
      // deprecated, use the `kind` property instead,
      isReasonStringType: kind === 'Error',
      kind,
      parameters,
      body: this.visitBlock(ctx.block()),
    }

    return this._addMeta(node, ctx)
  }

  public visitExpressionStatement(
    ctx: SP.ExpressionStatementContext
  ): AST.ExpressionStatement & WithMeta {
    if (!ctx) {
      return null as any
    }
    const node: AST.ExpressionStatement = {
      type: 'ExpressionStatement',
      expression: this.visitExpression(ctx.expression()),
    }

    return this._addMeta(node, ctx)
  }

  public visitNumberLiteral(
    ctx: SP.NumberLiteralContext
  ): AST.NumberLiteral & WithMeta {
    const number = this._toText(ctx.getChild(0))
    let subdenomination = null

    if (ctx.children?.length === 2) {
      subdenomination = this._toText(ctx.getChild(1))
    }

    const node: AST.NumberLiteral = {
      type: 'NumberLiteral',
      number,
      subdenomination: subdenomination as AST.NumberLiteral['subdenomination'],
    }

    return this._addMeta(node, ctx)
  }

  public visitMappingKey(
    ctx: SP.MappingKeyContext
  ): (AST.ElementaryTypeName | AST.UserDefinedTypeName) & WithMeta {
    if (ctx.elementaryTypeName()) {
      return this.visitElementaryTypeName(ctx.elementaryTypeName()!)
    } else if (ctx.userDefinedTypeName()) {
      return this.visitUserDefinedTypeName(ctx.userDefinedTypeName()!)
    } else {
      throw new Error(
        'Expected MappingKey to have either ' +
          'elementaryTypeName or userDefinedTypeName'
      )
    }
  }

  public visitMapping(ctx: SP.MappingContext): AST.Mapping & WithMeta {
    const node: AST.Mapping = {
      type: 'Mapping',
      keyType: this.visitMappingKey(ctx.mappingKey()),
      valueType: this.visitTypeName(ctx.typeName()),
    }

    return this._addMeta(node, ctx)
  }

  public visitModifierDefinition(
    ctx: SP.ModifierDefinitionContext
  ): AST.ModifierDefinition & WithMeta {
    let parameters = null
    if (ctx.parameterList()) {
      parameters = this.visitParameterList(ctx.parameterList()!)
    }

    let isVirtual = false
    if (ctx.VirtualKeyword().length > 0) {
      isVirtual = true
    }

    let override
    const overrideSpecifier = ctx.overrideSpecifier()
    if (overrideSpecifier.length === 0) {
      override = null
    } else {
      override = overrideSpecifier[0]
        .userDefinedTypeName()
        .map((x) => this.visitUserDefinedTypeName(x))
    }

    let body = null
    const blockCtx = ctx.block()
    if (blockCtx !== undefined) {
      body = this.visitBlock(blockCtx)
    }

    const node: AST.ModifierDefinition = {
      type: 'ModifierDefinition',
      name: this._toText(ctx.identifier()),
      parameters,
      body,
      isVirtual,
      override,
    }

    return this._addMeta(node, ctx)
  }

  public visitUncheckedStatement(
    ctx: SP.UncheckedStatementContext
  ): AST.UncheckedStatement & WithMeta {
    const node: AST.UncheckedStatement = {
      type: 'UncheckedStatement',
      block: this.visitBlock(ctx.block()),
    }

    return this._addMeta(node, ctx)
  }

  public visitExpression(ctx: SP.ExpressionContext): AST.Expression & WithMeta {
    let op: string

    switch (ctx.children!.length) {
      case 1: {
        // primary expression
        const primaryExpressionCtx = ctx.tryGetRuleContext(
          0,
          SP.PrimaryExpressionContext
        )
        if (primaryExpressionCtx === undefined) {
          throw new Error(
            'Assertion error: primary expression should exist when children length is 1'
          )
        }
        return this.visitPrimaryExpression(primaryExpressionCtx)
      }
      case 2:
        op = this._toText(ctx.getChild(0))

        // new expression
        if (op === 'new') {
          const node: AST.NewExpression = {
            type: 'NewExpression',
            typeName: this.visitTypeName(ctx.typeName()!),
          }
          return this._addMeta(node, ctx)
        }

        // prefix operators
        if (AST.unaryOpValues.includes(op as AST.UnaryOp)) {
          const node: AST.UnaryOperation = {
            type: 'UnaryOperation',
            operator: op as AST.UnaryOp,
            subExpression: this.visitExpression(
              ctx.getRuleContext(0, SP.ExpressionContext)
            ),
            isPrefix: true,
          }
          return this._addMeta(node, ctx)
        }

        op = this._toText(ctx.getChild(1))!

        // postfix operators
        if (['++', '--'].includes(op)) {
          const node: AST.UnaryOperation = {
            type: 'UnaryOperation',
            operator: op as AST.UnaryOp,
            subExpression: this.visitExpression(
              ctx.getRuleContext(0, SP.ExpressionContext)
            ),
            isPrefix: false,
          }
          return this._addMeta(node, ctx)
        }
        break

      case 3:
        // treat parenthesis as no-op
        if (
          this._toText(ctx.getChild(0)) === '(' &&
          this._toText(ctx.getChild(2)) === ')'
        ) {
          const node: AST.TupleExpression = {
            type: 'TupleExpression',
            components: [
              this.visitExpression(ctx.getRuleContext(0, SP.ExpressionContext)),
            ],
            isArray: false,
          }
          return this._addMeta(node, ctx)
        }

        op = this._toText(ctx.getChild(1))!

        // member access
        if (op === '.') {
          const node: AST.MemberAccess = {
            type: 'MemberAccess',
            expression: this.visitExpression(ctx.expression(0)),
            memberName: this._toText(ctx.identifier()!),
          }
          return this._addMeta(node, ctx)
        }

        if (isBinOp(op)) {
          const node: AST.BinaryOperation = {
            type: 'BinaryOperation',
            operator: op,
            left: this.visitExpression(ctx.expression(0)),
            right: this.visitExpression(ctx.expression(1)),
          }
          return this._addMeta(node, ctx)
        }
        break

      case 4:
        // function call
        if (
          this._toText(ctx.getChild(1)) === '(' &&
          this._toText(ctx.getChild(3)) === ')'
        ) {
          let args: AST.Expression[] = []
          const names = []
          const identifiers = []

          const ctxArgs = ctx.functionCallArguments()!
          if (ctxArgs.expressionList()) {
            args = ctxArgs
              .expressionList()!
              .expression()
              .map((exprCtx) => this.visitExpression(exprCtx))
          } else if (ctxArgs.nameValueList()) {
            for (const nameValue of ctxArgs.nameValueList()!.nameValue()) {
              args.push(this.visitExpression(nameValue.expression()))
              names.push(this._toText(nameValue.identifier()))
              identifiers.push(this.visitIdentifier(nameValue.identifier()))
            }
          }

          const node: AST.FunctionCall = {
            type: 'FunctionCall',
            expression: this.visitExpression(ctx.expression(0)),
            arguments: args,
            names,
            identifiers,
          }

          return this._addMeta(node, ctx)
        }

        // index access
        if (
          this._toText(ctx.getChild(1)) === '[' &&
          this._toText(ctx.getChild(3)) === ']'
        ) {
          if (ctx.getChild(2).text === ':') {
            const node: AST.IndexRangeAccess = {
              type: 'IndexRangeAccess',
              base: this.visitExpression(ctx.expression(0)),
            }

            return this._addMeta(node, ctx)
          }

          const node: AST.IndexAccess = {
            type: 'IndexAccess',
            base: this.visitExpression(ctx.expression(0)),
            index: this.visitExpression(ctx.expression(1)),
          }

          return this._addMeta(node, ctx)
        }

        // expression with nameValueList
        if (
          this._toText(ctx.getChild(1)) === '{' &&
          this._toText(ctx.getChild(3)) === '}'
        ) {
          const node: AST.NameValueExpression = {
            type: 'NameValueExpression',
            expression: this.visitExpression(ctx.expression(0)),
            arguments: this.visitNameValueList(ctx.nameValueList()!),
          }

          return this._addMeta(node, ctx)
        }

        break

      case 5:
        // ternary operator
        if (
          this._toText(ctx.getChild(1)) === '?' &&
          this._toText(ctx.getChild(3)) === ':'
        ) {
          const node: AST.Conditional = {
            type: 'Conditional',
            condition: this.visitExpression(ctx.expression(0)),
            trueExpression: this.visitExpression(ctx.expression(1)),
            falseExpression: this.visitExpression(ctx.expression(2)),
          }

          return this._addMeta(node, ctx)
        }

        // index range access
        if (
          this._toText(ctx.getChild(1)) === '[' &&
          this._toText(ctx.getChild(2)) === ':' &&
          this._toText(ctx.getChild(4)) === ']'
        ) {
          const node: AST.IndexRangeAccess = {
            type: 'IndexRangeAccess',
            base: this.visitExpression(ctx.expression(0)),
            indexEnd: this.visitExpression(ctx.expression(1)),
          }

          return this._addMeta(node, ctx)
        } else if (
          this._toText(ctx.getChild(1)) === '[' &&
          this._toText(ctx.getChild(3)) === ':' &&
          this._toText(ctx.getChild(4)) === ']'
        ) {
          const node: AST.IndexRangeAccess = {
            type: 'IndexRangeAccess',
            base: this.visitExpression(ctx.expression(0)),
            indexStart: this.visitExpression(ctx.expression(1)),
          }

          return this._addMeta(node, ctx)
        }
        break

      case 6:
        // index range access
        if (
          this._toText(ctx.getChild(1)) === '[' &&
          this._toText(ctx.getChild(3)) === ':' &&
          this._toText(ctx.getChild(5)) === ']'
        ) {
          const node: AST.IndexRangeAccess = {
            type: 'IndexRangeAccess',
            base: this.visitExpression(ctx.expression(0)),
            indexStart: this.visitExpression(ctx.expression(1)),
            indexEnd: this.visitExpression(ctx.expression(2)),
          }

          return this._addMeta(node, ctx)
        }
        break
    }

    throw new Error('Unrecognized expression')
  }

  public visitNameValueList(
    ctx: SP.NameValueListContext
  ): AST.NameValueList & WithMeta {
    const names: string[] = []
    const identifiers: AST.Identifier[] = []
    const args: AST.Expression[] = []

    for (const nameValue of ctx.nameValue()) {
      names.push(this._toText(nameValue.identifier()))
      identifiers.push(this.visitIdentifier(nameValue.identifier()))
      args.push(this.visitExpression(nameValue.expression()))
    }

    const node: AST.NameValueList = {
      type: 'NameValueList',
      names,
      identifiers,
      arguments: args,
    }

    return this._addMeta(node, ctx)
  }

  public visitFileLevelConstant(ctx: SP.FileLevelConstantContext) {
    const type = this.visitTypeName(ctx.typeName())
    const iden = ctx.identifier()
    const name = this._toText(iden)

    const expression = this.visitExpression(ctx.expression())

    const node: AST.FileLevelConstant = {
      type: 'FileLevelConstant',
      typeName: type,
      name,
      initialValue: expression,
      isDeclaredConst: true,
      isImmutable: false,
    }

    return this._addMeta(node, ctx)
  }

  public visitForStatement(ctx: SP.ForStatementContext) {
    let conditionExpression: any = this.visitExpressionStatement(
      ctx.expressionStatement()!
    )
    if (conditionExpression) {
      conditionExpression = conditionExpression.expression
    }
    const node: AST.ForStatement = {
      type: 'ForStatement',
      initExpression: ctx.simpleStatement()
        ? this.visitSimpleStatement(ctx.simpleStatement()!)
        : null,
      conditionExpression,
      loopExpression: {
        type: 'ExpressionStatement',
        expression:
          ctx.expression() !== undefined
            ? this.visitExpression(ctx.expression()!)
            : null,
      },
      body: this.visitStatement(ctx.statement()),
    }

    return this._addMeta(node, ctx)
  }

  public visitHexLiteral(ctx: SP.HexLiteralContext) {
    const parts = ctx
      .HexLiteralFragment()
      .map((x) => this._toText(x))
      .map((x) => x.substring(4, x.length - 1))

    const node: AST.HexLiteral = {
      type: 'HexLiteral',
      value: parts.join(''),
      parts,
    }

    return this._addMeta(node, ctx)
  }

  public visitPrimaryExpression(
    ctx: SP.PrimaryExpressionContext
  ): AST.PrimaryExpression & WithMeta {
    if (ctx.BooleanLiteral()) {
      const node: AST.BooleanLiteral = {
        type: 'BooleanLiteral',
        value: this._toText(ctx.BooleanLiteral()!) === 'true',
      }

      return this._addMeta(node, ctx)
    }

    if (ctx.hexLiteral()) {
      return this.visitHexLiteral(ctx.hexLiteral()!)
    }

    if (ctx.stringLiteral()) {
      const fragments = ctx
        .stringLiteral()!
        .StringLiteralFragment()
        .map((stringLiteralFragmentCtx: any) => {
          let text = this._toText(stringLiteralFragmentCtx)!

          const isUnicode = text.slice(0, 7) === 'unicode'
          if (isUnicode) {
            text = text.slice(7)
          }
          const singleQuotes = text[0] === "'"
          const textWithoutQuotes = text.substring(1, text.length - 1)
          const value = singleQuotes
            ? textWithoutQuotes.replace(new RegExp("\\\\'", 'g'), "'")
            : textWithoutQuotes.replace(new RegExp('\\\\"', 'g'), '"')

          return { value, isUnicode }
        })

      const parts = fragments.map((x: any) => x.value)

      const node: AST.StringLiteral = {
        type: 'StringLiteral',
        value: parts.join(''),
        parts,
        isUnicode: fragments.map((x: any) => x.isUnicode),
      }

      return this._addMeta(node, ctx)
    }

    if (ctx.numberLiteral()) {
      return this.visitNumberLiteral(ctx.numberLiteral()!)
    }

    if (ctx.TypeKeyword()) {
      const node: AST.Identifier = {
        type: 'Identifier',
        name: 'type',
      }

      return this._addMeta(node, ctx)
    }

    if (ctx.typeName()) {
      return this.visitTypeName(ctx.typeName()!)
    }

    return this.visit(ctx.getChild(0)) as any
  }

  public visitTupleExpression(
    ctx: SP.TupleExpressionContext
  ): AST.TupleExpression & WithMeta {
    // remove parentheses
    const children = ctx.children!.slice(1, -1)
    const components = this._mapCommasToNulls(children).map((expr) => {
      // add a null for each empty value
      if (expr === null) {
        return null
      }
      return this.visit(expr)
    })

    const node: AST.TupleExpression = {
      type: 'TupleExpression',
      components,
      isArray: this._toText(ctx.getChild(0)) === '[',
    }

    return this._addMeta(node, ctx)
  }

  public buildIdentifierList(ctx: SP.IdentifierListContext) {
    // remove parentheses
    const children = ctx.children!.slice(1, -1)
    const identifiers = ctx.identifier()
    let i = 0
    return this._mapCommasToNulls(children).map((idenOrNull) => {
      // add a null for each empty value
      if (!idenOrNull) {
        return null
      }

      const iden = identifiers[i]
      i++

      const node: AST.VariableDeclaration = {
        type: 'VariableDeclaration',
        name: this._toText(iden),
        identifier: this.visitIdentifier(iden),
        isStateVar: false,
        isIndexed: false,
        typeName: null,
        storageLocation: null,
        expression: null,
      }

      return this._addMeta(node, iden)
    })
  }

  public buildVariableDeclarationList(
    ctx: SP.VariableDeclarationListContext
  ): Array<(AST.VariableDeclaration & WithMeta) | null> {
    const variableDeclarations = ctx.variableDeclaration()
    let i = 0
    return this._mapCommasToNulls(ctx.children ?? []).map((declOrNull) => {
      // add a null for each empty value
      if (!declOrNull) {
        return null
      }

      const decl = variableDeclarations[i]
      i++

      let storageLocation: string | null = null
      if (decl.storageLocation()) {
        storageLocation = this._toText(decl.storageLocation()!)
      }

      const identifierCtx = decl.identifier()

      const result: AST.VariableDeclaration = {
        type: 'VariableDeclaration',
        name: this._toText(identifierCtx),
        identifier: this.visitIdentifier(identifierCtx),
        typeName: this.visitTypeName(decl.typeName()),
        storageLocation,
        isStateVar: false,
        isIndexed: false,
        expression: null,
      }

      return this._addMeta(result, decl)
    })
  }

  public visitImportDirective(ctx: SP.ImportDirectiveContext) {
    const pathString = this._toText(ctx.importPath())
    let unitAlias = null
    let unitAliasIdentifier = null
    let symbolAliases = null
    let symbolAliasesIdentifiers = null

    if (ctx.importDeclaration().length > 0) {
      symbolAliases = ctx.importDeclaration().map((decl) => {
        const symbol = this._toText(decl.identifier(0))
        let alias = null
        if (decl.identifier().length > 1) {
          alias = this._toText(decl.identifier(1))
        }
        return [symbol, alias] as [string, string | null]
      })
      symbolAliasesIdentifiers = ctx.importDeclaration().map((decl) => {
        const symbolIdentifier = this.visitIdentifier(decl.identifier(0))
        let aliasIdentifier = null
        if (decl.identifier().length > 1) {
          aliasIdentifier = this.visitIdentifier(decl.identifier(1))
        }
        return [symbolIdentifier, aliasIdentifier] as [
          AST.Identifier,
          AST.Identifier | null
        ]
      })
    } else {
      const identifierCtxList = ctx.identifier()
      if (identifierCtxList.length === 0) {
        // nothing to do
      } else if (identifierCtxList.length === 1) {
        const aliasIdentifierCtx = ctx.identifier(0)
        unitAlias = this._toText(aliasIdentifierCtx)
        unitAliasIdentifier = this.visitIdentifier(aliasIdentifierCtx)
      } else if (identifierCtxList.length === 2) {
        const aliasIdentifierCtx = ctx.identifier(1)
        unitAlias = this._toText(aliasIdentifierCtx)
        unitAliasIdentifier = this.visitIdentifier(aliasIdentifierCtx)
      } else {
        throw new Error(
          'Assertion error: an import should have one or two identifiers'
        )
      }
    }

    const path = pathString.substring(1, pathString.length - 1)

    const pathLiteral: AST.StringLiteral = {
      type: 'StringLiteral',
      value: path,
      parts: [path],
      isUnicode: [false], // paths in imports don't seem to support unicode literals
    }

    const node: AST.ImportDirective = {
      type: 'ImportDirective',
      path,
      pathLiteral: this._addMeta(pathLiteral, ctx.importPath()),
      unitAlias,
      unitAliasIdentifier,
      symbolAliases,
      symbolAliasesIdentifiers,
    }

    return this._addMeta(node, ctx)
  }

  public buildEventParameterList(ctx: SP.EventParameterListContext) {
    return ctx.eventParameter().map((paramCtx: any) => {
      const type = this.visit(paramCtx.typeName())
      let name = null
      if (paramCtx.identifier()) {
        name = this._toText(paramCtx.identifier())
      }

      return {
        type: 'VariableDeclaration',
        typeName: type,
        name,
        isStateVar: false,
        isIndexed: !!paramCtx.IndexedKeyword(0),
      }
    })
  }

  public visitReturnParameters(
    ctx: SP.ReturnParametersContext
  ): (AST.VariableDeclaration & WithMeta)[] {
    return this.visitParameterList(ctx.parameterList())
  }

  public visitParameterList(
    ctx: SP.ParameterListContext
  ): (AST.VariableDeclaration & WithMeta)[] {
    return ctx.parameter().map((paramCtx: any) => this.visitParameter(paramCtx))
  }

  public visitInlineAssemblyStatement(ctx: SP.InlineAssemblyStatementContext) {
    let language: string | null = null
    if (ctx.StringLiteralFragment()) {
      language = this._toText(ctx.StringLiteralFragment()!)!
      language = language.substring(1, language.length - 1)
    }

    const flags = []
    const flag = ctx.inlineAssemblyStatementFlag()
    if (flag !== undefined) {
      const flagString = this._toText(flag.stringLiteral())
      flags.push(flagString.slice(1, flagString.length - 1))
    }

    const node: AST.InlineAssemblyStatement = {
      type: 'InlineAssemblyStatement',
      language,
      flags,
      body: this.visitAssemblyBlock(ctx.assemblyBlock()),
    }

    return this._addMeta(node, ctx)
  }

  public visitAssemblyBlock(
    ctx: SP.AssemblyBlockContext
  ): AST.AssemblyBlock & WithMeta {
    const operations = ctx
      .assemblyItem()
      .map((item) => this.visitAssemblyItem(item))

    const node: AST.AssemblyBlock = {
      type: 'AssemblyBlock',
      operations,
    }

    return this._addMeta(node, ctx)
  }

  public visitAssemblyItem(
    ctx: SP.AssemblyItemContext
  ): AST.AssemblyItem & WithMeta {
    let text

    if (ctx.hexLiteral()) {
      return this.visitHexLiteral(ctx.hexLiteral()!)
    }

    if (ctx.stringLiteral()) {
      text = this._toText(ctx.stringLiteral()!)!
      const value = text.substring(1, text.length - 1)
      const node: AST.StringLiteral = {
        type: 'StringLiteral',
        value,
        parts: [value],
        isUnicode: [false], // assembly doesn't seem to support unicode literals right now
      }

      return this._addMeta(node, ctx)
    }

    if (ctx.BreakKeyword()) {
      const node: AST.Break = {
        type: 'Break',
      }

      return this._addMeta(node, ctx)
    }

    if (ctx.ContinueKeyword()) {
      const node: AST.Continue = {
        type: 'Continue',
      }

      return this._addMeta(node, ctx)
    }

    return this.visit(ctx.getChild(0)) as AST.AssemblyItem & WithMeta
  }

  public visitAssemblyExpression(ctx: SP.AssemblyExpressionContext) {
    return this.visit(ctx.getChild(0)) as AST.AssemblyExpression & WithMeta
  }

  public visitAssemblyCall(ctx: SP.AssemblyCallContext) {
    const functionName = this._toText(ctx.getChild(0))
    const args = ctx
      .assemblyExpression()
      .map((assemblyExpr) => this.visitAssemblyExpression(assemblyExpr))

    const node: AST.AssemblyCall = {
      type: 'AssemblyCall',
      functionName,
      arguments: args,
    }

    return this._addMeta(node, ctx)
  }

  public visitAssemblyLiteral(
    ctx: SP.AssemblyLiteralContext
  ): AST.AssemblyLiteral & WithMeta {
    let text

    if (ctx.stringLiteral()) {
      text = this._toText(ctx)!
      const value = text.substring(1, text.length - 1)
      const node: AST.StringLiteral = {
        type: 'StringLiteral',
        value,
        parts: [value],
        isUnicode: [false], // assembly doesn't seem to support unicode literals right now
      }

      return this._addMeta(node, ctx)
    }

    if (ctx.BooleanLiteral()) {
      const node: AST.BooleanLiteral = {
        type: 'BooleanLiteral',
        value: this._toText(ctx.BooleanLiteral()!) === 'true',
      }

      return this._addMeta(node, ctx);
    }

    if (ctx.DecimalNumber()) {
      const node: AST.DecimalNumber = {
        type: 'DecimalNumber',
        value: this._toText(ctx),
      }

      return this._addMeta(node, ctx)
    }

    if (ctx.HexNumber()) {
      const node: AST.HexNumber = {
        type: 'HexNumber',
        value: this._toText(ctx),
      }

      return this._addMeta(node, ctx)
    }

    if (ctx.hexLiteral()) {
      return this.visitHexLiteral(ctx.hexLiteral()!)
    }

    throw new Error('Should never reach here')
  }

  public visitAssemblySwitch(ctx: SP.AssemblySwitchContext) {
    const node: AST.AssemblySwitch = {
      type: 'AssemblySwitch',
      expression: this.visitAssemblyExpression(ctx.assemblyExpression()),
      cases: ctx.assemblyCase().map((c) => this.visitAssemblyCase(c)),
    }

    return this._addMeta(node, ctx)
  }

  public visitAssemblyCase(
    ctx: SP.AssemblyCaseContext
  ): AST.AssemblyCase & WithMeta {
    let value = null
    if (this._toText(ctx.getChild(0)) === 'case') {
      value = this.visitAssemblyLiteral(ctx.assemblyLiteral()!)
    }

    const node: AST.AssemblyCase = {
      type: 'AssemblyCase',
      block: this.visitAssemblyBlock(ctx.assemblyBlock()),
      value,
      default: value === null,
    }

    return this._addMeta(node, ctx)
  }

  public visitAssemblyLocalDefinition(
    ctx: SP.AssemblyLocalDefinitionContext
  ): AST.AssemblyLocalDefinition & WithMeta {
    const ctxAssemblyIdentifierOrList = ctx.assemblyIdentifierOrList()
    let names
    if (ctxAssemblyIdentifierOrList.identifier()) {
      names = [this.visitIdentifier(ctxAssemblyIdentifierOrList.identifier()!)]
    } else if (ctxAssemblyIdentifierOrList.assemblyMember()) {
      names = [
        this.visitAssemblyMember(ctxAssemblyIdentifierOrList.assemblyMember()!),
      ]
    } else {
      names = ctxAssemblyIdentifierOrList
        .assemblyIdentifierList()!
        .identifier()!
        .map((x) => this.visitIdentifier(x))
    }

    let expression: AST.AssemblyExpression | null = null
    if (ctx.assemblyExpression() !== undefined) {
      expression = this.visitAssemblyExpression(ctx.assemblyExpression()!)
    }

    const node: AST.AssemblyLocalDefinition = {
      type: 'AssemblyLocalDefinition',
      names,
      expression,
    }

    return this._addMeta(node, ctx)
  }

  public visitAssemblyFunctionDefinition(
    ctx: SP.AssemblyFunctionDefinitionContext
  ) {
    const ctxAssemblyIdentifierList = ctx.assemblyIdentifierList()
    const args =
      ctxAssemblyIdentifierList !== undefined
        ? ctxAssemblyIdentifierList
            .identifier()
            .map((x) => this.visitIdentifier(x))
        : []

    const ctxAssemblyFunctionReturns = ctx.assemblyFunctionReturns()
    const returnArgs = ctxAssemblyFunctionReturns
      ? ctxAssemblyFunctionReturns
          .assemblyIdentifierList()!
          .identifier()
          .map((x) => this.visitIdentifier(x))
      : []

    const node: AST.AssemblyFunctionDefinition = {
      type: 'AssemblyFunctionDefinition',
      name: this._toText(ctx.identifier()),
      arguments: args,
      returnArguments: returnArgs,
      body: this.visitAssemblyBlock(ctx.assemblyBlock()),
    }

    return this._addMeta(node, ctx)
  }

  public visitAssemblyAssignment(ctx: SP.AssemblyAssignmentContext) {
    const ctxAssemblyIdentifierOrList = ctx.assemblyIdentifierOrList()
    let names
    if (ctxAssemblyIdentifierOrList.identifier()) {
      names = [this.visitIdentifier(ctxAssemblyIdentifierOrList.identifier()!)]
    } else if (ctxAssemblyIdentifierOrList.assemblyMember()) {
      names = [
        this.visitAssemblyMember(ctxAssemblyIdentifierOrList.assemblyMember()!),
      ]
    } else {
      names = ctxAssemblyIdentifierOrList
        .assemblyIdentifierList()!
        .identifier()
        .map((x) => this.visitIdentifier(x))
    }

    const node: AST.AssemblyAssignment = {
      type: 'AssemblyAssignment',
      names,
      expression: this.visitAssemblyExpression(ctx.assemblyExpression()),
    }

    return this._addMeta(node, ctx)
  }

  public visitAssemblyMember(
    ctx: SP.AssemblyMemberContext
  ): AST.AssemblyMemberAccess & WithMeta {
    const [accessed, member] = ctx.identifier()
    const node: AST.AssemblyMemberAccess = {
      type: 'AssemblyMemberAccess',
      expression: this.visitIdentifier(accessed),
      memberName: this.visitIdentifier(member),
    }

    return this._addMeta(node, ctx)
  }

  public visitLabelDefinition(ctx: SP.LabelDefinitionContext) {
    const node: AST.LabelDefinition = {
      type: 'LabelDefinition',
      name: this._toText(ctx.identifier()),
    }

    return this._addMeta(node, ctx)
  }

  public visitAssemblyStackAssignment(ctx: SP.AssemblyStackAssignmentContext) {
    const node: AST.AssemblyStackAssignment = {
      type: 'AssemblyStackAssignment',
      name: this._toText(ctx.identifier()),
      expression: this.visitAssemblyExpression(ctx.assemblyExpression()),
    }

    return this._addMeta(node, ctx)
  }

  public visitAssemblyFor(ctx: SP.AssemblyForContext) {
    // TODO remove these type assertions
    const node: AST.AssemblyFor = {
      type: 'AssemblyFor',
      pre: this.visit(ctx.getChild(1)) as
        | AST.AssemblyBlock
        | AST.AssemblyExpression,
      condition: this.visit(ctx.getChild(2)) as AST.AssemblyExpression,
      post: this.visit(ctx.getChild(3)) as
        | AST.AssemblyBlock
        | AST.AssemblyExpression,
      body: this.visit(ctx.getChild(4)) as AST.AssemblyBlock,
    }

    return this._addMeta(node, ctx)
  }

  public visitAssemblyIf(ctx: SP.AssemblyIfContext) {
    const node: AST.AssemblyIf = {
      type: 'AssemblyIf',
      condition: this.visitAssemblyExpression(ctx.assemblyExpression()),
      body: this.visitAssemblyBlock(ctx.assemblyBlock()),
    }

    return this._addMeta(node, ctx)
  }

  public visitContinueStatement(
    ctx: SP.ContinueStatementContext
  ): AST.ContinueStatement & WithMeta {
    const node: AST.ContinueStatement = {
      type: 'ContinueStatement',
    }

    return this._addMeta(node, ctx)
  }

  public visitBreakStatement(
    ctx: SP.BreakStatementContext
  ): AST.BreakStatement & WithMeta {
    const node: AST.BreakStatement = {
      type: 'BreakStatement',
    }

    return this._addMeta(node, ctx)
  }

  private _toText(ctx: ParserRuleContext | ParseTree): string {
    const text = ctx.text
    if (text === undefined) {
      throw new Error('Assertion error: text should never be undefiend')
    }

    return text
  }

  private _stateMutabilityToText(
    ctx: SP.StateMutabilityContext
  ): AST.FunctionDefinition['stateMutability'] {
    if (ctx.PureKeyword() !== undefined) {
      return 'pure'
    }
    if (ctx.ConstantKeyword() !== undefined) {
      return 'constant'
    }
    if (ctx.PayableKeyword() !== undefined) {
      return 'payable'
    }
    if (ctx.ViewKeyword() !== undefined) {
      return 'view'
    }

    throw new Error('Assertion error: non-exhaustive stateMutability check')
  }

  private _loc(ctx: ParserRuleContext): SourceLocation {
    const sourceLocation: SourceLocation = {
      start: {
        line: ctx.start.line,
        column: ctx.start.charPositionInLine,
      },
      end: {
        line: ctx.stop ? ctx.stop.line : ctx.start.line,
        column: ctx.stop
          ? ctx.stop.charPositionInLine
          : ctx.start.charPositionInLine,
      },
    }
    return sourceLocation
  }

  _range(ctx: ParserRuleContext): [number, number] {
    return [ctx.start.startIndex, ctx.stop?.stopIndex ?? ctx.start.startIndex]
  }

  private _addMeta<T extends AST.BaseASTNode>(
    node: T,
    ctx: ParserRuleContext
  ): T & WithMeta {
    const nodeWithMeta: AST.BaseASTNode = {
      type: node.type,
    }

    if (this.options.loc === true) {
      node.loc = this._loc(ctx)
    }
    if (this.options.range === true) {
      node.range = this._range(ctx)
    }

    return {
      ...nodeWithMeta,
      ...node,
    } as T & WithMeta
  }

  private _mapCommasToNulls(children: ParseTree[]) {
    if (children.length === 0) {
      return []
    }

    const values: Array<ParseTree | null> = []
    let comma = true

    for (const el of children) {
      if (comma) {
        if (this._toText(el) === ',') {
          values.push(null)
        } else {
          values.push(el)
          comma = false
        }
      } else {
        if (this._toText(el) !== ',') {
          throw new Error('expected comma')
        }
        comma = true
      }
    }

    if (comma) {
      values.push(null)
    }

    return values
  }
}

function isBinOp(op: string): op is AST.BinOp {
  return AST.binaryOpValues.includes(op as AST.BinOp)
}
