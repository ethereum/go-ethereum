import { ParseTreeVisitor } from 'antlr4';
import { SourceUnitContext } from "./SolidityParser.js";
import { PragmaDirectiveContext } from "./SolidityParser.js";
import { PragmaNameContext } from "./SolidityParser.js";
import { PragmaValueContext } from "./SolidityParser.js";
import { VersionContext } from "./SolidityParser.js";
import { VersionOperatorContext } from "./SolidityParser.js";
import { VersionConstraintContext } from "./SolidityParser.js";
import { ImportDeclarationContext } from "./SolidityParser.js";
import { ImportDirectiveContext } from "./SolidityParser.js";
import { ImportPathContext } from "./SolidityParser.js";
import { ContractDefinitionContext } from "./SolidityParser.js";
import { InheritanceSpecifierContext } from "./SolidityParser.js";
import { CustomStorageLayoutContext } from "./SolidityParser.js";
import { ContractPartContext } from "./SolidityParser.js";
import { StateVariableDeclarationContext } from "./SolidityParser.js";
import { FileLevelConstantContext } from "./SolidityParser.js";
import { CustomErrorDefinitionContext } from "./SolidityParser.js";
import { TypeDefinitionContext } from "./SolidityParser.js";
import { UsingForDeclarationContext } from "./SolidityParser.js";
import { UsingForObjectContext } from "./SolidityParser.js";
import { UsingForObjectDirectiveContext } from "./SolidityParser.js";
import { UserDefinableOperatorsContext } from "./SolidityParser.js";
import { StructDefinitionContext } from "./SolidityParser.js";
import { ModifierDefinitionContext } from "./SolidityParser.js";
import { ModifierInvocationContext } from "./SolidityParser.js";
import { FunctionDefinitionContext } from "./SolidityParser.js";
import { FunctionDescriptorContext } from "./SolidityParser.js";
import { ReturnParametersContext } from "./SolidityParser.js";
import { ModifierListContext } from "./SolidityParser.js";
import { EventDefinitionContext } from "./SolidityParser.js";
import { EnumValueContext } from "./SolidityParser.js";
import { EnumDefinitionContext } from "./SolidityParser.js";
import { ParameterListContext } from "./SolidityParser.js";
import { ParameterContext } from "./SolidityParser.js";
import { EventParameterListContext } from "./SolidityParser.js";
import { EventParameterContext } from "./SolidityParser.js";
import { FunctionTypeParameterListContext } from "./SolidityParser.js";
import { FunctionTypeParameterContext } from "./SolidityParser.js";
import { VariableDeclarationContext } from "./SolidityParser.js";
import { TypeNameContext } from "./SolidityParser.js";
import { UserDefinedTypeNameContext } from "./SolidityParser.js";
import { MappingKeyContext } from "./SolidityParser.js";
import { MappingContext } from "./SolidityParser.js";
import { MappingKeyNameContext } from "./SolidityParser.js";
import { MappingValueNameContext } from "./SolidityParser.js";
import { FunctionTypeNameContext } from "./SolidityParser.js";
import { StorageLocationContext } from "./SolidityParser.js";
import { StateMutabilityContext } from "./SolidityParser.js";
import { BlockContext } from "./SolidityParser.js";
import { StatementContext } from "./SolidityParser.js";
import { ExpressionStatementContext } from "./SolidityParser.js";
import { IfStatementContext } from "./SolidityParser.js";
import { TryStatementContext } from "./SolidityParser.js";
import { CatchClauseContext } from "./SolidityParser.js";
import { WhileStatementContext } from "./SolidityParser.js";
import { SimpleStatementContext } from "./SolidityParser.js";
import { UncheckedStatementContext } from "./SolidityParser.js";
import { ForStatementContext } from "./SolidityParser.js";
import { InlineAssemblyStatementContext } from "./SolidityParser.js";
import { InlineAssemblyStatementFlagContext } from "./SolidityParser.js";
import { DoWhileStatementContext } from "./SolidityParser.js";
import { ContinueStatementContext } from "./SolidityParser.js";
import { BreakStatementContext } from "./SolidityParser.js";
import { ReturnStatementContext } from "./SolidityParser.js";
import { ThrowStatementContext } from "./SolidityParser.js";
import { EmitStatementContext } from "./SolidityParser.js";
import { RevertStatementContext } from "./SolidityParser.js";
import { VariableDeclarationStatementContext } from "./SolidityParser.js";
import { VariableDeclarationListContext } from "./SolidityParser.js";
import { IdentifierListContext } from "./SolidityParser.js";
import { ElementaryTypeNameContext } from "./SolidityParser.js";
import { ExpressionContext } from "./SolidityParser.js";
import { PrimaryExpressionContext } from "./SolidityParser.js";
import { ExpressionListContext } from "./SolidityParser.js";
import { NameValueListContext } from "./SolidityParser.js";
import { NameValueContext } from "./SolidityParser.js";
import { FunctionCallArgumentsContext } from "./SolidityParser.js";
import { FunctionCallContext } from "./SolidityParser.js";
import { AssemblyBlockContext } from "./SolidityParser.js";
import { AssemblyItemContext } from "./SolidityParser.js";
import { AssemblyExpressionContext } from "./SolidityParser.js";
import { AssemblyMemberContext } from "./SolidityParser.js";
import { AssemblyCallContext } from "./SolidityParser.js";
import { AssemblyLocalDefinitionContext } from "./SolidityParser.js";
import { AssemblyAssignmentContext } from "./SolidityParser.js";
import { AssemblyIdentifierOrListContext } from "./SolidityParser.js";
import { AssemblyIdentifierListContext } from "./SolidityParser.js";
import { AssemblyStackAssignmentContext } from "./SolidityParser.js";
import { LabelDefinitionContext } from "./SolidityParser.js";
import { AssemblySwitchContext } from "./SolidityParser.js";
import { AssemblyCaseContext } from "./SolidityParser.js";
import { AssemblyFunctionDefinitionContext } from "./SolidityParser.js";
import { AssemblyFunctionReturnsContext } from "./SolidityParser.js";
import { AssemblyForContext } from "./SolidityParser.js";
import { AssemblyIfContext } from "./SolidityParser.js";
import { AssemblyLiteralContext } from "./SolidityParser.js";
import { TupleExpressionContext } from "./SolidityParser.js";
import { NumberLiteralContext } from "./SolidityParser.js";
import { IdentifierContext } from "./SolidityParser.js";
import { HexLiteralContext } from "./SolidityParser.js";
import { OverrideSpecifierContext } from "./SolidityParser.js";
import { StringLiteralContext } from "./SolidityParser.js";
/**
 * This interface defines a complete generic visitor for a parse tree produced
 * by `SolidityParser`.
 *
 * @param <Result> The return type of the visit operation. Use `void` for
 * operations with no return type.
 */
