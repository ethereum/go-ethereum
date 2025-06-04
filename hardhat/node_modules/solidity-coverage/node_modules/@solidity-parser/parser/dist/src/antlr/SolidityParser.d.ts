import { ATN, DFA, FailedPredicateException, Parser, RuleContext, ParserRuleContext, TerminalNode, TokenStream } from 'antlr4';
import SolidityListener from "./SolidityListener.js";
import SolidityVisitor from "./SolidityVisitor.js";
export default class SolidityParser extends Parser {
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
    static readonly T__97 = 98;
    static readonly Int = 99;
    static readonly Uint = 100;
    static readonly Byte = 101;
    static readonly Fixed = 102;
    static readonly Ufixed = 103;
    static readonly BooleanLiteral = 104;
    static readonly DecimalNumber = 105;
    static readonly HexNumber = 106;
    static readonly NumberUnit = 107;
    static readonly HexLiteralFragment = 108;
    static readonly ReservedKeyword = 109;
    static readonly AnonymousKeyword = 110;
    static readonly BreakKeyword = 111;
    static readonly ConstantKeyword = 112;
    static readonly TransientKeyword = 113;
    static readonly ImmutableKeyword = 114;
    static readonly ContinueKeyword = 115;
    static readonly LeaveKeyword = 116;
    static readonly ExternalKeyword = 117;
    static readonly IndexedKeyword = 118;
    static readonly InternalKeyword = 119;
    static readonly PayableKeyword = 120;
    static readonly PrivateKeyword = 121;
    static readonly PublicKeyword = 122;
    static readonly VirtualKeyword = 123;
    static readonly PureKeyword = 124;
    static readonly TypeKeyword = 125;
    static readonly ViewKeyword = 126;
    static readonly GlobalKeyword = 127;
    static readonly ConstructorKeyword = 128;
    static readonly FallbackKeyword = 129;
    static readonly ReceiveKeyword = 130;
    static readonly Identifier = 131;
    static readonly StringLiteralFragment = 132;
    static readonly VersionLiteral = 133;
    static readonly WS = 134;
    static readonly COMMENT = 135;
    static readonly LINE_COMMENT = 136;
    static readonly EOF: number;
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
    static readonly RULE_customStorageLayout = 12;
    static readonly RULE_contractPart = 13;
    static readonly RULE_stateVariableDeclaration = 14;
    static readonly RULE_fileLevelConstant = 15;
    static readonly RULE_customErrorDefinition = 16;
    static readonly RULE_typeDefinition = 17;
    static readonly RULE_usingForDeclaration = 18;
    static readonly RULE_usingForObject = 19;
    static readonly RULE_usingForObjectDirective = 20;
    static readonly RULE_userDefinableOperators = 21;
    static readonly RULE_structDefinition = 22;
    static readonly RULE_modifierDefinition = 23;
    static readonly RULE_modifierInvocation = 24;
    static readonly RULE_functionDefinition = 25;
    static readonly RULE_functionDescriptor = 26;
    static readonly RULE_returnParameters = 27;
    static readonly RULE_modifierList = 28;
    static readonly RULE_eventDefinition = 29;
    static readonly RULE_enumValue = 30;
    static readonly RULE_enumDefinition = 31;
    static readonly RULE_parameterList = 32;
    static readonly RULE_parameter = 33;
    static readonly RULE_eventParameterList = 34;
    static readonly RULE_eventParameter = 35;
    static readonly RULE_functionTypeParameterList = 36;
    static readonly RULE_functionTypeParameter = 37;
    static readonly RULE_variableDeclaration = 38;
    static readonly RULE_typeName = 39;
    static readonly RULE_userDefinedTypeName = 40;
    static readonly RULE_mappingKey = 41;
    static readonly RULE_mapping = 42;
    static readonly RULE_mappingKeyName = 43;
    static readonly RULE_mappingValueName = 44;
    static readonly RULE_functionTypeName = 45;
    static readonly RULE_storageLocation = 46;
    static readonly RULE_stateMutability = 47;
    static readonly RULE_block = 48;
    static readonly RULE_statement = 49;
    static readonly RULE_expressionStatement = 50;
    static readonly RULE_ifStatement = 51;
    static readonly RULE_tryStatement = 52;
    static readonly RULE_catchClause = 53;
    static readonly RULE_whileStatement = 54;
    static readonly RULE_simpleStatement = 55;
    static readonly RULE_uncheckedStatement = 56;
    static readonly RULE_forStatement = 57;
    static readonly RULE_inlineAssemblyStatement = 58;
    static readonly RULE_inlineAssemblyStatementFlag = 59;
    static readonly RULE_doWhileStatement = 60;
    static readonly RULE_continueStatement = 61;
    static readonly RULE_breakStatement = 62;
    static readonly RULE_returnStatement = 63;
    static readonly RULE_throwStatement = 64;
    static readonly RULE_emitStatement = 65;
    static readonly RULE_revertStatement = 66;
    static readonly RULE_variableDeclarationStatement = 67;
    static readonly RULE_variableDeclarationList = 68;
    static readonly RULE_identifierList = 69;
    static readonly RULE_elementaryTypeName = 70;
    static readonly RULE_expression = 71;
    static readonly RULE_primaryExpression = 72;
    static readonly RULE_expressionList = 73;
    static readonly RULE_nameValueList = 74;
    static readonly RULE_nameValue = 75;
    static readonly RULE_functionCallArguments = 76;
    static readonly RULE_functionCall = 77;
    static readonly RULE_assemblyBlock = 78;
    static readonly RULE_assemblyItem = 79;
    static readonly RULE_assemblyExpression = 80;
    static readonly RULE_assemblyMember = 81;
    static readonly RULE_assemblyCall = 82;
    static readonly RULE_assemblyLocalDefinition = 83;
    static readonly RULE_assemblyAssignment = 84;
    static readonly RULE_assemblyIdentifierOrList = 85;
    static readonly RULE_assemblyIdentifierList = 86;
    static readonly RULE_assemblyStackAssignment = 87;
    static readonly RULE_labelDefinition = 88;
    static readonly RULE_assemblySwitch = 89;
    static readonly RULE_assemblyCase = 90;
    static readonly RULE_assemblyFunctionDefinition = 91;
    static readonly RULE_assemblyFunctionReturns = 92;
    static readonly RULE_assemblyFor = 93;
    static readonly RULE_assemblyIf = 94;
    static readonly RULE_assemblyLiteral = 95;
    static readonly RULE_tupleExpression = 96;
    static readonly RULE_numberLiteral = 97;
    static readonly RULE_identifier = 98;
    static readonly RULE_hexLiteral = 99;
    static readonly RULE_overrideSpecifier = 100;
    static readonly RULE_stringLiteral = 101;
    static readonly literalNames: (string | null)[];
    static readonly symbolicNames: (string | null)[];
    static readonly ruleNames: string[];
    get grammarFileName(): string;
    get literalNames(): (string | null)[];
    get symbolicNames(): (string | null)[];
    get ruleNames(): string[];
    get serializedATN(): number[];
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
    customStorageLayout(): CustomStorageLayoutContext;
    contractPart(): ContractPartContext;
    stateVariableDeclaration(): StateVariableDeclarationContext;
    fileLevelConstant(): FileLevelConstantContext;
    customErrorDefinition(): CustomErrorDefinitionContext;
    typeDefinition(): TypeDefinitionContext;
    usingForDeclaration(): UsingForDeclarationContext;
    usingForObject(): UsingForObjectContext;
    usingForObjectDirective(): UsingForObjectDirectiveContext;
    userDefinableOperators(): UserDefinableOperatorsContext;
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
    mappingKeyName(): MappingKeyNameContext;
    mappingValueName(): MappingValueNameContext;
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
    tupleExpression(): TupleExpressionContext;
    numberLiteral(): NumberLiteralContext;
    identifier(): IdentifierContext;
    hexLiteral(): HexLiteralContext;
    overrideSpecifier(): OverrideSpecifierContext;
    stringLiteral(): StringLiteralContext;
    sempred(localctx: RuleContext, ruleIndex: number, predIndex: number): boolean;
    private typeName_sempred;
    private expression_sempred;
    static readonly _serializedATN: number[];
    private static __ATN;
    static get _ATN(): ATN;
    static DecisionsToDFA: DFA[];
}
export declare class SourceUnitContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    EOF(): TerminalNode;
    pragmaDirective_list(): PragmaDirectiveContext[];
    pragmaDirective(i: number): PragmaDirectiveContext;
    importDirective_list(): ImportDirectiveContext[];
    importDirective(i: number): ImportDirectiveContext;
    contractDefinition_list(): ContractDefinitionContext[];
    contractDefinition(i: number): ContractDefinitionContext;
    enumDefinition_list(): EnumDefinitionContext[];
    enumDefinition(i: number): EnumDefinitionContext;
    eventDefinition_list(): EventDefinitionContext[];
    eventDefinition(i: number): EventDefinitionContext;
    structDefinition_list(): StructDefinitionContext[];
    structDefinition(i: number): StructDefinitionContext;
    functionDefinition_list(): FunctionDefinitionContext[];
    functionDefinition(i: number): FunctionDefinitionContext;
    fileLevelConstant_list(): FileLevelConstantContext[];
    fileLevelConstant(i: number): FileLevelConstantContext;
    customErrorDefinition_list(): CustomErrorDefinitionContext[];
    customErrorDefinition(i: number): CustomErrorDefinitionContext;
    typeDefinition_list(): TypeDefinitionContext[];
    typeDefinition(i: number): TypeDefinitionContext;
    usingForDeclaration_list(): UsingForDeclarationContext[];
    usingForDeclaration(i: number): UsingForDeclarationContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class PragmaDirectiveContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    pragmaName(): PragmaNameContext;
    pragmaValue(): PragmaValueContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class PragmaNameContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    identifier(): IdentifierContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class PragmaValueContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    version(): VersionContext;
    expression(): ExpressionContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class VersionContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    versionConstraint_list(): VersionConstraintContext[];
    versionConstraint(i: number): VersionConstraintContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class VersionOperatorContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class VersionConstraintContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    VersionLiteral(): TerminalNode;
    versionOperator(): VersionOperatorContext;
    DecimalNumber(): TerminalNode;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class ImportDeclarationContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    identifier_list(): IdentifierContext[];
    identifier(i: number): IdentifierContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class ImportDirectiveContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    importPath(): ImportPathContext;
    identifier_list(): IdentifierContext[];
    identifier(i: number): IdentifierContext;
    importDeclaration_list(): ImportDeclarationContext[];
    importDeclaration(i: number): ImportDeclarationContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class ImportPathContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    StringLiteralFragment(): TerminalNode;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class ContractDefinitionContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    identifier(): IdentifierContext;
    customStorageLayout_list(): CustomStorageLayoutContext[];
    customStorageLayout(i: number): CustomStorageLayoutContext;
    inheritanceSpecifier_list(): InheritanceSpecifierContext[];
    inheritanceSpecifier(i: number): InheritanceSpecifierContext;
    contractPart_list(): ContractPartContext[];
    contractPart(i: number): ContractPartContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class InheritanceSpecifierContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    userDefinedTypeName(): UserDefinedTypeNameContext;
    expressionList(): ExpressionListContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class CustomStorageLayoutContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    expression(): ExpressionContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class ContractPartContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    stateVariableDeclaration(): StateVariableDeclarationContext;
    usingForDeclaration(): UsingForDeclarationContext;
    structDefinition(): StructDefinitionContext;
    modifierDefinition(): ModifierDefinitionContext;
    functionDefinition(): FunctionDefinitionContext;
    eventDefinition(): EventDefinitionContext;
    enumDefinition(): EnumDefinitionContext;
    customErrorDefinition(): CustomErrorDefinitionContext;
    typeDefinition(): TypeDefinitionContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class StateVariableDeclarationContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    typeName(): TypeNameContext;
    identifier(): IdentifierContext;
    PublicKeyword_list(): TerminalNode[];
    PublicKeyword(i: number): TerminalNode;
    InternalKeyword_list(): TerminalNode[];
    InternalKeyword(i: number): TerminalNode;
    PrivateKeyword_list(): TerminalNode[];
    PrivateKeyword(i: number): TerminalNode;
    ConstantKeyword_list(): TerminalNode[];
    ConstantKeyword(i: number): TerminalNode;
    TransientKeyword_list(): TerminalNode[];
    TransientKeyword(i: number): TerminalNode;
    ImmutableKeyword_list(): TerminalNode[];
    ImmutableKeyword(i: number): TerminalNode;
    overrideSpecifier_list(): OverrideSpecifierContext[];
    overrideSpecifier(i: number): OverrideSpecifierContext;
    expression(): ExpressionContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class FileLevelConstantContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    typeName(): TypeNameContext;
    ConstantKeyword(): TerminalNode;
    identifier(): IdentifierContext;
    expression(): ExpressionContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class CustomErrorDefinitionContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    identifier(): IdentifierContext;
    parameterList(): ParameterListContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class TypeDefinitionContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    TypeKeyword(): TerminalNode;
    identifier(): IdentifierContext;
    elementaryTypeName(): ElementaryTypeNameContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class UsingForDeclarationContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    usingForObject(): UsingForObjectContext;
    typeName(): TypeNameContext;
    GlobalKeyword(): TerminalNode;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class UsingForObjectContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    userDefinedTypeName(): UserDefinedTypeNameContext;
    usingForObjectDirective_list(): UsingForObjectDirectiveContext[];
    usingForObjectDirective(i: number): UsingForObjectDirectiveContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class UsingForObjectDirectiveContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    userDefinedTypeName(): UserDefinedTypeNameContext;
    userDefinableOperators(): UserDefinableOperatorsContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class UserDefinableOperatorsContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class StructDefinitionContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    identifier(): IdentifierContext;
    variableDeclaration_list(): VariableDeclarationContext[];
    variableDeclaration(i: number): VariableDeclarationContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class ModifierDefinitionContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    identifier(): IdentifierContext;
    block(): BlockContext;
    parameterList(): ParameterListContext;
    VirtualKeyword_list(): TerminalNode[];
    VirtualKeyword(i: number): TerminalNode;
    overrideSpecifier_list(): OverrideSpecifierContext[];
    overrideSpecifier(i: number): OverrideSpecifierContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class ModifierInvocationContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    identifier(): IdentifierContext;
    expressionList(): ExpressionListContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class FunctionDefinitionContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    functionDescriptor(): FunctionDescriptorContext;
    parameterList(): ParameterListContext;
    modifierList(): ModifierListContext;
    block(): BlockContext;
    returnParameters(): ReturnParametersContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class FunctionDescriptorContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    identifier(): IdentifierContext;
    ConstructorKeyword(): TerminalNode;
    FallbackKeyword(): TerminalNode;
    ReceiveKeyword(): TerminalNode;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class ReturnParametersContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    parameterList(): ParameterListContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class ModifierListContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    ExternalKeyword_list(): TerminalNode[];
    ExternalKeyword(i: number): TerminalNode;
    PublicKeyword_list(): TerminalNode[];
    PublicKeyword(i: number): TerminalNode;
    InternalKeyword_list(): TerminalNode[];
    InternalKeyword(i: number): TerminalNode;
    PrivateKeyword_list(): TerminalNode[];
    PrivateKeyword(i: number): TerminalNode;
    VirtualKeyword_list(): TerminalNode[];
    VirtualKeyword(i: number): TerminalNode;
    stateMutability_list(): StateMutabilityContext[];
    stateMutability(i: number): StateMutabilityContext;
    modifierInvocation_list(): ModifierInvocationContext[];
    modifierInvocation(i: number): ModifierInvocationContext;
    overrideSpecifier_list(): OverrideSpecifierContext[];
    overrideSpecifier(i: number): OverrideSpecifierContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class EventDefinitionContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    identifier(): IdentifierContext;
    eventParameterList(): EventParameterListContext;
    AnonymousKeyword(): TerminalNode;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class EnumValueContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    identifier(): IdentifierContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class EnumDefinitionContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    identifier(): IdentifierContext;
    enumValue_list(): EnumValueContext[];
    enumValue(i: number): EnumValueContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class ParameterListContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    parameter_list(): ParameterContext[];
    parameter(i: number): ParameterContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class ParameterContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    typeName(): TypeNameContext;
    storageLocation(): StorageLocationContext;
    identifier(): IdentifierContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class EventParameterListContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    eventParameter_list(): EventParameterContext[];
    eventParameter(i: number): EventParameterContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class EventParameterContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    typeName(): TypeNameContext;
    IndexedKeyword(): TerminalNode;
    identifier(): IdentifierContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class FunctionTypeParameterListContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    functionTypeParameter_list(): FunctionTypeParameterContext[];
    functionTypeParameter(i: number): FunctionTypeParameterContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class FunctionTypeParameterContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    typeName(): TypeNameContext;
    storageLocation(): StorageLocationContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class VariableDeclarationContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    typeName(): TypeNameContext;
    identifier(): IdentifierContext;
    storageLocation(): StorageLocationContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class TypeNameContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    elementaryTypeName(): ElementaryTypeNameContext;
    userDefinedTypeName(): UserDefinedTypeNameContext;
    mapping(): MappingContext;
    functionTypeName(): FunctionTypeNameContext;
    PayableKeyword(): TerminalNode;
    typeName(): TypeNameContext;
    expression(): ExpressionContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class UserDefinedTypeNameContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    identifier_list(): IdentifierContext[];
    identifier(i: number): IdentifierContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class MappingKeyContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    elementaryTypeName(): ElementaryTypeNameContext;
    userDefinedTypeName(): UserDefinedTypeNameContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class MappingContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    mappingKey(): MappingKeyContext;
    typeName(): TypeNameContext;
    mappingKeyName(): MappingKeyNameContext;
    mappingValueName(): MappingValueNameContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class MappingKeyNameContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    identifier(): IdentifierContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class MappingValueNameContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    identifier(): IdentifierContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class FunctionTypeNameContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    functionTypeParameterList_list(): FunctionTypeParameterListContext[];
    functionTypeParameterList(i: number): FunctionTypeParameterListContext;
    InternalKeyword_list(): TerminalNode[];
    InternalKeyword(i: number): TerminalNode;
    ExternalKeyword_list(): TerminalNode[];
    ExternalKeyword(i: number): TerminalNode;
    stateMutability_list(): StateMutabilityContext[];
    stateMutability(i: number): StateMutabilityContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class StorageLocationContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class StateMutabilityContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    PureKeyword(): TerminalNode;
    ConstantKeyword(): TerminalNode;
    ViewKeyword(): TerminalNode;
    PayableKeyword(): TerminalNode;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class BlockContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    statement_list(): StatementContext[];
    statement(i: number): StatementContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class StatementContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    ifStatement(): IfStatementContext;
    tryStatement(): TryStatementContext;
    whileStatement(): WhileStatementContext;
    forStatement(): ForStatementContext;
    block(): BlockContext;
    inlineAssemblyStatement(): InlineAssemblyStatementContext;
    doWhileStatement(): DoWhileStatementContext;
    continueStatement(): ContinueStatementContext;
    breakStatement(): BreakStatementContext;
    returnStatement(): ReturnStatementContext;
    throwStatement(): ThrowStatementContext;
    emitStatement(): EmitStatementContext;
    simpleStatement(): SimpleStatementContext;
    uncheckedStatement(): UncheckedStatementContext;
    revertStatement(): RevertStatementContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class ExpressionStatementContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    expression(): ExpressionContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class IfStatementContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    expression(): ExpressionContext;
    statement_list(): StatementContext[];
    statement(i: number): StatementContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class TryStatementContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    expression(): ExpressionContext;
    block(): BlockContext;
    returnParameters(): ReturnParametersContext;
    catchClause_list(): CatchClauseContext[];
    catchClause(i: number): CatchClauseContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class CatchClauseContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    block(): BlockContext;
    parameterList(): ParameterListContext;
    identifier(): IdentifierContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class WhileStatementContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    expression(): ExpressionContext;
    statement(): StatementContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class SimpleStatementContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    variableDeclarationStatement(): VariableDeclarationStatementContext;
    expressionStatement(): ExpressionStatementContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class UncheckedStatementContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    block(): BlockContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class ForStatementContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    statement(): StatementContext;
    simpleStatement(): SimpleStatementContext;
    expressionStatement(): ExpressionStatementContext;
    expression(): ExpressionContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class InlineAssemblyStatementContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    assemblyBlock(): AssemblyBlockContext;
    StringLiteralFragment(): TerminalNode;
    inlineAssemblyStatementFlag(): InlineAssemblyStatementFlagContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class InlineAssemblyStatementFlagContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    stringLiteral(): StringLiteralContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class DoWhileStatementContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    statement(): StatementContext;
    expression(): ExpressionContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class ContinueStatementContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    ContinueKeyword(): TerminalNode;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class BreakStatementContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    BreakKeyword(): TerminalNode;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class ReturnStatementContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    expression(): ExpressionContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class ThrowStatementContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class EmitStatementContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    functionCall(): FunctionCallContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class RevertStatementContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    functionCall(): FunctionCallContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class VariableDeclarationStatementContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    identifierList(): IdentifierListContext;
    variableDeclaration(): VariableDeclarationContext;
    variableDeclarationList(): VariableDeclarationListContext;
    expression(): ExpressionContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class VariableDeclarationListContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    variableDeclaration_list(): VariableDeclarationContext[];
    variableDeclaration(i: number): VariableDeclarationContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class IdentifierListContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    identifier_list(): IdentifierContext[];
    identifier(i: number): IdentifierContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class ElementaryTypeNameContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    Int(): TerminalNode;
    Uint(): TerminalNode;
    Byte(): TerminalNode;
    Fixed(): TerminalNode;
    Ufixed(): TerminalNode;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class ExpressionContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    typeName(): TypeNameContext;
    expression_list(): ExpressionContext[];
    expression(i: number): ExpressionContext;
    primaryExpression(): PrimaryExpressionContext;
    identifier(): IdentifierContext;
    nameValueList(): NameValueListContext;
    functionCallArguments(): FunctionCallArgumentsContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class PrimaryExpressionContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    BooleanLiteral(): TerminalNode;
    numberLiteral(): NumberLiteralContext;
    hexLiteral(): HexLiteralContext;
    stringLiteral(): StringLiteralContext;
    identifier(): IdentifierContext;
    TypeKeyword(): TerminalNode;
    PayableKeyword(): TerminalNode;
    tupleExpression(): TupleExpressionContext;
    typeName(): TypeNameContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class ExpressionListContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    expression_list(): ExpressionContext[];
    expression(i: number): ExpressionContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class NameValueListContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    nameValue_list(): NameValueContext[];
    nameValue(i: number): NameValueContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class NameValueContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    identifier(): IdentifierContext;
    expression(): ExpressionContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class FunctionCallArgumentsContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    nameValueList(): NameValueListContext;
    expressionList(): ExpressionListContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class FunctionCallContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    expression(): ExpressionContext;
    functionCallArguments(): FunctionCallArgumentsContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class AssemblyBlockContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    assemblyItem_list(): AssemblyItemContext[];
    assemblyItem(i: number): AssemblyItemContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class AssemblyItemContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    identifier(): IdentifierContext;
    assemblyBlock(): AssemblyBlockContext;
    assemblyExpression(): AssemblyExpressionContext;
    assemblyLocalDefinition(): AssemblyLocalDefinitionContext;
    assemblyAssignment(): AssemblyAssignmentContext;
    assemblyStackAssignment(): AssemblyStackAssignmentContext;
    labelDefinition(): LabelDefinitionContext;
    assemblySwitch(): AssemblySwitchContext;
    assemblyFunctionDefinition(): AssemblyFunctionDefinitionContext;
    assemblyFor(): AssemblyForContext;
    assemblyIf(): AssemblyIfContext;
    BreakKeyword(): TerminalNode;
    ContinueKeyword(): TerminalNode;
    LeaveKeyword(): TerminalNode;
    numberLiteral(): NumberLiteralContext;
    stringLiteral(): StringLiteralContext;
    hexLiteral(): HexLiteralContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class AssemblyExpressionContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    assemblyCall(): AssemblyCallContext;
    assemblyLiteral(): AssemblyLiteralContext;
    assemblyMember(): AssemblyMemberContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class AssemblyMemberContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    identifier_list(): IdentifierContext[];
    identifier(i: number): IdentifierContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class AssemblyCallContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    identifier(): IdentifierContext;
    assemblyExpression_list(): AssemblyExpressionContext[];
    assemblyExpression(i: number): AssemblyExpressionContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class AssemblyLocalDefinitionContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    assemblyIdentifierOrList(): AssemblyIdentifierOrListContext;
    assemblyExpression(): AssemblyExpressionContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class AssemblyAssignmentContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    assemblyIdentifierOrList(): AssemblyIdentifierOrListContext;
    assemblyExpression(): AssemblyExpressionContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class AssemblyIdentifierOrListContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    identifier(): IdentifierContext;
    assemblyMember(): AssemblyMemberContext;
    assemblyIdentifierList(): AssemblyIdentifierListContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class AssemblyIdentifierListContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    identifier_list(): IdentifierContext[];
    identifier(i: number): IdentifierContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class AssemblyStackAssignmentContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    assemblyExpression(): AssemblyExpressionContext;
    identifier(): IdentifierContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class LabelDefinitionContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    identifier(): IdentifierContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class AssemblySwitchContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    assemblyExpression(): AssemblyExpressionContext;
    assemblyCase_list(): AssemblyCaseContext[];
    assemblyCase(i: number): AssemblyCaseContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class AssemblyCaseContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    assemblyLiteral(): AssemblyLiteralContext;
    assemblyBlock(): AssemblyBlockContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class AssemblyFunctionDefinitionContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    identifier(): IdentifierContext;
    assemblyBlock(): AssemblyBlockContext;
    assemblyIdentifierList(): AssemblyIdentifierListContext;
    assemblyFunctionReturns(): AssemblyFunctionReturnsContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class AssemblyFunctionReturnsContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    assemblyIdentifierList(): AssemblyIdentifierListContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class AssemblyForContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    assemblyExpression_list(): AssemblyExpressionContext[];
    assemblyExpression(i: number): AssemblyExpressionContext;
    assemblyBlock_list(): AssemblyBlockContext[];
    assemblyBlock(i: number): AssemblyBlockContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class AssemblyIfContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    assemblyExpression(): AssemblyExpressionContext;
    assemblyBlock(): AssemblyBlockContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class AssemblyLiteralContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    stringLiteral(): StringLiteralContext;
    DecimalNumber(): TerminalNode;
    HexNumber(): TerminalNode;
    hexLiteral(): HexLiteralContext;
    BooleanLiteral(): TerminalNode;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class TupleExpressionContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    expression_list(): ExpressionContext[];
    expression(i: number): ExpressionContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class NumberLiteralContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    DecimalNumber(): TerminalNode;
    HexNumber(): TerminalNode;
    NumberUnit(): TerminalNode;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class IdentifierContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    ReceiveKeyword(): TerminalNode;
    GlobalKeyword(): TerminalNode;
    ConstructorKeyword(): TerminalNode;
    PayableKeyword(): TerminalNode;
    LeaveKeyword(): TerminalNode;
    Identifier(): TerminalNode;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class HexLiteralContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    HexLiteralFragment_list(): TerminalNode[];
    HexLiteralFragment(i: number): TerminalNode;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class OverrideSpecifierContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    userDefinedTypeName_list(): UserDefinedTypeNameContext[];
    userDefinedTypeName(i: number): UserDefinedTypeNameContext;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
export declare class StringLiteralContext extends ParserRuleContext {
    constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number);
    StringLiteralFragment_list(): TerminalNode[];
    StringLiteralFragment(i: number): TerminalNode;
    get ruleIndex(): number;
    enterRule(listener: SolidityListener): void;
    exitRule(listener: SolidityListener): void;
    accept<Result>(visitor: SolidityVisitor<Result>): Result;
}
