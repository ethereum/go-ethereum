import { ATN } from "antlr4ts/atn/ATN";
import { FailedPredicateException } from "antlr4ts/FailedPredicateException";
import { Parser } from "antlr4ts/Parser";
import { ParserRuleContext } from "antlr4ts/ParserRuleContext";
import { RuleContext } from "antlr4ts/RuleContext";
import { TerminalNode } from "antlr4ts/tree/TerminalNode";
import { TokenStream } from "antlr4ts/TokenStream";
import { Vocabulary } from "antlr4ts/Vocabulary";
import { SolidityListener } from "./SolidityListener";
import { SolidityVisitor } from "./SolidityVisitor";
export declare class SolidityParser extends Parser {
    static readonly T__0 = 1;
    static readonly T__1 = 2;
    static readonly T__2 = 3;
    static readonly T__3 = 4;
    static readonly T__4 = 5;
    static readonly T__5 = 6;
    static readonly T__6 = 7;
    static readonly T__7 = 8;
    static readonly T__8 = 9;
    static readonly T__9 = 10;
    static readonly T__10 = 11;
    static readonly T__11 = 12;
    static readonly T__12 = 13;
    static readonly T__13 = 14;
    static readonly T__14 = 15;
    static readonly T__15 = 16;
    static readonly T__16 = 17;
    static readonly T__17 = 18;
    static readonly T__18 = 19;
    static readonly T__19 = 20;
    static readonly T__20 = 21;
    static readonly T__21 = 22;
    static readonly T__22 = 23;
    static readonly T__23 = 24;
    static readonly T__24 = 25;
    static readonly T__25 = 26;
    static readonly T__26 = 27;
    static readonly T__27 = 28;
    static readonly T__28 = 29;
    static readonly T__29 = 30;
    static readonly T__30 = 31;
    static readonly T__31 = 32;
    static readonly T__32 = 33;
    static readonly T__33 = 34;
    static readonly T__34 = 35;
    static readonly T__35 = 36;
    static readonly T__36 = 37;
    static readonly T__37 = 38;
    static readonly T__38 = 39;
    static readonly T__39 = 40;
    static readonly T__40 = 41;
    static readonly T__41 = 42;
    static readonly T__42 = 43;
    static readonly T__43 = 44;
    static readonly T__44 = 45;
    static readonly T__45 = 46;
    static readonly T__46 = 47;
    static readonly T__47 = 48;
    static readonly T__48 = 49;
    static readonly T__49 = 50;
    static readonly T__50 = 51;
    static readonly T__51 = 52;
    static readonly T__52 = 53;
    static readonly T__53 = 54;
    static readonly T__54 = 55;
    static readonly T__55 = 56;
    static readonly T__56 = 57;
    static readonly T__57 = 58;
    static readonly T__58 = 59;
    static readonly T__59 = 60;
    static readonly T__60 = 61;
    static readonly T__61 = 62;
    static readonly T__62 = 63;
    static readonly T__63 = 64;
    static readonly T__64 = 65;
    static readonly T__65 = 66;
    static readonly T__66 = 67;
    static readonly T__67 = 68;
    static readonly T__68 = 69;
    static readonly T__69 = 70;
    static readonly T__70 = 71;
    static readonly T__71 = 72;
    static readonly T__72 = 73;
    static readonly T__73 = 74;
    static readonly T__74 = 75;
    static readonly T__75 = 76;
    static readonly T__76 = 77;
    static readonly T__77 = 78;
    static readonly T__78 = 79;
    static readonly T__79 = 80;
    static readonly T__80 = 81;
    static readonly T__81 = 82;
    static readonly T__82 = 83;
    static readonly T__83 = 84;
    static readonly T__84 = 85;
    static readonly T__85 = 86;
    static readonly T__86 = 87;
    static readonly T__87 = 88;
    static readonly T__88 = 89;
    static readonly T__89 = 90;
    static readonly T__90 = 91;
    static readonly T__91 = 92;
    static readonly T__92 = 93;
    static readonly T__93 = 94;
    static readonly T__94 = 95;
    static readonly T__95 = 96;
    static readonly T__96 = 97;
    static readonly Int = 98;
    static readonly Uint = 99;
    static readonly Byte = 100;
    static readonly Fixed = 101;
    static readonly Ufixed = 102;
    static readonly BooleanLiteral = 103;
    static readonly DecimalNumber = 104;
    static readonly HexNumber = 105;
    static readonly NumberUnit = 106;
    static readonly HexLiteralFragment = 107;
    static readonly ReservedKeyword = 108;
    static readonly AnonymousKeyword = 109;
    static readonly BreakKeyword = 110;
    static readonly ConstantKeyword = 111;
    static readonly ImmutableKeyword = 112;
    static readonly ContinueKeyword = 113;
    static readonly LeaveKeyword = 114;
    static readonly ExternalKeyword = 115;
    static readonly IndexedKeyword = 116;
    static readonly InternalKeyword = 117;
    static readonly PayableKeyword = 118;
    static readonly PrivateKeyword = 119;
    static readonly PublicKeyword = 120;
    static readonly VirtualKeyword = 121;
    static readonly PureKeyword = 122;
    static readonly TypeKeyword = 123;
    static readonly ViewKeyword = 124;
    static readonly GlobalKeyword = 125;
    static readonly ConstructorKeyword = 126;
    static readonly FallbackKeyword = 127;
    static readonly ReceiveKeyword = 128;
    static readonly Identifier = 129;
    static readonly StringLiteralFragment = 130;
    static readonly VersionLiteral = 131;
    static readonly WS = 132;
    static readonly COMMENT = 133;
    static readonly LINE_COMMENT = 134;
    static readonly RULE_sourceUnit = 0;
    static readonly RULE_pragmaDirective = 1;
    static readonly RULE_pragmaName = 2;
    static readonly RULE_pragmaValue = 3;
    static readonly RULE_version = 4;
    static readonly RULE_versionOperator = 5;
    static readonly RULE_versionConstraint = 6;
    static readonly RULE_importDeclaration = 7;
    static readonly RULE_importDirective = 8;
    static readonly RULE_importPath = 9;
    static readonly RULE_contractDefinition = 10;
    static readonly RULE_inheritanceSpecifier = 11;
    static readonly RULE_contractPart = 12;
    static readonly RULE_stateVariableDeclaration = 13;
    static readonly RULE_fileLevelConstant = 14;
    static readonly RULE_customErrorDefinition = 15;
    static readonly RULE_typeDefinition = 16;
    static readonly RULE_usingForDeclaration = 17;
    static readonly RULE_usingForObject = 18;
    static readonly RULE_structDefinition = 19;
    static readonly RULE_modifierDefinition = 20;
    static readonly RULE_modifierInvocation = 21;
    static readonly RULE_functionDefinition = 22;
    static readonly RULE_functionDescriptor = 23;
    static readonly RULE_returnParameters = 24;
    static readonly RULE_modifierList = 25;
    static readonly RULE_eventDefinition = 26;
    static readonly RULE_enumValue = 27;
    static readonly RULE_enumDefinition = 28;
    static readonly RULE_parameterList = 29;
    static readonly RULE_parameter = 30;
    static readonly RULE_eventParameterList = 31;
    static readonly RULE_eventParameter = 32;
    static readonly RULE_functionTypeParameterList = 33;
    static readonly RULE_functionTypeParameter = 34;
    static readonly RULE_variableDeclaration = 35;
    static readonly RULE_typeName = 36;
    static readonly RULE_userDefinedTypeName = 37;
    static readonly RULE_mappingKey = 38;
    static readonly RULE_mapping = 39;
    static readonly RULE_functionTypeName = 40;
    static readonly RULE_storageLocation = 41;
    static readonly RULE_stateMutability = 42;
    static readonly RULE_block = 43;
    static readonly RULE_statement = 44;
    static readonly RULE_expressionStatement = 45;
    static readonly RULE_ifStatement = 46;
    static readonly RULE_tryStatement = 47;
    static readonly RULE_catchClause = 48;
    static readonly RULE_whileStatement = 49;
    static readonly RULE_simpleStatement = 50;
    static readonly RULE_uncheckedStatement = 51;
    static readonly RULE_forStatement = 52;
    static readonly RULE_inlineAssemblyStatement = 53;
    static readonly RULE_inlineAssemblyStatementFlag = 54;
    static readonly RULE_doWhileStatement = 55;
    static readonly RULE_continueStatement = 56;
    static readonly RULE_breakStatement = 57;
    static readonly RULE_returnStatement = 58;
    static readonly RULE_throwStatement = 59;
    static readonly RULE_emitStatement = 60;
    static readonly RULE_revertStatement = 61;
    static readonly RULE_variableDeclarationStatement = 62;
    static readonly RULE_variableDeclarationList = 63;
    static readonly RULE_identifierList = 64;
    static readonly RULE_elementaryTypeName = 65;
    static readonly RULE_expression = 66;
    static readonly RULE_primaryExpression = 67;
    static readonly RULE_expressionList = 68;
    static readonly RULE_nameValueList = 69;
    static readonly RULE_nameValue = 70;
    static readonly RULE_functionCallArguments = 71;
    static readonly RULE_functionCall = 72;
    static readonly RULE_assemblyBlock = 73;
    static readonly RULE_assemblyItem = 74;
    static readonly RULE_assemblyExpression = 75;
    static readonly RULE_assemblyMember = 76;
    static readonly RULE_assemblyCall = 77;
    static readonly RULE_assemblyLocalDefinition = 78;
    static readonly RULE_assemblyAssignment = 79;
    static readonly RULE_assemblyIdentifierOrList = 80;
    static readonly RULE_assemblyIdentifierList = 81;
    static readonly RULE_assemblyStackAssignment = 82;
    static readonly RULE_labelDefinition = 83;
    static readonly RULE_assemblySwitch = 84;
    static readonly RULE_assemblyCase = 85;
    static readonly RULE_assemblyFunctionDefinition = 86;
    static readonly RULE_assemblyFunctionReturns = 87;
    static readonly RULE_assemblyFor = 88;
    static readonly RULE_assemblyIf = 89;
    static readonly RULE_assemblyLiteral = 90;
    static readonly RULE_subAssembly = 91;
    static readonly RULE_tupleExpression = 92;
    static readonly RULE_numberLiteral = 93;
    static readonly RULE_identifier = 94;
    static readonly RULE_hexLiteral = 95;
    static readonly RULE_overrideSpecifier = 96;
    static readonly RULE_stringLiteral = 97;
    static readonly ruleNames: string[];
    private static readonly _LITERAL_NAMES;
    private static readonly _SYMBOLIC_NAMES;
    static readonly VOCABULARY: Vocabulary;
    get vocabulary(): Vocabulary;
    get grammarFileName(): string;
    get ruleNames(): string[];
    get serializedATN(): string;
    protected createFailedPredicateException(predicate?: string, message?: string): FailedPredicateException;
    constructor(input: TokenStream);
    sourceUnit(): SourceUnitContext;
    pragmaDirective(): PragmaDirectiveContext;
    pragmaName(): PragmaNameContext;
    pragmaValue(): PragmaValueContext;
    version(): VersionContext;
    versionOperator(): VersionOperatorContext;
    versionConstraint(): VersionConstraintContext;
    importDeclaration(): ImportDeclarationContext;
    importDirective(): ImportDirectiveContext;
    importPath(): ImportPathContext;
    contractDefinition(): ContractDefinitionContext;
    inheritanceSpecifier(): InheritanceSpecifierContext;
    contractPart(): ContractPartContext;
    stateVariableDeclaration(): StateVariableDeclarationContext;
    fileLevelConstant(): FileLevelConstantContext;
    customErrorDefinition(): CustomErrorDefinitionContext;
    typeDefinition(): TypeDefinitionContext;
    usingForDeclaration(): UsingForDeclarationContext;
    usingForObject(): UsingForObjectContext;
    structDefinition(): StructDefinitionContext;
    modifierDefinition(): ModifierDefinitionContext;
    modifierInvocation(): ModifierInvocationContext;
    functionDefinition(): FunctionDefinitionContext;
    functionDescriptor(): FunctionDescriptorContext;
    returnParameters(): ReturnParametersContext;
    modifierList(): ModifierListContext;
    eventDefinition(): EventDefinitionContext;
    enumValue(): EnumValueContext;
    enumDefinition(): EnumDefinitionContext;
    parameterList(): ParameterListContext;
    parameter(): ParameterContext;
    eventParameterList(): EventParameterListContext;
    eventParameter(): EventParameterContext;
    functionTypeParameterList(): FunctionTypeParameterListContext;
    functionTypeParameter(): FunctionTypeParameterContext;
    variableDeclaration(): VariableDeclarationContext;
    typeName(): TypeNameContext;
    typeName(_p: number): TypeNameContext;
    userDefinedTypeName(): UserDefinedTypeNameContext;
    mappingKey(): MappingKeyContext;
    mapping(): MappingContext;
    functionTypeName(): FunctionTypeNameContext;
    storageLocation(): StorageLocationContext;
    stateMutability(): StateMutabilityContext;
    block(): BlockContext;
    statement(): StatementContext;
    expressionStatement(): ExpressionStatementContext;
    ifStatement(): IfStatementContext;
    tryStatement(): TryStatementContext;
    catchClause(): CatchClauseContext;
    whileStatement(): WhileStatementContext;
    simpleStatement(): SimpleStatementContext;
    uncheckedStatement(): UncheckedStatementContext;
    forStatement(): ForStatementContext;
    inlineAssemblyStatement(): InlineAssemblyStatementContext;
    inlineAssemblyStatementFlag(): InlineAssemblyStatementFlagContext;
    doWhileStatement(): DoWhileStatementContext;
    continueStatement(): ContinueStatementContext;
    breakStatement(): BreakStatementContext;
    returnStatement(): ReturnStatementContext;
    throwStatement(): ThrowStatementContext;
    emitStatement(): EmitStatementContext;
    revertStatement(): RevertStatementContext;
    variableDeclarationStatement(): VariableDeclarationStatementContext;
    variableDeclarationList(): VariableDeclarationListContext;
    identifierList(): IdentifierListContext;
    elementaryTypeName(): ElementaryTypeNameContext;
    expression(): ExpressionContext;
    expression(_p: number): ExpressionContext;
    primaryExpression(): PrimaryExpressionContext;
    expressionList(): ExpressionListContext;
    nameValueList(): NameValueListContext;
    nameValue(): NameValueContext;
    functionCallArguments(): FunctionCallArgumentsContext;
    functionCall(): FunctionCallContext;
    assemblyBlock(): AssemblyBlockContext;
    assemblyItem(): AssemblyItemContext;
    assemblyExpression(): AssemblyExpressionContext;
    assemblyMember(): AssemblyMemberContext;
    assemblyCall(): AssemblyCallContext;
    assemblyLocalDefinition(): AssemblyLocalDefinitionContext;
    assemblyAssignment(): AssemblyAssignmentContext;
    assemblyIdentifierOrList(): AssemblyIdentifierOrListContext;
    assemblyIdentifierList(): AssemblyIdentifierListContext;
    assemblyStackAssignment(): AssemblyStackAssignmentContext;
    labelDefinition(): LabelDefinitionContext;
    assemblySwitch(): AssemblySwitchContext;
    assemblyCase(): AssemblyCaseContext;
    assemblyFunctionDefinition(): AssemblyFunctionDefinitionContext;
    assemblyFunctionReturns(): AssemblyFunctionReturnsContext;
    assemblyFor(): AssemblyForContext;
    assemblyIf(): AssemblyIfContext;
    assemblyLiteral(): AssemblyLiteralContext;
    subAssembly(): SubAssemblyContext;
    tupleExpression(): TupleExpressionContext;
    numberLiteral(): NumberLiteralContext;
    identifier(): IdentifierContext;
    hexLiteral(): HexLiteralContext;
    overrideSpecifier(): OverrideSpecifierContext;
    stringLiteral(): StringLiteralContext;
    sempred(_localctx: RuleContext, ruleIndex: number, predIndex: number): boolean;
    private typeName_sempred;
    private expression_sempred;
    private static readonly _serializedATNSegments;
    private static readonly _serializedATNSegment0;
    private static readonly _serializedATNSegment1;
    private static readonly _serializedATNSegment2;
    static readonly _serializedATN: string;
    static __ATN: ATN;
    static get _ATN(): ATN;
}
export declare class SourceUnitContext extends ParserRuleContext {
    EOF(): TerminalNode;
    pragmaDirective(): PragmaDirectiveContext[];
    pragmaDirective(i: number): PragmaDirectiveContext;
    importDirective(): ImportDirectiveContext[];
    importDirective(i: number): ImportDirectiveContext;
    contractDefinition(): ContractDefinitionContext[];
    contractDefinition(i: number): ContractDefinitionContext;
    enumDefinition(): EnumDefinitionContext[];
    enumDefinition(i: number): EnumDefinitionContext;
    structDefinition(): StructDefinitionContext[];
    structDefinition(i: number): StructDefinitionContext;
    functionDefinition(): FunctionDefinitionContext[];
    functionDefinition(i: number): FunctionDefinitionContext;
    fileLevelConstant(): FileLevelConstantContext[];
    fileLevelConstant(i: number): FileLevelConstantContext;
    customErrorDefinition(): CustomErrorDefinitionContext[];
    customErrorDefinition(i: number): CustomErrorDefinitionContext;
    typeDefinition(): TypeDefinitionContext[];
    typeDefinition(i: number): TypeDefinitionContext;
    usingForDeclaration(): UsingForDeclarationContext[];
    usingForDeclaration(i: number): UsingForDeclarationContext;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class PragmaDirectiveContext extends ParserRuleContext {
    pragmaName(): PragmaNameContext;
    pragmaValue(): PragmaValueContext;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class PragmaNameContext extends ParserRuleContext {
    identifier(): IdentifierContext;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class PragmaValueContext extends ParserRuleContext {
    version(): VersionContext | undefined;
    expression(): ExpressionContext | undefined;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class VersionContext extends ParserRuleContext {
    versionConstraint(): VersionConstraintContext[];
    versionConstraint(i: number): VersionConstraintContext;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class VersionOperatorContext extends ParserRuleContext {
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class VersionConstraintContext extends ParserRuleContext {
    VersionLiteral(): TerminalNode | undefined;
    versionOperator(): VersionOperatorContext | undefined;
    DecimalNumber(): TerminalNode | undefined;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class ImportDeclarationContext extends ParserRuleContext {
    identifier(): IdentifierContext[];
    identifier(i: number): IdentifierContext;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class ImportDirectiveContext extends ParserRuleContext {
    importPath(): ImportPathContext;
    identifier(): IdentifierContext[];
    identifier(i: number): IdentifierContext;
    importDeclaration(): ImportDeclarationContext[];
    importDeclaration(i: number): ImportDeclarationContext;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class ImportPathContext extends ParserRuleContext {
    StringLiteralFragment(): TerminalNode;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class ContractDefinitionContext extends ParserRuleContext {
    identifier(): IdentifierContext;
    inheritanceSpecifier(): InheritanceSpecifierContext[];
    inheritanceSpecifier(i: number): InheritanceSpecifierContext;
    contractPart(): ContractPartContext[];
    contractPart(i: number): ContractPartContext;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class InheritanceSpecifierContext extends ParserRuleContext {
    userDefinedTypeName(): UserDefinedTypeNameContext;
    expressionList(): ExpressionListContext | undefined;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class ContractPartContext extends ParserRuleContext {
    stateVariableDeclaration(): StateVariableDeclarationContext | undefined;
    usingForDeclaration(): UsingForDeclarationContext | undefined;
    structDefinition(): StructDefinitionContext | undefined;
    modifierDefinition(): ModifierDefinitionContext | undefined;
    functionDefinition(): FunctionDefinitionContext | undefined;
    eventDefinition(): EventDefinitionContext | undefined;
    enumDefinition(): EnumDefinitionContext | undefined;
    customErrorDefinition(): CustomErrorDefinitionContext | undefined;
    typeDefinition(): TypeDefinitionContext | undefined;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class StateVariableDeclarationContext extends ParserRuleContext {
    typeName(): TypeNameContext;
    identifier(): IdentifierContext;
    PublicKeyword(): TerminalNode[];
    PublicKeyword(i: number): TerminalNode;
    InternalKeyword(): TerminalNode[];
    InternalKeyword(i: number): TerminalNode;
    PrivateKeyword(): TerminalNode[];
    PrivateKeyword(i: number): TerminalNode;
    ConstantKeyword(): TerminalNode[];
    ConstantKeyword(i: number): TerminalNode;
    ImmutableKeyword(): TerminalNode[];
    ImmutableKeyword(i: number): TerminalNode;
    overrideSpecifier(): OverrideSpecifierContext[];
    overrideSpecifier(i: number): OverrideSpecifierContext;
    expression(): ExpressionContext | undefined;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class FileLevelConstantContext extends ParserRuleContext {
    typeName(): TypeNameContext;
    ConstantKeyword(): TerminalNode;
    identifier(): IdentifierContext;
    expression(): ExpressionContext;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class CustomErrorDefinitionContext extends ParserRuleContext {
    identifier(): IdentifierContext;
    parameterList(): ParameterListContext;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class TypeDefinitionContext extends ParserRuleContext {
    TypeKeyword(): TerminalNode;
    identifier(): IdentifierContext;
    elementaryTypeName(): ElementaryTypeNameContext;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class UsingForDeclarationContext extends ParserRuleContext {
    usingForObject(): UsingForObjectContext;
    typeName(): TypeNameContext | undefined;
    GlobalKeyword(): TerminalNode | undefined;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class UsingForObjectContext extends ParserRuleContext {
    userDefinedTypeName(): UserDefinedTypeNameContext[];
    userDefinedTypeName(i: number): UserDefinedTypeNameContext;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class StructDefinitionContext extends ParserRuleContext {
    identifier(): IdentifierContext;
    variableDeclaration(): VariableDeclarationContext[];
    variableDeclaration(i: number): VariableDeclarationContext;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class ModifierDefinitionContext extends ParserRuleContext {
    identifier(): IdentifierContext;
    block(): BlockContext | undefined;
    parameterList(): ParameterListContext | undefined;
    VirtualKeyword(): TerminalNode[];
    VirtualKeyword(i: number): TerminalNode;
    overrideSpecifier(): OverrideSpecifierContext[];
    overrideSpecifier(i: number): OverrideSpecifierContext;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class ModifierInvocationContext extends ParserRuleContext {
    identifier(): IdentifierContext;
    expressionList(): ExpressionListContext | undefined;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class FunctionDefinitionContext extends ParserRuleContext {
    functionDescriptor(): FunctionDescriptorContext;
    parameterList(): ParameterListContext;
    modifierList(): ModifierListContext;
    block(): BlockContext | undefined;
    returnParameters(): ReturnParametersContext | undefined;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class FunctionDescriptorContext extends ParserRuleContext {
    identifier(): IdentifierContext | undefined;
    ConstructorKeyword(): TerminalNode | undefined;
    FallbackKeyword(): TerminalNode | undefined;
    ReceiveKeyword(): TerminalNode | undefined;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class ReturnParametersContext extends ParserRuleContext {
    parameterList(): ParameterListContext;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class ModifierListContext extends ParserRuleContext {
    ExternalKeyword(): TerminalNode[];
    ExternalKeyword(i: number): TerminalNode;
    PublicKeyword(): TerminalNode[];
    PublicKeyword(i: number): TerminalNode;
    InternalKeyword(): TerminalNode[];
    InternalKeyword(i: number): TerminalNode;
    PrivateKeyword(): TerminalNode[];
    PrivateKeyword(i: number): TerminalNode;
    VirtualKeyword(): TerminalNode[];
    VirtualKeyword(i: number): TerminalNode;
    stateMutability(): StateMutabilityContext[];
    stateMutability(i: number): StateMutabilityContext;
    modifierInvocation(): ModifierInvocationContext[];
    modifierInvocation(i: number): ModifierInvocationContext;
    overrideSpecifier(): OverrideSpecifierContext[];
    overrideSpecifier(i: number): OverrideSpecifierContext;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class EventDefinitionContext extends ParserRuleContext {
    identifier(): IdentifierContext;
    eventParameterList(): EventParameterListContext;
    AnonymousKeyword(): TerminalNode | undefined;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class EnumValueContext extends ParserRuleContext {
    identifier(): IdentifierContext;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class EnumDefinitionContext extends ParserRuleContext {
    identifier(): IdentifierContext;
    enumValue(): EnumValueContext[];
    enumValue(i: number): EnumValueContext;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class ParameterListContext extends ParserRuleContext {
    parameter(): ParameterContext[];
    parameter(i: number): ParameterContext;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class ParameterContext extends ParserRuleContext {
    typeName(): TypeNameContext;
    storageLocation(): StorageLocationContext | undefined;
    identifier(): IdentifierContext | undefined;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class EventParameterListContext extends ParserRuleContext {
    eventParameter(): EventParameterContext[];
    eventParameter(i: number): EventParameterContext;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class EventParameterContext extends ParserRuleContext {
    typeName(): TypeNameContext;
    IndexedKeyword(): TerminalNode | undefined;
    identifier(): IdentifierContext | undefined;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class FunctionTypeParameterListContext extends ParserRuleContext {
    functionTypeParameter(): FunctionTypeParameterContext[];
    functionTypeParameter(i: number): FunctionTypeParameterContext;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class FunctionTypeParameterContext extends ParserRuleContext {
    typeName(): TypeNameContext;
    storageLocation(): StorageLocationContext | undefined;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class VariableDeclarationContext extends ParserRuleContext {
    typeName(): TypeNameContext;
    identifier(): IdentifierContext;
    storageLocation(): StorageLocationContext | undefined;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class TypeNameContext extends ParserRuleContext {
    elementaryTypeName(): ElementaryTypeNameContext | undefined;
    userDefinedTypeName(): UserDefinedTypeNameContext | undefined;
    mapping(): MappingContext | undefined;
    typeName(): TypeNameContext | undefined;
    expression(): ExpressionContext | undefined;
    functionTypeName(): FunctionTypeNameContext | undefined;
    PayableKeyword(): TerminalNode | undefined;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class UserDefinedTypeNameContext extends ParserRuleContext {
    identifier(): IdentifierContext[];
    identifier(i: number): IdentifierContext;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class MappingKeyContext extends ParserRuleContext {
    elementaryTypeName(): ElementaryTypeNameContext | undefined;
    userDefinedTypeName(): UserDefinedTypeNameContext | undefined;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class MappingContext extends ParserRuleContext {
    mappingKey(): MappingKeyContext;
    typeName(): TypeNameContext;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class FunctionTypeNameContext extends ParserRuleContext {
    functionTypeParameterList(): FunctionTypeParameterListContext[];
    functionTypeParameterList(i: number): FunctionTypeParameterListContext;
    InternalKeyword(): TerminalNode[];
    InternalKeyword(i: number): TerminalNode;
    ExternalKeyword(): TerminalNode[];
    ExternalKeyword(i: number): TerminalNode;
    stateMutability(): StateMutabilityContext[];
    stateMutability(i: number): StateMutabilityContext;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class StorageLocationContext extends ParserRuleContext {
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class StateMutabilityContext extends ParserRuleContext {
    PureKeyword(): TerminalNode | undefined;
    ConstantKeyword(): TerminalNode | undefined;
    ViewKeyword(): TerminalNode | undefined;
    PayableKeyword(): TerminalNode | undefined;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class BlockContext extends ParserRuleContext {
    statement(): StatementContext[];
    statement(i: number): StatementContext;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class StatementContext extends ParserRuleContext {
    ifStatement(): IfStatementContext | undefined;
    tryStatement(): TryStatementContext | undefined;
    whileStatement(): WhileStatementContext | undefined;
    forStatement(): ForStatementContext | undefined;
    block(): BlockContext | undefined;
    inlineAssemblyStatement(): InlineAssemblyStatementContext | undefined;
    doWhileStatement(): DoWhileStatementContext | undefined;
    continueStatement(): ContinueStatementContext | undefined;
    breakStatement(): BreakStatementContext | undefined;
    returnStatement(): ReturnStatementContext | undefined;
    throwStatement(): ThrowStatementContext | undefined;
    emitStatement(): EmitStatementContext | undefined;
    simpleStatement(): SimpleStatementContext | undefined;
    uncheckedStatement(): UncheckedStatementContext | undefined;
    revertStatement(): RevertStatementContext | undefined;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class ExpressionStatementContext extends ParserRuleContext {
    expression(): ExpressionContext;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class IfStatementContext extends ParserRuleContext {
    expression(): ExpressionContext;
    statement(): StatementContext[];
    statement(i: number): StatementContext;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class TryStatementContext extends ParserRuleContext {
    expression(): ExpressionContext;
    block(): BlockContext;
    returnParameters(): ReturnParametersContext | undefined;
    catchClause(): CatchClauseContext[];
    catchClause(i: number): CatchClauseContext;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class CatchClauseContext extends ParserRuleContext {
    block(): BlockContext;
    parameterList(): ParameterListContext | undefined;
    identifier(): IdentifierContext | undefined;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class WhileStatementContext extends ParserRuleContext {
    expression(): ExpressionContext;
    statement(): StatementContext;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class SimpleStatementContext extends ParserRuleContext {
    variableDeclarationStatement(): VariableDeclarationStatementContext | undefined;
    expressionStatement(): ExpressionStatementContext | undefined;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class UncheckedStatementContext extends ParserRuleContext {
    block(): BlockContext;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class ForStatementContext extends ParserRuleContext {
    statement(): StatementContext;
    simpleStatement(): SimpleStatementContext | undefined;
    expressionStatement(): ExpressionStatementContext | undefined;
    expression(): ExpressionContext | undefined;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class InlineAssemblyStatementContext extends ParserRuleContext {
    assemblyBlock(): AssemblyBlockContext;
    StringLiteralFragment(): TerminalNode | undefined;
    inlineAssemblyStatementFlag(): InlineAssemblyStatementFlagContext | undefined;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class InlineAssemblyStatementFlagContext extends ParserRuleContext {
    stringLiteral(): StringLiteralContext;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class DoWhileStatementContext extends ParserRuleContext {
    statement(): StatementContext;
    expression(): ExpressionContext;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class ContinueStatementContext extends ParserRuleContext {
    ContinueKeyword(): TerminalNode;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class BreakStatementContext extends ParserRuleContext {
    BreakKeyword(): TerminalNode;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class ReturnStatementContext extends ParserRuleContext {
    expression(): ExpressionContext | undefined;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class ThrowStatementContext extends ParserRuleContext {
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class EmitStatementContext extends ParserRuleContext {
    functionCall(): FunctionCallContext;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class RevertStatementContext extends ParserRuleContext {
    functionCall(): FunctionCallContext;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class VariableDeclarationStatementContext extends ParserRuleContext {
    identifierList(): IdentifierListContext | undefined;
    variableDeclaration(): VariableDeclarationContext | undefined;
    variableDeclarationList(): VariableDeclarationListContext | undefined;
    expression(): ExpressionContext | undefined;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class VariableDeclarationListContext extends ParserRuleContext {
    variableDeclaration(): VariableDeclarationContext[];
    variableDeclaration(i: number): VariableDeclarationContext;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class IdentifierListContext extends ParserRuleContext {
    identifier(): IdentifierContext[];
    identifier(i: number): IdentifierContext;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class ElementaryTypeNameContext extends ParserRuleContext {
    Int(): TerminalNode | undefined;
    Uint(): TerminalNode | undefined;
    Byte(): TerminalNode | undefined;
    Fixed(): TerminalNode | undefined;
    Ufixed(): TerminalNode | undefined;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class ExpressionContext extends ParserRuleContext {
    expression(): ExpressionContext[];
    expression(i: number): ExpressionContext;
    typeName(): TypeNameContext | undefined;
    identifier(): IdentifierContext | undefined;
    nameValueList(): NameValueListContext | undefined;
    functionCallArguments(): FunctionCallArgumentsContext | undefined;
    primaryExpression(): PrimaryExpressionContext | undefined;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class PrimaryExpressionContext extends ParserRuleContext {
    BooleanLiteral(): TerminalNode | undefined;
    numberLiteral(): NumberLiteralContext | undefined;
    hexLiteral(): HexLiteralContext | undefined;
    stringLiteral(): StringLiteralContext | undefined;
    identifier(): IdentifierContext | undefined;
    TypeKeyword(): TerminalNode | undefined;
    PayableKeyword(): TerminalNode | undefined;
    tupleExpression(): TupleExpressionContext | undefined;
    typeName(): TypeNameContext | undefined;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class ExpressionListContext extends ParserRuleContext {
    expression(): ExpressionContext[];
    expression(i: number): ExpressionContext;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class NameValueListContext extends ParserRuleContext {
    nameValue(): NameValueContext[];
    nameValue(i: number): NameValueContext;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class NameValueContext extends ParserRuleContext {
    identifier(): IdentifierContext;
    expression(): ExpressionContext;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class FunctionCallArgumentsContext extends ParserRuleContext {
    nameValueList(): NameValueListContext | undefined;
    expressionList(): ExpressionListContext | undefined;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class FunctionCallContext extends ParserRuleContext {
    expression(): ExpressionContext;
    functionCallArguments(): FunctionCallArgumentsContext;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class AssemblyBlockContext extends ParserRuleContext {
    assemblyItem(): AssemblyItemContext[];
    assemblyItem(i: number): AssemblyItemContext;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class AssemblyItemContext extends ParserRuleContext {
    identifier(): IdentifierContext | undefined;
    assemblyBlock(): AssemblyBlockContext | undefined;
    assemblyExpression(): AssemblyExpressionContext | undefined;
    assemblyLocalDefinition(): AssemblyLocalDefinitionContext | undefined;
    assemblyAssignment(): AssemblyAssignmentContext | undefined;
    assemblyStackAssignment(): AssemblyStackAssignmentContext | undefined;
    labelDefinition(): LabelDefinitionContext | undefined;
    assemblySwitch(): AssemblySwitchContext | undefined;
    assemblyFunctionDefinition(): AssemblyFunctionDefinitionContext | undefined;
    assemblyFor(): AssemblyForContext | undefined;
    assemblyIf(): AssemblyIfContext | undefined;
    BreakKeyword(): TerminalNode | undefined;
    ContinueKeyword(): TerminalNode | undefined;
    LeaveKeyword(): TerminalNode | undefined;
    subAssembly(): SubAssemblyContext | undefined;
    numberLiteral(): NumberLiteralContext | undefined;
    stringLiteral(): StringLiteralContext | undefined;
    hexLiteral(): HexLiteralContext | undefined;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class AssemblyExpressionContext extends ParserRuleContext {
    assemblyCall(): AssemblyCallContext | undefined;
    assemblyLiteral(): AssemblyLiteralContext | undefined;
    assemblyMember(): AssemblyMemberContext | undefined;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class AssemblyMemberContext extends ParserRuleContext {
    identifier(): IdentifierContext[];
    identifier(i: number): IdentifierContext;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class AssemblyCallContext extends ParserRuleContext {
    identifier(): IdentifierContext | undefined;
    assemblyExpression(): AssemblyExpressionContext[];
    assemblyExpression(i: number): AssemblyExpressionContext;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class AssemblyLocalDefinitionContext extends ParserRuleContext {
    assemblyIdentifierOrList(): AssemblyIdentifierOrListContext;
    assemblyExpression(): AssemblyExpressionContext | undefined;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class AssemblyAssignmentContext extends ParserRuleContext {
    assemblyIdentifierOrList(): AssemblyIdentifierOrListContext;
    assemblyExpression(): AssemblyExpressionContext;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class AssemblyIdentifierOrListContext extends ParserRuleContext {
    identifier(): IdentifierContext | undefined;
    assemblyMember(): AssemblyMemberContext | undefined;
    assemblyIdentifierList(): AssemblyIdentifierListContext | undefined;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class AssemblyIdentifierListContext extends ParserRuleContext {
    identifier(): IdentifierContext[];
    identifier(i: number): IdentifierContext;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class AssemblyStackAssignmentContext extends ParserRuleContext {
    assemblyExpression(): AssemblyExpressionContext;
    identifier(): IdentifierContext;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class LabelDefinitionContext extends ParserRuleContext {
    identifier(): IdentifierContext;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class AssemblySwitchContext extends ParserRuleContext {
    assemblyExpression(): AssemblyExpressionContext;
    assemblyCase(): AssemblyCaseContext[];
    assemblyCase(i: number): AssemblyCaseContext;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class AssemblyCaseContext extends ParserRuleContext {
    assemblyLiteral(): AssemblyLiteralContext | undefined;
    assemblyBlock(): AssemblyBlockContext;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class AssemblyFunctionDefinitionContext extends ParserRuleContext {
    identifier(): IdentifierContext;
    assemblyBlock(): AssemblyBlockContext;
    assemblyIdentifierList(): AssemblyIdentifierListContext | undefined;
    assemblyFunctionReturns(): AssemblyFunctionReturnsContext | undefined;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class AssemblyFunctionReturnsContext extends ParserRuleContext {
    assemblyIdentifierList(): AssemblyIdentifierListContext | undefined;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class AssemblyForContext extends ParserRuleContext {
    assemblyExpression(): AssemblyExpressionContext[];
    assemblyExpression(i: number): AssemblyExpressionContext;
    assemblyBlock(): AssemblyBlockContext[];
    assemblyBlock(i: number): AssemblyBlockContext;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class AssemblyIfContext extends ParserRuleContext {
    assemblyExpression(): AssemblyExpressionContext;
    assemblyBlock(): AssemblyBlockContext;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class AssemblyLiteralContext extends ParserRuleContext {
    stringLiteral(): StringLiteralContext | undefined;
    DecimalNumber(): TerminalNode | undefined;
    HexNumber(): TerminalNode | undefined;
    hexLiteral(): HexLiteralContext | undefined;
    BooleanLiteral(): TerminalNode | undefined;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class SubAssemblyContext extends ParserRuleContext {
    identifier(): IdentifierContext;
    assemblyBlock(): AssemblyBlockContext;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class TupleExpressionContext extends ParserRuleContext {
    expression(): ExpressionContext[];
    expression(i: number): ExpressionContext;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class NumberLiteralContext extends ParserRuleContext {
    DecimalNumber(): TerminalNode | undefined;
    HexNumber(): TerminalNode | undefined;
    NumberUnit(): TerminalNode | undefined;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class IdentifierContext extends ParserRuleContext {
    ReceiveKeyword(): TerminalNode | undefined;
    ConstructorKeyword(): TerminalNode | undefined;
    PayableKeyword(): TerminalNode | undefined;
    LeaveKeyword(): TerminalNode | undefined;
    Identifier(): TerminalNode | undefined;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class HexLiteralContext extends ParserRuleContext {
    HexLiteralFragment(): TerminalNode[];
    HexLiteralFragment(i: number): TerminalNode;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class OverrideSpecifierContext extends ParserRuleContext {
    userDefinedTypeName(): UserDefinedTypeNameContext[];
    userDefinedTypeName(i: number): UserDefinedTypeNameContext;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class StringLiteralContext extends ParserRuleContext {
    StringLiteralFragment(): TerminalNode[];
    StringLiteralFragment(i: number): TerminalNode;
    constructor(parent: ParserRuleContext | undefined, invokingState: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