export default class SolidityVisitor<Result> extends ParseTreeVisitor<Result> {
    /**
     * Visit a parse tree produced by `SolidityParser.sourceUnit`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitSourceUnit?: (ctx: SourceUnitContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.pragmaDirective`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitPragmaDirective?: (ctx: PragmaDirectiveContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.pragmaName`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitPragmaName?: (ctx: PragmaNameContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.pragmaValue`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitPragmaValue?: (ctx: PragmaValueContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.version`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitVersion?: (ctx: VersionContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.versionOperator`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitVersionOperator?: (ctx: VersionOperatorContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.versionConstraint`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitVersionConstraint?: (ctx: VersionConstraintContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.importDeclaration`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitImportDeclaration?: (ctx: ImportDeclarationContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.importDirective`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitImportDirective?: (ctx: ImportDirectiveContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.importPath`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitImportPath?: (ctx: ImportPathContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.contractDefinition`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitContractDefinition?: (ctx: ContractDefinitionContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.inheritanceSpecifier`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitInheritanceSpecifier?: (ctx: InheritanceSpecifierContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.customStorageLayout`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitCustomStorageLayout?: (ctx: CustomStorageLayoutContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.contractPart`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitContractPart?: (ctx: ContractPartContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.stateVariableDeclaration`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitStateVariableDeclaration?: (ctx: StateVariableDeclarationContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.fileLevelConstant`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitFileLevelConstant?: (ctx: FileLevelConstantContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.customErrorDefinition`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitCustomErrorDefinition?: (ctx: CustomErrorDefinitionContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.typeDefinition`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitTypeDefinition?: (ctx: TypeDefinitionContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.usingForDeclaration`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitUsingForDeclaration?: (ctx: UsingForDeclarationContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.usingForObject`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitUsingForObject?: (ctx: UsingForObjectContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.usingForObjectDirective`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitUsingForObjectDirective?: (ctx: UsingForObjectDirectiveContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.userDefinableOperators`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitUserDefinableOperators?: (ctx: UserDefinableOperatorsContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.structDefinition`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitStructDefinition?: (ctx: StructDefinitionContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.modifierDefinition`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitModifierDefinition?: (ctx: ModifierDefinitionContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.modifierInvocation`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitModifierInvocation?: (ctx: ModifierInvocationContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.functionDefinition`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitFunctionDefinition?: (ctx: FunctionDefinitionContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.functionDescriptor`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitFunctionDescriptor?: (ctx: FunctionDescriptorContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.returnParameters`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitReturnParameters?: (ctx: ReturnParametersContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.modifierList`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitModifierList?: (ctx: ModifierListContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.eventDefinition`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitEventDefinition?: (ctx: EventDefinitionContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.enumValue`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitEnumValue?: (ctx: EnumValueContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.enumDefinition`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitEnumDefinition?: (ctx: EnumDefinitionContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.parameterList`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitParameterList?: (ctx: ParameterListContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.parameter`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitParameter?: (ctx: ParameterContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.eventParameterList`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitEventParameterList?: (ctx: EventParameterListContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.eventParameter`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitEventParameter?: (ctx: EventParameterContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.functionTypeParameterList`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitFunctionTypeParameterList?: (ctx: FunctionTypeParameterListContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.functionTypeParameter`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitFunctionTypeParameter?: (ctx: FunctionTypeParameterContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.variableDeclaration`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitVariableDeclaration?: (ctx: VariableDeclarationContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.typeName`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitTypeName?: (ctx: TypeNameContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.userDefinedTypeName`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitUserDefinedTypeName?: (ctx: UserDefinedTypeNameContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.mappingKey`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitMappingKey?: (ctx: MappingKeyContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.mapping`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitMapping?: (ctx: MappingContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.mappingKeyName`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitMappingKeyName?: (ctx: MappingKeyNameContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.mappingValueName`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitMappingValueName?: (ctx: MappingValueNameContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.functionTypeName`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitFunctionTypeName?: (ctx: FunctionTypeNameContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.storageLocation`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitStorageLocation?: (ctx: StorageLocationContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.stateMutability`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitStateMutability?: (ctx: StateMutabilityContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.block`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitBlock?: (ctx: BlockContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.statement`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitStatement?: (ctx: StatementContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.expressionStatement`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitExpressionStatement?: (ctx: ExpressionStatementContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.ifStatement`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitIfStatement?: (ctx: IfStatementContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.tryStatement`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitTryStatement?: (ctx: TryStatementContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.catchClause`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitCatchClause?: (ctx: CatchClauseContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.whileStatement`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitWhileStatement?: (ctx: WhileStatementContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.simpleStatement`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitSimpleStatement?: (ctx: SimpleStatementContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.uncheckedStatement`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitUncheckedStatement?: (ctx: UncheckedStatementContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.forStatement`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitForStatement?: (ctx: ForStatementContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.inlineAssemblyStatement`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitInlineAssemblyStatement?: (ctx: InlineAssemblyStatementContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.inlineAssemblyStatementFlag`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitInlineAssemblyStatementFlag?: (ctx: InlineAssemblyStatementFlagContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.doWhileStatement`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitDoWhileStatement?: (ctx: DoWhileStatementContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.continueStatement`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitContinueStatement?: (ctx: ContinueStatementContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.breakStatement`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitBreakStatement?: (ctx: BreakStatementContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.returnStatement`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitReturnStatement?: (ctx: ReturnStatementContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.throwStatement`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitThrowStatement?: (ctx: ThrowStatementContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.emitStatement`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitEmitStatement?: (ctx: EmitStatementContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.revertStatement`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitRevertStatement?: (ctx: RevertStatementContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.variableDeclarationStatement`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitVariableDeclarationStatement?: (ctx: VariableDeclarationStatementContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.variableDeclarationList`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitVariableDeclarationList?: (ctx: VariableDeclarationListContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.identifierList`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitIdentifierList?: (ctx: IdentifierListContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.elementaryTypeName`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitElementaryTypeName?: (ctx: ElementaryTypeNameContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.expression`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitExpression?: (ctx: ExpressionContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.primaryExpression`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitPrimaryExpression?: (ctx: PrimaryExpressionContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.expressionList`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitExpressionList?: (ctx: ExpressionListContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.nameValueList`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitNameValueList?: (ctx: NameValueListContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.nameValue`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitNameValue?: (ctx: NameValueContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.functionCallArguments`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitFunctionCallArguments?: (ctx: FunctionCallArgumentsContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.functionCall`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitFunctionCall?: (ctx: FunctionCallContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.assemblyBlock`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitAssemblyBlock?: (ctx: AssemblyBlockContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.assemblyItem`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitAssemblyItem?: (ctx: AssemblyItemContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.assemblyExpression`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitAssemblyExpression?: (ctx: AssemblyExpressionContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.assemblyMember`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitAssemblyMember?: (ctx: AssemblyMemberContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.assemblyCall`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitAssemblyCall?: (ctx: AssemblyCallContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.assemblyLocalDefinition`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitAssemblyLocalDefinition?: (ctx: AssemblyLocalDefinitionContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.assemblyAssignment`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitAssemblyAssignment?: (ctx: AssemblyAssignmentContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.assemblyIdentifierOrList`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitAssemblyIdentifierOrList?: (ctx: AssemblyIdentifierOrListContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.assemblyIdentifierList`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitAssemblyIdentifierList?: (ctx: AssemblyIdentifierListContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.assemblyStackAssignment`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitAssemblyStackAssignment?: (ctx: AssemblyStackAssignmentContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.labelDefinition`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitLabelDefinition?: (ctx: LabelDefinitionContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.assemblySwitch`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitAssemblySwitch?: (ctx: AssemblySwitchContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.assemblyCase`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitAssemblyCase?: (ctx: AssemblyCaseContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.assemblyFunctionDefinition`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitAssemblyFunctionDefinition?: (ctx: AssemblyFunctionDefinitionContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.assemblyFunctionReturns`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitAssemblyFunctionReturns?: (ctx: AssemblyFunctionReturnsContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.assemblyFor`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitAssemblyFor?: (ctx: AssemblyForContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.assemblyIf`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitAssemblyIf?: (ctx: AssemblyIfContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.assemblyLiteral`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitAssemblyLiteral?: (ctx: AssemblyLiteralContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.tupleExpression`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitTupleExpression?: (ctx: TupleExpressionContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.numberLiteral`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitNumberLiteral?: (ctx: NumberLiteralContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.identifier`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitIdentifier?: (ctx: IdentifierContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.hexLiteral`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitHexLiteral?: (ctx: HexLiteralContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.overrideSpecifier`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitOverrideSpecifier?: (ctx: OverrideSpecifierContext) => Result;
    /**
     * Visit a parse tree produced by `SolidityParser.stringLiteral`.
     * @param ctx the parse tree
     * @return the visitor result
     */
    visitStringLiteral?: (ctx: StringLiteralContext) => Result;
}
