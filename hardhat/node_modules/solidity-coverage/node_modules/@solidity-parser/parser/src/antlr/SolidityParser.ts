// Generated from antlr/Solidity.g4 by ANTLR 4.13.2
// noinspection ES6UnusedImports,JSUnusedGlobalSymbols,JSUnusedLocalSymbols

import {
	ATN,
	ATNDeserializer, DecisionState, DFA, FailedPredicateException,
	RecognitionException, NoViableAltException, BailErrorStrategy,
	Parser, ParserATNSimulator,
	RuleContext, ParserRuleContext, PredictionMode, PredictionContextCache,
	TerminalNode, RuleNode,
	Token, TokenStream,
	Interval, IntervalSet
} from 'antlr4';
import SolidityListener from "./SolidityListener.js";
import SolidityVisitor from "./SolidityVisitor.js";

// for running tests with parameters, TODO: discuss strategy for typed parameters in CI
// eslint-disable-next-line no-unused-vars
type int = number;

export default class SolidityParser extends Parser {
	public static readonly T__0 = 1;
	public static readonly T__1 = 2;
	public static readonly T__2 = 3;
	public static readonly T__3 = 4;
	public static readonly T__4 = 5;
	public static readonly T__5 = 6;
	public static readonly T__6 = 7;
	public static readonly T__7 = 8;
	public static readonly T__8 = 9;
	public static readonly T__9 = 10;
	public static readonly T__10 = 11;
	public static readonly T__11 = 12;
	public static readonly T__12 = 13;
	public static readonly T__13 = 14;
	public static readonly T__14 = 15;
	public static readonly T__15 = 16;
	public static readonly T__16 = 17;
	public static readonly T__17 = 18;
	public static readonly T__18 = 19;
	public static readonly T__19 = 20;
	public static readonly T__20 = 21;
	public static readonly T__21 = 22;
	public static readonly T__22 = 23;
	public static readonly T__23 = 24;
	public static readonly T__24 = 25;
	public static readonly T__25 = 26;
	public static readonly T__26 = 27;
	public static readonly T__27 = 28;
	public static readonly T__28 = 29;
	public static readonly T__29 = 30;
	public static readonly T__30 = 31;
	public static readonly T__31 = 32;
	public static readonly T__32 = 33;
	public static readonly T__33 = 34;
	public static readonly T__34 = 35;
	public static readonly T__35 = 36;
	public static readonly T__36 = 37;
	public static readonly T__37 = 38;
	public static readonly T__38 = 39;
	public static readonly T__39 = 40;
	public static readonly T__40 = 41;
	public static readonly T__41 = 42;
	public static readonly T__42 = 43;
	public static readonly T__43 = 44;
	public static readonly T__44 = 45;
	public static readonly T__45 = 46;
	public static readonly T__46 = 47;
	public static readonly T__47 = 48;
	public static readonly T__48 = 49;
	public static readonly T__49 = 50;
	public static readonly T__50 = 51;
	public static readonly T__51 = 52;
	public static readonly T__52 = 53;
	public static readonly T__53 = 54;
	public static readonly T__54 = 55;
	public static readonly T__55 = 56;
	public static readonly T__56 = 57;
	public static readonly T__57 = 58;
	public static readonly T__58 = 59;
	public static readonly T__59 = 60;
	public static readonly T__60 = 61;
	public static readonly T__61 = 62;
	public static readonly T__62 = 63;
	public static readonly T__63 = 64;
	public static readonly T__64 = 65;
	public static readonly T__65 = 66;
	public static readonly T__66 = 67;
	public static readonly T__67 = 68;
	public static readonly T__68 = 69;
	public static readonly T__69 = 70;
	public static readonly T__70 = 71;
	public static readonly T__71 = 72;
	public static readonly T__72 = 73;
	public static readonly T__73 = 74;
	public static readonly T__74 = 75;
	public static readonly T__75 = 76;
	public static readonly T__76 = 77;
	public static readonly T__77 = 78;
	public static readonly T__78 = 79;
	public static readonly T__79 = 80;
	public static readonly T__80 = 81;
	public static readonly T__81 = 82;
	public static readonly T__82 = 83;
	public static readonly T__83 = 84;
	public static readonly T__84 = 85;
	public static readonly T__85 = 86;
	public static readonly T__86 = 87;
	public static readonly T__87 = 88;
	public static readonly T__88 = 89;
	public static readonly T__89 = 90;
	public static readonly T__90 = 91;
	public static readonly T__91 = 92;
	public static readonly T__92 = 93;
	public static readonly T__93 = 94;
	public static readonly T__94 = 95;
	public static readonly T__95 = 96;
	public static readonly T__96 = 97;
	public static readonly T__97 = 98;
	public static readonly Int = 99;
	public static readonly Uint = 100;
	public static readonly Byte = 101;
	public static readonly Fixed = 102;
	public static readonly Ufixed = 103;
	public static readonly BooleanLiteral = 104;
	public static readonly DecimalNumber = 105;
	public static readonly HexNumber = 106;
	public static readonly NumberUnit = 107;
	public static readonly HexLiteralFragment = 108;
	public static readonly ReservedKeyword = 109;
	public static readonly AnonymousKeyword = 110;
	public static readonly BreakKeyword = 111;
	public static readonly ConstantKeyword = 112;
	public static readonly TransientKeyword = 113;
	public static readonly ImmutableKeyword = 114;
	public static readonly ContinueKeyword = 115;
	public static readonly LeaveKeyword = 116;
	public static readonly ExternalKeyword = 117;
	public static readonly IndexedKeyword = 118;
	public static readonly InternalKeyword = 119;
	public static readonly PayableKeyword = 120;
	public static readonly PrivateKeyword = 121;
	public static readonly PublicKeyword = 122;
	public static readonly VirtualKeyword = 123;
	public static readonly PureKeyword = 124;
	public static readonly TypeKeyword = 125;
	public static readonly ViewKeyword = 126;
	public static readonly GlobalKeyword = 127;
	public static readonly ConstructorKeyword = 128;
	public static readonly FallbackKeyword = 129;
	public static readonly ReceiveKeyword = 130;
	public static readonly Identifier = 131;
	public static readonly StringLiteralFragment = 132;
	public static readonly VersionLiteral = 133;
	public static readonly WS = 134;
	public static readonly COMMENT = 135;
	public static readonly LINE_COMMENT = 136;
	public static override readonly EOF = Token.EOF;
	public static readonly RULE_sourceUnit = 0;
	public static readonly RULE_pragmaDirective = 1;
	public static readonly RULE_pragmaName = 2;
	public static readonly RULE_pragmaValue = 3;
	public static readonly RULE_version = 4;
	public static readonly RULE_versionOperator = 5;
	public static readonly RULE_versionConstraint = 6;
	public static readonly RULE_importDeclaration = 7;
	public static readonly RULE_importDirective = 8;
	public static readonly RULE_importPath = 9;
	public static readonly RULE_contractDefinition = 10;
	public static readonly RULE_inheritanceSpecifier = 11;
	public static readonly RULE_customStorageLayout = 12;
	public static readonly RULE_contractPart = 13;
	public static readonly RULE_stateVariableDeclaration = 14;
	public static readonly RULE_fileLevelConstant = 15;
	public static readonly RULE_customErrorDefinition = 16;
	public static readonly RULE_typeDefinition = 17;
	public static readonly RULE_usingForDeclaration = 18;
	public static readonly RULE_usingForObject = 19;
	public static readonly RULE_usingForObjectDirective = 20;
	public static readonly RULE_userDefinableOperators = 21;
	public static readonly RULE_structDefinition = 22;
	public static readonly RULE_modifierDefinition = 23;
	public static readonly RULE_modifierInvocation = 24;
	public static readonly RULE_functionDefinition = 25;
	public static readonly RULE_functionDescriptor = 26;
	public static readonly RULE_returnParameters = 27;
	public static readonly RULE_modifierList = 28;
	public static readonly RULE_eventDefinition = 29;
	public static readonly RULE_enumValue = 30;
	public static readonly RULE_enumDefinition = 31;
	public static readonly RULE_parameterList = 32;
	public static readonly RULE_parameter = 33;
	public static readonly RULE_eventParameterList = 34;
	public static readonly RULE_eventParameter = 35;
	public static readonly RULE_functionTypeParameterList = 36;
	public static readonly RULE_functionTypeParameter = 37;
	public static readonly RULE_variableDeclaration = 38;
	public static readonly RULE_typeName = 39;
	public static readonly RULE_userDefinedTypeName = 40;
	public static readonly RULE_mappingKey = 41;
	public static readonly RULE_mapping = 42;
	public static readonly RULE_mappingKeyName = 43;
	public static readonly RULE_mappingValueName = 44;
	public static readonly RULE_functionTypeName = 45;
	public static readonly RULE_storageLocation = 46;
	public static readonly RULE_stateMutability = 47;
	public static readonly RULE_block = 48;
	public static readonly RULE_statement = 49;
	public static readonly RULE_expressionStatement = 50;
	public static readonly RULE_ifStatement = 51;
	public static readonly RULE_tryStatement = 52;
	public static readonly RULE_catchClause = 53;
	public static readonly RULE_whileStatement = 54;
	public static readonly RULE_simpleStatement = 55;
	public static readonly RULE_uncheckedStatement = 56;
	public static readonly RULE_forStatement = 57;
	public static readonly RULE_inlineAssemblyStatement = 58;
	public static readonly RULE_inlineAssemblyStatementFlag = 59;
	public static readonly RULE_doWhileStatement = 60;
	public static readonly RULE_continueStatement = 61;
	public static readonly RULE_breakStatement = 62;
	public static readonly RULE_returnStatement = 63;
	public static readonly RULE_throwStatement = 64;
	public static readonly RULE_emitStatement = 65;
	public static readonly RULE_revertStatement = 66;
	public static readonly RULE_variableDeclarationStatement = 67;
	public static readonly RULE_variableDeclarationList = 68;
	public static readonly RULE_identifierList = 69;
	public static readonly RULE_elementaryTypeName = 70;
	public static readonly RULE_expression = 71;
	public static readonly RULE_primaryExpression = 72;
	public static readonly RULE_expressionList = 73;
	public static readonly RULE_nameValueList = 74;
	public static readonly RULE_nameValue = 75;
	public static readonly RULE_functionCallArguments = 76;
	public static readonly RULE_functionCall = 77;
	public static readonly RULE_assemblyBlock = 78;
	public static readonly RULE_assemblyItem = 79;
	public static readonly RULE_assemblyExpression = 80;
	public static readonly RULE_assemblyMember = 81;
	public static readonly RULE_assemblyCall = 82;
	public static readonly RULE_assemblyLocalDefinition = 83;
	public static readonly RULE_assemblyAssignment = 84;
	public static readonly RULE_assemblyIdentifierOrList = 85;
	public static readonly RULE_assemblyIdentifierList = 86;
	public static readonly RULE_assemblyStackAssignment = 87;
	public static readonly RULE_labelDefinition = 88;
	public static readonly RULE_assemblySwitch = 89;
	public static readonly RULE_assemblyCase = 90;
	public static readonly RULE_assemblyFunctionDefinition = 91;
	public static readonly RULE_assemblyFunctionReturns = 92;
	public static readonly RULE_assemblyFor = 93;
	public static readonly RULE_assemblyIf = 94;
	public static readonly RULE_assemblyLiteral = 95;
	public static readonly RULE_tupleExpression = 96;
	public static readonly RULE_numberLiteral = 97;
	public static readonly RULE_identifier = 98;
	public static readonly RULE_hexLiteral = 99;
	public static readonly RULE_overrideSpecifier = 100;
	public static readonly RULE_stringLiteral = 101;
	public static readonly literalNames: (string | null)[] = [ null, "'pragma'", 
                                                            "';'", "'*'", 
                                                            "'||'", "'^'", 
                                                            "'~'", "'>='", 
                                                            "'>'", "'<'", 
                                                            "'<='", "'='", 
                                                            "'as'", "'import'", 
                                                            "'from'", "'{'", 
                                                            "','", "'}'", 
                                                            "'abstract'", 
                                                            "'contract'", 
                                                            "'interface'", 
                                                            "'library'", 
                                                            "'is'", "'('", 
                                                            "')'", "'layout'", 
                                                            "'at'", "'error'", 
                                                            "'using'", "'for'", 
                                                            "'|'", "'&'", 
                                                            "'+'", "'-'", 
                                                            "'/'", "'%'", 
                                                            "'=='", "'!='", 
                                                            "'struct'", 
                                                            "'modifier'", 
                                                            "'function'", 
                                                            "'returns'", 
                                                            "'event'", "'enum'", 
                                                            "'['", "']'", 
                                                            "'address'", 
                                                            "'.'", "'mapping'", 
                                                            "'=>'", "'memory'", 
                                                            "'storage'", 
                                                            "'calldata'", 
                                                            "'if'", "'else'", 
                                                            "'try'", "'catch'", 
                                                            "'while'", "'unchecked'", 
                                                            "'assembly'", 
                                                            "'do'", "'return'", 
                                                            "'throw'", "'emit'", 
                                                            "'revert'", 
                                                            "'var'", "'bool'", 
                                                            "'string'", 
                                                            "'byte'", "'++'", 
                                                            "'--'", "'new'", 
                                                            "':'", "'delete'", 
                                                            "'!'", "'**'", 
                                                            "'<<'", "'>>'", 
                                                            "'&&'", "'?'", 
                                                            "'|='", "'^='", 
                                                            "'&='", "'<<='", 
                                                            "'>>='", "'+='", 
                                                            "'-='", "'*='", 
                                                            "'/='", "'%='", 
                                                            "'let'", "':='", 
                                                            "'=:'", "'switch'", 
                                                            "'case'", "'default'", 
                                                            "'->'", "'callback'", 
                                                            "'override'", 
                                                            null, null, 
                                                            null, null, 
                                                            null, null, 
                                                            null, null, 
                                                            null, null, 
                                                            null, "'anonymous'", 
                                                            "'break'", "'constant'", 
                                                            "'transient'", 
                                                            "'immutable'", 
                                                            "'continue'", 
                                                            "'leave'", "'external'", 
                                                            "'indexed'", 
                                                            "'internal'", 
                                                            "'payable'", 
                                                            "'private'", 
                                                            "'public'", 
                                                            "'virtual'", 
                                                            "'pure'", "'type'", 
                                                            "'view'", "'global'", 
                                                            "'constructor'", 
                                                            "'fallback'", 
                                                            "'receive'" ];
	public static readonly symbolicNames: (string | null)[] = [ null, null, 
                                                             null, null, 
                                                             null, null, 
                                                             null, null, 
                                                             null, null, 
                                                             null, null, 
                                                             null, null, 
                                                             null, null, 
                                                             null, null, 
                                                             null, null, 
                                                             null, null, 
                                                             null, null, 
                                                             null, null, 
                                                             null, null, 
                                                             null, null, 
                                                             null, null, 
                                                             null, null, 
                                                             null, null, 
                                                             null, null, 
                                                             null, null, 
                                                             null, null, 
                                                             null, null, 
                                                             null, null, 
                                                             null, null, 
                                                             null, null, 
                                                             null, null, 
                                                             null, null, 
                                                             null, null, 
                                                             null, null, 
                                                             null, null, 
                                                             null, null, 
                                                             null, null, 
                                                             null, null, 
                                                             null, null, 
                                                             null, null, 
                                                             null, null, 
                                                             null, null, 
                                                             null, null, 
                                                             null, null, 
                                                             null, null, 
                                                             null, null, 
                                                             null, null, 
                                                             null, null, 
                                                             null, null, 
                                                             null, null, 
                                                             null, null, 
                                                             null, null, 
                                                             null, null, 
                                                             null, null, 
                                                             null, "Int", 
                                                             "Uint", "Byte", 
                                                             "Fixed", "Ufixed", 
                                                             "BooleanLiteral", 
                                                             "DecimalNumber", 
                                                             "HexNumber", 
                                                             "NumberUnit", 
                                                             "HexLiteralFragment", 
                                                             "ReservedKeyword", 
                                                             "AnonymousKeyword", 
                                                             "BreakKeyword", 
                                                             "ConstantKeyword", 
                                                             "TransientKeyword", 
                                                             "ImmutableKeyword", 
                                                             "ContinueKeyword", 
                                                             "LeaveKeyword", 
                                                             "ExternalKeyword", 
                                                             "IndexedKeyword", 
                                                             "InternalKeyword", 
                                                             "PayableKeyword", 
                                                             "PrivateKeyword", 
                                                             "PublicKeyword", 
                                                             "VirtualKeyword", 
                                                             "PureKeyword", 
                                                             "TypeKeyword", 
                                                             "ViewKeyword", 
                                                             "GlobalKeyword", 
                                                             "ConstructorKeyword", 
                                                             "FallbackKeyword", 
                                                             "ReceiveKeyword", 
                                                             "Identifier", 
                                                             "StringLiteralFragment", 
                                                             "VersionLiteral", 
                                                             "WS", "COMMENT", 
                                                             "LINE_COMMENT" ];
	// tslint:disable:no-trailing-whitespace
	public static readonly ruleNames: string[] = [
		"sourceUnit", "pragmaDirective", "pragmaName", "pragmaValue", "version", 
		"versionOperator", "versionConstraint", "importDeclaration", "importDirective", 
		"importPath", "contractDefinition", "inheritanceSpecifier", "customStorageLayout", 
		"contractPart", "stateVariableDeclaration", "fileLevelConstant", "customErrorDefinition", 
		"typeDefinition", "usingForDeclaration", "usingForObject", "usingForObjectDirective", 
		"userDefinableOperators", "structDefinition", "modifierDefinition", "modifierInvocation", 
		"functionDefinition", "functionDescriptor", "returnParameters", "modifierList", 
		"eventDefinition", "enumValue", "enumDefinition", "parameterList", "parameter", 
		"eventParameterList", "eventParameter", "functionTypeParameterList", "functionTypeParameter", 
		"variableDeclaration", "typeName", "userDefinedTypeName", "mappingKey", 
		"mapping", "mappingKeyName", "mappingValueName", "functionTypeName", "storageLocation", 
		"stateMutability", "block", "statement", "expressionStatement", "ifStatement", 
		"tryStatement", "catchClause", "whileStatement", "simpleStatement", "uncheckedStatement", 
		"forStatement", "inlineAssemblyStatement", "inlineAssemblyStatementFlag", 
		"doWhileStatement", "continueStatement", "breakStatement", "returnStatement", 
		"throwStatement", "emitStatement", "revertStatement", "variableDeclarationStatement", 
		"variableDeclarationList", "identifierList", "elementaryTypeName", "expression", 
		"primaryExpression", "expressionList", "nameValueList", "nameValue", "functionCallArguments", 
		"functionCall", "assemblyBlock", "assemblyItem", "assemblyExpression", 
		"assemblyMember", "assemblyCall", "assemblyLocalDefinition", "assemblyAssignment", 
		"assemblyIdentifierOrList", "assemblyIdentifierList", "assemblyStackAssignment", 
		"labelDefinition", "assemblySwitch", "assemblyCase", "assemblyFunctionDefinition", 
		"assemblyFunctionReturns", "assemblyFor", "assemblyIf", "assemblyLiteral", 
		"tupleExpression", "numberLiteral", "identifier", "hexLiteral", "overrideSpecifier", 
		"stringLiteral",
	];
	public get grammarFileName(): string { return "Solidity.g4"; }
	public get literalNames(): (string | null)[] { return SolidityParser.literalNames; }
	public get symbolicNames(): (string | null)[] { return SolidityParser.symbolicNames; }
	public get ruleNames(): string[] { return SolidityParser.ruleNames; }
	public get serializedATN(): number[] { return SolidityParser._serializedATN; }

	protected createFailedPredicateException(predicate?: string, message?: string): FailedPredicateException {
		return new FailedPredicateException(this, predicate, message);
	}

