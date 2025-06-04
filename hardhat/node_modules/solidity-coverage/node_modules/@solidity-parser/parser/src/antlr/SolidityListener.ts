// Generated from antlr/Solidity.g4 by ANTLR 4.13.2

import {ParseTreeListener} from "antlr4";


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
 * This interface defines a complete listener for a parse tree produced by
 * `SolidityParser`.
 */
export default class SolidityListener extends ParseTreeListener {
	/**
	 * Enter a parse tree produced by `SolidityParser.sourceUnit`.
	 * @param ctx the parse tree
	 */
	enterSourceUnit?: (ctx: SourceUnitContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.sourceUnit`.
	 * @param ctx the parse tree
	 */
	exitSourceUnit?: (ctx: SourceUnitContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.pragmaDirective`.
	 * @param ctx the parse tree
	 */
	enterPragmaDirective?: (ctx: PragmaDirectiveContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.pragmaDirective`.
	 * @param ctx the parse tree
	 */
	exitPragmaDirective?: (ctx: PragmaDirectiveContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.pragmaName`.
	 * @param ctx the parse tree
	 */
	enterPragmaName?: (ctx: PragmaNameContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.pragmaName`.
	 * @param ctx the parse tree
	 */
	exitPragmaName?: (ctx: PragmaNameContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.pragmaValue`.
	 * @param ctx the parse tree
	 */
	enterPragmaValue?: (ctx: PragmaValueContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.pragmaValue`.
	 * @param ctx the parse tree
	 */
	exitPragmaValue?: (ctx: PragmaValueContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.version`.
	 * @param ctx the parse tree
	 */
	enterVersion?: (ctx: VersionContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.version`.
	 * @param ctx the parse tree
	 */
	exitVersion?: (ctx: VersionContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.versionOperator`.
	 * @param ctx the parse tree
	 */
	enterVersionOperator?: (ctx: VersionOperatorContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.versionOperator`.
	 * @param ctx the parse tree
	 */
	exitVersionOperator?: (ctx: VersionOperatorContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.versionConstraint`.
	 * @param ctx the parse tree
	 */
	enterVersionConstraint?: (ctx: VersionConstraintContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.versionConstraint`.
	 * @param ctx the parse tree
	 */
	exitVersionConstraint?: (ctx: VersionConstraintContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.importDeclaration`.
	 * @param ctx the parse tree
	 */
	enterImportDeclaration?: (ctx: ImportDeclarationContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.importDeclaration`.
	 * @param ctx the parse tree
	 */
	exitImportDeclaration?: (ctx: ImportDeclarationContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.importDirective`.
	 * @param ctx the parse tree
	 */
	enterImportDirective?: (ctx: ImportDirectiveContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.importDirective`.
	 * @param ctx the parse tree
	 */
	exitImportDirective?: (ctx: ImportDirectiveContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.importPath`.
	 * @param ctx the parse tree
	 */
	enterImportPath?: (ctx: ImportPathContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.importPath`.
	 * @param ctx the parse tree
	 */
	exitImportPath?: (ctx: ImportPathContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.contractDefinition`.
	 * @param ctx the parse tree
	 */
	enterContractDefinition?: (ctx: ContractDefinitionContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.contractDefinition`.
	 * @param ctx the parse tree
	 */
	exitContractDefinition?: (ctx: ContractDefinitionContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.inheritanceSpecifier`.
	 * @param ctx the parse tree
	 */
	enterInheritanceSpecifier?: (ctx: InheritanceSpecifierContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.inheritanceSpecifier`.
	 * @param ctx the parse tree
	 */
	exitInheritanceSpecifier?: (ctx: InheritanceSpecifierContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.customStorageLayout`.
	 * @param ctx the parse tree
	 */
	enterCustomStorageLayout?: (ctx: CustomStorageLayoutContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.customStorageLayout`.
	 * @param ctx the parse tree
	 */
	exitCustomStorageLayout?: (ctx: CustomStorageLayoutContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.contractPart`.
	 * @param ctx the parse tree
	 */
	enterContractPart?: (ctx: ContractPartContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.contractPart`.
	 * @param ctx the parse tree
	 */
	exitContractPart?: (ctx: ContractPartContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.stateVariableDeclaration`.
	 * @param ctx the parse tree
	 */
	enterStateVariableDeclaration?: (ctx: StateVariableDeclarationContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.stateVariableDeclaration`.
	 * @param ctx the parse tree
	 */
	exitStateVariableDeclaration?: (ctx: StateVariableDeclarationContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.fileLevelConstant`.
	 * @param ctx the parse tree
	 */
	enterFileLevelConstant?: (ctx: FileLevelConstantContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.fileLevelConstant`.
	 * @param ctx the parse tree
	 */
	exitFileLevelConstant?: (ctx: FileLevelConstantContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.customErrorDefinition`.
	 * @param ctx the parse tree
	 */
	enterCustomErrorDefinition?: (ctx: CustomErrorDefinitionContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.customErrorDefinition`.
	 * @param ctx the parse tree
	 */
	exitCustomErrorDefinition?: (ctx: CustomErrorDefinitionContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.typeDefinition`.
	 * @param ctx the parse tree
	 */
	enterTypeDefinition?: (ctx: TypeDefinitionContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.typeDefinition`.
	 * @param ctx the parse tree
	 */
	exitTypeDefinition?: (ctx: TypeDefinitionContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.usingForDeclaration`.
	 * @param ctx the parse tree
	 */
	enterUsingForDeclaration?: (ctx: UsingForDeclarationContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.usingForDeclaration`.
	 * @param ctx the parse tree
	 */
	exitUsingForDeclaration?: (ctx: UsingForDeclarationContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.usingForObject`.
	 * @param ctx the parse tree
	 */
	enterUsingForObject?: (ctx: UsingForObjectContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.usingForObject`.
	 * @param ctx the parse tree
	 */
	exitUsingForObject?: (ctx: UsingForObjectContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.usingForObjectDirective`.
	 * @param ctx the parse tree
	 */
	enterUsingForObjectDirective?: (ctx: UsingForObjectDirectiveContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.usingForObjectDirective`.
	 * @param ctx the parse tree
	 */
	exitUsingForObjectDirective?: (ctx: UsingForObjectDirectiveContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.userDefinableOperators`.
	 * @param ctx the parse tree
	 */
	enterUserDefinableOperators?: (ctx: UserDefinableOperatorsContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.userDefinableOperators`.
	 * @param ctx the parse tree
	 */
	exitUserDefinableOperators?: (ctx: UserDefinableOperatorsContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.structDefinition`.
	 * @param ctx the parse tree
	 */
	enterStructDefinition?: (ctx: StructDefinitionContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.structDefinition`.
	 * @param ctx the parse tree
	 */
	exitStructDefinition?: (ctx: StructDefinitionContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.modifierDefinition`.
	 * @param ctx the parse tree
	 */
	enterModifierDefinition?: (ctx: ModifierDefinitionContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.modifierDefinition`.
	 * @param ctx the parse tree
	 */
	exitModifierDefinition?: (ctx: ModifierDefinitionContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.modifierInvocation`.
	 * @param ctx the parse tree
	 */
	enterModifierInvocation?: (ctx: ModifierInvocationContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.modifierInvocation`.
	 * @param ctx the parse tree
	 */
	exitModifierInvocation?: (ctx: ModifierInvocationContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.functionDefinition`.
	 * @param ctx the parse tree
	 */
	enterFunctionDefinition?: (ctx: FunctionDefinitionContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.functionDefinition`.
	 * @param ctx the parse tree
	 */
	exitFunctionDefinition?: (ctx: FunctionDefinitionContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.functionDescriptor`.
	 * @param ctx the parse tree
	 */
	enterFunctionDescriptor?: (ctx: FunctionDescriptorContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.functionDescriptor`.
	 * @param ctx the parse tree
	 */
	exitFunctionDescriptor?: (ctx: FunctionDescriptorContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.returnParameters`.
	 * @param ctx the parse tree
	 */
	enterReturnParameters?: (ctx: ReturnParametersContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.returnParameters`.
	 * @param ctx the parse tree
	 */
	exitReturnParameters?: (ctx: ReturnParametersContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.modifierList`.
	 * @param ctx the parse tree
	 */
	enterModifierList?: (ctx: ModifierListContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.modifierList`.
	 * @param ctx the parse tree
	 */
	exitModifierList?: (ctx: ModifierListContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.eventDefinition`.
	 * @param ctx the parse tree
	 */
	enterEventDefinition?: (ctx: EventDefinitionContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.eventDefinition`.
	 * @param ctx the parse tree
	 */
	exitEventDefinition?: (ctx: EventDefinitionContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.enumValue`.
	 * @param ctx the parse tree
	 */
	enterEnumValue?: (ctx: EnumValueContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.enumValue`.
	 * @param ctx the parse tree
	 */
	exitEnumValue?: (ctx: EnumValueContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.enumDefinition`.
	 * @param ctx the parse tree
	 */
	enterEnumDefinition?: (ctx: EnumDefinitionContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.enumDefinition`.
	 * @param ctx the parse tree
	 */
	exitEnumDefinition?: (ctx: EnumDefinitionContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.parameterList`.
	 * @param ctx the parse tree
	 */
	enterParameterList?: (ctx: ParameterListContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.parameterList`.
	 * @param ctx the parse tree
	 */
	exitParameterList?: (ctx: ParameterListContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.parameter`.
	 * @param ctx the parse tree
	 */
	enterParameter?: (ctx: ParameterContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.parameter`.
	 * @param ctx the parse tree
	 */
	exitParameter?: (ctx: ParameterContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.eventParameterList`.
	 * @param ctx the parse tree
	 */
	enterEventParameterList?: (ctx: EventParameterListContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.eventParameterList`.
	 * @param ctx the parse tree
	 */
	exitEventParameterList?: (ctx: EventParameterListContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.eventParameter`.
	 * @param ctx the parse tree
	 */
	enterEventParameter?: (ctx: EventParameterContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.eventParameter`.
	 * @param ctx the parse tree
	 */
	exitEventParameter?: (ctx: EventParameterContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.functionTypeParameterList`.
	 * @param ctx the parse tree
	 */
	enterFunctionTypeParameterList?: (ctx: FunctionTypeParameterListContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.functionTypeParameterList`.
	 * @param ctx the parse tree
	 */
	exitFunctionTypeParameterList?: (ctx: FunctionTypeParameterListContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.functionTypeParameter`.
	 * @param ctx the parse tree
	 */
	enterFunctionTypeParameter?: (ctx: FunctionTypeParameterContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.functionTypeParameter`.
	 * @param ctx the parse tree
	 */
	exitFunctionTypeParameter?: (ctx: FunctionTypeParameterContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.variableDeclaration`.
	 * @param ctx the parse tree
	 */
	enterVariableDeclaration?: (ctx: VariableDeclarationContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.variableDeclaration`.
	 * @param ctx the parse tree
	 */
	exitVariableDeclaration?: (ctx: VariableDeclarationContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.typeName`.
	 * @param ctx the parse tree
	 */
	enterTypeName?: (ctx: TypeNameContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.typeName`.
	 * @param ctx the parse tree
	 */
	exitTypeName?: (ctx: TypeNameContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.userDefinedTypeName`.
	 * @param ctx the parse tree
	 */
	enterUserDefinedTypeName?: (ctx: UserDefinedTypeNameContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.userDefinedTypeName`.
	 * @param ctx the parse tree
	 */
	exitUserDefinedTypeName?: (ctx: UserDefinedTypeNameContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.mappingKey`.
	 * @param ctx the parse tree
	 */
	enterMappingKey?: (ctx: MappingKeyContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.mappingKey`.
	 * @param ctx the parse tree
	 */
	exitMappingKey?: (ctx: MappingKeyContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.mapping`.
	 * @param ctx the parse tree
	 */
	enterMapping?: (ctx: MappingContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.mapping`.
	 * @param ctx the parse tree
	 */
	exitMapping?: (ctx: MappingContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.mappingKeyName`.
	 * @param ctx the parse tree
	 */
	enterMappingKeyName?: (ctx: MappingKeyNameContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.mappingKeyName`.
	 * @param ctx the parse tree
	 */
	exitMappingKeyName?: (ctx: MappingKeyNameContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.mappingValueName`.
	 * @param ctx the parse tree
	 */
	enterMappingValueName?: (ctx: MappingValueNameContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.mappingValueName`.
	 * @param ctx the parse tree
	 */
	exitMappingValueName?: (ctx: MappingValueNameContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.functionTypeName`.
	 * @param ctx the parse tree
	 */
	enterFunctionTypeName?: (ctx: FunctionTypeNameContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.functionTypeName`.
	 * @param ctx the parse tree
	 */
	exitFunctionTypeName?: (ctx: FunctionTypeNameContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.storageLocation`.
	 * @param ctx the parse tree
	 */
	enterStorageLocation?: (ctx: StorageLocationContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.storageLocation`.
	 * @param ctx the parse tree
	 */
	exitStorageLocation?: (ctx: StorageLocationContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.stateMutability`.
	 * @param ctx the parse tree
	 */
	enterStateMutability?: (ctx: StateMutabilityContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.stateMutability`.
	 * @param ctx the parse tree
	 */
	exitStateMutability?: (ctx: StateMutabilityContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.block`.
	 * @param ctx the parse tree
	 */
	enterBlock?: (ctx: BlockContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.block`.
	 * @param ctx the parse tree
	 */
	exitBlock?: (ctx: BlockContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.statement`.
	 * @param ctx the parse tree
	 */
	enterStatement?: (ctx: StatementContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.statement`.
	 * @param ctx the parse tree
	 */
	exitStatement?: (ctx: StatementContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.expressionStatement`.
	 * @param ctx the parse tree
	 */
	enterExpressionStatement?: (ctx: ExpressionStatementContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.expressionStatement`.
	 * @param ctx the parse tree
	 */
	exitExpressionStatement?: (ctx: ExpressionStatementContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.ifStatement`.
	 * @param ctx the parse tree
	 */
	enterIfStatement?: (ctx: IfStatementContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.ifStatement`.
	 * @param ctx the parse tree
	 */
	exitIfStatement?: (ctx: IfStatementContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.tryStatement`.
	 * @param ctx the parse tree
	 */
	enterTryStatement?: (ctx: TryStatementContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.tryStatement`.
	 * @param ctx the parse tree
	 */
	exitTryStatement?: (ctx: TryStatementContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.catchClause`.
	 * @param ctx the parse tree
	 */
	enterCatchClause?: (ctx: CatchClauseContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.catchClause`.
	 * @param ctx the parse tree
	 */
	exitCatchClause?: (ctx: CatchClauseContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.whileStatement`.
	 * @param ctx the parse tree
	 */
	enterWhileStatement?: (ctx: WhileStatementContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.whileStatement`.
	 * @param ctx the parse tree
	 */
	exitWhileStatement?: (ctx: WhileStatementContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.simpleStatement`.
	 * @param ctx the parse tree
	 */
	enterSimpleStatement?: (ctx: SimpleStatementContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.simpleStatement`.
	 * @param ctx the parse tree
	 */
	exitSimpleStatement?: (ctx: SimpleStatementContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.uncheckedStatement`.
	 * @param ctx the parse tree
	 */
	enterUncheckedStatement?: (ctx: UncheckedStatementContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.uncheckedStatement`.
	 * @param ctx the parse tree
	 */
	exitUncheckedStatement?: (ctx: UncheckedStatementContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.forStatement`.
	 * @param ctx the parse tree
	 */
	enterForStatement?: (ctx: ForStatementContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.forStatement`.
	 * @param ctx the parse tree
	 */
	exitForStatement?: (ctx: ForStatementContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.inlineAssemblyStatement`.
	 * @param ctx the parse tree
	 */
	enterInlineAssemblyStatement?: (ctx: InlineAssemblyStatementContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.inlineAssemblyStatement`.
	 * @param ctx the parse tree
	 */
	exitInlineAssemblyStatement?: (ctx: InlineAssemblyStatementContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.inlineAssemblyStatementFlag`.
	 * @param ctx the parse tree
	 */
	enterInlineAssemblyStatementFlag?: (ctx: InlineAssemblyStatementFlagContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.inlineAssemblyStatementFlag`.
	 * @param ctx the parse tree
	 */
	exitInlineAssemblyStatementFlag?: (ctx: InlineAssemblyStatementFlagContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.doWhileStatement`.
	 * @param ctx the parse tree
	 */
	enterDoWhileStatement?: (ctx: DoWhileStatementContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.doWhileStatement`.
	 * @param ctx the parse tree
	 */
	exitDoWhileStatement?: (ctx: DoWhileStatementContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.continueStatement`.
	 * @param ctx the parse tree
	 */
	enterContinueStatement?: (ctx: ContinueStatementContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.continueStatement`.
	 * @param ctx the parse tree
	 */
	exitContinueStatement?: (ctx: ContinueStatementContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.breakStatement`.
	 * @param ctx the parse tree
	 */
	enterBreakStatement?: (ctx: BreakStatementContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.breakStatement`.
	 * @param ctx the parse tree
	 */
	exitBreakStatement?: (ctx: BreakStatementContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.returnStatement`.
	 * @param ctx the parse tree
	 */
	enterReturnStatement?: (ctx: ReturnStatementContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.returnStatement`.
	 * @param ctx the parse tree
	 */
	exitReturnStatement?: (ctx: ReturnStatementContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.throwStatement`.
	 * @param ctx the parse tree
	 */
	enterThrowStatement?: (ctx: ThrowStatementContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.throwStatement`.
	 * @param ctx the parse tree
	 */
	exitThrowStatement?: (ctx: ThrowStatementContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.emitStatement`.
	 * @param ctx the parse tree
	 */
	enterEmitStatement?: (ctx: EmitStatementContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.emitStatement`.
	 * @param ctx the parse tree
	 */
	exitEmitStatement?: (ctx: EmitStatementContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.revertStatement`.
	 * @param ctx the parse tree
	 */
	enterRevertStatement?: (ctx: RevertStatementContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.revertStatement`.
	 * @param ctx the parse tree
	 */
	exitRevertStatement?: (ctx: RevertStatementContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.variableDeclarationStatement`.
	 * @param ctx the parse tree
	 */
	enterVariableDeclarationStatement?: (ctx: VariableDeclarationStatementContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.variableDeclarationStatement`.
	 * @param ctx the parse tree
	 */
	exitVariableDeclarationStatement?: (ctx: VariableDeclarationStatementContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.variableDeclarationList`.
	 * @param ctx the parse tree
	 */
	enterVariableDeclarationList?: (ctx: VariableDeclarationListContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.variableDeclarationList`.
	 * @param ctx the parse tree
	 */
	exitVariableDeclarationList?: (ctx: VariableDeclarationListContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.identifierList`.
	 * @param ctx the parse tree
	 */
	enterIdentifierList?: (ctx: IdentifierListContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.identifierList`.
	 * @param ctx the parse tree
	 */
	exitIdentifierList?: (ctx: IdentifierListContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.elementaryTypeName`.
	 * @param ctx the parse tree
	 */
	enterElementaryTypeName?: (ctx: ElementaryTypeNameContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.elementaryTypeName`.
	 * @param ctx the parse tree
	 */
	exitElementaryTypeName?: (ctx: ElementaryTypeNameContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.expression`.
	 * @param ctx the parse tree
	 */
	enterExpression?: (ctx: ExpressionContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.expression`.
	 * @param ctx the parse tree
	 */
	exitExpression?: (ctx: ExpressionContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.primaryExpression`.
	 * @param ctx the parse tree
	 */
	enterPrimaryExpression?: (ctx: PrimaryExpressionContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.primaryExpression`.
	 * @param ctx the parse tree
	 */
	exitPrimaryExpression?: (ctx: PrimaryExpressionContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.expressionList`.
	 * @param ctx the parse tree
	 */
	enterExpressionList?: (ctx: ExpressionListContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.expressionList`.
	 * @param ctx the parse tree
	 */
	exitExpressionList?: (ctx: ExpressionListContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.nameValueList`.
	 * @param ctx the parse tree
	 */
	enterNameValueList?: (ctx: NameValueListContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.nameValueList`.
	 * @param ctx the parse tree
	 */
	exitNameValueList?: (ctx: NameValueListContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.nameValue`.
	 * @param ctx the parse tree
	 */
	enterNameValue?: (ctx: NameValueContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.nameValue`.
	 * @param ctx the parse tree
	 */
	exitNameValue?: (ctx: NameValueContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.functionCallArguments`.
	 * @param ctx the parse tree
	 */
	enterFunctionCallArguments?: (ctx: FunctionCallArgumentsContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.functionCallArguments`.
	 * @param ctx the parse tree
	 */
	exitFunctionCallArguments?: (ctx: FunctionCallArgumentsContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.functionCall`.
	 * @param ctx the parse tree
	 */
	enterFunctionCall?: (ctx: FunctionCallContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.functionCall`.
	 * @param ctx the parse tree
	 */
	exitFunctionCall?: (ctx: FunctionCallContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.assemblyBlock`.
	 * @param ctx the parse tree
	 */
	enterAssemblyBlock?: (ctx: AssemblyBlockContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.assemblyBlock`.
	 * @param ctx the parse tree
	 */
	exitAssemblyBlock?: (ctx: AssemblyBlockContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.assemblyItem`.
	 * @param ctx the parse tree
	 */
	enterAssemblyItem?: (ctx: AssemblyItemContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.assemblyItem`.
	 * @param ctx the parse tree
	 */
	exitAssemblyItem?: (ctx: AssemblyItemContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.assemblyExpression`.
	 * @param ctx the parse tree
	 */
	enterAssemblyExpression?: (ctx: AssemblyExpressionContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.assemblyExpression`.
	 * @param ctx the parse tree
	 */
	exitAssemblyExpression?: (ctx: AssemblyExpressionContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.assemblyMember`.
	 * @param ctx the parse tree
	 */
	enterAssemblyMember?: (ctx: AssemblyMemberContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.assemblyMember`.
	 * @param ctx the parse tree
	 */
	exitAssemblyMember?: (ctx: AssemblyMemberContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.assemblyCall`.
	 * @param ctx the parse tree
	 */
	enterAssemblyCall?: (ctx: AssemblyCallContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.assemblyCall`.
	 * @param ctx the parse tree
	 */
	exitAssemblyCall?: (ctx: AssemblyCallContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.assemblyLocalDefinition`.
	 * @param ctx the parse tree
	 */
	enterAssemblyLocalDefinition?: (ctx: AssemblyLocalDefinitionContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.assemblyLocalDefinition`.
	 * @param ctx the parse tree
	 */
	exitAssemblyLocalDefinition?: (ctx: AssemblyLocalDefinitionContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.assemblyAssignment`.
	 * @param ctx the parse tree
	 */
	enterAssemblyAssignment?: (ctx: AssemblyAssignmentContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.assemblyAssignment`.
	 * @param ctx the parse tree
	 */
	exitAssemblyAssignment?: (ctx: AssemblyAssignmentContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.assemblyIdentifierOrList`.
	 * @param ctx the parse tree
	 */
	enterAssemblyIdentifierOrList?: (ctx: AssemblyIdentifierOrListContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.assemblyIdentifierOrList`.
	 * @param ctx the parse tree
	 */
	exitAssemblyIdentifierOrList?: (ctx: AssemblyIdentifierOrListContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.assemblyIdentifierList`.
	 * @param ctx the parse tree
	 */
	enterAssemblyIdentifierList?: (ctx: AssemblyIdentifierListContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.assemblyIdentifierList`.
	 * @param ctx the parse tree
	 */
	exitAssemblyIdentifierList?: (ctx: AssemblyIdentifierListContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.assemblyStackAssignment`.
	 * @param ctx the parse tree
	 */
	enterAssemblyStackAssignment?: (ctx: AssemblyStackAssignmentContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.assemblyStackAssignment`.
	 * @param ctx the parse tree
	 */
	exitAssemblyStackAssignment?: (ctx: AssemblyStackAssignmentContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.labelDefinition`.
	 * @param ctx the parse tree
	 */
	enterLabelDefinition?: (ctx: LabelDefinitionContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.labelDefinition`.
	 * @param ctx the parse tree
	 */
	exitLabelDefinition?: (ctx: LabelDefinitionContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.assemblySwitch`.
	 * @param ctx the parse tree
	 */
	enterAssemblySwitch?: (ctx: AssemblySwitchContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.assemblySwitch`.
	 * @param ctx the parse tree
	 */
	exitAssemblySwitch?: (ctx: AssemblySwitchContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.assemblyCase`.
	 * @param ctx the parse tree
	 */
	enterAssemblyCase?: (ctx: AssemblyCaseContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.assemblyCase`.
	 * @param ctx the parse tree
	 */
	exitAssemblyCase?: (ctx: AssemblyCaseContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.assemblyFunctionDefinition`.
	 * @param ctx the parse tree
	 */
	enterAssemblyFunctionDefinition?: (ctx: AssemblyFunctionDefinitionContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.assemblyFunctionDefinition`.
	 * @param ctx the parse tree
	 */
	exitAssemblyFunctionDefinition?: (ctx: AssemblyFunctionDefinitionContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.assemblyFunctionReturns`.
	 * @param ctx the parse tree
	 */
	enterAssemblyFunctionReturns?: (ctx: AssemblyFunctionReturnsContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.assemblyFunctionReturns`.
	 * @param ctx the parse tree
	 */
	exitAssemblyFunctionReturns?: (ctx: AssemblyFunctionReturnsContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.assemblyFor`.
	 * @param ctx the parse tree
	 */
	enterAssemblyFor?: (ctx: AssemblyForContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.assemblyFor`.
	 * @param ctx the parse tree
	 */
	exitAssemblyFor?: (ctx: AssemblyForContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.assemblyIf`.
	 * @param ctx the parse tree
	 */
	enterAssemblyIf?: (ctx: AssemblyIfContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.assemblyIf`.
	 * @param ctx the parse tree
	 */
	exitAssemblyIf?: (ctx: AssemblyIfContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.assemblyLiteral`.
	 * @param ctx the parse tree
	 */
	enterAssemblyLiteral?: (ctx: AssemblyLiteralContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.assemblyLiteral`.
	 * @param ctx the parse tree
	 */
	exitAssemblyLiteral?: (ctx: AssemblyLiteralContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.tupleExpression`.
	 * @param ctx the parse tree
	 */
	enterTupleExpression?: (ctx: TupleExpressionContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.tupleExpression`.
	 * @param ctx the parse tree
	 */
	exitTupleExpression?: (ctx: TupleExpressionContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.numberLiteral`.
	 * @param ctx the parse tree
	 */
	enterNumberLiteral?: (ctx: NumberLiteralContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.numberLiteral`.
	 * @param ctx the parse tree
	 */
	exitNumberLiteral?: (ctx: NumberLiteralContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.identifier`.
	 * @param ctx the parse tree
	 */
	enterIdentifier?: (ctx: IdentifierContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.identifier`.
	 * @param ctx the parse tree
	 */
	exitIdentifier?: (ctx: IdentifierContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.hexLiteral`.
	 * @param ctx the parse tree
	 */
	enterHexLiteral?: (ctx: HexLiteralContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.hexLiteral`.
	 * @param ctx the parse tree
	 */
	exitHexLiteral?: (ctx: HexLiteralContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.overrideSpecifier`.
	 * @param ctx the parse tree
	 */
	enterOverrideSpecifier?: (ctx: OverrideSpecifierContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.overrideSpecifier`.
	 * @param ctx the parse tree
	 */
	exitOverrideSpecifier?: (ctx: OverrideSpecifierContext) => void;
	/**
	 * Enter a parse tree produced by `SolidityParser.stringLiteral`.
	 * @param ctx the parse tree
	 */
	enterStringLiteral?: (ctx: StringLiteralContext) => void;
	/**
	 * Exit a parse tree produced by `SolidityParser.stringLiteral`.
	 * @param ctx the parse tree
	 */
	exitStringLiteral?: (ctx: StringLiteralContext) => void;
}