	constructor(input: TokenStream) {
		super(input);
		this._interp = new ParserATNSimulator(this, SolidityParser._ATN, SolidityParser.DecisionsToDFA, new PredictionContextCache());
	}
	// @RuleVersion(0)
	public sourceUnit(): SourceUnitContext {
		let localctx: SourceUnitContext = new SourceUnitContext(this, this._ctx, this.state);
		this.enterRule(localctx, 0, SolidityParser.RULE_sourceUnit);
		let _la: number;
		try {
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 217;
			this._errHandler.sync(this);
			_la = this._input.LA(1);
			while ((((_la) & ~0x1F) === 0 && ((1 << _la) & 507273218) !== 0) || ((((_la - 38)) & ~0x1F) === 0 && ((1 << (_la - 38)) & 2080392501) !== 0) || ((((_la - 97)) & ~0x1F) === 0 && ((1 << (_la - 97)) & 3498573949) !== 0) || ((((_la - 129)) & ~0x1F) === 0 && ((1 << (_la - 129)) & 7) !== 0)) {
				{
				this.state = 215;
				this._errHandler.sync(this);
				switch ( this._interp.adaptivePredict(this._input, 0, this._ctx) ) {
				case 1:
					{
					this.state = 204;
					this.pragmaDirective();
					}
					break;
				case 2:
					{
					this.state = 205;
					this.importDirective();
					}
					break;
				case 3:
					{
					this.state = 206;
					this.contractDefinition();
					}
					break;
				case 4:
					{
					this.state = 207;
					this.enumDefinition();
					}
					break;
				case 5:
					{
					this.state = 208;
					this.eventDefinition();
					}
					break;
				case 6:
					{
					this.state = 209;
					this.structDefinition();
					}
					break;
				case 7:
					{
					this.state = 210;
					this.functionDefinition();
					}
					break;
				case 8:
					{
					this.state = 211;
					this.fileLevelConstant();
					}
					break;
				case 9:
					{
					this.state = 212;
					this.customErrorDefinition();
					}
					break;
				case 10:
					{
					this.state = 213;
					this.typeDefinition();
					}
					break;
				case 11:
					{
					this.state = 214;
					this.usingForDeclaration();
					}
					break;
				}
				}
				this.state = 219;
				this._errHandler.sync(this);
				_la = this._input.LA(1);
			}
			this.state = 220;
			this.match(SolidityParser.EOF);
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public pragmaDirective(): PragmaDirectiveContext {
		let localctx: PragmaDirectiveContext = new PragmaDirectiveContext(this, this._ctx, this.state);
		this.enterRule(localctx, 2, SolidityParser.RULE_pragmaDirective);
		try {
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 222;
			this.match(SolidityParser.T__0);
			this.state = 223;
			this.pragmaName();
			this.state = 224;
			this.pragmaValue();
			this.state = 225;
			this.match(SolidityParser.T__1);
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public pragmaName(): PragmaNameContext {
		let localctx: PragmaNameContext = new PragmaNameContext(this, this._ctx, this.state);
		this.enterRule(localctx, 4, SolidityParser.RULE_pragmaName);
		try {
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 227;
			this.identifier();
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public pragmaValue(): PragmaValueContext {
		let localctx: PragmaValueContext = new PragmaValueContext(this, this._ctx, this.state);
		this.enterRule(localctx, 6, SolidityParser.RULE_pragmaValue);
		try {
			this.state = 232;
			this._errHandler.sync(this);
			switch ( this._interp.adaptivePredict(this._input, 2, this._ctx) ) {
			case 1:
				this.enterOuterAlt(localctx, 1);
				{
				this.state = 229;
				this.match(SolidityParser.T__2);
				}
				break;
			case 2:
				this.enterOuterAlt(localctx, 2);
				{
				this.state = 230;
				this.version();
				}
				break;
			case 3:
				this.enterOuterAlt(localctx, 3);
				{
				this.state = 231;
				this.expression(0);
				}
				break;
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public version(): VersionContext {
		let localctx: VersionContext = new VersionContext(this, this._ctx, this.state);
		this.enterRule(localctx, 8, SolidityParser.RULE_version);
		let _la: number;
		try {
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 234;
			this.versionConstraint();
			this.state = 241;
			this._errHandler.sync(this);
			_la = this._input.LA(1);
			while ((((_la) & ~0x1F) === 0 && ((1 << _la) & 4080) !== 0) || _la===105 || _la===133) {
				{
				{
				this.state = 236;
				this._errHandler.sync(this);
				_la = this._input.LA(1);
				if (_la===4) {
					{
					this.state = 235;
					this.match(SolidityParser.T__3);
					}
				}

				this.state = 238;
				this.versionConstraint();
				}
				}
				this.state = 243;
				this._errHandler.sync(this);
				_la = this._input.LA(1);
			}
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public versionOperator(): VersionOperatorContext {
		let localctx: VersionOperatorContext = new VersionOperatorContext(this, this._ctx, this.state);
		this.enterRule(localctx, 10, SolidityParser.RULE_versionOperator);
		let _la: number;
		try {
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 244;
			_la = this._input.LA(1);
			if(!((((_la) & ~0x1F) === 0 && ((1 << _la) & 4064) !== 0))) {
			this._errHandler.recoverInline(this);
			}
			else {
				this._errHandler.reportMatch(this);
			    this.consume();
			}
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public versionConstraint(): VersionConstraintContext {
		let localctx: VersionConstraintContext = new VersionConstraintContext(this, this._ctx, this.state);
		this.enterRule(localctx, 12, SolidityParser.RULE_versionConstraint);
		let _la: number;
		try {
			this.state = 254;
			this._errHandler.sync(this);
			switch ( this._interp.adaptivePredict(this._input, 7, this._ctx) ) {
			case 1:
				this.enterOuterAlt(localctx, 1);
				{
				this.state = 247;
				this._errHandler.sync(this);
				_la = this._input.LA(1);
				if ((((_la) & ~0x1F) === 0 && ((1 << _la) & 4064) !== 0)) {
					{
					this.state = 246;
					this.versionOperator();
					}
				}

				this.state = 249;
				this.match(SolidityParser.VersionLiteral);
				}
				break;
			case 2:
				this.enterOuterAlt(localctx, 2);
				{
				this.state = 251;
				this._errHandler.sync(this);
				_la = this._input.LA(1);
				if ((((_la) & ~0x1F) === 0 && ((1 << _la) & 4064) !== 0)) {
					{
					this.state = 250;
					this.versionOperator();
					}
				}

				this.state = 253;
				this.match(SolidityParser.DecimalNumber);
				}
				break;
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public importDeclaration(): ImportDeclarationContext {
		let localctx: ImportDeclarationContext = new ImportDeclarationContext(this, this._ctx, this.state);
		this.enterRule(localctx, 14, SolidityParser.RULE_importDeclaration);
		let _la: number;
		try {
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 256;
			this.identifier();
			this.state = 259;
			this._errHandler.sync(this);
			_la = this._input.LA(1);
			if (_la===12) {
				{
				this.state = 257;
				this.match(SolidityParser.T__11);
				this.state = 258;
				this.identifier();
				}
			}

			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public importDirective(): ImportDirectiveContext {
		let localctx: ImportDirectiveContext = new ImportDirectiveContext(this, this._ctx, this.state);
		this.enterRule(localctx, 16, SolidityParser.RULE_importDirective);
		let _la: number;
		try {
			this.state = 297;
			this._errHandler.sync(this);
			switch ( this._interp.adaptivePredict(this._input, 13, this._ctx) ) {
			case 1:
				this.enterOuterAlt(localctx, 1);
				{
				this.state = 261;
				this.match(SolidityParser.T__12);
				this.state = 262;
				this.importPath();
				this.state = 265;
				this._errHandler.sync(this);
				_la = this._input.LA(1);
				if (_la===12) {
					{
					this.state = 263;
					this.match(SolidityParser.T__11);
					this.state = 264;
					this.identifier();
					}
				}

				this.state = 267;
				this.match(SolidityParser.T__1);
				}
				break;
			case 2:
				this.enterOuterAlt(localctx, 2);
				{
				this.state = 269;
				this.match(SolidityParser.T__12);
				this.state = 272;
				this._errHandler.sync(this);
				switch (this._input.LA(1)) {
				case 3:
					{
					this.state = 270;
					this.match(SolidityParser.T__2);
					}
					break;
				case 14:
				case 25:
				case 26:
				case 27:
				case 46:
				case 52:
				case 64:
				case 97:
				case 116:
				case 120:
				case 127:
				case 128:
				case 130:
				case 131:
					{
					this.state = 271;
					this.identifier();
					}
					break;
				default:
					throw new NoViableAltException(this);
				}
				this.state = 276;
				this._errHandler.sync(this);
				_la = this._input.LA(1);
				if (_la===12) {
					{
					this.state = 274;
					this.match(SolidityParser.T__11);
					this.state = 275;
					this.identifier();
					}
				}

				this.state = 278;
				this.match(SolidityParser.T__13);
				this.state = 279;
				this.importPath();
				this.state = 280;
				this.match(SolidityParser.T__1);
				}
				break;
			case 3:
				this.enterOuterAlt(localctx, 3);
				{
				this.state = 282;
				this.match(SolidityParser.T__12);
				this.state = 283;
				this.match(SolidityParser.T__14);
				this.state = 284;
				this.importDeclaration();
				this.state = 289;
				this._errHandler.sync(this);
				_la = this._input.LA(1);
				while (_la===16) {
					{
					{
					this.state = 285;
					this.match(SolidityParser.T__15);
					this.state = 286;
					this.importDeclaration();
					}
					}
					this.state = 291;
					this._errHandler.sync(this);
					_la = this._input.LA(1);
				}
				this.state = 292;
				this.match(SolidityParser.T__16);
				this.state = 293;
				this.match(SolidityParser.T__13);
				this.state = 294;
				this.importPath();
				this.state = 295;
				this.match(SolidityParser.T__1);
				}
				break;
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public importPath(): ImportPathContext {
		let localctx: ImportPathContext = new ImportPathContext(this, this._ctx, this.state);
		this.enterRule(localctx, 18, SolidityParser.RULE_importPath);
		try {
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 299;
			this.match(SolidityParser.StringLiteralFragment);
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public contractDefinition(): ContractDefinitionContext {
		let localctx: ContractDefinitionContext = new ContractDefinitionContext(this, this._ctx, this.state);
		this.enterRule(localctx, 20, SolidityParser.RULE_contractDefinition);
		let _la: number;
		try {
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 302;
			this._errHandler.sync(this);
			_la = this._input.LA(1);
			if (_la===18) {
				{
				this.state = 301;
				this.match(SolidityParser.T__17);
				}
			}

			this.state = 304;
			_la = this._input.LA(1);
			if(!((((_la) & ~0x1F) === 0 && ((1 << _la) & 3670016) !== 0))) {
			this._errHandler.recoverInline(this);
			}
			else {
				this._errHandler.reportMatch(this);
			    this.consume();
			}
			this.state = 305;
			this.identifier();
			this.state = 307;
			this._errHandler.sync(this);
			switch ( this._interp.adaptivePredict(this._input, 15, this._ctx) ) {
			case 1:
				{
				this.state = 306;
				this.customStorageLayout();
				}
				break;
			}
			this.state = 318;
			this._errHandler.sync(this);
			_la = this._input.LA(1);
			if (_la===22) {
				{
				this.state = 309;
				this.match(SolidityParser.T__21);
				this.state = 310;
				this.inheritanceSpecifier();
				this.state = 315;
				this._errHandler.sync(this);
				_la = this._input.LA(1);
				while (_la===16) {
					{
					{
					this.state = 311;
					this.match(SolidityParser.T__15);
					this.state = 312;
					this.inheritanceSpecifier();
					}
					}
					this.state = 317;
					this._errHandler.sync(this);
					_la = this._input.LA(1);
				}
				}
			}

			this.state = 321;
			this._errHandler.sync(this);
			_la = this._input.LA(1);
			if (_la===25) {
				{
				this.state = 320;
				this.customStorageLayout();
				}
			}

			this.state = 323;
			this.match(SolidityParser.T__14);
			this.state = 327;
			this._errHandler.sync(this);
			_la = this._input.LA(1);
			while ((((_la) & ~0x1F) === 0 && ((1 << _la) & 503332864) !== 0) || ((((_la - 38)) & ~0x1F) === 0 && ((1 << (_la - 38)) & 2080392503) !== 0) || ((((_la - 97)) & ~0x1F) === 0 && ((1 << (_la - 97)) & 3498573949) !== 0) || ((((_la - 129)) & ~0x1F) === 0 && ((1 << (_la - 129)) & 7) !== 0)) {
				{
				{
				this.state = 324;
				this.contractPart();
				}
				}
				this.state = 329;
				this._errHandler.sync(this);
				_la = this._input.LA(1);
			}
			this.state = 330;
			this.match(SolidityParser.T__16);
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public inheritanceSpecifier(): InheritanceSpecifierContext {
		let localctx: InheritanceSpecifierContext = new InheritanceSpecifierContext(this, this._ctx, this.state);
		this.enterRule(localctx, 22, SolidityParser.RULE_inheritanceSpecifier);
		let _la: number;
		try {
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 332;
			this.userDefinedTypeName();
			this.state = 338;
			this._errHandler.sync(this);
			_la = this._input.LA(1);
			if (_la===23) {
				{
				this.state = 333;
				this.match(SolidityParser.T__22);
				this.state = 335;
				this._errHandler.sync(this);
				_la = this._input.LA(1);
				if (((((_la - 6)) & ~0x1F) === 0 && ((1 << (_la - 6)) & 205127937) !== 0) || ((((_la - 40)) & ~0x1F) === 0 && ((1 << (_la - 40)) & 4278194513) !== 0) || ((((_la - 73)) & ~0x1F) === 0 && ((1 << (_la - 73)) & 4244635651) !== 0) || ((((_la - 105)) & ~0x1F) === 0 && ((1 << (_la - 105)) & 248547339) !== 0)) {
					{
					this.state = 334;
					this.expressionList();
					}
				}

				this.state = 337;
				this.match(SolidityParser.T__23);
				}
			}

			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public customStorageLayout(): CustomStorageLayoutContext {
		let localctx: CustomStorageLayoutContext = new CustomStorageLayoutContext(this, this._ctx, this.state);
		this.enterRule(localctx, 24, SolidityParser.RULE_customStorageLayout);
		try {
			this.enterOuterAlt(localctx, 1);
			{
			{
			this.state = 340;
			this.match(SolidityParser.T__24);
			this.state = 341;
			this.match(SolidityParser.T__25);
			this.state = 342;
			this.expression(0);
			}
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public contractPart(): ContractPartContext {
		let localctx: ContractPartContext = new ContractPartContext(this, this._ctx, this.state);
		this.enterRule(localctx, 26, SolidityParser.RULE_contractPart);
		try {
			this.state = 353;
			this._errHandler.sync(this);
			switch ( this._interp.adaptivePredict(this._input, 22, this._ctx) ) {
			case 1:
				this.enterOuterAlt(localctx, 1);
				{
				this.state = 344;
				this.stateVariableDeclaration();
				}
				break;
			case 2:
				this.enterOuterAlt(localctx, 2);
				{
				this.state = 345;
				this.usingForDeclaration();
				}
				break;
			case 3:
				this.enterOuterAlt(localctx, 3);
				{
				this.state = 346;
				this.structDefinition();
				}
				break;
			case 4:
				this.enterOuterAlt(localctx, 4);
				{
				this.state = 347;
				this.modifierDefinition();
				}
				break;
			case 5:
				this.enterOuterAlt(localctx, 5);
				{
				this.state = 348;
				this.functionDefinition();
				}
				break;
			case 6:
				this.enterOuterAlt(localctx, 6);
				{
				this.state = 349;
				this.eventDefinition();
				}
				break;
			case 7:
				this.enterOuterAlt(localctx, 7);
				{
				this.state = 350;
				this.enumDefinition();
				}
				break;
			case 8:
				this.enterOuterAlt(localctx, 8);
				{
				this.state = 351;
				this.customErrorDefinition();
				}
				break;
			case 9:
				this.enterOuterAlt(localctx, 9);
				{
				this.state = 352;
				this.typeDefinition();
				}
				break;
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public stateVariableDeclaration(): StateVariableDeclarationContext {
		let localctx: StateVariableDeclarationContext = new StateVariableDeclarationContext(this, this._ctx, this.state);
		this.enterRule(localctx, 28, SolidityParser.RULE_stateVariableDeclaration);
		let _la: number;
		try {
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 355;
			this.typeName(0);
			this.state = 365;
			this._errHandler.sync(this);
			_la = this._input.LA(1);
			while (((((_la - 98)) & ~0x1F) === 0 && ((1 << (_la - 98)) & 27377665) !== 0)) {
				{
				this.state = 363;
				this._errHandler.sync(this);
				switch (this._input.LA(1)) {
				case 122:
					{
					this.state = 356;
					this.match(SolidityParser.PublicKeyword);
					}
					break;
				case 119:
					{
					this.state = 357;
					this.match(SolidityParser.InternalKeyword);
					}
					break;
				case 121:
					{
					this.state = 358;
					this.match(SolidityParser.PrivateKeyword);
					}
					break;
				case 112:
					{
					this.state = 359;
					this.match(SolidityParser.ConstantKeyword);
					}
					break;
				case 113:
					{
					this.state = 360;
					this.match(SolidityParser.TransientKeyword);
					}
					break;
				case 114:
					{
					this.state = 361;
					this.match(SolidityParser.ImmutableKeyword);
					}
					break;
				case 98:
					{
					this.state = 362;
					this.overrideSpecifier();
					}
					break;
				default:
					throw new NoViableAltException(this);
				}
				}
				this.state = 367;
				this._errHandler.sync(this);
				_la = this._input.LA(1);
			}
			this.state = 368;
			this.identifier();
			this.state = 371;
			this._errHandler.sync(this);
			_la = this._input.LA(1);
			if (_la===11) {
				{
				this.state = 369;
				this.match(SolidityParser.T__10);
				this.state = 370;
				this.expression(0);
				}
			}

			this.state = 373;
			this.match(SolidityParser.T__1);
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public fileLevelConstant(): FileLevelConstantContext {
		let localctx: FileLevelConstantContext = new FileLevelConstantContext(this, this._ctx, this.state);
		this.enterRule(localctx, 30, SolidityParser.RULE_fileLevelConstant);
		try {
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 375;
			this.typeName(0);
			this.state = 376;
			this.match(SolidityParser.ConstantKeyword);
			this.state = 377;
			this.identifier();
			this.state = 378;
			this.match(SolidityParser.T__10);
			this.state = 379;
			this.expression(0);
			this.state = 380;
			this.match(SolidityParser.T__1);
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public customErrorDefinition(): CustomErrorDefinitionContext {
		let localctx: CustomErrorDefinitionContext = new CustomErrorDefinitionContext(this, this._ctx, this.state);
		this.enterRule(localctx, 32, SolidityParser.RULE_customErrorDefinition);
		try {
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 382;
			this.match(SolidityParser.T__26);
			this.state = 383;
			this.identifier();
			this.state = 384;
			this.parameterList();
			this.state = 385;
			this.match(SolidityParser.T__1);
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public typeDefinition(): TypeDefinitionContext {
		let localctx: TypeDefinitionContext = new TypeDefinitionContext(this, this._ctx, this.state);
		this.enterRule(localctx, 34, SolidityParser.RULE_typeDefinition);
		try {
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 387;
			this.match(SolidityParser.TypeKeyword);
			this.state = 388;
			this.identifier();
			this.state = 389;
			this.match(SolidityParser.T__21);
			this.state = 390;
			this.elementaryTypeName();
			this.state = 391;
			this.match(SolidityParser.T__1);
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public usingForDeclaration(): UsingForDeclarationContext {
		let localctx: UsingForDeclarationContext = new UsingForDeclarationContext(this, this._ctx, this.state);
		this.enterRule(localctx, 36, SolidityParser.RULE_usingForDeclaration);
		let _la: number;
		try {
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 393;
			this.match(SolidityParser.T__27);
			this.state = 394;
			this.usingForObject();
			this.state = 395;
			this.match(SolidityParser.T__28);
			this.state = 398;
			this._errHandler.sync(this);
			switch (this._input.LA(1)) {
			case 3:
				{
				this.state = 396;
				this.match(SolidityParser.T__2);
				}
				break;
			case 14:
			case 25:
			case 26:
			case 27:
			case 40:
			case 46:
			case 48:
			case 52:
			case 64:
			case 65:
			case 66:
			case 67:
			case 68:
			case 97:
			case 99:
			case 100:
			case 101:
			case 102:
			case 103:
			case 116:
			case 120:
			case 127:
			case 128:
			case 130:
			case 131:
				{
				this.state = 397;
				this.typeName(0);
				}
				break;
			default:
				throw new NoViableAltException(this);
			}
			this.state = 401;
			this._errHandler.sync(this);
			_la = this._input.LA(1);
			if (_la===127) {
				{
				this.state = 400;
				this.match(SolidityParser.GlobalKeyword);
				}
			}

			this.state = 403;
			this.match(SolidityParser.T__1);
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public usingForObject(): UsingForObjectContext {
		let localctx: UsingForObjectContext = new UsingForObjectContext(this, this._ctx, this.state);
		this.enterRule(localctx, 38, SolidityParser.RULE_usingForObject);
		let _la: number;
		try {
			this.state = 417;
			this._errHandler.sync(this);
			switch (this._input.LA(1)) {
			case 14:
			case 25:
			case 26:
			case 27:
			case 46:
			case 52:
			case 64:
			case 97:
			case 116:
			case 120:
			case 127:
			case 128:
			case 130:
			case 131:
				this.enterOuterAlt(localctx, 1);
				{
				this.state = 405;
				this.userDefinedTypeName();
				}
				break;
			case 15:
				this.enterOuterAlt(localctx, 2);
				{
				this.state = 406;
				this.match(SolidityParser.T__14);
				this.state = 407;
				this.usingForObjectDirective();
				this.state = 412;
				this._errHandler.sync(this);
				_la = this._input.LA(1);
				while (_la===16) {
					{
					{
					this.state = 408;
					this.match(SolidityParser.T__15);
					this.state = 409;
					this.usingForObjectDirective();
					}
					}
					this.state = 414;
					this._errHandler.sync(this);
					_la = this._input.LA(1);
				}
				this.state = 415;
				this.match(SolidityParser.T__16);
				}
				break;
			default:
				throw new NoViableAltException(this);
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public usingForObjectDirective(): UsingForObjectDirectiveContext {
		let localctx: UsingForObjectDirectiveContext = new UsingForObjectDirectiveContext(this, this._ctx, this.state);
		this.enterRule(localctx, 40, SolidityParser.RULE_usingForObjectDirective);
		let _la: number;
		try {
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 419;
			this.userDefinedTypeName();
			this.state = 422;
			this._errHandler.sync(this);
			_la = this._input.LA(1);
			if (_la===12) {
				{
				this.state = 420;
				this.match(SolidityParser.T__11);
				this.state = 421;
				this.userDefinableOperators();
				}
			}

			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public userDefinableOperators(): UserDefinableOperatorsContext {
		let localctx: UserDefinableOperatorsContext = new UserDefinableOperatorsContext(this, this._ctx, this.state);
		this.enterRule(localctx, 42, SolidityParser.RULE_userDefinableOperators);
		let _la: number;
		try {
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 424;
			_la = this._input.LA(1);
			if(!((((_la) & ~0x1F) === 0 && ((1 << _la) & 3221227496) !== 0) || ((((_la - 32)) & ~0x1F) === 0 && ((1 << (_la - 32)) & 63) !== 0))) {
			this._errHandler.recoverInline(this);
			}
			else {
				this._errHandler.reportMatch(this);
			    this.consume();
			}
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public structDefinition(): StructDefinitionContext {
		let localctx: StructDefinitionContext = new StructDefinitionContext(this, this._ctx, this.state);
		this.enterRule(localctx, 44, SolidityParser.RULE_structDefinition);
		let _la: number;
		try {
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 426;
			this.match(SolidityParser.T__37);
			this.state = 427;
			this.identifier();
			this.state = 428;
			this.match(SolidityParser.T__14);
			this.state = 439;
			this._errHandler.sync(this);
			_la = this._input.LA(1);
			if ((((_la) & ~0x1F) === 0 && ((1 << _la) & 234897408) !== 0) || ((((_la - 40)) & ~0x1F) === 0 && ((1 << (_la - 40)) & 520098113) !== 0) || ((((_la - 97)) & ~0x1F) === 0 && ((1 << (_la - 97)) & 3230138493) !== 0) || _la===130 || _la===131) {
				{
				this.state = 429;
				this.variableDeclaration();
				this.state = 430;
				this.match(SolidityParser.T__1);
				this.state = 436;
				this._errHandler.sync(this);
				_la = this._input.LA(1);
				while ((((_la) & ~0x1F) === 0 && ((1 << _la) & 234897408) !== 0) || ((((_la - 40)) & ~0x1F) === 0 && ((1 << (_la - 40)) & 520098113) !== 0) || ((((_la - 97)) & ~0x1F) === 0 && ((1 << (_la - 97)) & 3230138493) !== 0) || _la===130 || _la===131) {
					{
					{
					this.state = 431;
					this.variableDeclaration();
					this.state = 432;
					this.match(SolidityParser.T__1);
					}
					}
					this.state = 438;
					this._errHandler.sync(this);
					_la = this._input.LA(1);
				}
				}
			}

			this.state = 441;
			this.match(SolidityParser.T__16);
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public modifierDefinition(): ModifierDefinitionContext {
		let localctx: ModifierDefinitionContext = new ModifierDefinitionContext(this, this._ctx, this.state);
		this.enterRule(localctx, 46, SolidityParser.RULE_modifierDefinition);
		let _la: number;
		try {
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 443;
			this.match(SolidityParser.T__38);
			this.state = 444;
			this.identifier();
			this.state = 446;
			this._errHandler.sync(this);
			_la = this._input.LA(1);
			if (_la===23) {
				{
				this.state = 445;
				this.parameterList();
				}
			}

			this.state = 452;
			this._errHandler.sync(this);
			_la = this._input.LA(1);
			while (_la===98 || _la===123) {
				{
				this.state = 450;
				this._errHandler.sync(this);
				switch (this._input.LA(1)) {
				case 123:
					{
					this.state = 448;
					this.match(SolidityParser.VirtualKeyword);
					}
					break;
				case 98:
					{
					this.state = 449;
					this.overrideSpecifier();
					}
					break;
				default:
					throw new NoViableAltException(this);
				}
				}
				this.state = 454;
				this._errHandler.sync(this);
				_la = this._input.LA(1);
			}
			this.state = 457;
			this._errHandler.sync(this);
			switch (this._input.LA(1)) {
			case 2:
				{
				this.state = 455;
				this.match(SolidityParser.T__1);
				}
				break;
			case 15:
				{
				this.state = 456;
				this.block();
				}
				break;
			default:
				throw new NoViableAltException(this);
			}
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public modifierInvocation(): ModifierInvocationContext {
		let localctx: ModifierInvocationContext = new ModifierInvocationContext(this, this._ctx, this.state);
		this.enterRule(localctx, 48, SolidityParser.RULE_modifierInvocation);
		let _la: number;
		try {
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 459;
			this.identifier();
			this.state = 465;
			this._errHandler.sync(this);
			_la = this._input.LA(1);
			if (_la===23) {
				{
				this.state = 460;
				this.match(SolidityParser.T__22);
				this.state = 462;
				this._errHandler.sync(this);
				_la = this._input.LA(1);
				if (((((_la - 6)) & ~0x1F) === 0 && ((1 << (_la - 6)) & 205127937) !== 0) || ((((_la - 40)) & ~0x1F) === 0 && ((1 << (_la - 40)) & 4278194513) !== 0) || ((((_la - 73)) & ~0x1F) === 0 && ((1 << (_la - 73)) & 4244635651) !== 0) || ((((_la - 105)) & ~0x1F) === 0 && ((1 << (_la - 105)) & 248547339) !== 0)) {
					{
					this.state = 461;
					this.expressionList();
					}
				}

				this.state = 464;
				this.match(SolidityParser.T__23);
				}
			}

			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public functionDefinition(): FunctionDefinitionContext {
		let localctx: FunctionDefinitionContext = new FunctionDefinitionContext(this, this._ctx, this.state);
		this.enterRule(localctx, 50, SolidityParser.RULE_functionDefinition);
		let _la: number;
		try {
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 467;
			this.functionDescriptor();
			this.state = 468;
			this.parameterList();
			this.state = 469;
			this.modifierList();
			this.state = 471;
			this._errHandler.sync(this);
			_la = this._input.LA(1);
			if (_la===41) {
				{
				this.state = 470;
				this.returnParameters();
				}
			}

			this.state = 475;
			this._errHandler.sync(this);
			switch (this._input.LA(1)) {
			case 2:
				{
				this.state = 473;
				this.match(SolidityParser.T__1);
				}
				break;
			case 15:
				{
				this.state = 474;
				this.block();
				}
				break;
			default:
				throw new NoViableAltException(this);
			}
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public functionDescriptor(): FunctionDescriptorContext {
		let localctx: FunctionDescriptorContext = new FunctionDescriptorContext(this, this._ctx, this.state);
		this.enterRule(localctx, 52, SolidityParser.RULE_functionDescriptor);
		let _la: number;
		try {
			this.state = 484;
			this._errHandler.sync(this);
			switch (this._input.LA(1)) {
			case 40:
				this.enterOuterAlt(localctx, 1);
				{
				this.state = 477;
				this.match(SolidityParser.T__39);
				this.state = 479;
				this._errHandler.sync(this);
				_la = this._input.LA(1);
				if ((((_la) & ~0x1F) === 0 && ((1 << _la) & 234897408) !== 0) || ((((_la - 46)) & ~0x1F) === 0 && ((1 << (_la - 46)) & 262209) !== 0) || ((((_la - 97)) & ~0x1F) === 0 && ((1 << (_la - 97)) & 3230138369) !== 0) || _la===130 || _la===131) {
					{
					this.state = 478;
					this.identifier();
					}
				}

				}
				break;
			case 128:
				this.enterOuterAlt(localctx, 2);
				{
				this.state = 481;
				this.match(SolidityParser.ConstructorKeyword);
				}
				break;
			case 129:
				this.enterOuterAlt(localctx, 3);
				{
				this.state = 482;
				this.match(SolidityParser.FallbackKeyword);
				}
				break;
			case 130:
				this.enterOuterAlt(localctx, 4);
				{
				this.state = 483;
				this.match(SolidityParser.ReceiveKeyword);
				}
				break;
			default:
				throw new NoViableAltException(this);
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public returnParameters(): ReturnParametersContext {
		let localctx: ReturnParametersContext = new ReturnParametersContext(this, this._ctx, this.state);
		this.enterRule(localctx, 54, SolidityParser.RULE_returnParameters);
		try {
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 486;
			this.match(SolidityParser.T__40);
			this.state = 487;
			this.parameterList();
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public modifierList(): ModifierListContext {
		let localctx: ModifierListContext = new ModifierListContext(this, this._ctx, this.state);
		this.enterRule(localctx, 56, SolidityParser.RULE_modifierList);
		let _la: number;
		try {
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 499;
			this._errHandler.sync(this);
			_la = this._input.LA(1);
			while ((((_la) & ~0x1F) === 0 && ((1 << _la) & 234897408) !== 0) || ((((_la - 46)) & ~0x1F) === 0 && ((1 << (_la - 46)) & 262209) !== 0) || ((((_la - 97)) & ~0x1F) === 0 && ((1 << (_la - 97)) & 4023943171) !== 0) || _la===130 || _la===131) {
				{
				this.state = 497;
				this._errHandler.sync(this);
				switch ( this._interp.adaptivePredict(this._input, 43, this._ctx) ) {
				case 1:
					{
					this.state = 489;
					this.match(SolidityParser.ExternalKeyword);
					}
					break;
				case 2:
					{
					this.state = 490;
					this.match(SolidityParser.PublicKeyword);
					}
					break;
				case 3:
					{
					this.state = 491;
					this.match(SolidityParser.InternalKeyword);
					}
					break;
				case 4:
					{
					this.state = 492;
					this.match(SolidityParser.PrivateKeyword);
					}
					break;
				case 5:
					{
					this.state = 493;
					this.match(SolidityParser.VirtualKeyword);
					}
					break;
				case 6:
					{
					this.state = 494;
					this.stateMutability();
					}
					break;
				case 7:
					{
					this.state = 495;
					this.modifierInvocation();
					}
					break;
				case 8:
					{
					this.state = 496;
					this.overrideSpecifier();
					}
					break;
				}
				}
				this.state = 501;
				this._errHandler.sync(this);
				_la = this._input.LA(1);
			}
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public eventDefinition(): EventDefinitionContext {
		let localctx: EventDefinitionContext = new EventDefinitionContext(this, this._ctx, this.state);
		this.enterRule(localctx, 58, SolidityParser.RULE_eventDefinition);
		let _la: number;
		try {
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 502;
			this.match(SolidityParser.T__41);
			this.state = 503;
			this.identifier();
			this.state = 504;
			this.eventParameterList();
			this.state = 506;
			this._errHandler.sync(this);
			_la = this._input.LA(1);
			if (_la===110) {
				{
				this.state = 505;
				this.match(SolidityParser.AnonymousKeyword);
				}
			}

			this.state = 508;
			this.match(SolidityParser.T__1);
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public enumValue(): EnumValueContext {
		let localctx: EnumValueContext = new EnumValueContext(this, this._ctx, this.state);
		this.enterRule(localctx, 60, SolidityParser.RULE_enumValue);
		try {
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 510;
			this.identifier();
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public enumDefinition(): EnumDefinitionContext {
		let localctx: EnumDefinitionContext = new EnumDefinitionContext(this, this._ctx, this.state);
		this.enterRule(localctx, 62, SolidityParser.RULE_enumDefinition);
		let _la: number;
		try {
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 512;
			this.match(SolidityParser.T__42);
			this.state = 513;
			this.identifier();
			this.state = 514;
			this.match(SolidityParser.T__14);
			this.state = 516;
			this._errHandler.sync(this);
			_la = this._input.LA(1);
			if ((((_la) & ~0x1F) === 0 && ((1 << _la) & 234897408) !== 0) || ((((_la - 46)) & ~0x1F) === 0 && ((1 << (_la - 46)) & 262209) !== 0) || ((((_la - 97)) & ~0x1F) === 0 && ((1 << (_la - 97)) & 3230138369) !== 0) || _la===130 || _la===131) {
				{
				this.state = 515;
				this.enumValue();
				}
			}

			this.state = 522;
			this._errHandler.sync(this);
			_la = this._input.LA(1);
			while (_la===16) {
				{
				{
				this.state = 518;
				this.match(SolidityParser.T__15);
				this.state = 519;
				this.enumValue();
				}
				}
				this.state = 524;
				this._errHandler.sync(this);
				_la = this._input.LA(1);
			}
			this.state = 525;
			this.match(SolidityParser.T__16);
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public parameterList(): ParameterListContext {
		let localctx: ParameterListContext = new ParameterListContext(this, this._ctx, this.state);
		this.enterRule(localctx, 64, SolidityParser.RULE_parameterList);
		let _la: number;
		try {
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 527;
			this.match(SolidityParser.T__22);
			this.state = 536;
			this._errHandler.sync(this);
			_la = this._input.LA(1);
			if ((((_la) & ~0x1F) === 0 && ((1 << _la) & 234897408) !== 0) || ((((_la - 40)) & ~0x1F) === 0 && ((1 << (_la - 40)) & 520098113) !== 0) || ((((_la - 97)) & ~0x1F) === 0 && ((1 << (_la - 97)) & 3230138493) !== 0) || _la===130 || _la===131) {
				{
				this.state = 528;
				this.parameter();
				this.state = 533;
				this._errHandler.sync(this);
				_la = this._input.LA(1);
				while (_la===16) {
					{
					{
					this.state = 529;
					this.match(SolidityParser.T__15);
					this.state = 530;
					this.parameter();
					}
					}
					this.state = 535;
					this._errHandler.sync(this);
					_la = this._input.LA(1);
				}
				}
			}

			this.state = 538;
			this.match(SolidityParser.T__23);
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public parameter(): ParameterContext {
		let localctx: ParameterContext = new ParameterContext(this, this._ctx, this.state);
		this.enterRule(localctx, 66, SolidityParser.RULE_parameter);
		let _la: number;
		try {
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 540;
			this.typeName(0);
			this.state = 542;
			this._errHandler.sync(this);
			switch ( this._interp.adaptivePredict(this._input, 50, this._ctx) ) {
			case 1:
				{
				this.state = 541;
				this.storageLocation();
				}
				break;
			}
			this.state = 545;
			this._errHandler.sync(this);
			_la = this._input.LA(1);
			if ((((_la) & ~0x1F) === 0 && ((1 << _la) & 234897408) !== 0) || ((((_la - 46)) & ~0x1F) === 0 && ((1 << (_la - 46)) & 262209) !== 0) || ((((_la - 97)) & ~0x1F) === 0 && ((1 << (_la - 97)) & 3230138369) !== 0) || _la===130 || _la===131) {
				{
				this.state = 544;
				this.identifier();
				}
			}

			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public eventParameterList(): EventParameterListContext {
		let localctx: EventParameterListContext = new EventParameterListContext(this, this._ctx, this.state);
		this.enterRule(localctx, 68, SolidityParser.RULE_eventParameterList);
		let _la: number;
		try {
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 547;
			this.match(SolidityParser.T__22);
			this.state = 556;
			this._errHandler.sync(this);
			_la = this._input.LA(1);
			if ((((_la) & ~0x1F) === 0 && ((1 << _la) & 234897408) !== 0) || ((((_la - 40)) & ~0x1F) === 0 && ((1 << (_la - 40)) & 520098113) !== 0) || ((((_la - 97)) & ~0x1F) === 0 && ((1 << (_la - 97)) & 3230138493) !== 0) || _la===130 || _la===131) {
				{
				this.state = 548;
				this.eventParameter();
				this.state = 553;
				this._errHandler.sync(this);
				_la = this._input.LA(1);
				while (_la===16) {
					{
					{
					this.state = 549;
					this.match(SolidityParser.T__15);
					this.state = 550;
					this.eventParameter();
					}
					}
					this.state = 555;
					this._errHandler.sync(this);
					_la = this._input.LA(1);
				}
				}
			}

			this.state = 558;
			this.match(SolidityParser.T__23);
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public eventParameter(): EventParameterContext {
		let localctx: EventParameterContext = new EventParameterContext(this, this._ctx, this.state);
		this.enterRule(localctx, 70, SolidityParser.RULE_eventParameter);
		let _la: number;
		try {
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 560;
			this.typeName(0);
			this.state = 562;
			this._errHandler.sync(this);
			_la = this._input.LA(1);
			if (_la===118) {
				{
				this.state = 561;
				this.match(SolidityParser.IndexedKeyword);
				}
			}

			this.state = 565;
			this._errHandler.sync(this);
			_la = this._input.LA(1);
			if ((((_la) & ~0x1F) === 0 && ((1 << _la) & 234897408) !== 0) || ((((_la - 46)) & ~0x1F) === 0 && ((1 << (_la - 46)) & 262209) !== 0) || ((((_la - 97)) & ~0x1F) === 0 && ((1 << (_la - 97)) & 3230138369) !== 0) || _la===130 || _la===131) {
				{
				this.state = 564;
				this.identifier();
				}
			}

			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public functionTypeParameterList(): FunctionTypeParameterListContext {
		let localctx: FunctionTypeParameterListContext = new FunctionTypeParameterListContext(this, this._ctx, this.state);
		this.enterRule(localctx, 72, SolidityParser.RULE_functionTypeParameterList);
		let _la: number;
		try {
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 567;
			this.match(SolidityParser.T__22);
			this.state = 576;
			this._errHandler.sync(this);
			_la = this._input.LA(1);
			if ((((_la) & ~0x1F) === 0 && ((1 << _la) & 234897408) !== 0) || ((((_la - 40)) & ~0x1F) === 0 && ((1 << (_la - 40)) & 520098113) !== 0) || ((((_la - 97)) & ~0x1F) === 0 && ((1 << (_la - 97)) & 3230138493) !== 0) || _la===130 || _la===131) {
				{
				this.state = 568;
				this.functionTypeParameter();
				this.state = 573;
				this._errHandler.sync(this);
				_la = this._input.LA(1);
				while (_la===16) {
					{
					{
					this.state = 569;
					this.match(SolidityParser.T__15);
					this.state = 570;
					this.functionTypeParameter();
					}
					}
					this.state = 575;
					this._errHandler.sync(this);
					_la = this._input.LA(1);
				}
				}
			}

			this.state = 578;
			this.match(SolidityParser.T__23);
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public functionTypeParameter(): FunctionTypeParameterContext {
		let localctx: FunctionTypeParameterContext = new FunctionTypeParameterContext(this, this._ctx, this.state);
		this.enterRule(localctx, 74, SolidityParser.RULE_functionTypeParameter);
		let _la: number;
		try {
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 580;
			this.typeName(0);
			this.state = 582;
			this._errHandler.sync(this);
			_la = this._input.LA(1);
			if (((((_la - 50)) & ~0x1F) === 0 && ((1 << (_la - 50)) & 7) !== 0)) {
				{
				this.state = 581;
				this.storageLocation();
				}
			}

			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public variableDeclaration(): VariableDeclarationContext {
		let localctx: VariableDeclarationContext = new VariableDeclarationContext(this, this._ctx, this.state);
		this.enterRule(localctx, 76, SolidityParser.RULE_variableDeclaration);
		try {
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 584;
			this.typeName(0);
			this.state = 586;
			this._errHandler.sync(this);
			switch ( this._interp.adaptivePredict(this._input, 59, this._ctx) ) {
			case 1:
				{
				this.state = 585;
				this.storageLocation();
				}
				break;
			}
			this.state = 588;
			this.identifier();
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}

	public typeName(): TypeNameContext;
	public typeName(_p: number): TypeNameContext;
	// @RuleVersion(0)
	public typeName(_p?: number): TypeNameContext {
		if (_p === undefined) {
			_p = 0;
		}

		let _parentctx: ParserRuleContext = this._ctx;
		let _parentState: number = this.state;
		let localctx: TypeNameContext = new TypeNameContext(this, this._ctx, _parentState);
		let _prevctx: TypeNameContext = localctx;
		let _startState: number = 78;
		this.enterRecursionRule(localctx, 78, SolidityParser.RULE_typeName, _p);
		let _la: number;
		try {
			let _alt: number;
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 597;
			this._errHandler.sync(this);
			switch ( this._interp.adaptivePredict(this._input, 60, this._ctx) ) {
			case 1:
				{
				this.state = 591;
				this.elementaryTypeName();
				}
				break;
			case 2:
				{
				this.state = 592;
				this.userDefinedTypeName();
				}
				break;
			case 3:
				{
				this.state = 593;
				this.mapping();
				}
				break;
			case 4:
				{
				this.state = 594;
				this.functionTypeName();
				}
				break;
			case 5:
				{
				this.state = 595;
				this.match(SolidityParser.T__45);
				this.state = 596;
				this.match(SolidityParser.PayableKeyword);
				}
				break;
			}
			this._ctx.stop = this._input.LT(-1);
			this.state = 607;
			this._errHandler.sync(this);
			_alt = this._interp.adaptivePredict(this._input, 62, this._ctx);
			while (_alt !== 2 && _alt !== ATN.INVALID_ALT_NUMBER) {
				if (_alt === 1) {
					if (this._parseListeners != null) {
						this.triggerExitRuleEvent();
					}
					_prevctx = localctx;
					{
					{
					localctx = new TypeNameContext(this, _parentctx, _parentState);
					this.pushNewRecursionContext(localctx, _startState, SolidityParser.RULE_typeName);
					this.state = 599;
					if (!(this.precpred(this._ctx, 3))) {
						throw this.createFailedPredicateException("this.precpred(this._ctx, 3)");
					}
					this.state = 600;
					this.match(SolidityParser.T__43);
					this.state = 602;
					this._errHandler.sync(this);
					_la = this._input.LA(1);
					if (((((_la - 6)) & ~0x1F) === 0 && ((1 << (_la - 6)) & 205127937) !== 0) || ((((_la - 40)) & ~0x1F) === 0 && ((1 << (_la - 40)) & 4278194513) !== 0) || ((((_la - 73)) & ~0x1F) === 0 && ((1 << (_la - 73)) & 4244635651) !== 0) || ((((_la - 105)) & ~0x1F) === 0 && ((1 << (_la - 105)) & 248547339) !== 0)) {
						{
						this.state = 601;
						this.expression(0);
						}
					}

					this.state = 604;
					this.match(SolidityParser.T__44);
					}
					}
				}
				this.state = 609;
				this._errHandler.sync(this);
				_alt = this._interp.adaptivePredict(this._input, 62, this._ctx);
			}
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.unrollRecursionContexts(_parentctx);
		}
		return localctx;
	}
	// @RuleVersion(0)
	public userDefinedTypeName(): UserDefinedTypeNameContext {
		let localctx: UserDefinedTypeNameContext = new UserDefinedTypeNameContext(this, this._ctx, this.state);
		this.enterRule(localctx, 80, SolidityParser.RULE_userDefinedTypeName);
		try {
			let _alt: number;
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 610;
			this.identifier();
			this.state = 615;
			this._errHandler.sync(this);
			_alt = this._interp.adaptivePredict(this._input, 63, this._ctx);
			while (_alt !== 2 && _alt !== ATN.INVALID_ALT_NUMBER) {
				if (_alt === 1) {
					{
					{
					this.state = 611;
					this.match(SolidityParser.T__46);
					this.state = 612;
					this.identifier();
					}
					}
				}
				this.state = 617;
				this._errHandler.sync(this);
				_alt = this._interp.adaptivePredict(this._input, 63, this._ctx);
			}
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public mappingKey(): MappingKeyContext {
		let localctx: MappingKeyContext = new MappingKeyContext(this, this._ctx, this.state);
		this.enterRule(localctx, 82, SolidityParser.RULE_mappingKey);
		try {
			this.state = 620;
			this._errHandler.sync(this);
			switch ( this._interp.adaptivePredict(this._input, 64, this._ctx) ) {
			case 1:
				this.enterOuterAlt(localctx, 1);
				{
				this.state = 618;
				this.elementaryTypeName();
				}
				break;
			case 2:
				this.enterOuterAlt(localctx, 2);
				{
				this.state = 619;
				this.userDefinedTypeName();
				}
				break;
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public mapping(): MappingContext {
		let localctx: MappingContext = new MappingContext(this, this._ctx, this.state);
		this.enterRule(localctx, 84, SolidityParser.RULE_mapping);
		let _la: number;
		try {
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 622;
			this.match(SolidityParser.T__47);
			this.state = 623;
			this.match(SolidityParser.T__22);
			this.state = 624;
			this.mappingKey();
			this.state = 626;
			this._errHandler.sync(this);
			_la = this._input.LA(1);
			if ((((_la) & ~0x1F) === 0 && ((1 << _la) & 234897408) !== 0) || ((((_la - 46)) & ~0x1F) === 0 && ((1 << (_la - 46)) & 262209) !== 0) || ((((_la - 97)) & ~0x1F) === 0 && ((1 << (_la - 97)) & 3230138369) !== 0) || _la===130 || _la===131) {
				{
				this.state = 625;
				this.mappingKeyName();
				}
			}

			this.state = 628;
			this.match(SolidityParser.T__48);
			this.state = 629;
			this.typeName(0);
			this.state = 631;
			this._errHandler.sync(this);
			_la = this._input.LA(1);
			if ((((_la) & ~0x1F) === 0 && ((1 << _la) & 234897408) !== 0) || ((((_la - 46)) & ~0x1F) === 0 && ((1 << (_la - 46)) & 262209) !== 0) || ((((_la - 97)) & ~0x1F) === 0 && ((1 << (_la - 97)) & 3230138369) !== 0) || _la===130 || _la===131) {
				{
				this.state = 630;
				this.mappingValueName();
				}
			}

			this.state = 633;
			this.match(SolidityParser.T__23);
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public mappingKeyName(): MappingKeyNameContext {
		let localctx: MappingKeyNameContext = new MappingKeyNameContext(this, this._ctx, this.state);
		this.enterRule(localctx, 86, SolidityParser.RULE_mappingKeyName);
		try {
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 635;
			this.identifier();
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public mappingValueName(): MappingValueNameContext {
		let localctx: MappingValueNameContext = new MappingValueNameContext(this, this._ctx, this.state);
		this.enterRule(localctx, 88, SolidityParser.RULE_mappingValueName);
		try {
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 637;
			this.identifier();
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public functionTypeName(): FunctionTypeNameContext {
		let localctx: FunctionTypeNameContext = new FunctionTypeNameContext(this, this._ctx, this.state);
		this.enterRule(localctx, 90, SolidityParser.RULE_functionTypeName);
		try {
			let _alt: number;
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 639;
			this.match(SolidityParser.T__39);
			this.state = 640;
			this.functionTypeParameterList();
			this.state = 646;
			this._errHandler.sync(this);
			_alt = this._interp.adaptivePredict(this._input, 68, this._ctx);
			while (_alt !== 2 && _alt !== ATN.INVALID_ALT_NUMBER) {
				if (_alt === 1) {
					{
					this.state = 644;
					this._errHandler.sync(this);
					switch (this._input.LA(1)) {
					case 119:
						{
						this.state = 641;
						this.match(SolidityParser.InternalKeyword);
						}
						break;
					case 117:
						{
						this.state = 642;
						this.match(SolidityParser.ExternalKeyword);
						}
						break;
					case 112:
					case 120:
					case 124:
					case 126:
						{
						this.state = 643;
						this.stateMutability();
						}
						break;
					default:
						throw new NoViableAltException(this);
					}
					}
				}
				this.state = 648;
				this._errHandler.sync(this);
				_alt = this._interp.adaptivePredict(this._input, 68, this._ctx);
			}
			this.state = 651;
			this._errHandler.sync(this);
			switch ( this._interp.adaptivePredict(this._input, 69, this._ctx) ) {
			case 1:
				{
				this.state = 649;
				this.match(SolidityParser.T__40);
				this.state = 650;
				this.functionTypeParameterList();
				}
				break;
			}
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public storageLocation(): StorageLocationContext {
		let localctx: StorageLocationContext = new StorageLocationContext(this, this._ctx, this.state);
		this.enterRule(localctx, 92, SolidityParser.RULE_storageLocation);
		let _la: number;
		try {
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 653;
			_la = this._input.LA(1);
			if(!(((((_la - 50)) & ~0x1F) === 0 && ((1 << (_la - 50)) & 7) !== 0))) {
			this._errHandler.recoverInline(this);
			}
			else {
				this._errHandler.reportMatch(this);
			    this.consume();
			}
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public stateMutability(): StateMutabilityContext {
		let localctx: StateMutabilityContext = new StateMutabilityContext(this, this._ctx, this.state);
		this.enterRule(localctx, 94, SolidityParser.RULE_stateMutability);
		let _la: number;
		try {
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 655;
			_la = this._input.LA(1);
			if(!(((((_la - 112)) & ~0x1F) === 0 && ((1 << (_la - 112)) & 20737) !== 0))) {
			this._errHandler.recoverInline(this);
			}
			else {
				this._errHandler.reportMatch(this);
			    this.consume();
			}
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public block(): BlockContext {
		let localctx: BlockContext = new BlockContext(this, this._ctx, this.state);
		this.enterRule(localctx, 96, SolidityParser.RULE_block);
		let _la: number;
		try {
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 657;
			this.match(SolidityParser.T__14);
			this.state = 661;
			this._errHandler.sync(this);
			_la = this._input.LA(1);
			while (((((_la - 6)) & ~0x1F) === 0 && ((1 << (_la - 6)) & 213517057) !== 0) || ((((_la - 40)) & ~0x1F) === 0 && ((1 << (_la - 40)) & 4294881617) !== 0) || ((((_la - 73)) & ~0x1F) === 0 && ((1 << (_la - 73)) & 4244635651) !== 0) || ((((_la - 105)) & ~0x1F) === 0 && ((1 << (_la - 105)) & 248548427) !== 0)) {
				{
				{
				this.state = 658;
				this.statement();
				}
				}
				this.state = 663;
				this._errHandler.sync(this);
				_la = this._input.LA(1);
			}
			this.state = 664;
			this.match(SolidityParser.T__16);
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public statement(): StatementContext {
		let localctx: StatementContext = new StatementContext(this, this._ctx, this.state);
		this.enterRule(localctx, 98, SolidityParser.RULE_statement);
		try {
			this.state = 681;
			this._errHandler.sync(this);
			switch ( this._interp.adaptivePredict(this._input, 71, this._ctx) ) {
			case 1:
				this.enterOuterAlt(localctx, 1);
				{
				this.state = 666;
				this.ifStatement();
				}
				break;
			case 2:
				this.enterOuterAlt(localctx, 2);
				{
				this.state = 667;
				this.tryStatement();
				}
				break;
			case 3:
				this.enterOuterAlt(localctx, 3);
				{
				this.state = 668;
				this.whileStatement();
				}
				break;
			case 4:
				this.enterOuterAlt(localctx, 4);
				{
				this.state = 669;
				this.forStatement();
				}
				break;
			case 5:
				this.enterOuterAlt(localctx, 5);
				{
				this.state = 670;
				this.block();
				}
				break;
			case 6:
				this.enterOuterAlt(localctx, 6);
				{
				this.state = 671;
				this.inlineAssemblyStatement();
				}
				break;
			case 7:
				this.enterOuterAlt(localctx, 7);
				{
				this.state = 672;
				this.doWhileStatement();
				}
				break;
			case 8:
				this.enterOuterAlt(localctx, 8);
				{
				this.state = 673;
				this.continueStatement();
				}
				break;
			case 9:
				this.enterOuterAlt(localctx, 9);
				{
				this.state = 674;
				this.breakStatement();
				}
				break;
			case 10:
				this.enterOuterAlt(localctx, 10);
				{
				this.state = 675;
				this.returnStatement();
				}
				break;
			case 11:
				this.enterOuterAlt(localctx, 11);
				{
				this.state = 676;
				this.throwStatement();
				}
				break;
			case 12:
				this.enterOuterAlt(localctx, 12);
				{
				this.state = 677;
				this.emitStatement();
				}
				break;
			case 13:
				this.enterOuterAlt(localctx, 13);
				{
				this.state = 678;
				this.simpleStatement();
				}
				break;
			case 14:
				this.enterOuterAlt(localctx, 14);
				{
				this.state = 679;
				this.uncheckedStatement();
				}
				break;
			case 15:
				this.enterOuterAlt(localctx, 15);
				{
				this.state = 680;
				this.revertStatement();
				}
				break;
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public expressionStatement(): ExpressionStatementContext {
		let localctx: ExpressionStatementContext = new ExpressionStatementContext(this, this._ctx, this.state);
		this.enterRule(localctx, 100, SolidityParser.RULE_expressionStatement);
		try {
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 683;
			this.expression(0);
			this.state = 684;
			this.match(SolidityParser.T__1);
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public ifStatement(): IfStatementContext {
		let localctx: IfStatementContext = new IfStatementContext(this, this._ctx, this.state);
		this.enterRule(localctx, 102, SolidityParser.RULE_ifStatement);
		try {
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 686;
			this.match(SolidityParser.T__52);
			this.state = 687;
			this.match(SolidityParser.T__22);
			this.state = 688;
			this.expression(0);
			this.state = 689;
			this.match(SolidityParser.T__23);
			this.state = 690;
			this.statement();
			this.state = 693;
			this._errHandler.sync(this);
			switch ( this._interp.adaptivePredict(this._input, 72, this._ctx) ) {
			case 1:
				{
				this.state = 691;
				this.match(SolidityParser.T__53);
				this.state = 692;
				this.statement();
				}
				break;
			}
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public tryStatement(): TryStatementContext {
		let localctx: TryStatementContext = new TryStatementContext(this, this._ctx, this.state);
		this.enterRule(localctx, 104, SolidityParser.RULE_tryStatement);
		let _la: number;
		try {
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 695;
			this.match(SolidityParser.T__54);
			this.state = 696;
			this.expression(0);
			this.state = 698;
			this._errHandler.sync(this);
			_la = this._input.LA(1);
			if (_la===41) {
				{
				this.state = 697;
				this.returnParameters();
				}
			}

			this.state = 700;
			this.block();
			this.state = 702;
			this._errHandler.sync(this);
			_la = this._input.LA(1);
			do {
				{
				{
				this.state = 701;
				this.catchClause();
				}
				}
				this.state = 704;
				this._errHandler.sync(this);
				_la = this._input.LA(1);
			} while (_la===56);
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public catchClause(): CatchClauseContext {
		let localctx: CatchClauseContext = new CatchClauseContext(this, this._ctx, this.state);
		this.enterRule(localctx, 106, SolidityParser.RULE_catchClause);
		let _la: number;
		try {
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 706;
			this.match(SolidityParser.T__55);
			this.state = 711;
			this._errHandler.sync(this);
			_la = this._input.LA(1);
			if ((((_la) & ~0x1F) === 0 && ((1 << _la) & 243286016) !== 0) || ((((_la - 46)) & ~0x1F) === 0 && ((1 << (_la - 46)) & 262209) !== 0) || ((((_la - 97)) & ~0x1F) === 0 && ((1 << (_la - 97)) & 3230138369) !== 0) || _la===130 || _la===131) {
				{
				this.state = 708;
				this._errHandler.sync(this);
				_la = this._input.LA(1);
				if ((((_la) & ~0x1F) === 0 && ((1 << _la) & 234897408) !== 0) || ((((_la - 46)) & ~0x1F) === 0 && ((1 << (_la - 46)) & 262209) !== 0) || ((((_la - 97)) & ~0x1F) === 0 && ((1 << (_la - 97)) & 3230138369) !== 0) || _la===130 || _la===131) {
					{
					this.state = 707;
					this.identifier();
					}
				}

				this.state = 710;
				this.parameterList();
				}
			}

			this.state = 713;
			this.block();
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public whileStatement(): WhileStatementContext {
		let localctx: WhileStatementContext = new WhileStatementContext(this, this._ctx, this.state);
		this.enterRule(localctx, 108, SolidityParser.RULE_whileStatement);
		try {
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 715;
			this.match(SolidityParser.T__56);
			this.state = 716;
			this.match(SolidityParser.T__22);
			this.state = 717;
			this.expression(0);
			this.state = 718;
			this.match(SolidityParser.T__23);
			this.state = 719;
			this.statement();
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public simpleStatement(): SimpleStatementContext {
		let localctx: SimpleStatementContext = new SimpleStatementContext(this, this._ctx, this.state);
		this.enterRule(localctx, 110, SolidityParser.RULE_simpleStatement);
		try {
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 723;
			this._errHandler.sync(this);
			switch ( this._interp.adaptivePredict(this._input, 77, this._ctx) ) {
			case 1:
				{
				this.state = 721;
				this.variableDeclarationStatement();
				}
				break;
			case 2:
				{
				this.state = 722;
				this.expressionStatement();
				}
				break;
			}
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public uncheckedStatement(): UncheckedStatementContext {
		let localctx: UncheckedStatementContext = new UncheckedStatementContext(this, this._ctx, this.state);
		this.enterRule(localctx, 112, SolidityParser.RULE_uncheckedStatement);
		try {
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 725;
			this.match(SolidityParser.T__57);
			this.state = 726;
			this.block();
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public forStatement(): ForStatementContext {
		let localctx: ForStatementContext = new ForStatementContext(this, this._ctx, this.state);
		this.enterRule(localctx, 114, SolidityParser.RULE_forStatement);
		let _la: number;
		try {
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 728;
			this.match(SolidityParser.T__28);
			this.state = 729;
			this.match(SolidityParser.T__22);
			this.state = 732;
			this._errHandler.sync(this);
			switch (this._input.LA(1)) {
			case 6:
			case 14:
			case 23:
			case 25:
			case 26:
			case 27:
			case 32:
			case 33:
			case 40:
			case 44:
			case 46:
			case 48:
			case 52:
			case 64:
			case 65:
			case 66:
			case 67:
			case 68:
			case 69:
			case 70:
			case 71:
			case 73:
			case 74:
			case 97:
			case 99:
			case 100:
			case 101:
			case 102:
			case 103:
			case 104:
			case 105:
			case 106:
			case 108:
			case 116:
			case 120:
			case 125:
			case 127:
			case 128:
			case 130:
			case 131:
			case 132:
				{
				this.state = 730;
				this.simpleStatement();
				}
				break;
			case 2:
				{
				this.state = 731;
				this.match(SolidityParser.T__1);
				}
				break;
			default:
				throw new NoViableAltException(this);
			}
			this.state = 736;
			this._errHandler.sync(this);
			switch (this._input.LA(1)) {
			case 6:
			case 14:
			case 23:
			case 25:
			case 26:
			case 27:
			case 32:
			case 33:
			case 40:
			case 44:
			case 46:
			case 48:
			case 52:
			case 64:
			case 65:
			case 66:
			case 67:
			case 68:
			case 69:
			case 70:
			case 71:
			case 73:
			case 74:
			case 97:
			case 99:
			case 100:
			case 101:
			case 102:
			case 103:
			case 104:
			case 105:
			case 106:
			case 108:
			case 116:
			case 120:
			case 125:
			case 127:
			case 128:
			case 130:
			case 131:
			case 132:
				{
				this.state = 734;
				this.expressionStatement();
				}
				break;
			case 2:
				{
				this.state = 735;
				this.match(SolidityParser.T__1);
				}
				break;
			default:
				throw new NoViableAltException(this);
			}
			this.state = 739;
			this._errHandler.sync(this);
			_la = this._input.LA(1);
			if (((((_la - 6)) & ~0x1F) === 0 && ((1 << (_la - 6)) & 205127937) !== 0) || ((((_la - 40)) & ~0x1F) === 0 && ((1 << (_la - 40)) & 4278194513) !== 0) || ((((_la - 73)) & ~0x1F) === 0 && ((1 << (_la - 73)) & 4244635651) !== 0) || ((((_la - 105)) & ~0x1F) === 0 && ((1 << (_la - 105)) & 248547339) !== 0)) {
				{
				this.state = 738;
				this.expression(0);
				}
			}

			this.state = 741;
			this.match(SolidityParser.T__23);
			this.state = 742;
			this.statement();
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public inlineAssemblyStatement(): InlineAssemblyStatementContext {
		let localctx: InlineAssemblyStatementContext = new InlineAssemblyStatementContext(this, this._ctx, this.state);
		this.enterRule(localctx, 116, SolidityParser.RULE_inlineAssemblyStatement);
		let _la: number;
		try {
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 744;
			this.match(SolidityParser.T__58);
			this.state = 746;
			this._errHandler.sync(this);
			_la = this._input.LA(1);
			if (_la===132) {
				{
				this.state = 745;
				this.match(SolidityParser.StringLiteralFragment);
				}
			}

			this.state = 752;
			this._errHandler.sync(this);
			_la = this._input.LA(1);
			if (_la===23) {
				{
				this.state = 748;
				this.match(SolidityParser.T__22);
				this.state = 749;
				this.inlineAssemblyStatementFlag();
				this.state = 750;
				this.match(SolidityParser.T__23);
				}
			}

			this.state = 754;
			this.assemblyBlock();
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public inlineAssemblyStatementFlag(): InlineAssemblyStatementFlagContext {
		let localctx: InlineAssemblyStatementFlagContext = new InlineAssemblyStatementFlagContext(this, this._ctx, this.state);
		this.enterRule(localctx, 118, SolidityParser.RULE_inlineAssemblyStatementFlag);
		try {
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 756;
			this.stringLiteral();
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public doWhileStatement(): DoWhileStatementContext {
		let localctx: DoWhileStatementContext = new DoWhileStatementContext(this, this._ctx, this.state);
		this.enterRule(localctx, 120, SolidityParser.RULE_doWhileStatement);
		try {
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 758;
			this.match(SolidityParser.T__59);
			this.state = 759;
			this.statement();
			this.state = 760;
			this.match(SolidityParser.T__56);
			this.state = 761;
			this.match(SolidityParser.T__22);
			this.state = 762;
			this.expression(0);
			this.state = 763;
			this.match(SolidityParser.T__23);
			this.state = 764;
			this.match(SolidityParser.T__1);
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public continueStatement(): ContinueStatementContext {
		let localctx: ContinueStatementContext = new ContinueStatementContext(this, this._ctx, this.state);
		this.enterRule(localctx, 122, SolidityParser.RULE_continueStatement);
		try {
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 766;
			this.match(SolidityParser.ContinueKeyword);
			this.state = 767;
			this.match(SolidityParser.T__1);
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public breakStatement(): BreakStatementContext {
		let localctx: BreakStatementContext = new BreakStatementContext(this, this._ctx, this.state);
		this.enterRule(localctx, 124, SolidityParser.RULE_breakStatement);
		try {
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 769;
			this.match(SolidityParser.BreakKeyword);
			this.state = 770;
			this.match(SolidityParser.T__1);
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public returnStatement(): ReturnStatementContext {
		let localctx: ReturnStatementContext = new ReturnStatementContext(this, this._ctx, this.state);
		this.enterRule(localctx, 126, SolidityParser.RULE_returnStatement);
		let _la: number;
		try {
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 772;
			this.match(SolidityParser.T__60);
			this.state = 774;
			this._errHandler.sync(this);
			_la = this._input.LA(1);
			if (((((_la - 6)) & ~0x1F) === 0 && ((1 << (_la - 6)) & 205127937) !== 0) || ((((_la - 40)) & ~0x1F) === 0 && ((1 << (_la - 40)) & 4278194513) !== 0) || ((((_la - 73)) & ~0x1F) === 0 && ((1 << (_la - 73)) & 4244635651) !== 0) || ((((_la - 105)) & ~0x1F) === 0 && ((1 << (_la - 105)) & 248547339) !== 0)) {
				{
				this.state = 773;
				this.expression(0);
				}
			}

			this.state = 776;
			this.match(SolidityParser.T__1);
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public throwStatement(): ThrowStatementContext {
		let localctx: ThrowStatementContext = new ThrowStatementContext(this, this._ctx, this.state);
		this.enterRule(localctx, 128, SolidityParser.RULE_throwStatement);
		try {
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 778;
			this.match(SolidityParser.T__61);
			this.state = 779;
			this.match(SolidityParser.T__1);
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public emitStatement(): EmitStatementContext {
		let localctx: EmitStatementContext = new EmitStatementContext(this, this._ctx, this.state);
		this.enterRule(localctx, 130, SolidityParser.RULE_emitStatement);
		try {
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 781;
			this.match(SolidityParser.T__62);
			this.state = 782;
			this.functionCall();
			this.state = 783;
			this.match(SolidityParser.T__1);
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public revertStatement(): RevertStatementContext {
		let localctx: RevertStatementContext = new RevertStatementContext(this, this._ctx, this.state);
		this.enterRule(localctx, 132, SolidityParser.RULE_revertStatement);
		try {
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 785;
			this.match(SolidityParser.T__63);
			this.state = 786;
			this.functionCall();
			this.state = 787;
			this.match(SolidityParser.T__1);
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public variableDeclarationStatement(): VariableDeclarationStatementContext {
		let localctx: VariableDeclarationStatementContext = new VariableDeclarationStatementContext(this, this._ctx, this.state);
		this.enterRule(localctx, 134, SolidityParser.RULE_variableDeclarationStatement);
		let _la: number;
		try {
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 796;
			this._errHandler.sync(this);
			switch ( this._interp.adaptivePredict(this._input, 84, this._ctx) ) {
			case 1:
				{
				this.state = 789;
				this.match(SolidityParser.T__64);
				this.state = 790;
				this.identifierList();
				}
				break;
			case 2:
				{
				this.state = 791;
				this.variableDeclaration();
				}
				break;
			case 3:
				{
				this.state = 792;
				this.match(SolidityParser.T__22);
				this.state = 793;
				this.variableDeclarationList();
				this.state = 794;
				this.match(SolidityParser.T__23);
				}
				break;
			}
			this.state = 800;
			this._errHandler.sync(this);
			_la = this._input.LA(1);
			if (_la===11) {
				{
				this.state = 798;
				this.match(SolidityParser.T__10);
				this.state = 799;
				this.expression(0);
				}
			}

			this.state = 802;
			this.match(SolidityParser.T__1);
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public variableDeclarationList(): VariableDeclarationListContext {
		let localctx: VariableDeclarationListContext = new VariableDeclarationListContext(this, this._ctx, this.state);
		this.enterRule(localctx, 136, SolidityParser.RULE_variableDeclarationList);
		let _la: number;
		try {
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 805;
			this._errHandler.sync(this);
			_la = this._input.LA(1);
			if ((((_la) & ~0x1F) === 0 && ((1 << _la) & 234897408) !== 0) || ((((_la - 40)) & ~0x1F) === 0 && ((1 << (_la - 40)) & 520098113) !== 0) || ((((_la - 97)) & ~0x1F) === 0 && ((1 << (_la - 97)) & 3230138493) !== 0) || _la===130 || _la===131) {
				{
				this.state = 804;
				this.variableDeclaration();
				}
			}

			this.state = 813;
			this._errHandler.sync(this);
			_la = this._input.LA(1);
			while (_la===16) {
				{
				{
				this.state = 807;
				this.match(SolidityParser.T__15);
				this.state = 809;
				this._errHandler.sync(this);
				_la = this._input.LA(1);
				if ((((_la) & ~0x1F) === 0 && ((1 << _la) & 234897408) !== 0) || ((((_la - 40)) & ~0x1F) === 0 && ((1 << (_la - 40)) & 520098113) !== 0) || ((((_la - 97)) & ~0x1F) === 0 && ((1 << (_la - 97)) & 3230138493) !== 0) || _la===130 || _la===131) {
					{
					this.state = 808;
					this.variableDeclaration();
					}
				}

				}
				}
				this.state = 815;
				this._errHandler.sync(this);
				_la = this._input.LA(1);
			}
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public identifierList(): IdentifierListContext {
		let localctx: IdentifierListContext = new IdentifierListContext(this, this._ctx, this.state);
		this.enterRule(localctx, 138, SolidityParser.RULE_identifierList);
		let _la: number;
		try {
			let _alt: number;
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 816;
			this.match(SolidityParser.T__22);
			this.state = 823;
			this._errHandler.sync(this);
			_alt = this._interp.adaptivePredict(this._input, 90, this._ctx);
			while (_alt !== 2 && _alt !== ATN.INVALID_ALT_NUMBER) {
				if (_alt === 1) {
					{
					{
					this.state = 818;
					this._errHandler.sync(this);
					_la = this._input.LA(1);
					if ((((_la) & ~0x1F) === 0 && ((1 << _la) & 234897408) !== 0) || ((((_la - 46)) & ~0x1F) === 0 && ((1 << (_la - 46)) & 262209) !== 0) || ((((_la - 97)) & ~0x1F) === 0 && ((1 << (_la - 97)) & 3230138369) !== 0) || _la===130 || _la===131) {
						{
						this.state = 817;
						this.identifier();
						}
					}

					this.state = 820;
					this.match(SolidityParser.T__15);
					}
					}
				}
				this.state = 825;
				this._errHandler.sync(this);
				_alt = this._interp.adaptivePredict(this._input, 90, this._ctx);
			}
			this.state = 827;
			this._errHandler.sync(this);
			_la = this._input.LA(1);
			if ((((_la) & ~0x1F) === 0 && ((1 << _la) & 234897408) !== 0) || ((((_la - 46)) & ~0x1F) === 0 && ((1 << (_la - 46)) & 262209) !== 0) || ((((_la - 97)) & ~0x1F) === 0 && ((1 << (_la - 97)) & 3230138369) !== 0) || _la===130 || _la===131) {
				{
				this.state = 826;
				this.identifier();
				}
			}

			this.state = 829;
			this.match(SolidityParser.T__23);
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public elementaryTypeName(): ElementaryTypeNameContext {
		let localctx: ElementaryTypeNameContext = new ElementaryTypeNameContext(this, this._ctx, this.state);
		this.enterRule(localctx, 140, SolidityParser.RULE_elementaryTypeName);
		let _la: number;
		try {
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 831;
			_la = this._input.LA(1);
			if(!(((((_la - 46)) & ~0x1F) === 0 && ((1 << (_la - 46)) & 7864321) !== 0) || ((((_la - 99)) & ~0x1F) === 0 && ((1 << (_la - 99)) & 31) !== 0))) {
			this._errHandler.recoverInline(this);
			}
			else {
				this._errHandler.reportMatch(this);
			    this.consume();
			}
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}

	public expression(): ExpressionContext;
	public expression(_p: number): ExpressionContext;
	// @RuleVersion(0)
	public expression(_p?: number): ExpressionContext {
		if (_p === undefined) {
			_p = 0;
		}

		let _parentctx: ParserRuleContext = this._ctx;
		let _parentState: number = this.state;
		let localctx: ExpressionContext = new ExpressionContext(this, this._ctx, _parentState);
		let _prevctx: ExpressionContext = localctx;
		let _startState: number = 142;
		this.enterRecursionRule(localctx, 142, SolidityParser.RULE_expression, _p);
		let _la: number;
		try {
			let _alt: number;
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 851;
			this._errHandler.sync(this);
			switch ( this._interp.adaptivePredict(this._input, 92, this._ctx) ) {
			case 1:
				{
				this.state = 834;
				this.match(SolidityParser.T__70);
				this.state = 835;
				this.typeName(0);
				}
				break;
			case 2:
				{
				this.state = 836;
				this.match(SolidityParser.T__22);
				this.state = 837;
				this.expression(0);
				this.state = 838;
				this.match(SolidityParser.T__23);
				}
				break;
			case 3:
				{
				this.state = 840;
				_la = this._input.LA(1);
				if(!(_la===69 || _la===70)) {
				this._errHandler.recoverInline(this);
				}
				else {
					this._errHandler.reportMatch(this);
				    this.consume();
				}
				this.state = 841;
				this.expression(19);
				}
				break;
			case 4:
				{
				this.state = 842;
				_la = this._input.LA(1);
				if(!(_la===32 || _la===33)) {
				this._errHandler.recoverInline(this);
				}
				else {
					this._errHandler.reportMatch(this);
				    this.consume();
				}
				this.state = 843;
				this.expression(18);
				}
				break;
			case 5:
				{
				this.state = 844;
				this.match(SolidityParser.T__72);
				this.state = 845;
				this.expression(17);
				}
				break;
			case 6:
				{
				this.state = 846;
				this.match(SolidityParser.T__73);
				this.state = 847;
				this.expression(16);
				}
				break;
			case 7:
				{
				this.state = 848;
				this.match(SolidityParser.T__5);
				this.state = 849;
				this.expression(15);
				}
				break;
			case 8:
				{
				this.state = 850;
				this.primaryExpression();
				}
				break;
			}
			this._ctx.stop = this._input.LT(-1);
			this.state = 927;
			this._errHandler.sync(this);
			_alt = this._interp.adaptivePredict(this._input, 96, this._ctx);
			while (_alt !== 2 && _alt !== ATN.INVALID_ALT_NUMBER) {
				if (_alt === 1) {
					if (this._parseListeners != null) {
						this.triggerExitRuleEvent();
					}
					_prevctx = localctx;
					{
					this.state = 925;
					this._errHandler.sync(this);
					switch ( this._interp.adaptivePredict(this._input, 95, this._ctx) ) {
					case 1:
						{
						localctx = new ExpressionContext(this, _parentctx, _parentState);
						this.pushNewRecursionContext(localctx, _startState, SolidityParser.RULE_expression);
						this.state = 853;
						if (!(this.precpred(this._ctx, 14))) {
							throw this.createFailedPredicateException("this.precpred(this._ctx, 14)");
						}
						this.state = 854;
						this.match(SolidityParser.T__74);
						this.state = 855;
						this.expression(14);
						}
						break;
					case 2:
						{
						localctx = new ExpressionContext(this, _parentctx, _parentState);
						this.pushNewRecursionContext(localctx, _startState, SolidityParser.RULE_expression);
						this.state = 856;
						if (!(this.precpred(this._ctx, 13))) {
							throw this.createFailedPredicateException("this.precpred(this._ctx, 13)");
						}
						this.state = 857;
						_la = this._input.LA(1);
						if(!(_la===3 || _la===34 || _la===35)) {
						this._errHandler.recoverInline(this);
						}
						else {
							this._errHandler.reportMatch(this);
						    this.consume();
						}
						this.state = 858;
						this.expression(14);
						}
						break;
					case 3:
						{
						localctx = new ExpressionContext(this, _parentctx, _parentState);
						this.pushNewRecursionContext(localctx, _startState, SolidityParser.RULE_expression);
						this.state = 859;
						if (!(this.precpred(this._ctx, 12))) {
							throw this.createFailedPredicateException("this.precpred(this._ctx, 12)");
						}
						this.state = 860;
						_la = this._input.LA(1);
						if(!(_la===32 || _la===33)) {
						this._errHandler.recoverInline(this);
						}
						else {
							this._errHandler.reportMatch(this);
						    this.consume();
						}
						this.state = 861;
						this.expression(13);
						}
						break;
					case 4:
						{
						localctx = new ExpressionContext(this, _parentctx, _parentState);
						this.pushNewRecursionContext(localctx, _startState, SolidityParser.RULE_expression);
						this.state = 862;
						if (!(this.precpred(this._ctx, 11))) {
							throw this.createFailedPredicateException("this.precpred(this._ctx, 11)");
						}
						this.state = 863;
						_la = this._input.LA(1);
						if(!(_la===76 || _la===77)) {
						this._errHandler.recoverInline(this);
						}
						else {
							this._errHandler.reportMatch(this);
						    this.consume();
						}
						this.state = 864;
						this.expression(12);
						}
						break;
					case 5:
						{
						localctx = new ExpressionContext(this, _parentctx, _parentState);
						this.pushNewRecursionContext(localctx, _startState, SolidityParser.RULE_expression);
						this.state = 865;
						if (!(this.precpred(this._ctx, 10))) {
							throw this.createFailedPredicateException("this.precpred(this._ctx, 10)");
						}
						this.state = 866;
						this.match(SolidityParser.T__30);
						this.state = 867;
						this.expression(11);
						}
						break;
					case 6:
						{
						localctx = new ExpressionContext(this, _parentctx, _parentState);
						this.pushNewRecursionContext(localctx, _startState, SolidityParser.RULE_expression);
						this.state = 868;
						if (!(this.precpred(this._ctx, 9))) {
							throw this.createFailedPredicateException("this.precpred(this._ctx, 9)");
						}
						this.state = 869;
						this.match(SolidityParser.T__4);
						this.state = 870;
						this.expression(10);
						}
						break;
					case 7:
						{
						localctx = new ExpressionContext(this, _parentctx, _parentState);
						this.pushNewRecursionContext(localctx, _startState, SolidityParser.RULE_expression);
						this.state = 871;
						if (!(this.precpred(this._ctx, 8))) {
							throw this.createFailedPredicateException("this.precpred(this._ctx, 8)");
						}
						this.state = 872;
						this.match(SolidityParser.T__29);
						this.state = 873;
						this.expression(9);
						}
						break;
					case 8:
						{
						localctx = new ExpressionContext(this, _parentctx, _parentState);
						this.pushNewRecursionContext(localctx, _startState, SolidityParser.RULE_expression);
						this.state = 874;
						if (!(this.precpred(this._ctx, 7))) {
							throw this.createFailedPredicateException("this.precpred(this._ctx, 7)");
						}
						this.state = 875;
						_la = this._input.LA(1);
						if(!((((_la) & ~0x1F) === 0 && ((1 << _la) & 1920) !== 0))) {
						this._errHandler.recoverInline(this);
						}
						else {
							this._errHandler.reportMatch(this);
						    this.consume();
						}
						this.state = 876;
						this.expression(8);
						}
						break;
					case 9:
						{
						localctx = new ExpressionContext(this, _parentctx, _parentState);
						this.pushNewRecursionContext(localctx, _startState, SolidityParser.RULE_expression);
						this.state = 877;
						if (!(this.precpred(this._ctx, 6))) {
							throw this.createFailedPredicateException("this.precpred(this._ctx, 6)");
						}
						this.state = 878;
						_la = this._input.LA(1);
						if(!(_la===36 || _la===37)) {
						this._errHandler.recoverInline(this);
						}
						else {
							this._errHandler.reportMatch(this);
						    this.consume();
						}
						this.state = 879;
						this.expression(7);
						}
						break;
					case 10:
						{
						localctx = new ExpressionContext(this, _parentctx, _parentState);
						this.pushNewRecursionContext(localctx, _startState, SolidityParser.RULE_expression);
						this.state = 880;
						if (!(this.precpred(this._ctx, 5))) {
							throw this.createFailedPredicateException("this.precpred(this._ctx, 5)");
						}
						this.state = 881;
						this.match(SolidityParser.T__77);
						this.state = 882;
						this.expression(6);
						}
						break;
					case 11:
						{
						localctx = new ExpressionContext(this, _parentctx, _parentState);
						this.pushNewRecursionContext(localctx, _startState, SolidityParser.RULE_expression);
						this.state = 883;
						if (!(this.precpred(this._ctx, 4))) {
							throw this.createFailedPredicateException("this.precpred(this._ctx, 4)");
						}
						this.state = 884;
						this.match(SolidityParser.T__3);
						this.state = 885;
						this.expression(5);
						}
						break;
					case 12:
						{
						localctx = new ExpressionContext(this, _parentctx, _parentState);
						this.pushNewRecursionContext(localctx, _startState, SolidityParser.RULE_expression);
						this.state = 886;
						if (!(this.precpred(this._ctx, 3))) {
							throw this.createFailedPredicateException("this.precpred(this._ctx, 3)");
						}
						this.state = 887;
						this.match(SolidityParser.T__78);
						this.state = 888;
						this.expression(0);
						this.state = 889;
						this.match(SolidityParser.T__71);
						this.state = 890;
						this.expression(3);
						}
						break;
					case 13:
						{
						localctx = new ExpressionContext(this, _parentctx, _parentState);
						this.pushNewRecursionContext(localctx, _startState, SolidityParser.RULE_expression);
						this.state = 892;
						if (!(this.precpred(this._ctx, 2))) {
							throw this.createFailedPredicateException("this.precpred(this._ctx, 2)");
						}
						this.state = 893;
						_la = this._input.LA(1);
						if(!(_la===11 || ((((_la - 80)) & ~0x1F) === 0 && ((1 << (_la - 80)) & 1023) !== 0))) {
						this._errHandler.recoverInline(this);
						}
						else {
							this._errHandler.reportMatch(this);
						    this.consume();
						}
						this.state = 894;
						this.expression(3);
						}
						break;
					case 14:
						{
						localctx = new ExpressionContext(this, _parentctx, _parentState);
						this.pushNewRecursionContext(localctx, _startState, SolidityParser.RULE_expression);
						this.state = 895;
						if (!(this.precpred(this._ctx, 27))) {
							throw this.createFailedPredicateException("this.precpred(this._ctx, 27)");
						}
						this.state = 896;
						_la = this._input.LA(1);
						if(!(_la===69 || _la===70)) {
						this._errHandler.recoverInline(this);
						}
						else {
							this._errHandler.reportMatch(this);
						    this.consume();
						}
						}
						break;
					case 15:
						{
						localctx = new ExpressionContext(this, _parentctx, _parentState);
						this.pushNewRecursionContext(localctx, _startState, SolidityParser.RULE_expression);
						this.state = 897;
						if (!(this.precpred(this._ctx, 25))) {
							throw this.createFailedPredicateException("this.precpred(this._ctx, 25)");
						}
						this.state = 898;
						this.match(SolidityParser.T__43);
						this.state = 899;
						this.expression(0);
						this.state = 900;
						this.match(SolidityParser.T__44);
						}
						break;
					case 16:
						{
						localctx = new ExpressionContext(this, _parentctx, _parentState);
						this.pushNewRecursionContext(localctx, _startState, SolidityParser.RULE_expression);
						this.state = 902;
						if (!(this.precpred(this._ctx, 24))) {
							throw this.createFailedPredicateException("this.precpred(this._ctx, 24)");
						}
						this.state = 903;
						this.match(SolidityParser.T__43);
						this.state = 905;
						this._errHandler.sync(this);
						_la = this._input.LA(1);
						if (((((_la - 6)) & ~0x1F) === 0 && ((1 << (_la - 6)) & 205127937) !== 0) || ((((_la - 40)) & ~0x1F) === 0 && ((1 << (_la - 40)) & 4278194513) !== 0) || ((((_la - 73)) & ~0x1F) === 0 && ((1 << (_la - 73)) & 4244635651) !== 0) || ((((_la - 105)) & ~0x1F) === 0 && ((1 << (_la - 105)) & 248547339) !== 0)) {
							{
							this.state = 904;
							this.expression(0);
							}
						}

						this.state = 907;
						this.match(SolidityParser.T__71);
						this.state = 909;
						this._errHandler.sync(this);
						_la = this._input.LA(1);
						if (((((_la - 6)) & ~0x1F) === 0 && ((1 << (_la - 6)) & 205127937) !== 0) || ((((_la - 40)) & ~0x1F) === 0 && ((1 << (_la - 40)) & 4278194513) !== 0) || ((((_la - 73)) & ~0x1F) === 0 && ((1 << (_la - 73)) & 4244635651) !== 0) || ((((_la - 105)) & ~0x1F) === 0 && ((1 << (_la - 105)) & 248547339) !== 0)) {
							{
							this.state = 908;
							this.expression(0);
							}
						}

						this.state = 911;
						this.match(SolidityParser.T__44);
						}
						break;
					case 17:
						{
						localctx = new ExpressionContext(this, _parentctx, _parentState);
						this.pushNewRecursionContext(localctx, _startState, SolidityParser.RULE_expression);
						this.state = 912;
						if (!(this.precpred(this._ctx, 23))) {
							throw this.createFailedPredicateException("this.precpred(this._ctx, 23)");
						}
						this.state = 913;
						this.match(SolidityParser.T__46);
						this.state = 914;
						this.identifier();
						}
						break;
					case 18:
						{
						localctx = new ExpressionContext(this, _parentctx, _parentState);
						this.pushNewRecursionContext(localctx, _startState, SolidityParser.RULE_expression);
						this.state = 915;
						if (!(this.precpred(this._ctx, 22))) {
							throw this.createFailedPredicateException("this.precpred(this._ctx, 22)");
						}
						this.state = 916;
						this.match(SolidityParser.T__14);
						this.state = 917;
						this.nameValueList();
						this.state = 918;
						this.match(SolidityParser.T__16);
						}
						break;
					case 19:
						{
						localctx = new ExpressionContext(this, _parentctx, _parentState);
						this.pushNewRecursionContext(localctx, _startState, SolidityParser.RULE_expression);
						this.state = 920;
						if (!(this.precpred(this._ctx, 21))) {
							throw this.createFailedPredicateException("this.precpred(this._ctx, 21)");
						}
						this.state = 921;
						this.match(SolidityParser.T__22);
						this.state = 922;
						this.functionCallArguments();
						this.state = 923;
						this.match(SolidityParser.T__23);
						}
						break;
					}
					}
				}
				this.state = 929;
				this._errHandler.sync(this);
				_alt = this._interp.adaptivePredict(this._input, 96, this._ctx);
			}
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.unrollRecursionContexts(_parentctx);
		}
		return localctx;
	}
	// @RuleVersion(0)
	public primaryExpression(): PrimaryExpressionContext {
		let localctx: PrimaryExpressionContext = new PrimaryExpressionContext(this, this._ctx, this.state);
		this.enterRule(localctx, 144, SolidityParser.RULE_primaryExpression);
		try {
			this.state = 939;
			this._errHandler.sync(this);
			switch ( this._interp.adaptivePredict(this._input, 97, this._ctx) ) {
			case 1:
				this.enterOuterAlt(localctx, 1);
				{
				this.state = 930;
				this.match(SolidityParser.BooleanLiteral);
				}
				break;
			case 2:
				this.enterOuterAlt(localctx, 2);
				{
				this.state = 931;
				this.numberLiteral();
				}
				break;
			case 3:
				this.enterOuterAlt(localctx, 3);
				{
				this.state = 932;
				this.hexLiteral();
				}
				break;
			case 4:
				this.enterOuterAlt(localctx, 4);
				{
				this.state = 933;
				this.stringLiteral();
				}
				break;
			case 5:
				this.enterOuterAlt(localctx, 5);
				{
				this.state = 934;
				this.identifier();
				}
				break;
			case 6:
				this.enterOuterAlt(localctx, 6);
				{
				this.state = 935;
				this.match(SolidityParser.TypeKeyword);
				}
				break;
			case 7:
				this.enterOuterAlt(localctx, 7);
				{
				this.state = 936;
				this.match(SolidityParser.PayableKeyword);
				}
				break;
			case 8:
				this.enterOuterAlt(localctx, 8);
				{
				this.state = 937;
				this.tupleExpression();
				}
				break;
			case 9:
				this.enterOuterAlt(localctx, 9);
				{
				this.state = 938;
				this.typeName(0);
				}
				break;
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public expressionList(): ExpressionListContext {
		let localctx: ExpressionListContext = new ExpressionListContext(this, this._ctx, this.state);
		this.enterRule(localctx, 146, SolidityParser.RULE_expressionList);
		let _la: number;
		try {
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 941;
			this.expression(0);
			this.state = 946;
			this._errHandler.sync(this);
			_la = this._input.LA(1);
			while (_la===16) {
				{
				{
				this.state = 942;
				this.match(SolidityParser.T__15);
				this.state = 943;
				this.expression(0);
				}
				}
				this.state = 948;
				this._errHandler.sync(this);
				_la = this._input.LA(1);
			}
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public nameValueList(): NameValueListContext {
		let localctx: NameValueListContext = new NameValueListContext(this, this._ctx, this.state);
		this.enterRule(localctx, 148, SolidityParser.RULE_nameValueList);
		let _la: number;
		try {
			let _alt: number;
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 949;
			this.nameValue();
			this.state = 954;
			this._errHandler.sync(this);
			_alt = this._interp.adaptivePredict(this._input, 99, this._ctx);
			while (_alt !== 2 && _alt !== ATN.INVALID_ALT_NUMBER) {
				if (_alt === 1) {
					{
					{
					this.state = 950;
					this.match(SolidityParser.T__15);
					this.state = 951;
					this.nameValue();
					}
					}
				}
				this.state = 956;
				this._errHandler.sync(this);
				_alt = this._interp.adaptivePredict(this._input, 99, this._ctx);
			}
			this.state = 958;
			this._errHandler.sync(this);
			_la = this._input.LA(1);
			if (_la===16) {
				{
				this.state = 957;
				this.match(SolidityParser.T__15);
				}
			}

			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public nameValue(): NameValueContext {
		let localctx: NameValueContext = new NameValueContext(this, this._ctx, this.state);
		this.enterRule(localctx, 150, SolidityParser.RULE_nameValue);
		try {
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 960;
			this.identifier();
			this.state = 961;
			this.match(SolidityParser.T__71);
			this.state = 962;
			this.expression(0);
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public functionCallArguments(): FunctionCallArgumentsContext {
		let localctx: FunctionCallArgumentsContext = new FunctionCallArgumentsContext(this, this._ctx, this.state);
		this.enterRule(localctx, 152, SolidityParser.RULE_functionCallArguments);
		let _la: number;
		try {
			this.state = 972;
			this._errHandler.sync(this);
			switch (this._input.LA(1)) {
			case 15:
				this.enterOuterAlt(localctx, 1);
				{
				this.state = 964;
				this.match(SolidityParser.T__14);
				this.state = 966;
				this._errHandler.sync(this);
				_la = this._input.LA(1);
				if ((((_la) & ~0x1F) === 0 && ((1 << _la) & 234897408) !== 0) || ((((_la - 46)) & ~0x1F) === 0 && ((1 << (_la - 46)) & 262209) !== 0) || ((((_la - 97)) & ~0x1F) === 0 && ((1 << (_la - 97)) & 3230138369) !== 0) || _la===130 || _la===131) {
					{
					this.state = 965;
					this.nameValueList();
					}
				}

				this.state = 968;
				this.match(SolidityParser.T__16);
				}
				break;
			case 6:
			case 14:
			case 23:
			case 24:
			case 25:
			case 26:
			case 27:
			case 32:
			case 33:
			case 40:
			case 44:
			case 46:
			case 48:
			case 52:
			case 64:
			case 65:
			case 66:
			case 67:
			case 68:
			case 69:
			case 70:
			case 71:
			case 73:
			case 74:
			case 97:
			case 99:
			case 100:
			case 101:
			case 102:
			case 103:
			case 104:
			case 105:
			case 106:
			case 108:
			case 116:
			case 120:
			case 125:
			case 127:
			case 128:
			case 130:
			case 131:
			case 132:
				this.enterOuterAlt(localctx, 2);
				{
				this.state = 970;
				this._errHandler.sync(this);
				_la = this._input.LA(1);
				if (((((_la - 6)) & ~0x1F) === 0 && ((1 << (_la - 6)) & 205127937) !== 0) || ((((_la - 40)) & ~0x1F) === 0 && ((1 << (_la - 40)) & 4278194513) !== 0) || ((((_la - 73)) & ~0x1F) === 0 && ((1 << (_la - 73)) & 4244635651) !== 0) || ((((_la - 105)) & ~0x1F) === 0 && ((1 << (_la - 105)) & 248547339) !== 0)) {
					{
					this.state = 969;
					this.expressionList();
					}
				}

				}
				break;
			default:
				throw new NoViableAltException(this);
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public functionCall(): FunctionCallContext {
		let localctx: FunctionCallContext = new FunctionCallContext(this, this._ctx, this.state);
		this.enterRule(localctx, 154, SolidityParser.RULE_functionCall);
		try {
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 974;
			this.expression(0);
			this.state = 975;
			this.match(SolidityParser.T__22);
			this.state = 976;
			this.functionCallArguments();
			this.state = 977;
			this.match(SolidityParser.T__23);
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public assemblyBlock(): AssemblyBlockContext {
		let localctx: AssemblyBlockContext = new AssemblyBlockContext(this, this._ctx, this.state);
		this.enterRule(localctx, 156, SolidityParser.RULE_assemblyBlock);
		let _la: number;
		try {
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 979;
			this.match(SolidityParser.T__14);
			this.state = 983;
			this._errHandler.sync(this);
			_la = this._input.LA(1);
			while ((((_la) & ~0x1F) === 0 && ((1 << _la) & 780189696) !== 0) || ((((_la - 40)) & ~0x1F) === 0 && ((1 << (_la - 40)) & 287322177) !== 0) || ((((_la - 90)) & ~0x1F) === 0 && ((1 << (_la - 90)) & 1176879241) !== 0) || ((((_la - 127)) & ~0x1F) === 0 && ((1 << (_la - 127)) & 59) !== 0)) {
				{
				{
				this.state = 980;
				this.assemblyItem();
				}
				}
				this.state = 985;
				this._errHandler.sync(this);
				_la = this._input.LA(1);
			}
			this.state = 986;
			this.match(SolidityParser.T__16);
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public assemblyItem(): AssemblyItemContext {
		let localctx: AssemblyItemContext = new AssemblyItemContext(this, this._ctx, this.state);
		this.enterRule(localctx, 158, SolidityParser.RULE_assemblyItem);
		try {
			this.state = 1005;
			this._errHandler.sync(this);
			switch ( this._interp.adaptivePredict(this._input, 105, this._ctx) ) {
			case 1:
				this.enterOuterAlt(localctx, 1);
				{
				this.state = 988;
				this.identifier();
				}
				break;
			case 2:
				this.enterOuterAlt(localctx, 2);
				{
				this.state = 989;
				this.assemblyBlock();
				}
				break;
			case 3:
				this.enterOuterAlt(localctx, 3);
				{
				this.state = 990;
				this.assemblyExpression();
				}
				break;
			case 4:
				this.enterOuterAlt(localctx, 4);
				{
				this.state = 991;
				this.assemblyLocalDefinition();
				}
				break;
			case 5:
				this.enterOuterAlt(localctx, 5);
				{
				this.state = 992;
				this.assemblyAssignment();
				}
				break;
			case 6:
				this.enterOuterAlt(localctx, 6);
				{
				this.state = 993;
				this.assemblyStackAssignment();
				}
				break;
			case 7:
				this.enterOuterAlt(localctx, 7);
				{
				this.state = 994;
				this.labelDefinition();
				}
				break;
			case 8:
				this.enterOuterAlt(localctx, 8);
				{
				this.state = 995;
				this.assemblySwitch();
				}
				break;
			case 9:
				this.enterOuterAlt(localctx, 9);
				{
				this.state = 996;
				this.assemblyFunctionDefinition();
				}
				break;
			case 10:
				this.enterOuterAlt(localctx, 10);
				{
				this.state = 997;
				this.assemblyFor();
				}
				break;
			case 11:
				this.enterOuterAlt(localctx, 11);
				{
				this.state = 998;
				this.assemblyIf();
				}
				break;
			case 12:
				this.enterOuterAlt(localctx, 12);
				{
				this.state = 999;
				this.match(SolidityParser.BreakKeyword);
				}
				break;
			case 13:
				this.enterOuterAlt(localctx, 13);
				{
				this.state = 1000;
				this.match(SolidityParser.ContinueKeyword);
				}
				break;
			case 14:
				this.enterOuterAlt(localctx, 14);
				{
				this.state = 1001;
				this.match(SolidityParser.LeaveKeyword);
				}
				break;
			case 15:
				this.enterOuterAlt(localctx, 15);
				{
				this.state = 1002;
				this.numberLiteral();
				}
				break;
			case 16:
				this.enterOuterAlt(localctx, 16);
				{
				this.state = 1003;
				this.stringLiteral();
				}
				break;
			case 17:
				this.enterOuterAlt(localctx, 17);
				{
				this.state = 1004;
				this.hexLiteral();
				}
				break;
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public assemblyExpression(): AssemblyExpressionContext {
		let localctx: AssemblyExpressionContext = new AssemblyExpressionContext(this, this._ctx, this.state);
		this.enterRule(localctx, 160, SolidityParser.RULE_assemblyExpression);
		try {
			this.state = 1010;
			this._errHandler.sync(this);
			switch ( this._interp.adaptivePredict(this._input, 106, this._ctx) ) {
			case 1:
				this.enterOuterAlt(localctx, 1);
				{
				this.state = 1007;
				this.assemblyCall();
				}
				break;
			case 2:
				this.enterOuterAlt(localctx, 2);
				{
				this.state = 1008;
				this.assemblyLiteral();
				}
				break;
			case 3:
				this.enterOuterAlt(localctx, 3);
				{
				this.state = 1009;
				this.assemblyMember();
				}
				break;
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public assemblyMember(): AssemblyMemberContext {
		let localctx: AssemblyMemberContext = new AssemblyMemberContext(this, this._ctx, this.state);
		this.enterRule(localctx, 162, SolidityParser.RULE_assemblyMember);
		try {
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 1012;
			this.identifier();
			this.state = 1013;
			this.match(SolidityParser.T__46);
			this.state = 1014;
			this.identifier();
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public assemblyCall(): AssemblyCallContext {
		let localctx: AssemblyCallContext = new AssemblyCallContext(this, this._ctx, this.state);
		this.enterRule(localctx, 164, SolidityParser.RULE_assemblyCall);
		let _la: number;
		try {
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 1020;
			this._errHandler.sync(this);
			switch ( this._interp.adaptivePredict(this._input, 107, this._ctx) ) {
			case 1:
				{
				this.state = 1016;
				this.match(SolidityParser.T__60);
				}
				break;
			case 2:
				{
				this.state = 1017;
				this.match(SolidityParser.T__45);
				}
				break;
			case 3:
				{
				this.state = 1018;
				this.match(SolidityParser.T__67);
				}
				break;
			case 4:
				{
				this.state = 1019;
				this.identifier();
				}
				break;
			}
			this.state = 1034;
			this._errHandler.sync(this);
			switch ( this._interp.adaptivePredict(this._input, 110, this._ctx) ) {
			case 1:
				{
				this.state = 1022;
				this.match(SolidityParser.T__22);
				this.state = 1024;
				this._errHandler.sync(this);
				_la = this._input.LA(1);
				if ((((_la) & ~0x1F) === 0 && ((1 << _la) & 234897408) !== 0) || ((((_la - 46)) & ~0x1F) === 0 && ((1 << (_la - 46)) & 4489281) !== 0) || ((((_la - 97)) & ~0x1F) === 0 && ((1 << (_la - 97)) & 3230141313) !== 0) || ((((_la - 130)) & ~0x1F) === 0 && ((1 << (_la - 130)) & 7) !== 0)) {
					{
					this.state = 1023;
					this.assemblyExpression();
					}
				}

				this.state = 1030;
				this._errHandler.sync(this);
				_la = this._input.LA(1);
				while (_la===16) {
					{
					{
					this.state = 1026;
					this.match(SolidityParser.T__15);
					this.state = 1027;
					this.assemblyExpression();
					}
					}
					this.state = 1032;
					this._errHandler.sync(this);
					_la = this._input.LA(1);
				}
				this.state = 1033;
				this.match(SolidityParser.T__23);
				}
				break;
			}
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public assemblyLocalDefinition(): AssemblyLocalDefinitionContext {
		let localctx: AssemblyLocalDefinitionContext = new AssemblyLocalDefinitionContext(this, this._ctx, this.state);
		this.enterRule(localctx, 166, SolidityParser.RULE_assemblyLocalDefinition);
		let _la: number;
		try {
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 1036;
			this.match(SolidityParser.T__89);
			this.state = 1037;
			this.assemblyIdentifierOrList();
			this.state = 1040;
			this._errHandler.sync(this);
			_la = this._input.LA(1);
			if (_la===91) {
				{
				this.state = 1038;
				this.match(SolidityParser.T__90);
				this.state = 1039;
				this.assemblyExpression();
				}
			}

			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public assemblyAssignment(): AssemblyAssignmentContext {
		let localctx: AssemblyAssignmentContext = new AssemblyAssignmentContext(this, this._ctx, this.state);
		this.enterRule(localctx, 168, SolidityParser.RULE_assemblyAssignment);
		try {
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 1042;
			this.assemblyIdentifierOrList();
			this.state = 1043;
			this.match(SolidityParser.T__90);
			this.state = 1044;
			this.assemblyExpression();
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public assemblyIdentifierOrList(): AssemblyIdentifierOrListContext {
		let localctx: AssemblyIdentifierOrListContext = new AssemblyIdentifierOrListContext(this, this._ctx, this.state);
		this.enterRule(localctx, 170, SolidityParser.RULE_assemblyIdentifierOrList);
		try {
			this.state = 1053;
			this._errHandler.sync(this);
			switch ( this._interp.adaptivePredict(this._input, 112, this._ctx) ) {
			case 1:
				this.enterOuterAlt(localctx, 1);
				{
				this.state = 1046;
				this.identifier();
				}
				break;
			case 2:
				this.enterOuterAlt(localctx, 2);
				{
				this.state = 1047;
				this.assemblyMember();
				}
				break;
			case 3:
				this.enterOuterAlt(localctx, 3);
				{
				this.state = 1048;
				this.assemblyIdentifierList();
				}
				break;
			case 4:
				this.enterOuterAlt(localctx, 4);
				{
				this.state = 1049;
				this.match(SolidityParser.T__22);
				this.state = 1050;
				this.assemblyIdentifierList();
				this.state = 1051;
				this.match(SolidityParser.T__23);
				}
				break;
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public assemblyIdentifierList(): AssemblyIdentifierListContext {
		let localctx: AssemblyIdentifierListContext = new AssemblyIdentifierListContext(this, this._ctx, this.state);
		this.enterRule(localctx, 172, SolidityParser.RULE_assemblyIdentifierList);
		let _la: number;
		try {
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 1055;
			this.identifier();
			this.state = 1060;
			this._errHandler.sync(this);
			_la = this._input.LA(1);
			while (_la===16) {
				{
				{
				this.state = 1056;
				this.match(SolidityParser.T__15);
				this.state = 1057;
				this.identifier();
				}
				}
				this.state = 1062;
				this._errHandler.sync(this);
				_la = this._input.LA(1);
			}
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public assemblyStackAssignment(): AssemblyStackAssignmentContext {
		let localctx: AssemblyStackAssignmentContext = new AssemblyStackAssignmentContext(this, this._ctx, this.state);
		this.enterRule(localctx, 174, SolidityParser.RULE_assemblyStackAssignment);
		try {
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 1063;
			this.assemblyExpression();
			this.state = 1064;
			this.match(SolidityParser.T__91);
			this.state = 1065;
			this.identifier();
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public labelDefinition(): LabelDefinitionContext {
		let localctx: LabelDefinitionContext = new LabelDefinitionContext(this, this._ctx, this.state);
		this.enterRule(localctx, 176, SolidityParser.RULE_labelDefinition);
		try {
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 1067;
			this.identifier();
			this.state = 1068;
			this.match(SolidityParser.T__71);
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public assemblySwitch(): AssemblySwitchContext {
		let localctx: AssemblySwitchContext = new AssemblySwitchContext(this, this._ctx, this.state);
		this.enterRule(localctx, 178, SolidityParser.RULE_assemblySwitch);
		let _la: number;
		try {
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 1070;
			this.match(SolidityParser.T__92);
			this.state = 1071;
			this.assemblyExpression();
			this.state = 1075;
			this._errHandler.sync(this);
			_la = this._input.LA(1);
			while (_la===94 || _la===95) {
				{
				{
				this.state = 1072;
				this.assemblyCase();
				}
				}
				this.state = 1077;
				this._errHandler.sync(this);
				_la = this._input.LA(1);
			}
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public assemblyCase(): AssemblyCaseContext {
		let localctx: AssemblyCaseContext = new AssemblyCaseContext(this, this._ctx, this.state);
		this.enterRule(localctx, 180, SolidityParser.RULE_assemblyCase);
		try {
			this.state = 1084;
			this._errHandler.sync(this);
			switch (this._input.LA(1)) {
			case 94:
				this.enterOuterAlt(localctx, 1);
				{
				this.state = 1078;
				this.match(SolidityParser.T__93);
				this.state = 1079;
				this.assemblyLiteral();
				this.state = 1080;
				this.assemblyBlock();
				}
				break;
			case 95:
				this.enterOuterAlt(localctx, 2);
				{
				this.state = 1082;
				this.match(SolidityParser.T__94);
				this.state = 1083;
				this.assemblyBlock();
				}
				break;
			default:
				throw new NoViableAltException(this);
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public assemblyFunctionDefinition(): AssemblyFunctionDefinitionContext {
		let localctx: AssemblyFunctionDefinitionContext = new AssemblyFunctionDefinitionContext(this, this._ctx, this.state);
		this.enterRule(localctx, 182, SolidityParser.RULE_assemblyFunctionDefinition);
		let _la: number;
		try {
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 1086;
			this.match(SolidityParser.T__39);
			this.state = 1087;
			this.identifier();
			this.state = 1088;
			this.match(SolidityParser.T__22);
			this.state = 1090;
			this._errHandler.sync(this);
			_la = this._input.LA(1);
			if ((((_la) & ~0x1F) === 0 && ((1 << _la) & 234897408) !== 0) || ((((_la - 46)) & ~0x1F) === 0 && ((1 << (_la - 46)) & 262209) !== 0) || ((((_la - 97)) & ~0x1F) === 0 && ((1 << (_la - 97)) & 3230138369) !== 0) || _la===130 || _la===131) {
				{
				this.state = 1089;
				this.assemblyIdentifierList();
				}
			}

			this.state = 1092;
			this.match(SolidityParser.T__23);
			this.state = 1094;
			this._errHandler.sync(this);
			_la = this._input.LA(1);
			if (_la===96) {
				{
				this.state = 1093;
				this.assemblyFunctionReturns();
				}
			}

			this.state = 1096;
			this.assemblyBlock();
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public assemblyFunctionReturns(): AssemblyFunctionReturnsContext {
		let localctx: AssemblyFunctionReturnsContext = new AssemblyFunctionReturnsContext(this, this._ctx, this.state);
		this.enterRule(localctx, 184, SolidityParser.RULE_assemblyFunctionReturns);
		try {
			this.enterOuterAlt(localctx, 1);
			{
			{
			this.state = 1098;
			this.match(SolidityParser.T__95);
			this.state = 1099;
			this.assemblyIdentifierList();
			}
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public assemblyFor(): AssemblyForContext {
		let localctx: AssemblyForContext = new AssemblyForContext(this, this._ctx, this.state);
		this.enterRule(localctx, 186, SolidityParser.RULE_assemblyFor);
		try {
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 1101;
			this.match(SolidityParser.T__28);
			this.state = 1104;
			this._errHandler.sync(this);
			switch (this._input.LA(1)) {
			case 15:
				{
				this.state = 1102;
				this.assemblyBlock();
				}
				break;
			case 14:
			case 25:
			case 26:
			case 27:
			case 46:
			case 52:
			case 61:
			case 64:
			case 68:
			case 97:
			case 104:
			case 105:
			case 106:
			case 108:
			case 116:
			case 120:
			case 127:
			case 128:
			case 130:
			case 131:
			case 132:
				{
				this.state = 1103;
				this.assemblyExpression();
				}
				break;
			default:
				throw new NoViableAltException(this);
			}
			this.state = 1106;
			this.assemblyExpression();
			this.state = 1109;
			this._errHandler.sync(this);
			switch (this._input.LA(1)) {
			case 15:
				{
				this.state = 1107;
				this.assemblyBlock();
				}
				break;
			case 14:
			case 25:
			case 26:
			case 27:
			case 46:
			case 52:
			case 61:
			case 64:
			case 68:
			case 97:
			case 104:
			case 105:
			case 106:
			case 108:
			case 116:
			case 120:
			case 127:
			case 128:
			case 130:
			case 131:
			case 132:
				{
				this.state = 1108;
				this.assemblyExpression();
				}
				break;
			default:
				throw new NoViableAltException(this);
			}
			this.state = 1111;
			this.assemblyBlock();
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public assemblyIf(): AssemblyIfContext {
		let localctx: AssemblyIfContext = new AssemblyIfContext(this, this._ctx, this.state);
		this.enterRule(localctx, 188, SolidityParser.RULE_assemblyIf);
		try {
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 1113;
			this.match(SolidityParser.T__52);
			this.state = 1114;
			this.assemblyExpression();
			this.state = 1115;
			this.assemblyBlock();
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public assemblyLiteral(): AssemblyLiteralContext {
		let localctx: AssemblyLiteralContext = new AssemblyLiteralContext(this, this._ctx, this.state);
		this.enterRule(localctx, 190, SolidityParser.RULE_assemblyLiteral);
		try {
			this.state = 1122;
			this._errHandler.sync(this);
			switch (this._input.LA(1)) {
			case 132:
				this.enterOuterAlt(localctx, 1);
				{
				this.state = 1117;
				this.stringLiteral();
				}
				break;
			case 105:
				this.enterOuterAlt(localctx, 2);
				{
				this.state = 1118;
				this.match(SolidityParser.DecimalNumber);
				}
				break;
			case 106:
				this.enterOuterAlt(localctx, 3);
				{
				this.state = 1119;
				this.match(SolidityParser.HexNumber);
				}
				break;
			case 108:
				this.enterOuterAlt(localctx, 4);
				{
				this.state = 1120;
				this.hexLiteral();
				}
				break;
			case 104:
				this.enterOuterAlt(localctx, 5);
				{
				this.state = 1121;
				this.match(SolidityParser.BooleanLiteral);
				}
				break;
			default:
				throw new NoViableAltException(this);
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public tupleExpression(): TupleExpressionContext {
		let localctx: TupleExpressionContext = new TupleExpressionContext(this, this._ctx, this.state);
		this.enterRule(localctx, 192, SolidityParser.RULE_tupleExpression);
		let _la: number;
		try {
			this.state = 1150;
			this._errHandler.sync(this);
			switch (this._input.LA(1)) {
			case 23:
				this.enterOuterAlt(localctx, 1);
				{
				this.state = 1124;
				this.match(SolidityParser.T__22);
				{
				this.state = 1126;
				this._errHandler.sync(this);
				_la = this._input.LA(1);
				if (((((_la - 6)) & ~0x1F) === 0 && ((1 << (_la - 6)) & 205127937) !== 0) || ((((_la - 40)) & ~0x1F) === 0 && ((1 << (_la - 40)) & 4278194513) !== 0) || ((((_la - 73)) & ~0x1F) === 0 && ((1 << (_la - 73)) & 4244635651) !== 0) || ((((_la - 105)) & ~0x1F) === 0 && ((1 << (_la - 105)) & 248547339) !== 0)) {
					{
					this.state = 1125;
					this.expression(0);
					}
				}

				this.state = 1134;
				this._errHandler.sync(this);
				_la = this._input.LA(1);
				while (_la===16) {
					{
					{
					this.state = 1128;
					this.match(SolidityParser.T__15);
					this.state = 1130;
					this._errHandler.sync(this);
					_la = this._input.LA(1);
					if (((((_la - 6)) & ~0x1F) === 0 && ((1 << (_la - 6)) & 205127937) !== 0) || ((((_la - 40)) & ~0x1F) === 0 && ((1 << (_la - 40)) & 4278194513) !== 0) || ((((_la - 73)) & ~0x1F) === 0 && ((1 << (_la - 73)) & 4244635651) !== 0) || ((((_la - 105)) & ~0x1F) === 0 && ((1 << (_la - 105)) & 248547339) !== 0)) {
						{
						this.state = 1129;
						this.expression(0);
						}
					}

					}
					}
					this.state = 1136;
					this._errHandler.sync(this);
					_la = this._input.LA(1);
				}
				}
				this.state = 1137;
				this.match(SolidityParser.T__23);
				}
				break;
			case 44:
				this.enterOuterAlt(localctx, 2);
				{
				this.state = 1138;
				this.match(SolidityParser.T__43);
				this.state = 1147;
				this._errHandler.sync(this);
				_la = this._input.LA(1);
				if (((((_la - 6)) & ~0x1F) === 0 && ((1 << (_la - 6)) & 205127937) !== 0) || ((((_la - 40)) & ~0x1F) === 0 && ((1 << (_la - 40)) & 4278194513) !== 0) || ((((_la - 73)) & ~0x1F) === 0 && ((1 << (_la - 73)) & 4244635651) !== 0) || ((((_la - 105)) & ~0x1F) === 0 && ((1 << (_la - 105)) & 248547339) !== 0)) {
					{
					this.state = 1139;
					this.expression(0);
					this.state = 1144;
					this._errHandler.sync(this);
					_la = this._input.LA(1);
					while (_la===16) {
						{
						{
						this.state = 1140;
						this.match(SolidityParser.T__15);
						this.state = 1141;
						this.expression(0);
						}
						}
						this.state = 1146;
						this._errHandler.sync(this);
						_la = this._input.LA(1);
					}
					}
				}

				this.state = 1149;
				this.match(SolidityParser.T__44);
				}
				break;
			default:
				throw new NoViableAltException(this);
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public numberLiteral(): NumberLiteralContext {
		let localctx: NumberLiteralContext = new NumberLiteralContext(this, this._ctx, this.state);
		this.enterRule(localctx, 194, SolidityParser.RULE_numberLiteral);
		let _la: number;
		try {
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 1152;
			_la = this._input.LA(1);
			if(!(_la===105 || _la===106)) {
			this._errHandler.recoverInline(this);
			}
			else {
				this._errHandler.reportMatch(this);
			    this.consume();
			}
			this.state = 1154;
			this._errHandler.sync(this);
			switch ( this._interp.adaptivePredict(this._input, 127, this._ctx) ) {
			case 1:
				{
				this.state = 1153;
				this.match(SolidityParser.NumberUnit);
				}
				break;
			}
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public identifier(): IdentifierContext {
		let localctx: IdentifierContext = new IdentifierContext(this, this._ctx, this.state);
		this.enterRule(localctx, 196, SolidityParser.RULE_identifier);
		let _la: number;
		try {
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 1156;
			_la = this._input.LA(1);
			if(!((((_la) & ~0x1F) === 0 && ((1 << _la) & 234897408) !== 0) || ((((_la - 46)) & ~0x1F) === 0 && ((1 << (_la - 46)) & 262209) !== 0) || ((((_la - 97)) & ~0x1F) === 0 && ((1 << (_la - 97)) & 3230138369) !== 0) || _la===130 || _la===131)) {
			this._errHandler.recoverInline(this);
			}
			else {
				this._errHandler.reportMatch(this);
			    this.consume();
			}
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public hexLiteral(): HexLiteralContext {
		let localctx: HexLiteralContext = new HexLiteralContext(this, this._ctx, this.state);
		this.enterRule(localctx, 198, SolidityParser.RULE_hexLiteral);
		try {
			let _alt: number;
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 1159;
			this._errHandler.sync(this);
			_alt = 1;
			do {
				switch (_alt) {
				case 1:
					{
					{
					this.state = 1158;
					this.match(SolidityParser.HexLiteralFragment);
					}
					}
					break;
				default:
					throw new NoViableAltException(this);
				}
				this.state = 1161;
				this._errHandler.sync(this);
				_alt = this._interp.adaptivePredict(this._input, 128, this._ctx);
			} while (_alt !== 2 && _alt !== ATN.INVALID_ALT_NUMBER);
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public overrideSpecifier(): OverrideSpecifierContext {
		let localctx: OverrideSpecifierContext = new OverrideSpecifierContext(this, this._ctx, this.state);
		this.enterRule(localctx, 200, SolidityParser.RULE_overrideSpecifier);
		let _la: number;
		try {
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 1163;
			this.match(SolidityParser.T__97);
			this.state = 1175;
			this._errHandler.sync(this);
			_la = this._input.LA(1);
			if (_la===23) {
				{
				this.state = 1164;
				this.match(SolidityParser.T__22);
				this.state = 1165;
				this.userDefinedTypeName();
				this.state = 1170;
				this._errHandler.sync(this);
				_la = this._input.LA(1);
				while (_la===16) {
					{
					{
					this.state = 1166;
					this.match(SolidityParser.T__15);
					this.state = 1167;
					this.userDefinedTypeName();
					}
					}
					this.state = 1172;
					this._errHandler.sync(this);
					_la = this._input.LA(1);
				}
				this.state = 1173;
				this.match(SolidityParser.T__23);
				}
			}

			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}
	// @RuleVersion(0)
	public stringLiteral(): StringLiteralContext {
		let localctx: StringLiteralContext = new StringLiteralContext(this, this._ctx, this.state);
		this.enterRule(localctx, 202, SolidityParser.RULE_stringLiteral);
		try {
			let _alt: number;
			this.enterOuterAlt(localctx, 1);
			{
			this.state = 1178;
			this._errHandler.sync(this);
			_alt = 1;
			do {
				switch (_alt) {
				case 1:
					{
					{
					this.state = 1177;
					this.match(SolidityParser.StringLiteralFragment);
					}
					}
					break;
				default:
					throw new NoViableAltException(this);
				}
				this.state = 1180;
				this._errHandler.sync(this);
				_alt = this._interp.adaptivePredict(this._input, 131, this._ctx);
			} while (_alt !== 2 && _alt !== ATN.INVALID_ALT_NUMBER);
			}
		}
		catch (re) {
			if (re instanceof RecognitionException) {
				localctx.exception = re;
				this._errHandler.reportError(this, re);
				this._errHandler.recover(this, re);
			} else {
				throw re;
			}
		}
		finally {
			this.exitRule();
		}
		return localctx;
	}

	public sempred(localctx: RuleContext, ruleIndex: number, predIndex: number): boolean {
		switch (ruleIndex) {
		case 39:
			return this.typeName_sempred(localctx as TypeNameContext, predIndex);
		case 71:
			return this.expression_sempred(localctx as ExpressionContext, predIndex);
		}
		return true;
	}
	private typeName_sempred(localctx: TypeNameContext, predIndex: number): boolean {
		switch (predIndex) {
		case 0:
			return this.precpred(this._ctx, 3);
		}
		return true;
	}
	private expression_sempred(localctx: ExpressionContext, predIndex: number): boolean {
		switch (predIndex) {
		case 1:
			return this.precpred(this._ctx, 14);
		case 2:
			return this.precpred(this._ctx, 13);
		case 3:
			return this.precpred(this._ctx, 12);
		case 4:
			return this.precpred(this._ctx, 11);
		case 5:
			return this.precpred(this._ctx, 10);
		case 6:
			return this.precpred(this._ctx, 9);
		case 7:
			return this.precpred(this._ctx, 8);
		case 8:
			return this.precpred(this._ctx, 7);
		case 9:
			return this.precpred(this._ctx, 6);
		case 10:
			return this.precpred(this._ctx, 5);
		case 11:
			return this.precpred(this._ctx, 4);
		case 12:
			return this.precpred(this._ctx, 3);
		case 13:
			return this.precpred(this._ctx, 2);
		case 14:
			return this.precpred(this._ctx, 27);
		case 15:
			return this.precpred(this._ctx, 25);
		case 16:
			return this.precpred(this._ctx, 24);
		case 17:
			return this.precpred(this._ctx, 23);
		case 18:
			return this.precpred(this._ctx, 22);
		case 19:
			return this.precpred(this._ctx, 21);
		}
		return true;
	}

	public static readonly _serializedATN: number[] = [4,1,136,1183,2,0,7,0,
	2,1,7,1,2,2,7,2,2,3,7,3,2,4,7,4,2,5,7,5,2,6,7,6,2,7,7,7,2,8,7,8,2,9,7,9,
	2,10,7,10,2,11,7,11,2,12,7,12,2,13,7,13,2,14,7,14,2,15,7,15,2,16,7,16,2,
	17,7,17,2,18,7,18,2,19,7,19,2,20,7,20,2,21,7,21,2,22,7,22,2,23,7,23,2,24,
	7,24,2,25,7,25,2,26,7,26,2,27,7,27,2,28,7,28,2,29,7,29,2,30,7,30,2,31,7,
	31,2,32,7,32,2,33,7,33,2,34,7,34,2,35,7,35,2,36,7,36,2,37,7,37,2,38,7,38,
	2,39,7,39,2,40,7,40,2,41,7,41,2,42,7,42,2,43,7,43,2,44,7,44,2,45,7,45,2,
	46,7,46,2,47,7,47,2,48,7,48,2,49,7,49,2,50,7,50,2,51,7,51,2,52,7,52,2,53,
	7,53,2,54,7,54,2,55,7,55,2,56,7,56,2,57,7,57,2,58,7,58,2,59,7,59,2,60,7,
	60,2,61,7,61,2,62,7,62,2,63,7,63,2,64,7,64,2,65,7,65,2,66,7,66,2,67,7,67,
	2,68,7,68,2,69,7,69,2,70,7,70,2,71,7,71,2,72,7,72,2,73,7,73,2,74,7,74,2,
	75,7,75,2,76,7,76,2,77,7,77,2,78,7,78,2,79,7,79,2,80,7,80,2,81,7,81,2,82,
	7,82,2,83,7,83,2,84,7,84,2,85,7,85,2,86,7,86,2,87,7,87,2,88,7,88,2,89,7,
	89,2,90,7,90,2,91,7,91,2,92,7,92,2,93,7,93,2,94,7,94,2,95,7,95,2,96,7,96,
	2,97,7,97,2,98,7,98,2,99,7,99,2,100,7,100,2,101,7,101,1,0,1,0,1,0,1,0,1,
	0,1,0,1,0,1,0,1,0,1,0,1,0,5,0,216,8,0,10,0,12,0,219,9,0,1,0,1,0,1,1,1,1,
	1,1,1,1,1,1,1,2,1,2,1,3,1,3,1,3,3,3,233,8,3,1,4,1,4,3,4,237,8,4,1,4,5,4,
	240,8,4,10,4,12,4,243,9,4,1,5,1,5,1,6,3,6,248,8,6,1,6,1,6,3,6,252,8,6,1,
	6,3,6,255,8,6,1,7,1,7,1,7,3,7,260,8,7,1,8,1,8,1,8,1,8,3,8,266,8,8,1,8,1,
	8,1,8,1,8,1,8,3,8,273,8,8,1,8,1,8,3,8,277,8,8,1,8,1,8,1,8,1,8,1,8,1,8,1,
	8,1,8,1,8,5,8,288,8,8,10,8,12,8,291,9,8,1,8,1,8,1,8,1,8,1,8,3,8,298,8,8,
	1,9,1,9,1,10,3,10,303,8,10,1,10,1,10,1,10,3,10,308,8,10,1,10,1,10,1,10,
	1,10,5,10,314,8,10,10,10,12,10,317,9,10,3,10,319,8,10,1,10,3,10,322,8,10,
	1,10,1,10,5,10,326,8,10,10,10,12,10,329,9,10,1,10,1,10,1,11,1,11,1,11,3,
	11,336,8,11,1,11,3,11,339,8,11,1,12,1,12,1,12,1,12,1,13,1,13,1,13,1,13,
	1,13,1,13,1,13,1,13,1,13,3,13,354,8,13,1,14,1,14,1,14,1,14,1,14,1,14,1,
	14,1,14,5,14,364,8,14,10,14,12,14,367,9,14,1,14,1,14,1,14,3,14,372,8,14,
	1,14,1,14,1,15,1,15,1,15,1,15,1,15,1,15,1,15,1,16,1,16,1,16,1,16,1,16,1,
	17,1,17,1,17,1,17,1,17,1,17,1,18,1,18,1,18,1,18,1,18,3,18,399,8,18,1,18,
	3,18,402,8,18,1,18,1,18,1,19,1,19,1,19,1,19,1,19,5,19,411,8,19,10,19,12,
	19,414,9,19,1,19,1,19,3,19,418,8,19,1,20,1,20,1,20,3,20,423,8,20,1,21,1,
	21,1,22,1,22,1,22,1,22,1,22,1,22,1,22,1,22,5,22,435,8,22,10,22,12,22,438,
	9,22,3,22,440,8,22,1,22,1,22,1,23,1,23,1,23,3,23,447,8,23,1,23,1,23,5,23,
	451,8,23,10,23,12,23,454,9,23,1,23,1,23,3,23,458,8,23,1,24,1,24,1,24,3,
	24,463,8,24,1,24,3,24,466,8,24,1,25,1,25,1,25,1,25,3,25,472,8,25,1,25,1,
	25,3,25,476,8,25,1,26,1,26,3,26,480,8,26,1,26,1,26,1,26,3,26,485,8,26,1,
	27,1,27,1,27,1,28,1,28,1,28,1,28,1,28,1,28,1,28,1,28,5,28,498,8,28,10,28,
	12,28,501,9,28,1,29,1,29,1,29,1,29,3,29,507,8,29,1,29,1,29,1,30,1,30,1,
	31,1,31,1,31,1,31,3,31,517,8,31,1,31,1,31,5,31,521,8,31,10,31,12,31,524,
	9,31,1,31,1,31,1,32,1,32,1,32,1,32,5,32,532,8,32,10,32,12,32,535,9,32,3,
	32,537,8,32,1,32,1,32,1,33,1,33,3,33,543,8,33,1,33,3,33,546,8,33,1,34,1,
	34,1,34,1,34,5,34,552,8,34,10,34,12,34,555,9,34,3,34,557,8,34,1,34,1,34,
	1,35,1,35,3,35,563,8,35,1,35,3,35,566,8,35,1,36,1,36,1,36,1,36,5,36,572,
	8,36,10,36,12,36,575,9,36,3,36,577,8,36,1,36,1,36,1,37,1,37,3,37,583,8,
	37,1,38,1,38,3,38,587,8,38,1,38,1,38,1,39,1,39,1,39,1,39,1,39,1,39,1,39,
	3,39,598,8,39,1,39,1,39,1,39,3,39,603,8,39,1,39,5,39,606,8,39,10,39,12,
	39,609,9,39,1,40,1,40,1,40,5,40,614,8,40,10,40,12,40,617,9,40,1,41,1,41,
	3,41,621,8,41,1,42,1,42,1,42,1,42,3,42,627,8,42,1,42,1,42,1,42,3,42,632,
	8,42,1,42,1,42,1,43,1,43,1,44,1,44,1,45,1,45,1,45,1,45,1,45,5,45,645,8,
	45,10,45,12,45,648,9,45,1,45,1,45,3,45,652,8,45,1,46,1,46,1,47,1,47,1,48,
	1,48,5,48,660,8,48,10,48,12,48,663,9,48,1,48,1,48,1,49,1,49,1,49,1,49,1,
	49,1,49,1,49,1,49,1,49,1,49,1,49,1,49,1,49,1,49,1,49,3,49,682,8,49,1,50,
	1,50,1,50,1,51,1,51,1,51,1,51,1,51,1,51,1,51,3,51,694,8,51,1,52,1,52,1,
	52,3,52,699,8,52,1,52,1,52,4,52,703,8,52,11,52,12,52,704,1,53,1,53,3,53,
	709,8,53,1,53,3,53,712,8,53,1,53,1,53,1,54,1,54,1,54,1,54,1,54,1,54,1,55,
	1,55,3,55,724,8,55,1,56,1,56,1,56,1,57,1,57,1,57,1,57,3,57,733,8,57,1,57,
	1,57,3,57,737,8,57,1,57,3,57,740,8,57,1,57,1,57,1,57,1,58,1,58,3,58,747,
	8,58,1,58,1,58,1,58,1,58,3,58,753,8,58,1,58,1,58,1,59,1,59,1,60,1,60,1,
	60,1,60,1,60,1,60,1,60,1,60,1,61,1,61,1,61,1,62,1,62,1,62,1,63,1,63,3,63,
	775,8,63,1,63,1,63,1,64,1,64,1,64,1,65,1,65,1,65,1,65,1,66,1,66,1,66,1,
	66,1,67,1,67,1,67,1,67,1,67,1,67,1,67,3,67,797,8,67,1,67,1,67,3,67,801,
	8,67,1,67,1,67,1,68,3,68,806,8,68,1,68,1,68,3,68,810,8,68,5,68,812,8,68,
	10,68,12,68,815,9,68,1,69,1,69,3,69,819,8,69,1,69,5,69,822,8,69,10,69,12,
	69,825,9,69,1,69,3,69,828,8,69,1,69,1,69,1,70,1,70,1,71,1,71,1,71,1,71,
	1,71,1,71,1,71,1,71,1,71,1,71,1,71,1,71,1,71,1,71,1,71,1,71,1,71,1,71,3,
	71,852,8,71,1,71,1,71,1,71,1,71,1,71,1,71,1,71,1,71,1,71,1,71,1,71,1,71,
	1,71,1,71,1,71,1,71,1,71,1,71,1,71,1,71,1,71,1,71,1,71,1,71,1,71,1,71,1,
	71,1,71,1,71,1,71,1,71,1,71,1,71,1,71,1,71,1,71,1,71,1,71,1,71,1,71,1,71,
	1,71,1,71,1,71,1,71,1,71,1,71,1,71,1,71,1,71,1,71,1,71,3,71,906,8,71,1,
	71,1,71,3,71,910,8,71,1,71,1,71,1,71,1,71,1,71,1,71,1,71,1,71,1,71,1,71,
	1,71,1,71,1,71,1,71,5,71,926,8,71,10,71,12,71,929,9,71,1,72,1,72,1,72,1,
	72,1,72,1,72,1,72,1,72,1,72,3,72,940,8,72,1,73,1,73,1,73,5,73,945,8,73,
	10,73,12,73,948,9,73,1,74,1,74,1,74,5,74,953,8,74,10,74,12,74,956,9,74,
	1,74,3,74,959,8,74,1,75,1,75,1,75,1,75,1,76,1,76,3,76,967,8,76,1,76,1,76,
	3,76,971,8,76,3,76,973,8,76,1,77,1,77,1,77,1,77,1,77,1,78,1,78,5,78,982,
	8,78,10,78,12,78,985,9,78,1,78,1,78,1,79,1,79,1,79,1,79,1,79,1,79,1,79,
	1,79,1,79,1,79,1,79,1,79,1,79,1,79,1,79,1,79,1,79,3,79,1006,8,79,1,80,1,
	80,1,80,3,80,1011,8,80,1,81,1,81,1,81,1,81,1,82,1,82,1,82,1,82,3,82,1021,
	8,82,1,82,1,82,3,82,1025,8,82,1,82,1,82,5,82,1029,8,82,10,82,12,82,1032,
	9,82,1,82,3,82,1035,8,82,1,83,1,83,1,83,1,83,3,83,1041,8,83,1,84,1,84,1,
	84,1,84,1,85,1,85,1,85,1,85,1,85,1,85,1,85,3,85,1054,8,85,1,86,1,86,1,86,
	5,86,1059,8,86,10,86,12,86,1062,9,86,1,87,1,87,1,87,1,87,1,88,1,88,1,88,
	1,89,1,89,1,89,5,89,1074,8,89,10,89,12,89,1077,9,89,1,90,1,90,1,90,1,90,
	1,90,1,90,3,90,1085,8,90,1,91,1,91,1,91,1,91,3,91,1091,8,91,1,91,1,91,3,
	91,1095,8,91,1,91,1,91,1,92,1,92,1,92,1,93,1,93,1,93,3,93,1105,8,93,1,93,
	1,93,1,93,3,93,1110,8,93,1,93,1,93,1,94,1,94,1,94,1,94,1,95,1,95,1,95,1,
	95,1,95,3,95,1123,8,95,1,96,1,96,3,96,1127,8,96,1,96,1,96,3,96,1131,8,96,
	5,96,1133,8,96,10,96,12,96,1136,9,96,1,96,1,96,1,96,1,96,1,96,5,96,1143,
	8,96,10,96,12,96,1146,9,96,3,96,1148,8,96,1,96,3,96,1151,8,96,1,97,1,97,
	3,97,1155,8,97,1,98,1,98,1,99,4,99,1160,8,99,11,99,12,99,1161,1,100,1,100,
	1,100,1,100,1,100,5,100,1169,8,100,10,100,12,100,1172,9,100,1,100,1,100,
	3,100,1176,8,100,1,101,4,101,1179,8,101,11,101,12,101,1180,1,101,0,2,78,
	142,102,0,2,4,6,8,10,12,14,16,18,20,22,24,26,28,30,32,34,36,38,40,42,44,
	46,48,50,52,54,56,58,60,62,64,66,68,70,72,74,76,78,80,82,84,86,88,90,92,
	94,96,98,100,102,104,106,108,110,112,114,116,118,120,122,124,126,128,130,
	132,134,136,138,140,142,144,146,148,150,152,154,156,158,160,162,164,166,
	168,170,172,174,176,178,180,182,184,186,188,190,192,194,196,198,200,202,
	0,15,1,0,5,11,1,0,19,21,3,0,3,3,5,10,30,37,1,0,50,52,4,0,112,112,120,120,
	124,124,126,126,3,0,46,46,65,68,99,103,1,0,69,70,1,0,32,33,2,0,3,3,34,35,
	1,0,76,77,1,0,7,10,1,0,36,37,2,0,11,11,80,89,1,0,105,106,10,0,14,14,25,
	27,46,46,52,52,64,64,97,97,116,116,120,120,127,128,130,131,1314,0,217,1,
	0,0,0,2,222,1,0,0,0,4,227,1,0,0,0,6,232,1,0,0,0,8,234,1,0,0,0,10,244,1,
	0,0,0,12,254,1,0,0,0,14,256,1,0,0,0,16,297,1,0,0,0,18,299,1,0,0,0,20,302,
	1,0,0,0,22,332,1,0,0,0,24,340,1,0,0,0,26,353,1,0,0,0,28,355,1,0,0,0,30,
	375,1,0,0,0,32,382,1,0,0,0,34,387,1,0,0,0,36,393,1,0,0,0,38,417,1,0,0,0,
	40,419,1,0,0,0,42,424,1,0,0,0,44,426,1,0,0,0,46,443,1,0,0,0,48,459,1,0,
	0,0,50,467,1,0,0,0,52,484,1,0,0,0,54,486,1,0,0,0,56,499,1,0,0,0,58,502,
	1,0,0,0,60,510,1,0,0,0,62,512,1,0,0,0,64,527,1,0,0,0,66,540,1,0,0,0,68,
	547,1,0,0,0,70,560,1,0,0,0,72,567,1,0,0,0,74,580,1,0,0,0,76,584,1,0,0,0,
	78,597,1,0,0,0,80,610,1,0,0,0,82,620,1,0,0,0,84,622,1,0,0,0,86,635,1,0,
	0,0,88,637,1,0,0,0,90,639,1,0,0,0,92,653,1,0,0,0,94,655,1,0,0,0,96,657,
	1,0,0,0,98,681,1,0,0,0,100,683,1,0,0,0,102,686,1,0,0,0,104,695,1,0,0,0,
	106,706,1,0,0,0,108,715,1,0,0,0,110,723,1,0,0,0,112,725,1,0,0,0,114,728,
	1,0,0,0,116,744,1,0,0,0,118,756,1,0,0,0,120,758,1,0,0,0,122,766,1,0,0,0,
	124,769,1,0,0,0,126,772,1,0,0,0,128,778,1,0,0,0,130,781,1,0,0,0,132,785,
	1,0,0,0,134,796,1,0,0,0,136,805,1,0,0,0,138,816,1,0,0,0,140,831,1,0,0,0,
	142,851,1,0,0,0,144,939,1,0,0,0,146,941,1,0,0,0,148,949,1,0,0,0,150,960,
	1,0,0,0,152,972,1,0,0,0,154,974,1,0,0,0,156,979,1,0,0,0,158,1005,1,0,0,
	0,160,1010,1,0,0,0,162,1012,1,0,0,0,164,1020,1,0,0,0,166,1036,1,0,0,0,168,
	1042,1,0,0,0,170,1053,1,0,0,0,172,1055,1,0,0,0,174,1063,1,0,0,0,176,1067,
	1,0,0,0,178,1070,1,0,0,0,180,1084,1,0,0,0,182,1086,1,0,0,0,184,1098,1,0,
	0,0,186,1101,1,0,0,0,188,1113,1,0,0,0,190,1122,1,0,0,0,192,1150,1,0,0,0,
	194,1152,1,0,0,0,196,1156,1,0,0,0,198,1159,1,0,0,0,200,1163,1,0,0,0,202,
	1178,1,0,0,0,204,216,3,2,1,0,205,216,3,16,8,0,206,216,3,20,10,0,207,216,
	3,62,31,0,208,216,3,58,29,0,209,216,3,44,22,0,210,216,3,50,25,0,211,216,
	3,30,15,0,212,216,3,32,16,0,213,216,3,34,17,0,214,216,3,36,18,0,215,204,
	1,0,0,0,215,205,1,0,0,0,215,206,1,0,0,0,215,207,1,0,0,0,215,208,1,0,0,0,
	215,209,1,0,0,0,215,210,1,0,0,0,215,211,1,0,0,0,215,212,1,0,0,0,215,213,
	1,0,0,0,215,214,1,0,0,0,216,219,1,0,0,0,217,215,1,0,0,0,217,218,1,0,0,0,
	218,220,1,0,0,0,219,217,1,0,0,0,220,221,5,0,0,1,221,1,1,0,0,0,222,223,5,
	1,0,0,223,224,3,4,2,0,224,225,3,6,3,0,225,226,5,2,0,0,226,3,1,0,0,0,227,
	228,3,196,98,0,228,5,1,0,0,0,229,233,5,3,0,0,230,233,3,8,4,0,231,233,3,
	142,71,0,232,229,1,0,0,0,232,230,1,0,0,0,232,231,1,0,0,0,233,7,1,0,0,0,
	234,241,3,12,6,0,235,237,5,4,0,0,236,235,1,0,0,0,236,237,1,0,0,0,237,238,
	1,0,0,0,238,240,3,12,6,0,239,236,1,0,0,0,240,243,1,0,0,0,241,239,1,0,0,
	0,241,242,1,0,0,0,242,9,1,0,0,0,243,241,1,0,0,0,244,245,7,0,0,0,245,11,
	1,0,0,0,246,248,3,10,5,0,247,246,1,0,0,0,247,248,1,0,0,0,248,249,1,0,0,
	0,249,255,5,133,0,0,250,252,3,10,5,0,251,250,1,0,0,0,251,252,1,0,0,0,252,
	253,1,0,0,0,253,255,5,105,0,0,254,247,1,0,0,0,254,251,1,0,0,0,255,13,1,
	0,0,0,256,259,3,196,98,0,257,258,5,12,0,0,258,260,3,196,98,0,259,257,1,
	0,0,0,259,260,1,0,0,0,260,15,1,0,0,0,261,262,5,13,0,0,262,265,3,18,9,0,
	263,264,5,12,0,0,264,266,3,196,98,0,265,263,1,0,0,0,265,266,1,0,0,0,266,
	267,1,0,0,0,267,268,5,2,0,0,268,298,1,0,0,0,269,272,5,13,0,0,270,273,5,
	3,0,0,271,273,3,196,98,0,272,270,1,0,0,0,272,271,1,0,0,0,273,276,1,0,0,
	0,274,275,5,12,0,0,275,277,3,196,98,0,276,274,1,0,0,0,276,277,1,0,0,0,277,
	278,1,0,0,0,278,279,5,14,0,0,279,280,3,18,9,0,280,281,5,2,0,0,281,298,1,
	0,0,0,282,283,5,13,0,0,283,284,5,15,0,0,284,289,3,14,7,0,285,286,5,16,0,
	0,286,288,3,14,7,0,287,285,1,0,0,0,288,291,1,0,0,0,289,287,1,0,0,0,289,
	290,1,0,0,0,290,292,1,0,0,0,291,289,1,0,0,0,292,293,5,17,0,0,293,294,5,
	14,0,0,294,295,3,18,9,0,295,296,5,2,0,0,296,298,1,0,0,0,297,261,1,0,0,0,
	297,269,1,0,0,0,297,282,1,0,0,0,298,17,1,0,0,0,299,300,5,132,0,0,300,19,
	1,0,0,0,301,303,5,18,0,0,302,301,1,0,0,0,302,303,1,0,0,0,303,304,1,0,0,
	0,304,305,7,1,0,0,305,307,3,196,98,0,306,308,3,24,12,0,307,306,1,0,0,0,
	307,308,1,0,0,0,308,318,1,0,0,0,309,310,5,22,0,0,310,315,3,22,11,0,311,
	312,5,16,0,0,312,314,3,22,11,0,313,311,1,0,0,0,314,317,1,0,0,0,315,313,
	1,0,0,0,315,316,1,0,0,0,316,319,1,0,0,0,317,315,1,0,0,0,318,309,1,0,0,0,
	318,319,1,0,0,0,319,321,1,0,0,0,320,322,3,24,12,0,321,320,1,0,0,0,321,322,
	1,0,0,0,322,323,1,0,0,0,323,327,5,15,0,0,324,326,3,26,13,0,325,324,1,0,
	0,0,326,329,1,0,0,0,327,325,1,0,0,0,327,328,1,0,0,0,328,330,1,0,0,0,329,
	327,1,0,0,0,330,331,5,17,0,0,331,21,1,0,0,0,332,338,3,80,40,0,333,335,5,
	23,0,0,334,336,3,146,73,0,335,334,1,0,0,0,335,336,1,0,0,0,336,337,1,0,0,
	0,337,339,5,24,0,0,338,333,1,0,0,0,338,339,1,0,0,0,339,23,1,0,0,0,340,341,
	5,25,0,0,341,342,5,26,0,0,342,343,3,142,71,0,343,25,1,0,0,0,344,354,3,28,
	14,0,345,354,3,36,18,0,346,354,3,44,22,0,347,354,3,46,23,0,348,354,3,50,
	25,0,349,354,3,58,29,0,350,354,3,62,31,0,351,354,3,32,16,0,352,354,3,34,
	17,0,353,344,1,0,0,0,353,345,1,0,0,0,353,346,1,0,0,0,353,347,1,0,0,0,353,
	348,1,0,0,0,353,349,1,0,0,0,353,350,1,0,0,0,353,351,1,0,0,0,353,352,1,0,
	0,0,354,27,1,0,0,0,355,365,3,78,39,0,356,364,5,122,0,0,357,364,5,119,0,
	0,358,364,5,121,0,0,359,364,5,112,0,0,360,364,5,113,0,0,361,364,5,114,0,
	0,362,364,3,200,100,0,363,356,1,0,0,0,363,357,1,0,0,0,363,358,1,0,0,0,363,
	359,1,0,0,0,363,360,1,0,0,0,363,361,1,0,0,0,363,362,1,0,0,0,364,367,1,0,
	0,0,365,363,1,0,0,0,365,366,1,0,0,0,366,368,1,0,0,0,367,365,1,0,0,0,368,
	371,3,196,98,0,369,370,5,11,0,0,370,372,3,142,71,0,371,369,1,0,0,0,371,
	372,1,0,0,0,372,373,1,0,0,0,373,374,5,2,0,0,374,29,1,0,0,0,375,376,3,78,
	39,0,376,377,5,112,0,0,377,378,3,196,98,0,378,379,5,11,0,0,379,380,3,142,
	71,0,380,381,5,2,0,0,381,31,1,0,0,0,382,383,5,27,0,0,383,384,3,196,98,0,
	384,385,3,64,32,0,385,386,5,2,0,0,386,33,1,0,0,0,387,388,5,125,0,0,388,
	389,3,196,98,0,389,390,5,22,0,0,390,391,3,140,70,0,391,392,5,2,0,0,392,
	35,1,0,0,0,393,394,5,28,0,0,394,395,3,38,19,0,395,398,5,29,0,0,396,399,
	5,3,0,0,397,399,3,78,39,0,398,396,1,0,0,0,398,397,1,0,0,0,399,401,1,0,0,
	0,400,402,5,127,0,0,401,400,1,0,0,0,401,402,1,0,0,0,402,403,1,0,0,0,403,
	404,5,2,0,0,404,37,1,0,0,0,405,418,3,80,40,0,406,407,5,15,0,0,407,412,3,
	40,20,0,408,409,5,16,0,0,409,411,3,40,20,0,410,408,1,0,0,0,411,414,1,0,
	0,0,412,410,1,0,0,0,412,413,1,0,0,0,413,415,1,0,0,0,414,412,1,0,0,0,415,
	416,5,17,0,0,416,418,1,0,0,0,417,405,1,0,0,0,417,406,1,0,0,0,418,39,1,0,
	0,0,419,422,3,80,40,0,420,421,5,12,0,0,421,423,3,42,21,0,422,420,1,0,0,
	0,422,423,1,0,0,0,423,41,1,0,0,0,424,425,7,2,0,0,425,43,1,0,0,0,426,427,
	5,38,0,0,427,428,3,196,98,0,428,439,5,15,0,0,429,430,3,76,38,0,430,436,
	5,2,0,0,431,432,3,76,38,0,432,433,5,2,0,0,433,435,1,0,0,0,434,431,1,0,0,
	0,435,438,1,0,0,0,436,434,1,0,0,0,436,437,1,0,0,0,437,440,1,0,0,0,438,436,
	1,0,0,0,439,429,1,0,0,0,439,440,1,0,0,0,440,441,1,0,0,0,441,442,5,17,0,
	0,442,45,1,0,0,0,443,444,5,39,0,0,444,446,3,196,98,0,445,447,3,64,32,0,
	446,445,1,0,0,0,446,447,1,0,0,0,447,452,1,0,0,0,448,451,5,123,0,0,449,451,
	3,200,100,0,450,448,1,0,0,0,450,449,1,0,0,0,451,454,1,0,0,0,452,450,1,0,
	0,0,452,453,1,0,0,0,453,457,1,0,0,0,454,452,1,0,0,0,455,458,5,2,0,0,456,
	458,3,96,48,0,457,455,1,0,0,0,457,456,1,0,0,0,458,47,1,0,0,0,459,465,3,
	196,98,0,460,462,5,23,0,0,461,463,3,146,73,0,462,461,1,0,0,0,462,463,1,
	0,0,0,463,464,1,0,0,0,464,466,5,24,0,0,465,460,1,0,0,0,465,466,1,0,0,0,
	466,49,1,0,0,0,467,468,3,52,26,0,468,469,3,64,32,0,469,471,3,56,28,0,470,
	472,3,54,27,0,471,470,1,0,0,0,471,472,1,0,0,0,472,475,1,0,0,0,473,476,5,
	2,0,0,474,476,3,96,48,0,475,473,1,0,0,0,475,474,1,0,0,0,476,51,1,0,0,0,
	477,479,5,40,0,0,478,480,3,196,98,0,479,478,1,0,0,0,479,480,1,0,0,0,480,
	485,1,0,0,0,481,485,5,128,0,0,482,485,5,129,0,0,483,485,5,130,0,0,484,477,
	1,0,0,0,484,481,1,0,0,0,484,482,1,0,0,0,484,483,1,0,0,0,485,53,1,0,0,0,
	486,487,5,41,0,0,487,488,3,64,32,0,488,55,1,0,0,0,489,498,5,117,0,0,490,
	498,5,122,0,0,491,498,5,119,0,0,492,498,5,121,0,0,493,498,5,123,0,0,494,
	498,3,94,47,0,495,498,3,48,24,0,496,498,3,200,100,0,497,489,1,0,0,0,497,
	490,1,0,0,0,497,491,1,0,0,0,497,492,1,0,0,0,497,493,1,0,0,0,497,494,1,0,
	0,0,497,495,1,0,0,0,497,496,1,0,0,0,498,501,1,0,0,0,499,497,1,0,0,0,499,
	500,1,0,0,0,500,57,1,0,0,0,501,499,1,0,0,0,502,503,5,42,0,0,503,504,3,196,
	98,0,504,506,3,68,34,0,505,507,5,110,0,0,506,505,1,0,0,0,506,507,1,0,0,
	0,507,508,1,0,0,0,508,509,5,2,0,0,509,59,1,0,0,0,510,511,3,196,98,0,511,
	61,1,0,0,0,512,513,5,43,0,0,513,514,3,196,98,0,514,516,5,15,0,0,515,517,
	3,60,30,0,516,515,1,0,0,0,516,517,1,0,0,0,517,522,1,0,0,0,518,519,5,16,
	0,0,519,521,3,60,30,0,520,518,1,0,0,0,521,524,1,0,0,0,522,520,1,0,0,0,522,
	523,1,0,0,0,523,525,1,0,0,0,524,522,1,0,0,0,525,526,5,17,0,0,526,63,1,0,
	0,0,527,536,5,23,0,0,528,533,3,66,33,0,529,530,5,16,0,0,530,532,3,66,33,
	0,531,529,1,0,0,0,532,535,1,0,0,0,533,531,1,0,0,0,533,534,1,0,0,0,534,537,
	1,0,0,0,535,533,1,0,0,0,536,528,1,0,0,0,536,537,1,0,0,0,537,538,1,0,0,0,
	538,539,5,24,0,0,539,65,1,0,0,0,540,542,3,78,39,0,541,543,3,92,46,0,542,
	541,1,0,0,0,542,543,1,0,0,0,543,545,1,0,0,0,544,546,3,196,98,0,545,544,
	1,0,0,0,545,546,1,0,0,0,546,67,1,0,0,0,547,556,5,23,0,0,548,553,3,70,35,
	0,549,550,5,16,0,0,550,552,3,70,35,0,551,549,1,0,0,0,552,555,1,0,0,0,553,
	551,1,0,0,0,553,554,1,0,0,0,554,557,1,0,0,0,555,553,1,0,0,0,556,548,1,0,
	0,0,556,557,1,0,0,0,557,558,1,0,0,0,558,559,5,24,0,0,559,69,1,0,0,0,560,
	562,3,78,39,0,561,563,5,118,0,0,562,561,1,0,0,0,562,563,1,0,0,0,563,565,
	1,0,0,0,564,566,3,196,98,0,565,564,1,0,0,0,565,566,1,0,0,0,566,71,1,0,0,
	0,567,576,5,23,0,0,568,573,3,74,37,0,569,570,5,16,0,0,570,572,3,74,37,0,
	571,569,1,0,0,0,572,575,1,0,0,0,573,571,1,0,0,0,573,574,1,0,0,0,574,577,
	1,0,0,0,575,573,1,0,0,0,576,568,1,0,0,0,576,577,1,0,0,0,577,578,1,0,0,0,
	578,579,5,24,0,0,579,73,1,0,0,0,580,582,3,78,39,0,581,583,3,92,46,0,582,
	581,1,0,0,0,582,583,1,0,0,0,583,75,1,0,0,0,584,586,3,78,39,0,585,587,3,
	92,46,0,586,585,1,0,0,0,586,587,1,0,0,0,587,588,1,0,0,0,588,589,3,196,98,
	0,589,77,1,0,0,0,590,591,6,39,-1,0,591,598,3,140,70,0,592,598,3,80,40,0,
	593,598,3,84,42,0,594,598,3,90,45,0,595,596,5,46,0,0,596,598,5,120,0,0,
	597,590,1,0,0,0,597,592,1,0,0,0,597,593,1,0,0,0,597,594,1,0,0,0,597,595,
	1,0,0,0,598,607,1,0,0,0,599,600,10,3,0,0,600,602,5,44,0,0,601,603,3,142,
	71,0,602,601,1,0,0,0,602,603,1,0,0,0,603,604,1,0,0,0,604,606,5,45,0,0,605,
	599,1,0,0,0,606,609,1,0,0,0,607,605,1,0,0,0,607,608,1,0,0,0,608,79,1,0,
	0,0,609,607,1,0,0,0,610,615,3,196,98,0,611,612,5,47,0,0,612,614,3,196,98,
	0,613,611,1,0,0,0,614,617,1,0,0,0,615,613,1,0,0,0,615,616,1,0,0,0,616,81,
	1,0,0,0,617,615,1,0,0,0,618,621,3,140,70,0,619,621,3,80,40,0,620,618,1,
	0,0,0,620,619,1,0,0,0,621,83,1,0,0,0,622,623,5,48,0,0,623,624,5,23,0,0,
	624,626,3,82,41,0,625,627,3,86,43,0,626,625,1,0,0,0,626,627,1,0,0,0,627,
	628,1,0,0,0,628,629,5,49,0,0,629,631,3,78,39,0,630,632,3,88,44,0,631,630,
	1,0,0,0,631,632,1,0,0,0,632,633,1,0,0,0,633,634,5,24,0,0,634,85,1,0,0,0,
	635,636,3,196,98,0,636,87,1,0,0,0,637,638,3,196,98,0,638,89,1,0,0,0,639,
	640,5,40,0,0,640,646,3,72,36,0,641,645,5,119,0,0,642,645,5,117,0,0,643,
	645,3,94,47,0,644,641,1,0,0,0,644,642,1,0,0,0,644,643,1,0,0,0,645,648,1,
	0,0,0,646,644,1,0,0,0,646,647,1,0,0,0,647,651,1,0,0,0,648,646,1,0,0,0,649,
	650,5,41,0,0,650,652,3,72,36,0,651,649,1,0,0,0,651,652,1,0,0,0,652,91,1,
	0,0,0,653,654,7,3,0,0,654,93,1,0,0,0,655,656,7,4,0,0,656,95,1,0,0,0,657,
	661,5,15,0,0,658,660,3,98,49,0,659,658,1,0,0,0,660,663,1,0,0,0,661,659,
	1,0,0,0,661,662,1,0,0,0,662,664,1,0,0,0,663,661,1,0,0,0,664,665,5,17,0,
	0,665,97,1,0,0,0,666,682,3,102,51,0,667,682,3,104,52,0,668,682,3,108,54,
	0,669,682,3,114,57,0,670,682,3,96,48,0,671,682,3,116,58,0,672,682,3,120,
	60,0,673,682,3,122,61,0,674,682,3,124,62,0,675,682,3,126,63,0,676,682,3,
	128,64,0,677,682,3,130,65,0,678,682,3,110,55,0,679,682,3,112,56,0,680,682,
	3,132,66,0,681,666,1,0,0,0,681,667,1,0,0,0,681,668,1,0,0,0,681,669,1,0,
	0,0,681,670,1,0,0,0,681,671,1,0,0,0,681,672,1,0,0,0,681,673,1,0,0,0,681,
	674,1,0,0,0,681,675,1,0,0,0,681,676,1,0,0,0,681,677,1,0,0,0,681,678,1,0,
	0,0,681,679,1,0,0,0,681,680,1,0,0,0,682,99,1,0,0,0,683,684,3,142,71,0,684,
	685,5,2,0,0,685,101,1,0,0,0,686,687,5,53,0,0,687,688,5,23,0,0,688,689,3,
	142,71,0,689,690,5,24,0,0,690,693,3,98,49,0,691,692,5,54,0,0,692,694,3,
	98,49,0,693,691,1,0,0,0,693,694,1,0,0,0,694,103,1,0,0,0,695,696,5,55,0,
	0,696,698,3,142,71,0,697,699,3,54,27,0,698,697,1,0,0,0,698,699,1,0,0,0,
	699,700,1,0,0,0,700,702,3,96,48,0,701,703,3,106,53,0,702,701,1,0,0,0,703,
	704,1,0,0,0,704,702,1,0,0,0,704,705,1,0,0,0,705,105,1,0,0,0,706,711,5,56,
	0,0,707,709,3,196,98,0,708,707,1,0,0,0,708,709,1,0,0,0,709,710,1,0,0,0,
	710,712,3,64,32,0,711,708,1,0,0,0,711,712,1,0,0,0,712,713,1,0,0,0,713,714,
	3,96,48,0,714,107,1,0,0,0,715,716,5,57,0,0,716,717,5,23,0,0,717,718,3,142,
	71,0,718,719,5,24,0,0,719,720,3,98,49,0,720,109,1,0,0,0,721,724,3,134,67,
	0,722,724,3,100,50,0,723,721,1,0,0,0,723,722,1,0,0,0,724,111,1,0,0,0,725,
	726,5,58,0,0,726,727,3,96,48,0,727,113,1,0,0,0,728,729,5,29,0,0,729,732,
	5,23,0,0,730,733,3,110,55,0,731,733,5,2,0,0,732,730,1,0,0,0,732,731,1,0,
	0,0,733,736,1,0,0,0,734,737,3,100,50,0,735,737,5,2,0,0,736,734,1,0,0,0,
	736,735,1,0,0,0,737,739,1,0,0,0,738,740,3,142,71,0,739,738,1,0,0,0,739,
	740,1,0,0,0,740,741,1,0,0,0,741,742,5,24,0,0,742,743,3,98,49,0,743,115,
	1,0,0,0,744,746,5,59,0,0,745,747,5,132,0,0,746,745,1,0,0,0,746,747,1,0,
	0,0,747,752,1,0,0,0,748,749,5,23,0,0,749,750,3,118,59,0,750,751,5,24,0,
	0,751,753,1,0,0,0,752,748,1,0,0,0,752,753,1,0,0,0,753,754,1,0,0,0,754,755,
	3,156,78,0,755,117,1,0,0,0,756,757,3,202,101,0,757,119,1,0,0,0,758,759,
	5,60,0,0,759,760,3,98,49,0,760,761,5,57,0,0,761,762,5,23,0,0,762,763,3,
	142,71,0,763,764,5,24,0,0,764,765,5,2,0,0,765,121,1,0,0,0,766,767,5,115,
	0,0,767,768,5,2,0,0,768,123,1,0,0,0,769,770,5,111,0,0,770,771,5,2,0,0,771,
	125,1,0,0,0,772,774,5,61,0,0,773,775,3,142,71,0,774,773,1,0,0,0,774,775,
	1,0,0,0,775,776,1,0,0,0,776,777,5,2,0,0,777,127,1,0,0,0,778,779,5,62,0,
	0,779,780,5,2,0,0,780,129,1,0,0,0,781,782,5,63,0,0,782,783,3,154,77,0,783,
	784,5,2,0,0,784,131,1,0,0,0,785,786,5,64,0,0,786,787,3,154,77,0,787,788,
	5,2,0,0,788,133,1,0,0,0,789,790,5,65,0,0,790,797,3,138,69,0,791,797,3,76,
	38,0,792,793,5,23,0,0,793,794,3,136,68,0,794,795,5,24,0,0,795,797,1,0,0,
	0,796,789,1,0,0,0,796,791,1,0,0,0,796,792,1,0,0,0,797,800,1,0,0,0,798,799,
	5,11,0,0,799,801,3,142,71,0,800,798,1,0,0,0,800,801,1,0,0,0,801,802,1,0,
	0,0,802,803,5,2,0,0,803,135,1,0,0,0,804,806,3,76,38,0,805,804,1,0,0,0,805,
	806,1,0,0,0,806,813,1,0,0,0,807,809,5,16,0,0,808,810,3,76,38,0,809,808,
	1,0,0,0,809,810,1,0,0,0,810,812,1,0,0,0,811,807,1,0,0,0,812,815,1,0,0,0,
	813,811,1,0,0,0,813,814,1,0,0,0,814,137,1,0,0,0,815,813,1,0,0,0,816,823,
	5,23,0,0,817,819,3,196,98,0,818,817,1,0,0,0,818,819,1,0,0,0,819,820,1,0,
	0,0,820,822,5,16,0,0,821,818,1,0,0,0,822,825,1,0,0,0,823,821,1,0,0,0,823,
	824,1,0,0,0,824,827,1,0,0,0,825,823,1,0,0,0,826,828,3,196,98,0,827,826,
	1,0,0,0,827,828,1,0,0,0,828,829,1,0,0,0,829,830,5,24,0,0,830,139,1,0,0,
	0,831,832,7,5,0,0,832,141,1,0,0,0,833,834,6,71,-1,0,834,835,5,71,0,0,835,
	852,3,78,39,0,836,837,5,23,0,0,837,838,3,142,71,0,838,839,5,24,0,0,839,
	852,1,0,0,0,840,841,7,6,0,0,841,852,3,142,71,19,842,843,7,7,0,0,843,852,
	3,142,71,18,844,845,5,73,0,0,845,852,3,142,71,17,846,847,5,74,0,0,847,852,
	3,142,71,16,848,849,5,6,0,0,849,852,3,142,71,15,850,852,3,144,72,0,851,
	833,1,0,0,0,851,836,1,0,0,0,851,840,1,0,0,0,851,842,1,0,0,0,851,844,1,0,
	0,0,851,846,1,0,0,0,851,848,1,0,0,0,851,850,1,0,0,0,852,927,1,0,0,0,853,
	854,10,14,0,0,854,855,5,75,0,0,855,926,3,142,71,14,856,857,10,13,0,0,857,
	858,7,8,0,0,858,926,3,142,71,14,859,860,10,12,0,0,860,861,7,7,0,0,861,926,
	3,142,71,13,862,863,10,11,0,0,863,864,7,9,0,0,864,926,3,142,71,12,865,866,
	10,10,0,0,866,867,5,31,0,0,867,926,3,142,71,11,868,869,10,9,0,0,869,870,
	5,5,0,0,870,926,3,142,71,10,871,872,10,8,0,0,872,873,5,30,0,0,873,926,3,
	142,71,9,874,875,10,7,0,0,875,876,7,10,0,0,876,926,3,142,71,8,877,878,10,
	6,0,0,878,879,7,11,0,0,879,926,3,142,71,7,880,881,10,5,0,0,881,882,5,78,
	0,0,882,926,3,142,71,6,883,884,10,4,0,0,884,885,5,4,0,0,885,926,3,142,71,
	5,886,887,10,3,0,0,887,888,5,79,0,0,888,889,3,142,71,0,889,890,5,72,0,0,
	890,891,3,142,71,3,891,926,1,0,0,0,892,893,10,2,0,0,893,894,7,12,0,0,894,
	926,3,142,71,3,895,896,10,27,0,0,896,926,7,6,0,0,897,898,10,25,0,0,898,
	899,5,44,0,0,899,900,3,142,71,0,900,901,5,45,0,0,901,926,1,0,0,0,902,903,
	10,24,0,0,903,905,5,44,0,0,904,906,3,142,71,0,905,904,1,0,0,0,905,906,1,
	0,0,0,906,907,1,0,0,0,907,909,5,72,0,0,908,910,3,142,71,0,909,908,1,0,0,
	0,909,910,1,0,0,0,910,911,1,0,0,0,911,926,5,45,0,0,912,913,10,23,0,0,913,
	914,5,47,0,0,914,926,3,196,98,0,915,916,10,22,0,0,916,917,5,15,0,0,917,
	918,3,148,74,0,918,919,5,17,0,0,919,926,1,0,0,0,920,921,10,21,0,0,921,922,
	5,23,0,0,922,923,3,152,76,0,923,924,5,24,0,0,924,926,1,0,0,0,925,853,1,
	0,0,0,925,856,1,0,0,0,925,859,1,0,0,0,925,862,1,0,0,0,925,865,1,0,0,0,925,
	868,1,0,0,0,925,871,1,0,0,0,925,874,1,0,0,0,925,877,1,0,0,0,925,880,1,0,
	0,0,925,883,1,0,0,0,925,886,1,0,0,0,925,892,1,0,0,0,925,895,1,0,0,0,925,
	897,1,0,0,0,925,902,1,0,0,0,925,912,1,0,0,0,925,915,1,0,0,0,925,920,1,0,
	0,0,926,929,1,0,0,0,927,925,1,0,0,0,927,928,1,0,0,0,928,143,1,0,0,0,929,
	927,1,0,0,0,930,940,5,104,0,0,931,940,3,194,97,0,932,940,3,198,99,0,933,
	940,3,202,101,0,934,940,3,196,98,0,935,940,5,125,0,0,936,940,5,120,0,0,
	937,940,3,192,96,0,938,940,3,78,39,0,939,930,1,0,0,0,939,931,1,0,0,0,939,
	932,1,0,0,0,939,933,1,0,0,0,939,934,1,0,0,0,939,935,1,0,0,0,939,936,1,0,
	0,0,939,937,1,0,0,0,939,938,1,0,0,0,940,145,1,0,0,0,941,946,3,142,71,0,
	942,943,5,16,0,0,943,945,3,142,71,0,944,942,1,0,0,0,945,948,1,0,0,0,946,
	944,1,0,0,0,946,947,1,0,0,0,947,147,1,0,0,0,948,946,1,0,0,0,949,954,3,150,
	75,0,950,951,5,16,0,0,951,953,3,150,75,0,952,950,1,0,0,0,953,956,1,0,0,
	0,954,952,1,0,0,0,954,955,1,0,0,0,955,958,1,0,0,0,956,954,1,0,0,0,957,959,
	5,16,0,0,958,957,1,0,0,0,958,959,1,0,0,0,959,149,1,0,0,0,960,961,3,196,
	98,0,961,962,5,72,0,0,962,963,3,142,71,0,963,151,1,0,0,0,964,966,5,15,0,
	0,965,967,3,148,74,0,966,965,1,0,0,0,966,967,1,0,0,0,967,968,1,0,0,0,968,
	973,5,17,0,0,969,971,3,146,73,0,970,969,1,0,0,0,970,971,1,0,0,0,971,973,
	1,0,0,0,972,964,1,0,0,0,972,970,1,0,0,0,973,153,1,0,0,0,974,975,3,142,71,
	0,975,976,5,23,0,0,976,977,3,152,76,0,977,978,5,24,0,0,978,155,1,0,0,0,
	979,983,5,15,0,0,980,982,3,158,79,0,981,980,1,0,0,0,982,985,1,0,0,0,983,
	981,1,0,0,0,983,984,1,0,0,0,984,986,1,0,0,0,985,983,1,0,0,0,986,987,5,17,
	0,0,987,157,1,0,0,0,988,1006,3,196,98,0,989,1006,3,156,78,0,990,1006,3,
	160,80,0,991,1006,3,166,83,0,992,1006,3,168,84,0,993,1006,3,174,87,0,994,
	1006,3,176,88,0,995,1006,3,178,89,0,996,1006,3,182,91,0,997,1006,3,186,
	93,0,998,1006,3,188,94,0,999,1006,5,111,0,0,1000,1006,5,115,0,0,1001,1006,
	5,116,0,0,1002,1006,3,194,97,0,1003,1006,3,202,101,0,1004,1006,3,198,99,
	0,1005,988,1,0,0,0,1005,989,1,0,0,0,1005,990,1,0,0,0,1005,991,1,0,0,0,1005,
	992,1,0,0,0,1005,993,1,0,0,0,1005,994,1,0,0,0,1005,995,1,0,0,0,1005,996,
	1,0,0,0,1005,997,1,0,0,0,1005,998,1,0,0,0,1005,999,1,0,0,0,1005,1000,1,
	0,0,0,1005,1001,1,0,0,0,1005,1002,1,0,0,0,1005,1003,1,0,0,0,1005,1004,1,
	0,0,0,1006,159,1,0,0,0,1007,1011,3,164,82,0,1008,1011,3,190,95,0,1009,1011,
	3,162,81,0,1010,1007,1,0,0,0,1010,1008,1,0,0,0,1010,1009,1,0,0,0,1011,161,
	1,0,0,0,1012,1013,3,196,98,0,1013,1014,5,47,0,0,1014,1015,3,196,98,0,1015,
	163,1,0,0,0,1016,1021,5,61,0,0,1017,1021,5,46,0,0,1018,1021,5,68,0,0,1019,
	1021,3,196,98,0,1020,1016,1,0,0,0,1020,1017,1,0,0,0,1020,1018,1,0,0,0,1020,
	1019,1,0,0,0,1021,1034,1,0,0,0,1022,1024,5,23,0,0,1023,1025,3,160,80,0,
	1024,1023,1,0,0,0,1024,1025,1,0,0,0,1025,1030,1,0,0,0,1026,1027,5,16,0,
	0,1027,1029,3,160,80,0,1028,1026,1,0,0,0,1029,1032,1,0,0,0,1030,1028,1,
	0,0,0,1030,1031,1,0,0,0,1031,1033,1,0,0,0,1032,1030,1,0,0,0,1033,1035,5,
	24,0,0,1034,1022,1,0,0,0,1034,1035,1,0,0,0,1035,165,1,0,0,0,1036,1037,5,
	90,0,0,1037,1040,3,170,85,0,1038,1039,5,91,0,0,1039,1041,3,160,80,0,1040,
	1038,1,0,0,0,1040,1041,1,0,0,0,1041,167,1,0,0,0,1042,1043,3,170,85,0,1043,
	1044,5,91,0,0,1044,1045,3,160,80,0,1045,169,1,0,0,0,1046,1054,3,196,98,
	0,1047,1054,3,162,81,0,1048,1054,3,172,86,0,1049,1050,5,23,0,0,1050,1051,
	3,172,86,0,1051,1052,5,24,0,0,1052,1054,1,0,0,0,1053,1046,1,0,0,0,1053,
	1047,1,0,0,0,1053,1048,1,0,0,0,1053,1049,1,0,0,0,1054,171,1,0,0,0,1055,
	1060,3,196,98,0,1056,1057,5,16,0,0,1057,1059,3,196,98,0,1058,1056,1,0,0,
	0,1059,1062,1,0,0,0,1060,1058,1,0,0,0,1060,1061,1,0,0,0,1061,173,1,0,0,
	0,1062,1060,1,0,0,0,1063,1064,3,160,80,0,1064,1065,5,92,0,0,1065,1066,3,
	196,98,0,1066,175,1,0,0,0,1067,1068,3,196,98,0,1068,1069,5,72,0,0,1069,
	177,1,0,0,0,1070,1071,5,93,0,0,1071,1075,3,160,80,0,1072,1074,3,180,90,
	0,1073,1072,1,0,0,0,1074,1077,1,0,0,0,1075,1073,1,0,0,0,1075,1076,1,0,0,
	0,1076,179,1,0,0,0,1077,1075,1,0,0,0,1078,1079,5,94,0,0,1079,1080,3,190,
	95,0,1080,1081,3,156,78,0,1081,1085,1,0,0,0,1082,1083,5,95,0,0,1083,1085,
	3,156,78,0,1084,1078,1,0,0,0,1084,1082,1,0,0,0,1085,181,1,0,0,0,1086,1087,
	5,40,0,0,1087,1088,3,196,98,0,1088,1090,5,23,0,0,1089,1091,3,172,86,0,1090,
	1089,1,0,0,0,1090,1091,1,0,0,0,1091,1092,1,0,0,0,1092,1094,5,24,0,0,1093,
	1095,3,184,92,0,1094,1093,1,0,0,0,1094,1095,1,0,0,0,1095,1096,1,0,0,0,1096,
	1097,3,156,78,0,1097,183,1,0,0,0,1098,1099,5,96,0,0,1099,1100,3,172,86,
	0,1100,185,1,0,0,0,1101,1104,5,29,0,0,1102,1105,3,156,78,0,1103,1105,3,
	160,80,0,1104,1102,1,0,0,0,1104,1103,1,0,0,0,1105,1106,1,0,0,0,1106,1109,
	3,160,80,0,1107,1110,3,156,78,0,1108,1110,3,160,80,0,1109,1107,1,0,0,0,
	1109,1108,1,0,0,0,1110,1111,1,0,0,0,1111,1112,3,156,78,0,1112,187,1,0,0,
	0,1113,1114,5,53,0,0,1114,1115,3,160,80,0,1115,1116,3,156,78,0,1116,189,
	1,0,0,0,1117,1123,3,202,101,0,1118,1123,5,105,0,0,1119,1123,5,106,0,0,1120,
	1123,3,198,99,0,1121,1123,5,104,0,0,1122,1117,1,0,0,0,1122,1118,1,0,0,0,
	1122,1119,1,0,0,0,1122,1120,1,0,0,0,1122,1121,1,0,0,0,1123,191,1,0,0,0,
	1124,1126,5,23,0,0,1125,1127,3,142,71,0,1126,1125,1,0,0,0,1126,1127,1,0,
	0,0,1127,1134,1,0,0,0,1128,1130,5,16,0,0,1129,1131,3,142,71,0,1130,1129,
	1,0,0,0,1130,1131,1,0,0,0,1131,1133,1,0,0,0,1132,1128,1,0,0,0,1133,1136,
	1,0,0,0,1134,1132,1,0,0,0,1134,1135,1,0,0,0,1135,1137,1,0,0,0,1136,1134,
	1,0,0,0,1137,1151,5,24,0,0,1138,1147,5,44,0,0,1139,1144,3,142,71,0,1140,
	1141,5,16,0,0,1141,1143,3,142,71,0,1142,1140,1,0,0,0,1143,1146,1,0,0,0,
	1144,1142,1,0,0,0,1144,1145,1,0,0,0,1145,1148,1,0,0,0,1146,1144,1,0,0,0,
	1147,1139,1,0,0,0,1147,1148,1,0,0,0,1148,1149,1,0,0,0,1149,1151,5,45,0,
	0,1150,1124,1,0,0,0,1150,1138,1,0,0,0,1151,193,1,0,0,0,1152,1154,7,13,0,
	0,1153,1155,5,107,0,0,1154,1153,1,0,0,0,1154,1155,1,0,0,0,1155,195,1,0,
	0,0,1156,1157,7,14,0,0,1157,197,1,0,0,0,1158,1160,5,108,0,0,1159,1158,1,
	0,0,0,1160,1161,1,0,0,0,1161,1159,1,0,0,0,1161,1162,1,0,0,0,1162,199,1,
	0,0,0,1163,1175,5,98,0,0,1164,1165,5,23,0,0,1165,1170,3,80,40,0,1166,1167,
	5,16,0,0,1167,1169,3,80,40,0,1168,1166,1,0,0,0,1169,1172,1,0,0,0,1170,1168,
	1,0,0,0,1170,1171,1,0,0,0,1171,1173,1,0,0,0,1172,1170,1,0,0,0,1173,1174,
	5,24,0,0,1174,1176,1,0,0,0,1175,1164,1,0,0,0,1175,1176,1,0,0,0,1176,201,
	1,0,0,0,1177,1179,5,132,0,0,1178,1177,1,0,0,0,1179,1180,1,0,0,0,1180,1178,
	1,0,0,0,1180,1181,1,0,0,0,1181,203,1,0,0,0,132,215,217,232,236,241,247,
	251,254,259,265,272,276,289,297,302,307,315,318,321,327,335,338,353,363,
	365,371,398,401,412,417,422,436,439,446,450,452,457,462,465,471,475,479,
	484,497,499,506,516,522,533,536,542,545,553,556,562,565,573,576,582,586,
	597,602,607,615,620,626,631,644,646,651,661,681,693,698,704,708,711,723,
	732,736,739,746,752,774,796,800,805,809,813,818,823,827,851,905,909,925,
	927,939,946,954,958,966,970,972,983,1005,1010,1020,1024,1030,1034,1040,
	1053,1060,1075,1084,1090,1094,1104,1109,1122,1126,1130,1134,1144,1147,1150,
	1154,1161,1170,1175,1180];

	private static __ATN: ATN;
	public static get _ATN(): ATN {
		if (!SolidityParser.__ATN) {
			SolidityParser.__ATN = new ATNDeserializer().deserialize(SolidityParser._serializedATN);
		}

		return SolidityParser.__ATN;
	}


	static DecisionsToDFA = SolidityParser._ATN.decisionToState.map( (ds: DecisionState, index: number) => new DFA(ds, index) );

}

export class SourceUnitContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public EOF(): TerminalNode {
		return this.getToken(SolidityParser.EOF, 0);
	}
	public pragmaDirective_list(): PragmaDirectiveContext[] {
		return this.getTypedRuleContexts(PragmaDirectiveContext) as PragmaDirectiveContext[];
	}
	public pragmaDirective(i: number): PragmaDirectiveContext {
		return this.getTypedRuleContext(PragmaDirectiveContext, i) as PragmaDirectiveContext;
	}
	public importDirective_list(): ImportDirectiveContext[] {
		return this.getTypedRuleContexts(ImportDirectiveContext) as ImportDirectiveContext[];
	}
	public importDirective(i: number): ImportDirectiveContext {
		return this.getTypedRuleContext(ImportDirectiveContext, i) as ImportDirectiveContext;
	}
	public contractDefinition_list(): ContractDefinitionContext[] {
		return this.getTypedRuleContexts(ContractDefinitionContext) as ContractDefinitionContext[];
	}
	public contractDefinition(i: number): ContractDefinitionContext {
		return this.getTypedRuleContext(ContractDefinitionContext, i) as ContractDefinitionContext;
	}
	public enumDefinition_list(): EnumDefinitionContext[] {
		return this.getTypedRuleContexts(EnumDefinitionContext) as EnumDefinitionContext[];
	}
	public enumDefinition(i: number): EnumDefinitionContext {
		return this.getTypedRuleContext(EnumDefinitionContext, i) as EnumDefinitionContext;
	}
	public eventDefinition_list(): EventDefinitionContext[] {
		return this.getTypedRuleContexts(EventDefinitionContext) as EventDefinitionContext[];
	}
	public eventDefinition(i: number): EventDefinitionContext {
		return this.getTypedRuleContext(EventDefinitionContext, i) as EventDefinitionContext;
	}
	public structDefinition_list(): StructDefinitionContext[] {
		return this.getTypedRuleContexts(StructDefinitionContext) as StructDefinitionContext[];
	}
	public structDefinition(i: number): StructDefinitionContext {
		return this.getTypedRuleContext(StructDefinitionContext, i) as StructDefinitionContext;
	}
	public functionDefinition_list(): FunctionDefinitionContext[] {
		return this.getTypedRuleContexts(FunctionDefinitionContext) as FunctionDefinitionContext[];
	}
	public functionDefinition(i: number): FunctionDefinitionContext {
		return this.getTypedRuleContext(FunctionDefinitionContext, i) as FunctionDefinitionContext;
	}
	public fileLevelConstant_list(): FileLevelConstantContext[] {
		return this.getTypedRuleContexts(FileLevelConstantContext) as FileLevelConstantContext[];
	}
	public fileLevelConstant(i: number): FileLevelConstantContext {
		return this.getTypedRuleContext(FileLevelConstantContext, i) as FileLevelConstantContext;
	}
	public customErrorDefinition_list(): CustomErrorDefinitionContext[] {
		return this.getTypedRuleContexts(CustomErrorDefinitionContext) as CustomErrorDefinitionContext[];
	}
	public customErrorDefinition(i: number): CustomErrorDefinitionContext {
		return this.getTypedRuleContext(CustomErrorDefinitionContext, i) as CustomErrorDefinitionContext;
	}
	public typeDefinition_list(): TypeDefinitionContext[] {
		return this.getTypedRuleContexts(TypeDefinitionContext) as TypeDefinitionContext[];
	}
	public typeDefinition(i: number): TypeDefinitionContext {
		return this.getTypedRuleContext(TypeDefinitionContext, i) as TypeDefinitionContext;
	}
	public usingForDeclaration_list(): UsingForDeclarationContext[] {
		return this.getTypedRuleContexts(UsingForDeclarationContext) as UsingForDeclarationContext[];
	}
	public usingForDeclaration(i: number): UsingForDeclarationContext {
		return this.getTypedRuleContext(UsingForDeclarationContext, i) as UsingForDeclarationContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_sourceUnit;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterSourceUnit) {
	 		listener.enterSourceUnit(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitSourceUnit) {
	 		listener.exitSourceUnit(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitSourceUnit) {
			return visitor.visitSourceUnit(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class PragmaDirectiveContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public pragmaName(): PragmaNameContext {
		return this.getTypedRuleContext(PragmaNameContext, 0) as PragmaNameContext;
	}
	public pragmaValue(): PragmaValueContext {
		return this.getTypedRuleContext(PragmaValueContext, 0) as PragmaValueContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_pragmaDirective;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterPragmaDirective) {
	 		listener.enterPragmaDirective(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitPragmaDirective) {
	 		listener.exitPragmaDirective(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitPragmaDirective) {
			return visitor.visitPragmaDirective(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class PragmaNameContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public identifier(): IdentifierContext {
		return this.getTypedRuleContext(IdentifierContext, 0) as IdentifierContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_pragmaName;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterPragmaName) {
	 		listener.enterPragmaName(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitPragmaName) {
	 		listener.exitPragmaName(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitPragmaName) {
			return visitor.visitPragmaName(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class PragmaValueContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public version(): VersionContext {
		return this.getTypedRuleContext(VersionContext, 0) as VersionContext;
	}
	public expression(): ExpressionContext {
		return this.getTypedRuleContext(ExpressionContext, 0) as ExpressionContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_pragmaValue;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterPragmaValue) {
	 		listener.enterPragmaValue(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitPragmaValue) {
	 		listener.exitPragmaValue(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitPragmaValue) {
			return visitor.visitPragmaValue(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class VersionContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public versionConstraint_list(): VersionConstraintContext[] {
		return this.getTypedRuleContexts(VersionConstraintContext) as VersionConstraintContext[];
	}
	public versionConstraint(i: number): VersionConstraintContext {
		return this.getTypedRuleContext(VersionConstraintContext, i) as VersionConstraintContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_version;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterVersion) {
	 		listener.enterVersion(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitVersion) {
	 		listener.exitVersion(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitVersion) {
			return visitor.visitVersion(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class VersionOperatorContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_versionOperator;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterVersionOperator) {
	 		listener.enterVersionOperator(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitVersionOperator) {
	 		listener.exitVersionOperator(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitVersionOperator) {
			return visitor.visitVersionOperator(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class VersionConstraintContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public VersionLiteral(): TerminalNode {
		return this.getToken(SolidityParser.VersionLiteral, 0);
	}
	public versionOperator(): VersionOperatorContext {
		return this.getTypedRuleContext(VersionOperatorContext, 0) as VersionOperatorContext;
	}
	public DecimalNumber(): TerminalNode {
		return this.getToken(SolidityParser.DecimalNumber, 0);
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_versionConstraint;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterVersionConstraint) {
	 		listener.enterVersionConstraint(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitVersionConstraint) {
	 		listener.exitVersionConstraint(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitVersionConstraint) {
			return visitor.visitVersionConstraint(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class ImportDeclarationContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public identifier_list(): IdentifierContext[] {
		return this.getTypedRuleContexts(IdentifierContext) as IdentifierContext[];
	}
	public identifier(i: number): IdentifierContext {
		return this.getTypedRuleContext(IdentifierContext, i) as IdentifierContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_importDeclaration;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterImportDeclaration) {
	 		listener.enterImportDeclaration(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitImportDeclaration) {
	 		listener.exitImportDeclaration(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitImportDeclaration) {
			return visitor.visitImportDeclaration(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class ImportDirectiveContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public importPath(): ImportPathContext {
		return this.getTypedRuleContext(ImportPathContext, 0) as ImportPathContext;
	}
	public identifier_list(): IdentifierContext[] {
		return this.getTypedRuleContexts(IdentifierContext) as IdentifierContext[];
	}
	public identifier(i: number): IdentifierContext {
		return this.getTypedRuleContext(IdentifierContext, i) as IdentifierContext;
	}
	public importDeclaration_list(): ImportDeclarationContext[] {
		return this.getTypedRuleContexts(ImportDeclarationContext) as ImportDeclarationContext[];
	}
	public importDeclaration(i: number): ImportDeclarationContext {
		return this.getTypedRuleContext(ImportDeclarationContext, i) as ImportDeclarationContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_importDirective;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterImportDirective) {
	 		listener.enterImportDirective(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitImportDirective) {
	 		listener.exitImportDirective(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitImportDirective) {
			return visitor.visitImportDirective(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class ImportPathContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public StringLiteralFragment(): TerminalNode {
		return this.getToken(SolidityParser.StringLiteralFragment, 0);
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_importPath;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterImportPath) {
	 		listener.enterImportPath(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitImportPath) {
	 		listener.exitImportPath(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitImportPath) {
			return visitor.visitImportPath(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class ContractDefinitionContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public identifier(): IdentifierContext {
		return this.getTypedRuleContext(IdentifierContext, 0) as IdentifierContext;
	}
	public customStorageLayout_list(): CustomStorageLayoutContext[] {
		return this.getTypedRuleContexts(CustomStorageLayoutContext) as CustomStorageLayoutContext[];
	}
	public customStorageLayout(i: number): CustomStorageLayoutContext {
		return this.getTypedRuleContext(CustomStorageLayoutContext, i) as CustomStorageLayoutContext;
	}
	public inheritanceSpecifier_list(): InheritanceSpecifierContext[] {
		return this.getTypedRuleContexts(InheritanceSpecifierContext) as InheritanceSpecifierContext[];
	}
	public inheritanceSpecifier(i: number): InheritanceSpecifierContext {
		return this.getTypedRuleContext(InheritanceSpecifierContext, i) as InheritanceSpecifierContext;
	}
	public contractPart_list(): ContractPartContext[] {
		return this.getTypedRuleContexts(ContractPartContext) as ContractPartContext[];
	}
	public contractPart(i: number): ContractPartContext {
		return this.getTypedRuleContext(ContractPartContext, i) as ContractPartContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_contractDefinition;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterContractDefinition) {
	 		listener.enterContractDefinition(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitContractDefinition) {
	 		listener.exitContractDefinition(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitContractDefinition) {
			return visitor.visitContractDefinition(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class InheritanceSpecifierContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public userDefinedTypeName(): UserDefinedTypeNameContext {
		return this.getTypedRuleContext(UserDefinedTypeNameContext, 0) as UserDefinedTypeNameContext;
	}
	public expressionList(): ExpressionListContext {
		return this.getTypedRuleContext(ExpressionListContext, 0) as ExpressionListContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_inheritanceSpecifier;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterInheritanceSpecifier) {
	 		listener.enterInheritanceSpecifier(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitInheritanceSpecifier) {
	 		listener.exitInheritanceSpecifier(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitInheritanceSpecifier) {
			return visitor.visitInheritanceSpecifier(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class CustomStorageLayoutContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public expression(): ExpressionContext {
		return this.getTypedRuleContext(ExpressionContext, 0) as ExpressionContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_customStorageLayout;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterCustomStorageLayout) {
	 		listener.enterCustomStorageLayout(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitCustomStorageLayout) {
	 		listener.exitCustomStorageLayout(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitCustomStorageLayout) {
			return visitor.visitCustomStorageLayout(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class ContractPartContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public stateVariableDeclaration(): StateVariableDeclarationContext {
		return this.getTypedRuleContext(StateVariableDeclarationContext, 0) as StateVariableDeclarationContext;
	}
	public usingForDeclaration(): UsingForDeclarationContext {
		return this.getTypedRuleContext(UsingForDeclarationContext, 0) as UsingForDeclarationContext;
	}
	public structDefinition(): StructDefinitionContext {
		return this.getTypedRuleContext(StructDefinitionContext, 0) as StructDefinitionContext;
	}
	public modifierDefinition(): ModifierDefinitionContext {
		return this.getTypedRuleContext(ModifierDefinitionContext, 0) as ModifierDefinitionContext;
	}
	public functionDefinition(): FunctionDefinitionContext {
		return this.getTypedRuleContext(FunctionDefinitionContext, 0) as FunctionDefinitionContext;
	}
	public eventDefinition(): EventDefinitionContext {
		return this.getTypedRuleContext(EventDefinitionContext, 0) as EventDefinitionContext;
	}
	public enumDefinition(): EnumDefinitionContext {
		return this.getTypedRuleContext(EnumDefinitionContext, 0) as EnumDefinitionContext;
	}
	public customErrorDefinition(): CustomErrorDefinitionContext {
		return this.getTypedRuleContext(CustomErrorDefinitionContext, 0) as CustomErrorDefinitionContext;
	}
	public typeDefinition(): TypeDefinitionContext {
		return this.getTypedRuleContext(TypeDefinitionContext, 0) as TypeDefinitionContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_contractPart;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterContractPart) {
	 		listener.enterContractPart(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitContractPart) {
	 		listener.exitContractPart(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitContractPart) {
			return visitor.visitContractPart(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class StateVariableDeclarationContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public typeName(): TypeNameContext {
		return this.getTypedRuleContext(TypeNameContext, 0) as TypeNameContext;
	}
	public identifier(): IdentifierContext {
		return this.getTypedRuleContext(IdentifierContext, 0) as IdentifierContext;
	}
	public PublicKeyword_list(): TerminalNode[] {
	    	return this.getTokens(SolidityParser.PublicKeyword);
	}
	public PublicKeyword(i: number): TerminalNode {
		return this.getToken(SolidityParser.PublicKeyword, i);
	}
	public InternalKeyword_list(): TerminalNode[] {
	    	return this.getTokens(SolidityParser.InternalKeyword);
	}
	public InternalKeyword(i: number): TerminalNode {
		return this.getToken(SolidityParser.InternalKeyword, i);
	}
	public PrivateKeyword_list(): TerminalNode[] {
	    	return this.getTokens(SolidityParser.PrivateKeyword);
	}
	public PrivateKeyword(i: number): TerminalNode {
		return this.getToken(SolidityParser.PrivateKeyword, i);
	}
	public ConstantKeyword_list(): TerminalNode[] {
	    	return this.getTokens(SolidityParser.ConstantKeyword);
	}
	public ConstantKeyword(i: number): TerminalNode {
		return this.getToken(SolidityParser.ConstantKeyword, i);
	}
	public TransientKeyword_list(): TerminalNode[] {
	    	return this.getTokens(SolidityParser.TransientKeyword);
	}
	public TransientKeyword(i: number): TerminalNode {
		return this.getToken(SolidityParser.TransientKeyword, i);
	}
	public ImmutableKeyword_list(): TerminalNode[] {
	    	return this.getTokens(SolidityParser.ImmutableKeyword);
	}
	public ImmutableKeyword(i: number): TerminalNode {
		return this.getToken(SolidityParser.ImmutableKeyword, i);
	}
	public overrideSpecifier_list(): OverrideSpecifierContext[] {
		return this.getTypedRuleContexts(OverrideSpecifierContext) as OverrideSpecifierContext[];
	}
	public overrideSpecifier(i: number): OverrideSpecifierContext {
		return this.getTypedRuleContext(OverrideSpecifierContext, i) as OverrideSpecifierContext;
	}
	public expression(): ExpressionContext {
		return this.getTypedRuleContext(ExpressionContext, 0) as ExpressionContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_stateVariableDeclaration;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterStateVariableDeclaration) {
	 		listener.enterStateVariableDeclaration(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitStateVariableDeclaration) {
	 		listener.exitStateVariableDeclaration(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitStateVariableDeclaration) {
			return visitor.visitStateVariableDeclaration(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class FileLevelConstantContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public typeName(): TypeNameContext {
		return this.getTypedRuleContext(TypeNameContext, 0) as TypeNameContext;
	}
	public ConstantKeyword(): TerminalNode {
		return this.getToken(SolidityParser.ConstantKeyword, 0);
	}
	public identifier(): IdentifierContext {
		return this.getTypedRuleContext(IdentifierContext, 0) as IdentifierContext;
	}
	public expression(): ExpressionContext {
		return this.getTypedRuleContext(ExpressionContext, 0) as ExpressionContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_fileLevelConstant;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterFileLevelConstant) {
	 		listener.enterFileLevelConstant(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitFileLevelConstant) {
	 		listener.exitFileLevelConstant(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitFileLevelConstant) {
			return visitor.visitFileLevelConstant(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class CustomErrorDefinitionContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public identifier(): IdentifierContext {
		return this.getTypedRuleContext(IdentifierContext, 0) as IdentifierContext;
	}
	public parameterList(): ParameterListContext {
		return this.getTypedRuleContext(ParameterListContext, 0) as ParameterListContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_customErrorDefinition;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterCustomErrorDefinition) {
	 		listener.enterCustomErrorDefinition(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitCustomErrorDefinition) {
	 		listener.exitCustomErrorDefinition(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitCustomErrorDefinition) {
			return visitor.visitCustomErrorDefinition(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class TypeDefinitionContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public TypeKeyword(): TerminalNode {
		return this.getToken(SolidityParser.TypeKeyword, 0);
	}
	public identifier(): IdentifierContext {
		return this.getTypedRuleContext(IdentifierContext, 0) as IdentifierContext;
	}
	public elementaryTypeName(): ElementaryTypeNameContext {
		return this.getTypedRuleContext(ElementaryTypeNameContext, 0) as ElementaryTypeNameContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_typeDefinition;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterTypeDefinition) {
	 		listener.enterTypeDefinition(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitTypeDefinition) {
	 		listener.exitTypeDefinition(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitTypeDefinition) {
			return visitor.visitTypeDefinition(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class UsingForDeclarationContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public usingForObject(): UsingForObjectContext {
		return this.getTypedRuleContext(UsingForObjectContext, 0) as UsingForObjectContext;
	}
	public typeName(): TypeNameContext {
		return this.getTypedRuleContext(TypeNameContext, 0) as TypeNameContext;
	}
	public GlobalKeyword(): TerminalNode {
		return this.getToken(SolidityParser.GlobalKeyword, 0);
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_usingForDeclaration;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterUsingForDeclaration) {
	 		listener.enterUsingForDeclaration(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitUsingForDeclaration) {
	 		listener.exitUsingForDeclaration(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitUsingForDeclaration) {
			return visitor.visitUsingForDeclaration(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class UsingForObjectContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public userDefinedTypeName(): UserDefinedTypeNameContext {
		return this.getTypedRuleContext(UserDefinedTypeNameContext, 0) as UserDefinedTypeNameContext;
	}
	public usingForObjectDirective_list(): UsingForObjectDirectiveContext[] {
		return this.getTypedRuleContexts(UsingForObjectDirectiveContext) as UsingForObjectDirectiveContext[];
	}
	public usingForObjectDirective(i: number): UsingForObjectDirectiveContext {
		return this.getTypedRuleContext(UsingForObjectDirectiveContext, i) as UsingForObjectDirectiveContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_usingForObject;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterUsingForObject) {
	 		listener.enterUsingForObject(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitUsingForObject) {
	 		listener.exitUsingForObject(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitUsingForObject) {
			return visitor.visitUsingForObject(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class UsingForObjectDirectiveContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public userDefinedTypeName(): UserDefinedTypeNameContext {
		return this.getTypedRuleContext(UserDefinedTypeNameContext, 0) as UserDefinedTypeNameContext;
	}
	public userDefinableOperators(): UserDefinableOperatorsContext {
		return this.getTypedRuleContext(UserDefinableOperatorsContext, 0) as UserDefinableOperatorsContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_usingForObjectDirective;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterUsingForObjectDirective) {
	 		listener.enterUsingForObjectDirective(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitUsingForObjectDirective) {
	 		listener.exitUsingForObjectDirective(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitUsingForObjectDirective) {
			return visitor.visitUsingForObjectDirective(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class UserDefinableOperatorsContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_userDefinableOperators;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterUserDefinableOperators) {
	 		listener.enterUserDefinableOperators(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitUserDefinableOperators) {
	 		listener.exitUserDefinableOperators(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitUserDefinableOperators) {
			return visitor.visitUserDefinableOperators(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class StructDefinitionContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public identifier(): IdentifierContext {
		return this.getTypedRuleContext(IdentifierContext, 0) as IdentifierContext;
	}
	public variableDeclaration_list(): VariableDeclarationContext[] {
		return this.getTypedRuleContexts(VariableDeclarationContext) as VariableDeclarationContext[];
	}
	public variableDeclaration(i: number): VariableDeclarationContext {
		return this.getTypedRuleContext(VariableDeclarationContext, i) as VariableDeclarationContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_structDefinition;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterStructDefinition) {
	 		listener.enterStructDefinition(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitStructDefinition) {
	 		listener.exitStructDefinition(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitStructDefinition) {
			return visitor.visitStructDefinition(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class ModifierDefinitionContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public identifier(): IdentifierContext {
		return this.getTypedRuleContext(IdentifierContext, 0) as IdentifierContext;
	}
	public block(): BlockContext {
		return this.getTypedRuleContext(BlockContext, 0) as BlockContext;
	}
	public parameterList(): ParameterListContext {
		return this.getTypedRuleContext(ParameterListContext, 0) as ParameterListContext;
	}
	public VirtualKeyword_list(): TerminalNode[] {
	    	return this.getTokens(SolidityParser.VirtualKeyword);
	}
	public VirtualKeyword(i: number): TerminalNode {
		return this.getToken(SolidityParser.VirtualKeyword, i);
	}
	public overrideSpecifier_list(): OverrideSpecifierContext[] {
		return this.getTypedRuleContexts(OverrideSpecifierContext) as OverrideSpecifierContext[];
	}
	public overrideSpecifier(i: number): OverrideSpecifierContext {
		return this.getTypedRuleContext(OverrideSpecifierContext, i) as OverrideSpecifierContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_modifierDefinition;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterModifierDefinition) {
	 		listener.enterModifierDefinition(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitModifierDefinition) {
	 		listener.exitModifierDefinition(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitModifierDefinition) {
			return visitor.visitModifierDefinition(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class ModifierInvocationContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public identifier(): IdentifierContext {
		return this.getTypedRuleContext(IdentifierContext, 0) as IdentifierContext;
	}
	public expressionList(): ExpressionListContext {
		return this.getTypedRuleContext(ExpressionListContext, 0) as ExpressionListContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_modifierInvocation;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterModifierInvocation) {
	 		listener.enterModifierInvocation(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitModifierInvocation) {
	 		listener.exitModifierInvocation(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitModifierInvocation) {
			return visitor.visitModifierInvocation(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class FunctionDefinitionContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public functionDescriptor(): FunctionDescriptorContext {
		return this.getTypedRuleContext(FunctionDescriptorContext, 0) as FunctionDescriptorContext;
	}
	public parameterList(): ParameterListContext {
		return this.getTypedRuleContext(ParameterListContext, 0) as ParameterListContext;
	}
	public modifierList(): ModifierListContext {
		return this.getTypedRuleContext(ModifierListContext, 0) as ModifierListContext;
	}
	public block(): BlockContext {
		return this.getTypedRuleContext(BlockContext, 0) as BlockContext;
	}
	public returnParameters(): ReturnParametersContext {
		return this.getTypedRuleContext(ReturnParametersContext, 0) as ReturnParametersContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_functionDefinition;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterFunctionDefinition) {
	 		listener.enterFunctionDefinition(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitFunctionDefinition) {
	 		listener.exitFunctionDefinition(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitFunctionDefinition) {
			return visitor.visitFunctionDefinition(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class FunctionDescriptorContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public identifier(): IdentifierContext {
		return this.getTypedRuleContext(IdentifierContext, 0) as IdentifierContext;
	}
	public ConstructorKeyword(): TerminalNode {
		return this.getToken(SolidityParser.ConstructorKeyword, 0);
	}
	public FallbackKeyword(): TerminalNode {
		return this.getToken(SolidityParser.FallbackKeyword, 0);
	}
	public ReceiveKeyword(): TerminalNode {
		return this.getToken(SolidityParser.ReceiveKeyword, 0);
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_functionDescriptor;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterFunctionDescriptor) {
	 		listener.enterFunctionDescriptor(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitFunctionDescriptor) {
	 		listener.exitFunctionDescriptor(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitFunctionDescriptor) {
			return visitor.visitFunctionDescriptor(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class ReturnParametersContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public parameterList(): ParameterListContext {
		return this.getTypedRuleContext(ParameterListContext, 0) as ParameterListContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_returnParameters;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterReturnParameters) {
	 		listener.enterReturnParameters(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitReturnParameters) {
	 		listener.exitReturnParameters(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitReturnParameters) {
			return visitor.visitReturnParameters(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class ModifierListContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public ExternalKeyword_list(): TerminalNode[] {
	    	return this.getTokens(SolidityParser.ExternalKeyword);
	}
	public ExternalKeyword(i: number): TerminalNode {
		return this.getToken(SolidityParser.ExternalKeyword, i);
	}
	public PublicKeyword_list(): TerminalNode[] {
	    	return this.getTokens(SolidityParser.PublicKeyword);
	}
	public PublicKeyword(i: number): TerminalNode {
		return this.getToken(SolidityParser.PublicKeyword, i);
	}
	public InternalKeyword_list(): TerminalNode[] {
	    	return this.getTokens(SolidityParser.InternalKeyword);
	}
	public InternalKeyword(i: number): TerminalNode {
		return this.getToken(SolidityParser.InternalKeyword, i);
	}
	public PrivateKeyword_list(): TerminalNode[] {
	    	return this.getTokens(SolidityParser.PrivateKeyword);
	}
	public PrivateKeyword(i: number): TerminalNode {
		return this.getToken(SolidityParser.PrivateKeyword, i);
	}
	public VirtualKeyword_list(): TerminalNode[] {
	    	return this.getTokens(SolidityParser.VirtualKeyword);
	}
	public VirtualKeyword(i: number): TerminalNode {
		return this.getToken(SolidityParser.VirtualKeyword, i);
	}
	public stateMutability_list(): StateMutabilityContext[] {
		return this.getTypedRuleContexts(StateMutabilityContext) as StateMutabilityContext[];
	}
	public stateMutability(i: number): StateMutabilityContext {
		return this.getTypedRuleContext(StateMutabilityContext, i) as StateMutabilityContext;
	}
	public modifierInvocation_list(): ModifierInvocationContext[] {
		return this.getTypedRuleContexts(ModifierInvocationContext) as ModifierInvocationContext[];
	}
	public modifierInvocation(i: number): ModifierInvocationContext {
		return this.getTypedRuleContext(ModifierInvocationContext, i) as ModifierInvocationContext;
	}
	public overrideSpecifier_list(): OverrideSpecifierContext[] {
		return this.getTypedRuleContexts(OverrideSpecifierContext) as OverrideSpecifierContext[];
	}
	public overrideSpecifier(i: number): OverrideSpecifierContext {
		return this.getTypedRuleContext(OverrideSpecifierContext, i) as OverrideSpecifierContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_modifierList;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterModifierList) {
	 		listener.enterModifierList(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitModifierList) {
	 		listener.exitModifierList(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitModifierList) {
			return visitor.visitModifierList(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class EventDefinitionContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public identifier(): IdentifierContext {
		return this.getTypedRuleContext(IdentifierContext, 0) as IdentifierContext;
	}
	public eventParameterList(): EventParameterListContext {
		return this.getTypedRuleContext(EventParameterListContext, 0) as EventParameterListContext;
	}
	public AnonymousKeyword(): TerminalNode {
		return this.getToken(SolidityParser.AnonymousKeyword, 0);
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_eventDefinition;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterEventDefinition) {
	 		listener.enterEventDefinition(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitEventDefinition) {
	 		listener.exitEventDefinition(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitEventDefinition) {
			return visitor.visitEventDefinition(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class EnumValueContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public identifier(): IdentifierContext {
		return this.getTypedRuleContext(IdentifierContext, 0) as IdentifierContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_enumValue;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterEnumValue) {
	 		listener.enterEnumValue(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitEnumValue) {
	 		listener.exitEnumValue(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitEnumValue) {
			return visitor.visitEnumValue(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class EnumDefinitionContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public identifier(): IdentifierContext {
		return this.getTypedRuleContext(IdentifierContext, 0) as IdentifierContext;
	}
	public enumValue_list(): EnumValueContext[] {
		return this.getTypedRuleContexts(EnumValueContext) as EnumValueContext[];
	}
	public enumValue(i: number): EnumValueContext {
		return this.getTypedRuleContext(EnumValueContext, i) as EnumValueContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_enumDefinition;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterEnumDefinition) {
	 		listener.enterEnumDefinition(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitEnumDefinition) {
	 		listener.exitEnumDefinition(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitEnumDefinition) {
			return visitor.visitEnumDefinition(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class ParameterListContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public parameter_list(): ParameterContext[] {
		return this.getTypedRuleContexts(ParameterContext) as ParameterContext[];
	}
	public parameter(i: number): ParameterContext {
		return this.getTypedRuleContext(ParameterContext, i) as ParameterContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_parameterList;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterParameterList) {
	 		listener.enterParameterList(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitParameterList) {
	 		listener.exitParameterList(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitParameterList) {
			return visitor.visitParameterList(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class ParameterContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public typeName(): TypeNameContext {
		return this.getTypedRuleContext(TypeNameContext, 0) as TypeNameContext;
	}
	public storageLocation(): StorageLocationContext {
		return this.getTypedRuleContext(StorageLocationContext, 0) as StorageLocationContext;
	}
	public identifier(): IdentifierContext {
		return this.getTypedRuleContext(IdentifierContext, 0) as IdentifierContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_parameter;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterParameter) {
	 		listener.enterParameter(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitParameter) {
	 		listener.exitParameter(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitParameter) {
			return visitor.visitParameter(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class EventParameterListContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public eventParameter_list(): EventParameterContext[] {
		return this.getTypedRuleContexts(EventParameterContext) as EventParameterContext[];
	}
	public eventParameter(i: number): EventParameterContext {
		return this.getTypedRuleContext(EventParameterContext, i) as EventParameterContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_eventParameterList;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterEventParameterList) {
	 		listener.enterEventParameterList(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitEventParameterList) {
	 		listener.exitEventParameterList(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitEventParameterList) {
			return visitor.visitEventParameterList(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class EventParameterContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public typeName(): TypeNameContext {
		return this.getTypedRuleContext(TypeNameContext, 0) as TypeNameContext;
	}
	public IndexedKeyword(): TerminalNode {
		return this.getToken(SolidityParser.IndexedKeyword, 0);
	}
	public identifier(): IdentifierContext {
		return this.getTypedRuleContext(IdentifierContext, 0) as IdentifierContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_eventParameter;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterEventParameter) {
	 		listener.enterEventParameter(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitEventParameter) {
	 		listener.exitEventParameter(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitEventParameter) {
			return visitor.visitEventParameter(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class FunctionTypeParameterListContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public functionTypeParameter_list(): FunctionTypeParameterContext[] {
		return this.getTypedRuleContexts(FunctionTypeParameterContext) as FunctionTypeParameterContext[];
	}
	public functionTypeParameter(i: number): FunctionTypeParameterContext {
		return this.getTypedRuleContext(FunctionTypeParameterContext, i) as FunctionTypeParameterContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_functionTypeParameterList;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterFunctionTypeParameterList) {
	 		listener.enterFunctionTypeParameterList(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitFunctionTypeParameterList) {
	 		listener.exitFunctionTypeParameterList(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitFunctionTypeParameterList) {
			return visitor.visitFunctionTypeParameterList(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class FunctionTypeParameterContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public typeName(): TypeNameContext {
		return this.getTypedRuleContext(TypeNameContext, 0) as TypeNameContext;
	}
	public storageLocation(): StorageLocationContext {
		return this.getTypedRuleContext(StorageLocationContext, 0) as StorageLocationContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_functionTypeParameter;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterFunctionTypeParameter) {
	 		listener.enterFunctionTypeParameter(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitFunctionTypeParameter) {
	 		listener.exitFunctionTypeParameter(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitFunctionTypeParameter) {
			return visitor.visitFunctionTypeParameter(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class VariableDeclarationContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public typeName(): TypeNameContext {
		return this.getTypedRuleContext(TypeNameContext, 0) as TypeNameContext;
	}
	public identifier(): IdentifierContext {
		return this.getTypedRuleContext(IdentifierContext, 0) as IdentifierContext;
	}
	public storageLocation(): StorageLocationContext {
		return this.getTypedRuleContext(StorageLocationContext, 0) as StorageLocationContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_variableDeclaration;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterVariableDeclaration) {
	 		listener.enterVariableDeclaration(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitVariableDeclaration) {
	 		listener.exitVariableDeclaration(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitVariableDeclaration) {
			return visitor.visitVariableDeclaration(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class TypeNameContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public elementaryTypeName(): ElementaryTypeNameContext {
		return this.getTypedRuleContext(ElementaryTypeNameContext, 0) as ElementaryTypeNameContext;
	}
	public userDefinedTypeName(): UserDefinedTypeNameContext {
		return this.getTypedRuleContext(UserDefinedTypeNameContext, 0) as UserDefinedTypeNameContext;
	}
	public mapping(): MappingContext {
		return this.getTypedRuleContext(MappingContext, 0) as MappingContext;
	}
	public functionTypeName(): FunctionTypeNameContext {
		return this.getTypedRuleContext(FunctionTypeNameContext, 0) as FunctionTypeNameContext;
	}
	public PayableKeyword(): TerminalNode {
		return this.getToken(SolidityParser.PayableKeyword, 0);
	}
	public typeName(): TypeNameContext {
		return this.getTypedRuleContext(TypeNameContext, 0) as TypeNameContext;
	}
	public expression(): ExpressionContext {
		return this.getTypedRuleContext(ExpressionContext, 0) as ExpressionContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_typeName;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterTypeName) {
	 		listener.enterTypeName(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitTypeName) {
	 		listener.exitTypeName(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitTypeName) {
			return visitor.visitTypeName(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class UserDefinedTypeNameContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public identifier_list(): IdentifierContext[] {
		return this.getTypedRuleContexts(IdentifierContext) as IdentifierContext[];
	}
	public identifier(i: number): IdentifierContext {
		return this.getTypedRuleContext(IdentifierContext, i) as IdentifierContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_userDefinedTypeName;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterUserDefinedTypeName) {
	 		listener.enterUserDefinedTypeName(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitUserDefinedTypeName) {
	 		listener.exitUserDefinedTypeName(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitUserDefinedTypeName) {
			return visitor.visitUserDefinedTypeName(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class MappingKeyContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public elementaryTypeName(): ElementaryTypeNameContext {
		return this.getTypedRuleContext(ElementaryTypeNameContext, 0) as ElementaryTypeNameContext;
	}
	public userDefinedTypeName(): UserDefinedTypeNameContext {
		return this.getTypedRuleContext(UserDefinedTypeNameContext, 0) as UserDefinedTypeNameContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_mappingKey;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterMappingKey) {
	 		listener.enterMappingKey(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitMappingKey) {
	 		listener.exitMappingKey(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitMappingKey) {
			return visitor.visitMappingKey(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class MappingContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public mappingKey(): MappingKeyContext {
		return this.getTypedRuleContext(MappingKeyContext, 0) as MappingKeyContext;
	}
	public typeName(): TypeNameContext {
		return this.getTypedRuleContext(TypeNameContext, 0) as TypeNameContext;
	}
	public mappingKeyName(): MappingKeyNameContext {
		return this.getTypedRuleContext(MappingKeyNameContext, 0) as MappingKeyNameContext;
	}
	public mappingValueName(): MappingValueNameContext {
		return this.getTypedRuleContext(MappingValueNameContext, 0) as MappingValueNameContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_mapping;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterMapping) {
	 		listener.enterMapping(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitMapping) {
	 		listener.exitMapping(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitMapping) {
			return visitor.visitMapping(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class MappingKeyNameContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public identifier(): IdentifierContext {
		return this.getTypedRuleContext(IdentifierContext, 0) as IdentifierContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_mappingKeyName;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterMappingKeyName) {
	 		listener.enterMappingKeyName(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitMappingKeyName) {
	 		listener.exitMappingKeyName(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitMappingKeyName) {
			return visitor.visitMappingKeyName(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class MappingValueNameContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public identifier(): IdentifierContext {
		return this.getTypedRuleContext(IdentifierContext, 0) as IdentifierContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_mappingValueName;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterMappingValueName) {
	 		listener.enterMappingValueName(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitMappingValueName) {
	 		listener.exitMappingValueName(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitMappingValueName) {
			return visitor.visitMappingValueName(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class FunctionTypeNameContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public functionTypeParameterList_list(): FunctionTypeParameterListContext[] {
		return this.getTypedRuleContexts(FunctionTypeParameterListContext) as FunctionTypeParameterListContext[];
	}
	public functionTypeParameterList(i: number): FunctionTypeParameterListContext {
		return this.getTypedRuleContext(FunctionTypeParameterListContext, i) as FunctionTypeParameterListContext;
	}
	public InternalKeyword_list(): TerminalNode[] {
	    	return this.getTokens(SolidityParser.InternalKeyword);
	}
	public InternalKeyword(i: number): TerminalNode {
		return this.getToken(SolidityParser.InternalKeyword, i);
	}
	public ExternalKeyword_list(): TerminalNode[] {
	    	return this.getTokens(SolidityParser.ExternalKeyword);
	}
	public ExternalKeyword(i: number): TerminalNode {
		return this.getToken(SolidityParser.ExternalKeyword, i);
	}
	public stateMutability_list(): StateMutabilityContext[] {
		return this.getTypedRuleContexts(StateMutabilityContext) as StateMutabilityContext[];
	}
	public stateMutability(i: number): StateMutabilityContext {
		return this.getTypedRuleContext(StateMutabilityContext, i) as StateMutabilityContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_functionTypeName;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterFunctionTypeName) {
	 		listener.enterFunctionTypeName(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitFunctionTypeName) {
	 		listener.exitFunctionTypeName(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitFunctionTypeName) {
			return visitor.visitFunctionTypeName(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class StorageLocationContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_storageLocation;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterStorageLocation) {
	 		listener.enterStorageLocation(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitStorageLocation) {
	 		listener.exitStorageLocation(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitStorageLocation) {
			return visitor.visitStorageLocation(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class StateMutabilityContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public PureKeyword(): TerminalNode {
		return this.getToken(SolidityParser.PureKeyword, 0);
	}
	public ConstantKeyword(): TerminalNode {
		return this.getToken(SolidityParser.ConstantKeyword, 0);
	}
	public ViewKeyword(): TerminalNode {
		return this.getToken(SolidityParser.ViewKeyword, 0);
	}
	public PayableKeyword(): TerminalNode {
		return this.getToken(SolidityParser.PayableKeyword, 0);
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_stateMutability;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterStateMutability) {
	 		listener.enterStateMutability(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitStateMutability) {
	 		listener.exitStateMutability(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitStateMutability) {
			return visitor.visitStateMutability(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class BlockContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public statement_list(): StatementContext[] {
		return this.getTypedRuleContexts(StatementContext) as StatementContext[];
	}
	public statement(i: number): StatementContext {
		return this.getTypedRuleContext(StatementContext, i) as StatementContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_block;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterBlock) {
	 		listener.enterBlock(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitBlock) {
	 		listener.exitBlock(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitBlock) {
			return visitor.visitBlock(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class StatementContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public ifStatement(): IfStatementContext {
		return this.getTypedRuleContext(IfStatementContext, 0) as IfStatementContext;
	}
	public tryStatement(): TryStatementContext {
		return this.getTypedRuleContext(TryStatementContext, 0) as TryStatementContext;
	}
	public whileStatement(): WhileStatementContext {
		return this.getTypedRuleContext(WhileStatementContext, 0) as WhileStatementContext;
	}
	public forStatement(): ForStatementContext {
		return this.getTypedRuleContext(ForStatementContext, 0) as ForStatementContext;
	}
	public block(): BlockContext {
		return this.getTypedRuleContext(BlockContext, 0) as BlockContext;
	}
	public inlineAssemblyStatement(): InlineAssemblyStatementContext {
		return this.getTypedRuleContext(InlineAssemblyStatementContext, 0) as InlineAssemblyStatementContext;
	}
	public doWhileStatement(): DoWhileStatementContext {
		return this.getTypedRuleContext(DoWhileStatementContext, 0) as DoWhileStatementContext;
	}
	public continueStatement(): ContinueStatementContext {
		return this.getTypedRuleContext(ContinueStatementContext, 0) as ContinueStatementContext;
	}
	public breakStatement(): BreakStatementContext {
		return this.getTypedRuleContext(BreakStatementContext, 0) as BreakStatementContext;
	}
	public returnStatement(): ReturnStatementContext {
		return this.getTypedRuleContext(ReturnStatementContext, 0) as ReturnStatementContext;
	}
	public throwStatement(): ThrowStatementContext {
		return this.getTypedRuleContext(ThrowStatementContext, 0) as ThrowStatementContext;
	}
	public emitStatement(): EmitStatementContext {
		return this.getTypedRuleContext(EmitStatementContext, 0) as EmitStatementContext;
	}
	public simpleStatement(): SimpleStatementContext {
		return this.getTypedRuleContext(SimpleStatementContext, 0) as SimpleStatementContext;
	}
	public uncheckedStatement(): UncheckedStatementContext {
		return this.getTypedRuleContext(UncheckedStatementContext, 0) as UncheckedStatementContext;
	}
	public revertStatement(): RevertStatementContext {
		return this.getTypedRuleContext(RevertStatementContext, 0) as RevertStatementContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_statement;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterStatement) {
	 		listener.enterStatement(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitStatement) {
	 		listener.exitStatement(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitStatement) {
			return visitor.visitStatement(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class ExpressionStatementContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public expression(): ExpressionContext {
		return this.getTypedRuleContext(ExpressionContext, 0) as ExpressionContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_expressionStatement;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterExpressionStatement) {
	 		listener.enterExpressionStatement(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitExpressionStatement) {
	 		listener.exitExpressionStatement(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitExpressionStatement) {
			return visitor.visitExpressionStatement(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class IfStatementContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public expression(): ExpressionContext {
		return this.getTypedRuleContext(ExpressionContext, 0) as ExpressionContext;
	}
	public statement_list(): StatementContext[] {
		return this.getTypedRuleContexts(StatementContext) as StatementContext[];
	}
	public statement(i: number): StatementContext {
		return this.getTypedRuleContext(StatementContext, i) as StatementContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_ifStatement;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterIfStatement) {
	 		listener.enterIfStatement(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitIfStatement) {
	 		listener.exitIfStatement(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitIfStatement) {
			return visitor.visitIfStatement(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class TryStatementContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public expression(): ExpressionContext {
		return this.getTypedRuleContext(ExpressionContext, 0) as ExpressionContext;
	}
	public block(): BlockContext {
		return this.getTypedRuleContext(BlockContext, 0) as BlockContext;
	}
	public returnParameters(): ReturnParametersContext {
		return this.getTypedRuleContext(ReturnParametersContext, 0) as ReturnParametersContext;
	}
	public catchClause_list(): CatchClauseContext[] {
		return this.getTypedRuleContexts(CatchClauseContext) as CatchClauseContext[];
	}
	public catchClause(i: number): CatchClauseContext {
		return this.getTypedRuleContext(CatchClauseContext, i) as CatchClauseContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_tryStatement;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterTryStatement) {
	 		listener.enterTryStatement(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitTryStatement) {
	 		listener.exitTryStatement(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitTryStatement) {
			return visitor.visitTryStatement(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class CatchClauseContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public block(): BlockContext {
		return this.getTypedRuleContext(BlockContext, 0) as BlockContext;
	}
	public parameterList(): ParameterListContext {
		return this.getTypedRuleContext(ParameterListContext, 0) as ParameterListContext;
	}
	public identifier(): IdentifierContext {
		return this.getTypedRuleContext(IdentifierContext, 0) as IdentifierContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_catchClause;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterCatchClause) {
	 		listener.enterCatchClause(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitCatchClause) {
	 		listener.exitCatchClause(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitCatchClause) {
			return visitor.visitCatchClause(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class WhileStatementContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public expression(): ExpressionContext {
		return this.getTypedRuleContext(ExpressionContext, 0) as ExpressionContext;
	}
	public statement(): StatementContext {
		return this.getTypedRuleContext(StatementContext, 0) as StatementContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_whileStatement;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterWhileStatement) {
	 		listener.enterWhileStatement(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitWhileStatement) {
	 		listener.exitWhileStatement(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitWhileStatement) {
			return visitor.visitWhileStatement(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class SimpleStatementContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public variableDeclarationStatement(): VariableDeclarationStatementContext {
		return this.getTypedRuleContext(VariableDeclarationStatementContext, 0) as VariableDeclarationStatementContext;
	}
	public expressionStatement(): ExpressionStatementContext {
		return this.getTypedRuleContext(ExpressionStatementContext, 0) as ExpressionStatementContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_simpleStatement;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterSimpleStatement) {
	 		listener.enterSimpleStatement(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitSimpleStatement) {
	 		listener.exitSimpleStatement(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitSimpleStatement) {
			return visitor.visitSimpleStatement(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class UncheckedStatementContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public block(): BlockContext {
		return this.getTypedRuleContext(BlockContext, 0) as BlockContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_uncheckedStatement;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterUncheckedStatement) {
	 		listener.enterUncheckedStatement(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitUncheckedStatement) {
	 		listener.exitUncheckedStatement(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitUncheckedStatement) {
			return visitor.visitUncheckedStatement(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class ForStatementContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public statement(): StatementContext {
		return this.getTypedRuleContext(StatementContext, 0) as StatementContext;
	}
	public simpleStatement(): SimpleStatementContext {
		return this.getTypedRuleContext(SimpleStatementContext, 0) as SimpleStatementContext;
	}
	public expressionStatement(): ExpressionStatementContext {
		return this.getTypedRuleContext(ExpressionStatementContext, 0) as ExpressionStatementContext;
	}
	public expression(): ExpressionContext {
		return this.getTypedRuleContext(ExpressionContext, 0) as ExpressionContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_forStatement;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterForStatement) {
	 		listener.enterForStatement(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitForStatement) {
	 		listener.exitForStatement(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitForStatement) {
			return visitor.visitForStatement(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class InlineAssemblyStatementContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public assemblyBlock(): AssemblyBlockContext {
		return this.getTypedRuleContext(AssemblyBlockContext, 0) as AssemblyBlockContext;
	}
	public StringLiteralFragment(): TerminalNode {
		return this.getToken(SolidityParser.StringLiteralFragment, 0);
	}
	public inlineAssemblyStatementFlag(): InlineAssemblyStatementFlagContext {
		return this.getTypedRuleContext(InlineAssemblyStatementFlagContext, 0) as InlineAssemblyStatementFlagContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_inlineAssemblyStatement;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterInlineAssemblyStatement) {
	 		listener.enterInlineAssemblyStatement(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitInlineAssemblyStatement) {
	 		listener.exitInlineAssemblyStatement(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitInlineAssemblyStatement) {
			return visitor.visitInlineAssemblyStatement(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class InlineAssemblyStatementFlagContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public stringLiteral(): StringLiteralContext {
		return this.getTypedRuleContext(StringLiteralContext, 0) as StringLiteralContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_inlineAssemblyStatementFlag;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterInlineAssemblyStatementFlag) {
	 		listener.enterInlineAssemblyStatementFlag(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitInlineAssemblyStatementFlag) {
	 		listener.exitInlineAssemblyStatementFlag(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitInlineAssemblyStatementFlag) {
			return visitor.visitInlineAssemblyStatementFlag(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class DoWhileStatementContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public statement(): StatementContext {
		return this.getTypedRuleContext(StatementContext, 0) as StatementContext;
	}
	public expression(): ExpressionContext {
		return this.getTypedRuleContext(ExpressionContext, 0) as ExpressionContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_doWhileStatement;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterDoWhileStatement) {
	 		listener.enterDoWhileStatement(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitDoWhileStatement) {
	 		listener.exitDoWhileStatement(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitDoWhileStatement) {
			return visitor.visitDoWhileStatement(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class ContinueStatementContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public ContinueKeyword(): TerminalNode {
		return this.getToken(SolidityParser.ContinueKeyword, 0);
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_continueStatement;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterContinueStatement) {
	 		listener.enterContinueStatement(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitContinueStatement) {
	 		listener.exitContinueStatement(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitContinueStatement) {
			return visitor.visitContinueStatement(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class BreakStatementContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public BreakKeyword(): TerminalNode {
		return this.getToken(SolidityParser.BreakKeyword, 0);
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_breakStatement;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterBreakStatement) {
	 		listener.enterBreakStatement(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitBreakStatement) {
	 		listener.exitBreakStatement(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitBreakStatement) {
			return visitor.visitBreakStatement(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class ReturnStatementContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public expression(): ExpressionContext {
		return this.getTypedRuleContext(ExpressionContext, 0) as ExpressionContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_returnStatement;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterReturnStatement) {
	 		listener.enterReturnStatement(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitReturnStatement) {
	 		listener.exitReturnStatement(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitReturnStatement) {
			return visitor.visitReturnStatement(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class ThrowStatementContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_throwStatement;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterThrowStatement) {
	 		listener.enterThrowStatement(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitThrowStatement) {
	 		listener.exitThrowStatement(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitThrowStatement) {
			return visitor.visitThrowStatement(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class EmitStatementContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public functionCall(): FunctionCallContext {
		return this.getTypedRuleContext(FunctionCallContext, 0) as FunctionCallContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_emitStatement;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterEmitStatement) {
	 		listener.enterEmitStatement(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitEmitStatement) {
	 		listener.exitEmitStatement(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitEmitStatement) {
			return visitor.visitEmitStatement(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class RevertStatementContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public functionCall(): FunctionCallContext {
		return this.getTypedRuleContext(FunctionCallContext, 0) as FunctionCallContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_revertStatement;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterRevertStatement) {
	 		listener.enterRevertStatement(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitRevertStatement) {
	 		listener.exitRevertStatement(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitRevertStatement) {
			return visitor.visitRevertStatement(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class VariableDeclarationStatementContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public identifierList(): IdentifierListContext {
		return this.getTypedRuleContext(IdentifierListContext, 0) as IdentifierListContext;
	}
	public variableDeclaration(): VariableDeclarationContext {
		return this.getTypedRuleContext(VariableDeclarationContext, 0) as VariableDeclarationContext;
	}
	public variableDeclarationList(): VariableDeclarationListContext {
		return this.getTypedRuleContext(VariableDeclarationListContext, 0) as VariableDeclarationListContext;
	}
	public expression(): ExpressionContext {
		return this.getTypedRuleContext(ExpressionContext, 0) as ExpressionContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_variableDeclarationStatement;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterVariableDeclarationStatement) {
	 		listener.enterVariableDeclarationStatement(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitVariableDeclarationStatement) {
	 		listener.exitVariableDeclarationStatement(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitVariableDeclarationStatement) {
			return visitor.visitVariableDeclarationStatement(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class VariableDeclarationListContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public variableDeclaration_list(): VariableDeclarationContext[] {
		return this.getTypedRuleContexts(VariableDeclarationContext) as VariableDeclarationContext[];
	}
	public variableDeclaration(i: number): VariableDeclarationContext {
		return this.getTypedRuleContext(VariableDeclarationContext, i) as VariableDeclarationContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_variableDeclarationList;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterVariableDeclarationList) {
	 		listener.enterVariableDeclarationList(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitVariableDeclarationList) {
	 		listener.exitVariableDeclarationList(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitVariableDeclarationList) {
			return visitor.visitVariableDeclarationList(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class IdentifierListContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public identifier_list(): IdentifierContext[] {
		return this.getTypedRuleContexts(IdentifierContext) as IdentifierContext[];
	}
	public identifier(i: number): IdentifierContext {
		return this.getTypedRuleContext(IdentifierContext, i) as IdentifierContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_identifierList;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterIdentifierList) {
	 		listener.enterIdentifierList(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitIdentifierList) {
	 		listener.exitIdentifierList(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitIdentifierList) {
			return visitor.visitIdentifierList(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class ElementaryTypeNameContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public Int(): TerminalNode {
		return this.getToken(SolidityParser.Int, 0);
	}
	public Uint(): TerminalNode {
		return this.getToken(SolidityParser.Uint, 0);
	}
	public Byte(): TerminalNode {
		return this.getToken(SolidityParser.Byte, 0);
	}
	public Fixed(): TerminalNode {
		return this.getToken(SolidityParser.Fixed, 0);
	}
	public Ufixed(): TerminalNode {
		return this.getToken(SolidityParser.Ufixed, 0);
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_elementaryTypeName;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterElementaryTypeName) {
	 		listener.enterElementaryTypeName(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitElementaryTypeName) {
	 		listener.exitElementaryTypeName(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitElementaryTypeName) {
			return visitor.visitElementaryTypeName(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class ExpressionContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public typeName(): TypeNameContext {
		return this.getTypedRuleContext(TypeNameContext, 0) as TypeNameContext;
	}
	public expression_list(): ExpressionContext[] {
		return this.getTypedRuleContexts(ExpressionContext) as ExpressionContext[];
	}
	public expression(i: number): ExpressionContext {
		return this.getTypedRuleContext(ExpressionContext, i) as ExpressionContext;
	}
	public primaryExpression(): PrimaryExpressionContext {
		return this.getTypedRuleContext(PrimaryExpressionContext, 0) as PrimaryExpressionContext;
	}
	public identifier(): IdentifierContext {
		return this.getTypedRuleContext(IdentifierContext, 0) as IdentifierContext;
	}
	public nameValueList(): NameValueListContext {
		return this.getTypedRuleContext(NameValueListContext, 0) as NameValueListContext;
	}
	public functionCallArguments(): FunctionCallArgumentsContext {
		return this.getTypedRuleContext(FunctionCallArgumentsContext, 0) as FunctionCallArgumentsContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_expression;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterExpression) {
	 		listener.enterExpression(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitExpression) {
	 		listener.exitExpression(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitExpression) {
			return visitor.visitExpression(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class PrimaryExpressionContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public BooleanLiteral(): TerminalNode {
		return this.getToken(SolidityParser.BooleanLiteral, 0);
	}
	public numberLiteral(): NumberLiteralContext {
		return this.getTypedRuleContext(NumberLiteralContext, 0) as NumberLiteralContext;
	}
	public hexLiteral(): HexLiteralContext {
		return this.getTypedRuleContext(HexLiteralContext, 0) as HexLiteralContext;
	}
	public stringLiteral(): StringLiteralContext {
		return this.getTypedRuleContext(StringLiteralContext, 0) as StringLiteralContext;
	}
	public identifier(): IdentifierContext {
		return this.getTypedRuleContext(IdentifierContext, 0) as IdentifierContext;
	}
	public TypeKeyword(): TerminalNode {
		return this.getToken(SolidityParser.TypeKeyword, 0);
	}
	public PayableKeyword(): TerminalNode {
		return this.getToken(SolidityParser.PayableKeyword, 0);
	}
	public tupleExpression(): TupleExpressionContext {
		return this.getTypedRuleContext(TupleExpressionContext, 0) as TupleExpressionContext;
	}
	public typeName(): TypeNameContext {
		return this.getTypedRuleContext(TypeNameContext, 0) as TypeNameContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_primaryExpression;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterPrimaryExpression) {
	 		listener.enterPrimaryExpression(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitPrimaryExpression) {
	 		listener.exitPrimaryExpression(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitPrimaryExpression) {
			return visitor.visitPrimaryExpression(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class ExpressionListContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public expression_list(): ExpressionContext[] {
		return this.getTypedRuleContexts(ExpressionContext) as ExpressionContext[];
	}
	public expression(i: number): ExpressionContext {
		return this.getTypedRuleContext(ExpressionContext, i) as ExpressionContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_expressionList;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterExpressionList) {
	 		listener.enterExpressionList(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitExpressionList) {
	 		listener.exitExpressionList(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitExpressionList) {
			return visitor.visitExpressionList(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class NameValueListContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public nameValue_list(): NameValueContext[] {
		return this.getTypedRuleContexts(NameValueContext) as NameValueContext[];
	}
	public nameValue(i: number): NameValueContext {
		return this.getTypedRuleContext(NameValueContext, i) as NameValueContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_nameValueList;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterNameValueList) {
	 		listener.enterNameValueList(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitNameValueList) {
	 		listener.exitNameValueList(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitNameValueList) {
			return visitor.visitNameValueList(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class NameValueContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public identifier(): IdentifierContext {
		return this.getTypedRuleContext(IdentifierContext, 0) as IdentifierContext;
	}
	public expression(): ExpressionContext {
		return this.getTypedRuleContext(ExpressionContext, 0) as ExpressionContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_nameValue;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterNameValue) {
	 		listener.enterNameValue(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitNameValue) {
	 		listener.exitNameValue(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitNameValue) {
			return visitor.visitNameValue(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class FunctionCallArgumentsContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public nameValueList(): NameValueListContext {
		return this.getTypedRuleContext(NameValueListContext, 0) as NameValueListContext;
	}
	public expressionList(): ExpressionListContext {
		return this.getTypedRuleContext(ExpressionListContext, 0) as ExpressionListContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_functionCallArguments;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterFunctionCallArguments) {
	 		listener.enterFunctionCallArguments(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitFunctionCallArguments) {
	 		listener.exitFunctionCallArguments(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitFunctionCallArguments) {
			return visitor.visitFunctionCallArguments(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class FunctionCallContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public expression(): ExpressionContext {
		return this.getTypedRuleContext(ExpressionContext, 0) as ExpressionContext;
	}
	public functionCallArguments(): FunctionCallArgumentsContext {
		return this.getTypedRuleContext(FunctionCallArgumentsContext, 0) as FunctionCallArgumentsContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_functionCall;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterFunctionCall) {
	 		listener.enterFunctionCall(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitFunctionCall) {
	 		listener.exitFunctionCall(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitFunctionCall) {
			return visitor.visitFunctionCall(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class AssemblyBlockContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public assemblyItem_list(): AssemblyItemContext[] {
		return this.getTypedRuleContexts(AssemblyItemContext) as AssemblyItemContext[];
	}
	public assemblyItem(i: number): AssemblyItemContext {
		return this.getTypedRuleContext(AssemblyItemContext, i) as AssemblyItemContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_assemblyBlock;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterAssemblyBlock) {
	 		listener.enterAssemblyBlock(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitAssemblyBlock) {
	 		listener.exitAssemblyBlock(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitAssemblyBlock) {
			return visitor.visitAssemblyBlock(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class AssemblyItemContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public identifier(): IdentifierContext {
		return this.getTypedRuleContext(IdentifierContext, 0) as IdentifierContext;
	}
	public assemblyBlock(): AssemblyBlockContext {
		return this.getTypedRuleContext(AssemblyBlockContext, 0) as AssemblyBlockContext;
	}
	public assemblyExpression(): AssemblyExpressionContext {
		return this.getTypedRuleContext(AssemblyExpressionContext, 0) as AssemblyExpressionContext;
	}
	public assemblyLocalDefinition(): AssemblyLocalDefinitionContext {
		return this.getTypedRuleContext(AssemblyLocalDefinitionContext, 0) as AssemblyLocalDefinitionContext;
	}
	public assemblyAssignment(): AssemblyAssignmentContext {
		return this.getTypedRuleContext(AssemblyAssignmentContext, 0) as AssemblyAssignmentContext;
	}
	public assemblyStackAssignment(): AssemblyStackAssignmentContext {
		return this.getTypedRuleContext(AssemblyStackAssignmentContext, 0) as AssemblyStackAssignmentContext;
	}
	public labelDefinition(): LabelDefinitionContext {
		return this.getTypedRuleContext(LabelDefinitionContext, 0) as LabelDefinitionContext;
	}
	public assemblySwitch(): AssemblySwitchContext {
		return this.getTypedRuleContext(AssemblySwitchContext, 0) as AssemblySwitchContext;
	}
	public assemblyFunctionDefinition(): AssemblyFunctionDefinitionContext {
		return this.getTypedRuleContext(AssemblyFunctionDefinitionContext, 0) as AssemblyFunctionDefinitionContext;
	}
	public assemblyFor(): AssemblyForContext {
		return this.getTypedRuleContext(AssemblyForContext, 0) as AssemblyForContext;
	}
	public assemblyIf(): AssemblyIfContext {
		return this.getTypedRuleContext(AssemblyIfContext, 0) as AssemblyIfContext;
	}
	public BreakKeyword(): TerminalNode {
		return this.getToken(SolidityParser.BreakKeyword, 0);
	}
	public ContinueKeyword(): TerminalNode {
		return this.getToken(SolidityParser.ContinueKeyword, 0);
	}
	public LeaveKeyword(): TerminalNode {
		return this.getToken(SolidityParser.LeaveKeyword, 0);
	}
	public numberLiteral(): NumberLiteralContext {
		return this.getTypedRuleContext(NumberLiteralContext, 0) as NumberLiteralContext;
	}
	public stringLiteral(): StringLiteralContext {
		return this.getTypedRuleContext(StringLiteralContext, 0) as StringLiteralContext;
	}
	public hexLiteral(): HexLiteralContext {
		return this.getTypedRuleContext(HexLiteralContext, 0) as HexLiteralContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_assemblyItem;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterAssemblyItem) {
	 		listener.enterAssemblyItem(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitAssemblyItem) {
	 		listener.exitAssemblyItem(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitAssemblyItem) {
			return visitor.visitAssemblyItem(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class AssemblyExpressionContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public assemblyCall(): AssemblyCallContext {
		return this.getTypedRuleContext(AssemblyCallContext, 0) as AssemblyCallContext;
	}
	public assemblyLiteral(): AssemblyLiteralContext {
		return this.getTypedRuleContext(AssemblyLiteralContext, 0) as AssemblyLiteralContext;
	}
	public assemblyMember(): AssemblyMemberContext {
		return this.getTypedRuleContext(AssemblyMemberContext, 0) as AssemblyMemberContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_assemblyExpression;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterAssemblyExpression) {
	 		listener.enterAssemblyExpression(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitAssemblyExpression) {
	 		listener.exitAssemblyExpression(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitAssemblyExpression) {
			return visitor.visitAssemblyExpression(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class AssemblyMemberContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public identifier_list(): IdentifierContext[] {
		return this.getTypedRuleContexts(IdentifierContext) as IdentifierContext[];
	}
	public identifier(i: number): IdentifierContext {
		return this.getTypedRuleContext(IdentifierContext, i) as IdentifierContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_assemblyMember;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterAssemblyMember) {
	 		listener.enterAssemblyMember(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitAssemblyMember) {
	 		listener.exitAssemblyMember(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitAssemblyMember) {
			return visitor.visitAssemblyMember(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class AssemblyCallContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public identifier(): IdentifierContext {
		return this.getTypedRuleContext(IdentifierContext, 0) as IdentifierContext;
	}
	public assemblyExpression_list(): AssemblyExpressionContext[] {
		return this.getTypedRuleContexts(AssemblyExpressionContext) as AssemblyExpressionContext[];
	}
	public assemblyExpression(i: number): AssemblyExpressionContext {
		return this.getTypedRuleContext(AssemblyExpressionContext, i) as AssemblyExpressionContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_assemblyCall;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterAssemblyCall) {
	 		listener.enterAssemblyCall(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitAssemblyCall) {
	 		listener.exitAssemblyCall(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitAssemblyCall) {
			return visitor.visitAssemblyCall(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class AssemblyLocalDefinitionContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public assemblyIdentifierOrList(): AssemblyIdentifierOrListContext {
		return this.getTypedRuleContext(AssemblyIdentifierOrListContext, 0) as AssemblyIdentifierOrListContext;
	}
	public assemblyExpression(): AssemblyExpressionContext {
		return this.getTypedRuleContext(AssemblyExpressionContext, 0) as AssemblyExpressionContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_assemblyLocalDefinition;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterAssemblyLocalDefinition) {
	 		listener.enterAssemblyLocalDefinition(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitAssemblyLocalDefinition) {
	 		listener.exitAssemblyLocalDefinition(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitAssemblyLocalDefinition) {
			return visitor.visitAssemblyLocalDefinition(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class AssemblyAssignmentContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public assemblyIdentifierOrList(): AssemblyIdentifierOrListContext {
		return this.getTypedRuleContext(AssemblyIdentifierOrListContext, 0) as AssemblyIdentifierOrListContext;
	}
	public assemblyExpression(): AssemblyExpressionContext {
		return this.getTypedRuleContext(AssemblyExpressionContext, 0) as AssemblyExpressionContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_assemblyAssignment;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterAssemblyAssignment) {
	 		listener.enterAssemblyAssignment(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitAssemblyAssignment) {
	 		listener.exitAssemblyAssignment(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitAssemblyAssignment) {
			return visitor.visitAssemblyAssignment(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class AssemblyIdentifierOrListContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public identifier(): IdentifierContext {
		return this.getTypedRuleContext(IdentifierContext, 0) as IdentifierContext;
	}
	public assemblyMember(): AssemblyMemberContext {
		return this.getTypedRuleContext(AssemblyMemberContext, 0) as AssemblyMemberContext;
	}
	public assemblyIdentifierList(): AssemblyIdentifierListContext {
		return this.getTypedRuleContext(AssemblyIdentifierListContext, 0) as AssemblyIdentifierListContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_assemblyIdentifierOrList;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterAssemblyIdentifierOrList) {
	 		listener.enterAssemblyIdentifierOrList(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitAssemblyIdentifierOrList) {
	 		listener.exitAssemblyIdentifierOrList(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitAssemblyIdentifierOrList) {
			return visitor.visitAssemblyIdentifierOrList(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class AssemblyIdentifierListContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public identifier_list(): IdentifierContext[] {
		return this.getTypedRuleContexts(IdentifierContext) as IdentifierContext[];
	}
	public identifier(i: number): IdentifierContext {
		return this.getTypedRuleContext(IdentifierContext, i) as IdentifierContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_assemblyIdentifierList;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterAssemblyIdentifierList) {
	 		listener.enterAssemblyIdentifierList(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitAssemblyIdentifierList) {
	 		listener.exitAssemblyIdentifierList(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitAssemblyIdentifierList) {
			return visitor.visitAssemblyIdentifierList(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class AssemblyStackAssignmentContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public assemblyExpression(): AssemblyExpressionContext {
		return this.getTypedRuleContext(AssemblyExpressionContext, 0) as AssemblyExpressionContext;
	}
	public identifier(): IdentifierContext {
		return this.getTypedRuleContext(IdentifierContext, 0) as IdentifierContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_assemblyStackAssignment;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterAssemblyStackAssignment) {
	 		listener.enterAssemblyStackAssignment(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitAssemblyStackAssignment) {
	 		listener.exitAssemblyStackAssignment(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitAssemblyStackAssignment) {
			return visitor.visitAssemblyStackAssignment(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class LabelDefinitionContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public identifier(): IdentifierContext {
		return this.getTypedRuleContext(IdentifierContext, 0) as IdentifierContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_labelDefinition;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterLabelDefinition) {
	 		listener.enterLabelDefinition(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitLabelDefinition) {
	 		listener.exitLabelDefinition(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitLabelDefinition) {
			return visitor.visitLabelDefinition(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class AssemblySwitchContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public assemblyExpression(): AssemblyExpressionContext {
		return this.getTypedRuleContext(AssemblyExpressionContext, 0) as AssemblyExpressionContext;
	}
	public assemblyCase_list(): AssemblyCaseContext[] {
		return this.getTypedRuleContexts(AssemblyCaseContext) as AssemblyCaseContext[];
	}
	public assemblyCase(i: number): AssemblyCaseContext {
		return this.getTypedRuleContext(AssemblyCaseContext, i) as AssemblyCaseContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_assemblySwitch;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterAssemblySwitch) {
	 		listener.enterAssemblySwitch(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitAssemblySwitch) {
	 		listener.exitAssemblySwitch(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitAssemblySwitch) {
			return visitor.visitAssemblySwitch(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class AssemblyCaseContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public assemblyLiteral(): AssemblyLiteralContext {
		return this.getTypedRuleContext(AssemblyLiteralContext, 0) as AssemblyLiteralContext;
	}
	public assemblyBlock(): AssemblyBlockContext {
		return this.getTypedRuleContext(AssemblyBlockContext, 0) as AssemblyBlockContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_assemblyCase;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterAssemblyCase) {
	 		listener.enterAssemblyCase(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitAssemblyCase) {
	 		listener.exitAssemblyCase(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitAssemblyCase) {
			return visitor.visitAssemblyCase(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class AssemblyFunctionDefinitionContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public identifier(): IdentifierContext {
		return this.getTypedRuleContext(IdentifierContext, 0) as IdentifierContext;
	}
	public assemblyBlock(): AssemblyBlockContext {
		return this.getTypedRuleContext(AssemblyBlockContext, 0) as AssemblyBlockContext;
	}
	public assemblyIdentifierList(): AssemblyIdentifierListContext {
		return this.getTypedRuleContext(AssemblyIdentifierListContext, 0) as AssemblyIdentifierListContext;
	}
	public assemblyFunctionReturns(): AssemblyFunctionReturnsContext {
		return this.getTypedRuleContext(AssemblyFunctionReturnsContext, 0) as AssemblyFunctionReturnsContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_assemblyFunctionDefinition;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterAssemblyFunctionDefinition) {
	 		listener.enterAssemblyFunctionDefinition(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitAssemblyFunctionDefinition) {
	 		listener.exitAssemblyFunctionDefinition(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitAssemblyFunctionDefinition) {
			return visitor.visitAssemblyFunctionDefinition(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class AssemblyFunctionReturnsContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public assemblyIdentifierList(): AssemblyIdentifierListContext {
		return this.getTypedRuleContext(AssemblyIdentifierListContext, 0) as AssemblyIdentifierListContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_assemblyFunctionReturns;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterAssemblyFunctionReturns) {
	 		listener.enterAssemblyFunctionReturns(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitAssemblyFunctionReturns) {
	 		listener.exitAssemblyFunctionReturns(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitAssemblyFunctionReturns) {
			return visitor.visitAssemblyFunctionReturns(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class AssemblyForContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public assemblyExpression_list(): AssemblyExpressionContext[] {
		return this.getTypedRuleContexts(AssemblyExpressionContext) as AssemblyExpressionContext[];
	}
	public assemblyExpression(i: number): AssemblyExpressionContext {
		return this.getTypedRuleContext(AssemblyExpressionContext, i) as AssemblyExpressionContext;
	}
	public assemblyBlock_list(): AssemblyBlockContext[] {
		return this.getTypedRuleContexts(AssemblyBlockContext) as AssemblyBlockContext[];
	}
	public assemblyBlock(i: number): AssemblyBlockContext {
		return this.getTypedRuleContext(AssemblyBlockContext, i) as AssemblyBlockContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_assemblyFor;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterAssemblyFor) {
	 		listener.enterAssemblyFor(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitAssemblyFor) {
	 		listener.exitAssemblyFor(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitAssemblyFor) {
			return visitor.visitAssemblyFor(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class AssemblyIfContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public assemblyExpression(): AssemblyExpressionContext {
		return this.getTypedRuleContext(AssemblyExpressionContext, 0) as AssemblyExpressionContext;
	}
	public assemblyBlock(): AssemblyBlockContext {
		return this.getTypedRuleContext(AssemblyBlockContext, 0) as AssemblyBlockContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_assemblyIf;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterAssemblyIf) {
	 		listener.enterAssemblyIf(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitAssemblyIf) {
	 		listener.exitAssemblyIf(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitAssemblyIf) {
			return visitor.visitAssemblyIf(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class AssemblyLiteralContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public stringLiteral(): StringLiteralContext {
		return this.getTypedRuleContext(StringLiteralContext, 0) as StringLiteralContext;
	}
	public DecimalNumber(): TerminalNode {
		return this.getToken(SolidityParser.DecimalNumber, 0);
	}
	public HexNumber(): TerminalNode {
		return this.getToken(SolidityParser.HexNumber, 0);
	}
	public hexLiteral(): HexLiteralContext {
		return this.getTypedRuleContext(HexLiteralContext, 0) as HexLiteralContext;
	}
	public BooleanLiteral(): TerminalNode {
		return this.getToken(SolidityParser.BooleanLiteral, 0);
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_assemblyLiteral;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterAssemblyLiteral) {
	 		listener.enterAssemblyLiteral(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitAssemblyLiteral) {
	 		listener.exitAssemblyLiteral(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitAssemblyLiteral) {
			return visitor.visitAssemblyLiteral(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class TupleExpressionContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public expression_list(): ExpressionContext[] {
		return this.getTypedRuleContexts(ExpressionContext) as ExpressionContext[];
	}
	public expression(i: number): ExpressionContext {
		return this.getTypedRuleContext(ExpressionContext, i) as ExpressionContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_tupleExpression;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterTupleExpression) {
	 		listener.enterTupleExpression(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitTupleExpression) {
	 		listener.exitTupleExpression(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitTupleExpression) {
			return visitor.visitTupleExpression(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class NumberLiteralContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public DecimalNumber(): TerminalNode {
		return this.getToken(SolidityParser.DecimalNumber, 0);
	}
	public HexNumber(): TerminalNode {
		return this.getToken(SolidityParser.HexNumber, 0);
	}
	public NumberUnit(): TerminalNode {
		return this.getToken(SolidityParser.NumberUnit, 0);
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_numberLiteral;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterNumberLiteral) {
	 		listener.enterNumberLiteral(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitNumberLiteral) {
	 		listener.exitNumberLiteral(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitNumberLiteral) {
			return visitor.visitNumberLiteral(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class IdentifierContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public ReceiveKeyword(): TerminalNode {
		return this.getToken(SolidityParser.ReceiveKeyword, 0);
	}
	public GlobalKeyword(): TerminalNode {
		return this.getToken(SolidityParser.GlobalKeyword, 0);
	}
	public ConstructorKeyword(): TerminalNode {
		return this.getToken(SolidityParser.ConstructorKeyword, 0);
	}
	public PayableKeyword(): TerminalNode {
		return this.getToken(SolidityParser.PayableKeyword, 0);
	}
	public LeaveKeyword(): TerminalNode {
		return this.getToken(SolidityParser.LeaveKeyword, 0);
	}
	public Identifier(): TerminalNode {
		return this.getToken(SolidityParser.Identifier, 0);
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_identifier;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterIdentifier) {
	 		listener.enterIdentifier(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitIdentifier) {
	 		listener.exitIdentifier(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitIdentifier) {
			return visitor.visitIdentifier(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class HexLiteralContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public HexLiteralFragment_list(): TerminalNode[] {
	    	return this.getTokens(SolidityParser.HexLiteralFragment);
	}
	public HexLiteralFragment(i: number): TerminalNode {
		return this.getToken(SolidityParser.HexLiteralFragment, i);
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_hexLiteral;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterHexLiteral) {
	 		listener.enterHexLiteral(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitHexLiteral) {
	 		listener.exitHexLiteral(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitHexLiteral) {
			return visitor.visitHexLiteral(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class OverrideSpecifierContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public userDefinedTypeName_list(): UserDefinedTypeNameContext[] {
		return this.getTypedRuleContexts(UserDefinedTypeNameContext) as UserDefinedTypeNameContext[];
	}
	public userDefinedTypeName(i: number): UserDefinedTypeNameContext {
		return this.getTypedRuleContext(UserDefinedTypeNameContext, i) as UserDefinedTypeNameContext;
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_overrideSpecifier;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterOverrideSpecifier) {
	 		listener.enterOverrideSpecifier(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitOverrideSpecifier) {
	 		listener.exitOverrideSpecifier(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitOverrideSpecifier) {
			return visitor.visitOverrideSpecifier(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}


export class StringLiteralContext extends ParserRuleContext {
	constructor(parser?: SolidityParser, parent?: ParserRuleContext, invokingState?: number) {
		super(parent, invokingState);
    	this.parser = parser;
	}
	public StringLiteralFragment_list(): TerminalNode[] {
	    	return this.getTokens(SolidityParser.StringLiteralFragment);
	}
	public StringLiteralFragment(i: number): TerminalNode {
		return this.getToken(SolidityParser.StringLiteralFragment, i);
	}
    public get ruleIndex(): number {
    	return SolidityParser.RULE_stringLiteral;
	}
	public enterRule(listener: SolidityListener): void {
	    if(listener.enterStringLiteral) {
	 		listener.enterStringLiteral(this);
		}
	}
	public exitRule(listener: SolidityListener): void {
	    if(listener.exitStringLiteral) {
	 		listener.exitStringLiteral(this);
		}
	}
	// @Override
	public accept<Result>(visitor: SolidityVisitor<Result>): Result {
		if (visitor.visitStringLiteral) {
			return visitor.visitStringLiteral(this);
		} else {
			return visitor.visitChildren(this);
		}
	}
}
