// Base on the original type definitions for solidity-parser-antlr 0.2
// by Leonid Logvinov <https://github.com/LogvinovLeon>
//    Alex Browne <https://github.com/albrow>
//    Xiao Liang <https://github.com/yxliang01>

export interface Position {
  line: number
  column: number
}

export interface Location {
  start: Position
  end: Position
}

export interface Comment {
  type: 'BlockComment' | 'LineComment'
  value: string
  range?: [number, number]
  loc?: Location
}

export interface BaseASTNode {
  type: ASTNodeTypeString
  comments?: Comment[]
  range?: [number, number]
  loc?: Location
}

export interface SourceUnit extends BaseASTNode {
  type: 'SourceUnit'
  children: ASTNode[]
}

export interface ContractDefinition extends BaseASTNode {
  type: 'ContractDefinition'
  name: string
  baseContracts: InheritanceSpecifier[]
  kind: string
  subNodes: BaseASTNode[]
  storageLayout?: Expression
}

export interface InheritanceSpecifier extends BaseASTNode {
  type: 'InheritanceSpecifier'
  baseName: UserDefinedTypeName
  arguments: Expression[]
}

export interface UserDefinedTypeName extends BaseASTNode {
  type: 'UserDefinedTypeName'
  namePath: string
}

export const astNodeTypes = [
  'SourceUnit',
  'PragmaDirective',
  'ImportDirective',
  'ContractDefinition',
  'InheritanceSpecifier',
  'StateVariableDeclaration',
  'UsingForDeclaration',
  'StructDefinition',
  'ModifierDefinition',
  'ModifierInvocation',
  'FunctionDefinition',
  'EventDefinition',
  'CustomErrorDefinition',
  'RevertStatement',
  'EnumValue',
  'EnumDefinition',
  'VariableDeclaration',
  'UserDefinedTypeName',
  'Mapping',
  'ArrayTypeName',
  'FunctionTypeName',
  'Block',
  'ExpressionStatement',
  'IfStatement',
  'WhileStatement',
  'ForStatement',
  'InlineAssemblyStatement',
  'DoWhileStatement',
  'ContinueStatement',
  'Break',
  'Continue',
  'BreakStatement',
  'ReturnStatement',
  'EmitStatement',
  'ThrowStatement',
  'VariableDeclarationStatement',
  'ElementaryTypeName',
  'FunctionCall',
  'AssemblyBlock',
  'AssemblyCall',
  'AssemblyLocalDefinition',
  'AssemblyAssignment',
  'AssemblyStackAssignment',
  'LabelDefinition',
  'AssemblySwitch',
  'AssemblyCase',
  'AssemblyFunctionDefinition',
  'AssemblyFor',
  'AssemblyIf',
  'TupleExpression',
  'NameValueExpression',
  'BooleanLiteral',
  'NumberLiteral',
  'Identifier',
  'BinaryOperation',
  'UnaryOperation',
  'NewExpression',
  'Conditional',
  'StringLiteral',
  'HexLiteral',
  'HexNumber',
  'DecimalNumber',
  'MemberAccess',
  'IndexAccess',
  'IndexRangeAccess',
  'NameValueList',
  'UncheckedStatement',
  'TryStatement',
  'CatchClause',
  'FileLevelConstant',
  'AssemblyMemberAccess',
  'TypeDefinition',
] as const

export type ASTNodeTypeString = (typeof astNodeTypes)[number]

export interface PragmaDirective extends BaseASTNode {
  type: 'PragmaDirective'
  name: string
  value: string
}
export interface ImportDirective extends BaseASTNode {
  type: 'ImportDirective'
  path: string
  pathLiteral: StringLiteral
  unitAlias: string | null
  unitAliasIdentifier: Identifier | null
  symbolAliases: Array<[string, string | null]> | null
  symbolAliasesIdentifiers: Array<[Identifier, Identifier | null]> | null
}
export interface StateVariableDeclaration extends BaseASTNode {
  type: 'StateVariableDeclaration'
  variables: StateVariableDeclarationVariable[]
  initialValue: Expression | null
}
export interface FileLevelConstant extends BaseASTNode {
  type: 'FileLevelConstant'
  typeName: TypeName
  name: string
  initialValue: Expression
  isDeclaredConst: boolean
  isImmutable: boolean
}
export interface UsingForDeclaration extends BaseASTNode {
  type: 'UsingForDeclaration'
  typeName: TypeName | null
  functions: string[]
  // for each item in `functions`, the item with the same index in `operators`
  // will be the defined operator, or null if it's just an attached function
  operators: Array<string | null>
  libraryName: string | null
  isGlobal: boolean
}
export interface StructDefinition extends BaseASTNode {
  type: 'StructDefinition'
  name: string
  members: VariableDeclaration[]
}
export interface ModifierDefinition extends BaseASTNode {
  type: 'ModifierDefinition'
  name: string
  parameters: null | VariableDeclaration[]
  isVirtual: boolean
  override: null | UserDefinedTypeName[]
  body: Block | null
}
export interface ModifierInvocation extends BaseASTNode {
  type: 'ModifierInvocation'
  name: string
  arguments: Expression[] | null
}
export interface FunctionDefinition extends BaseASTNode {
  type: 'FunctionDefinition'
  name: string | null
  parameters: VariableDeclaration[]
  modifiers: ModifierInvocation[]
  stateMutability: 'pure' | 'constant' | 'payable' | 'view' | null
  visibility: 'default' | 'external' | 'internal' | 'public' | 'private'
  returnParameters: VariableDeclaration[] | null
  body: Block | null
  override: UserDefinedTypeName[] | null
  isConstructor: boolean
  isReceiveEther: boolean
  isFallback: boolean
  isVirtual: boolean
}

export interface CustomErrorDefinition extends BaseASTNode {
  type: 'CustomErrorDefinition'
  name: string
  parameters: VariableDeclaration[]
}

export interface TypeDefinition extends BaseASTNode {
  type: 'TypeDefinition'
  name: string
  definition: ElementaryTypeName
}

export interface RevertStatement extends BaseASTNode {
  type: 'RevertStatement'
  revertCall: FunctionCall
}
export interface EventDefinition extends BaseASTNode {
  type: 'EventDefinition'
  name: string
  parameters: VariableDeclaration[]
  isAnonymous: boolean
}
export interface EnumValue extends BaseASTNode {
  type: 'EnumValue'
  name: string
}
export interface EnumDefinition extends BaseASTNode {
  type: 'EnumDefinition'
  name: string
  members: EnumValue[]
}
export interface VariableDeclaration extends BaseASTNode {
  type: 'VariableDeclaration'
  isIndexed: boolean
  isStateVar: boolean
  typeName: TypeName | null
  name: string | null
  identifier: Identifier | null
  isDeclaredConst?: boolean
  storageLocation: string | null
  expression: Expression | null
  visibility?: 'public' | 'private' | 'internal' | 'default'
}
export interface StateVariableDeclarationVariable extends VariableDeclaration {
  override: null | UserDefinedTypeName[]
  isImmutable: boolean
  isTransient: boolean
}
export interface ArrayTypeName extends BaseASTNode {
  type: 'ArrayTypeName'
  baseTypeName: TypeName
  length: Expression | null
}
export interface Mapping extends BaseASTNode {
  type: 'Mapping'
  keyType: ElementaryTypeName | UserDefinedTypeName
  keyName: Identifier | null
  valueType: TypeName
  valueName: Identifier | null
}
export interface FunctionTypeName extends BaseASTNode {
  type: 'FunctionTypeName'
  parameterTypes: VariableDeclaration[]
  returnTypes: VariableDeclaration[]
  visibility: string
  stateMutability: string | null
}

export interface Block extends BaseASTNode {
  type: 'Block'
  statements: BaseASTNode[]
}
export interface ExpressionStatement extends BaseASTNode {
  type: 'ExpressionStatement'
  expression: Expression | null
}
export interface IfStatement extends BaseASTNode {
  type: 'IfStatement'
  condition: Expression
  trueBody: Statement
  falseBody: Statement | null
}
export interface UncheckedStatement extends BaseASTNode {
  type: 'UncheckedStatement'
  block: Block
}
export interface TryStatement extends BaseASTNode {
  type: 'TryStatement'
  expression: Expression
  returnParameters: VariableDeclaration[] | null
  body: Block
  catchClauses: CatchClause[]
}
export interface CatchClause extends BaseASTNode {
  type: 'CatchClause'
  isReasonStringType: boolean
  kind: string | null
  parameters: VariableDeclaration[] | null
  body: Block
}
export interface WhileStatement extends BaseASTNode {
  type: 'WhileStatement'
  condition: Expression
  body: Statement
}
export interface ForStatement extends BaseASTNode {
  type: 'ForStatement'
  initExpression: SimpleStatement | null
  conditionExpression?: Expression
  loopExpression: ExpressionStatement
  body: Statement
}
export interface InlineAssemblyStatement extends BaseASTNode {
  type: 'InlineAssemblyStatement'
  language: string | null
  flags: string[]
  body: AssemblyBlock
}
export interface DoWhileStatement extends BaseASTNode {
  type: 'DoWhileStatement'
  condition: Expression
  body: Statement
}
export interface ContinueStatement extends BaseASTNode {
  type: 'ContinueStatement'
}
export interface Break extends BaseASTNode {
  type: 'Break'
}
export interface Continue extends BaseASTNode {
  type: 'Continue'
}
export interface BreakStatement extends BaseASTNode {
  type: 'BreakStatement'
}
export interface ReturnStatement extends BaseASTNode {
  type: 'ReturnStatement'
  expression: Expression | null
}
export interface EmitStatement extends BaseASTNode {
  type: 'EmitStatement'
  eventCall: FunctionCall
}
export interface ThrowStatement extends BaseASTNode {
  type: 'ThrowStatement'
}
export interface VariableDeclarationStatement extends BaseASTNode {
  type: 'VariableDeclarationStatement'
  variables: Array<BaseASTNode | null>
  initialValue: Expression | null
}
export interface ElementaryTypeName extends BaseASTNode {
  type: 'ElementaryTypeName'
  name: string
  stateMutability: string | null
}
export interface FunctionCall extends BaseASTNode {
  type: 'FunctionCall'
  expression: Expression
  arguments: Expression[]
  names: string[]
  identifiers: Identifier[]
}
export interface AssemblyBlock extends BaseASTNode {
  type: 'AssemblyBlock'
  operations: AssemblyItem[]
}
export interface AssemblyCall extends BaseASTNode {
  type: 'AssemblyCall'
  functionName: string
  arguments: AssemblyExpression[]
}
export interface AssemblyLocalDefinition extends BaseASTNode {
  type: 'AssemblyLocalDefinition'
  names: Identifier[] | AssemblyMemberAccess[]
  expression: AssemblyExpression | null
}
export interface AssemblyAssignment extends BaseASTNode {
  type: 'AssemblyAssignment'
  names: Identifier[] | AssemblyMemberAccess[]
  expression: AssemblyExpression
}
export interface AssemblyStackAssignment extends BaseASTNode {
  type: 'AssemblyStackAssignment'
  name: string
  expression: AssemblyExpression
}
export interface LabelDefinition extends BaseASTNode {
  type: 'LabelDefinition'
  name: string
}
export interface AssemblySwitch extends BaseASTNode {
  type: 'AssemblySwitch'
  expression: AssemblyExpression
  cases: AssemblyCase[]
}
export interface AssemblyCase extends BaseASTNode {
  type: 'AssemblyCase'
  value: AssemblyLiteral | null
  block: AssemblyBlock
  default: boolean
}
export interface AssemblyFunctionDefinition extends BaseASTNode {
  type: 'AssemblyFunctionDefinition'
  name: string
  arguments: Identifier[]
  returnArguments: Identifier[]
  body: AssemblyBlock
}
export interface AssemblyFor extends BaseASTNode {
  type: 'AssemblyFor'
  pre: AssemblyBlock | AssemblyExpression
  condition: AssemblyExpression
  post: AssemblyBlock | AssemblyExpression
  body: AssemblyBlock
}
export interface AssemblyIf extends BaseASTNode {
  type: 'AssemblyIf'
  condition: AssemblyExpression
  body: AssemblyBlock
}
export type AssemblyLiteral =
  | StringLiteral
  | BooleanLiteral
  | DecimalNumber
  | HexNumber
  | HexLiteral
export interface AssemblyMemberAccess extends BaseASTNode {
  type: 'AssemblyMemberAccess'
  expression: Identifier
  memberName: Identifier
}
export interface NewExpression extends BaseASTNode {
  type: 'NewExpression'
  typeName: TypeName
}
export interface TupleExpression extends BaseASTNode {
  type: 'TupleExpression'
  components: Array<BaseASTNode | null>
  isArray: boolean
}
export interface NameValueExpression extends BaseASTNode {
  type: 'NameValueExpression'
  expression: Expression
  arguments: NameValueList
}
export interface NumberLiteral extends BaseASTNode {
  type: 'NumberLiteral'
  number: string
  subdenomination:
    | null
    | 'wei'
    | 'szabo'
    | 'finney'
    | 'ether'
    | 'seconds'
    | 'minutes'
    | 'hours'
    | 'days'
    | 'weeks'
    | 'years'
}
export interface BooleanLiteral extends BaseASTNode {
  type: 'BooleanLiteral'
  value: boolean
}
export interface HexLiteral extends BaseASTNode {
  type: 'HexLiteral'
  value: string
  parts: string[]
}
export interface StringLiteral extends BaseASTNode {
  type: 'StringLiteral'
  value: string
  parts: string[]
  isUnicode: boolean[]
}
export interface Identifier extends BaseASTNode {
  type: 'Identifier'
  name: string
}

export const binaryOpValues = [
  '+',
  '-',
  '*',
  '/',
  '**',
  '%',
  '<<',
  '>>',
  '&&',
  '||',
  '&',
  '^',
  '<',
  '>',
  '<=',
  '>=',
  '==',
  '!=',
  '=',
  '^=',
  '&=',
  '<<=',
  '>>=',
  '+=',
  '-=',
  '*=',
  '/=',
  '%=',
  '|',
  '|=',
] as const
export type BinOp = (typeof binaryOpValues)[number]

export const unaryOpValues = [
  '-',
  '+',
  '++',
  '--',
  '~',
  'after',
  'delete',
  '!',
] as const
export type UnaryOp = (typeof unaryOpValues)[number]

export interface BinaryOperation extends BaseASTNode {
  type: 'BinaryOperation'
  left: Expression
  right: Expression
  operator: BinOp
}
export interface UnaryOperation extends BaseASTNode {
  type: 'UnaryOperation'
  operator: UnaryOp
  subExpression: Expression
  isPrefix: boolean
}
export interface Conditional extends BaseASTNode {
  type: 'Conditional'
  condition: Expression
  trueExpression: Expression
  falseExpression: Expression
}
export interface IndexAccess extends BaseASTNode {
  type: 'IndexAccess'
  base: Expression
  index: Expression
}
export interface IndexRangeAccess extends BaseASTNode {
  type: 'IndexRangeAccess'
  base: Expression
  indexStart?: Expression
  indexEnd?: Expression
}
export interface MemberAccess extends BaseASTNode {
  type: 'MemberAccess'
  expression: Expression
  memberName: string
}
export interface HexNumber extends BaseASTNode {
  type: 'HexNumber'
  value: string
}
export interface DecimalNumber extends BaseASTNode {
  type: 'DecimalNumber'
  value: string
}
export interface NameValueList extends BaseASTNode {
  type: 'NameValueList'
  names: string[]
  identifiers: Identifier[]
  arguments: Expression[]
}
export type ASTNode =
  | SourceUnit
  | PragmaDirective
  | ImportDirective
  | ContractDefinition
  | InheritanceSpecifier
  | StateVariableDeclaration
  | UsingForDeclaration
  | StructDefinition
  | ModifierDefinition
  | ModifierInvocation
  | FunctionDefinition
  | EventDefinition
  | CustomErrorDefinition
  | EnumValue
  | EnumDefinition
  | VariableDeclaration
  | TypeName
  | UserDefinedTypeName
  | Mapping
  | FunctionTypeName
  | Block
  | Statement
  | ElementaryTypeName
  | AssemblyBlock
  | AssemblyCall
  | AssemblyLocalDefinition
  | AssemblyAssignment
  | AssemblyStackAssignment
  | LabelDefinition
  | AssemblySwitch
  | AssemblyCase
  | AssemblyFunctionDefinition
  | AssemblyFor
  | AssemblyIf
  | AssemblyLiteral
  | TupleExpression
  | BinaryOperation
  | Conditional
  | IndexAccess
  | IndexRangeAccess
  | AssemblyItem
  | Expression
  | NameValueList
  | AssemblyMemberAccess
  | CatchClause
  | FileLevelConstant
  | TypeDefinition

export type AssemblyItem =
  | Identifier
  | AssemblyBlock
  | AssemblyExpression
  | AssemblyLocalDefinition
  | AssemblyAssignment
  | AssemblyStackAssignment
  | LabelDefinition
  | AssemblySwitch
  | AssemblyFunctionDefinition
  | AssemblyFor
  | AssemblyIf
  | Break
  | Continue
  | NumberLiteral
  | StringLiteral
  | HexNumber
  | HexLiteral
  | DecimalNumber
export type AssemblyExpression = AssemblyCall | AssemblyLiteral
export type Expression =
  | IndexAccess
  | IndexRangeAccess
  | TupleExpression
  | BinaryOperation
  | Conditional
  | MemberAccess
  | FunctionCall
  | UnaryOperation
  | NewExpression
  | PrimaryExpression
  | NameValueExpression
export type PrimaryExpression =
  | BooleanLiteral
  | HexLiteral
  | StringLiteral
  | NumberLiteral
  | Identifier
  | TupleExpression
  | TypeName
export type SimpleStatement = VariableDeclarationStatement | ExpressionStatement
export type TypeName =
  | ElementaryTypeName
  | UserDefinedTypeName
  | Mapping
  | ArrayTypeName
  | FunctionTypeName
export type Statement =
  | IfStatement
  | WhileStatement
  | ForStatement
  | Block
  | InlineAssemblyStatement
  | DoWhileStatement
  | ContinueStatement
  | BreakStatement
  | ReturnStatement
  | EmitStatement
  | ThrowStatement
  | SimpleStatement
  | VariableDeclarationStatement
  | UncheckedStatement
  | TryStatement
  | RevertStatement

type ASTMap<U> = { [K in ASTNodeTypeString]: U extends { type: K } ? U : never }
type ASTTypeMap = ASTMap<ASTNode>
type ASTVisitorEnter = {
  [K in keyof ASTTypeMap]?: (ast: ASTTypeMap[K], parent?: ASTNode) => any
}
type ASTVisitorExit = {
  [K in keyof ASTTypeMap as `${K}:exit`]?: (
    ast: ASTTypeMap[K],
    parent?: ASTNode
  ) => any
}

export type ASTVisitor = ASTVisitorEnter & ASTVisitorExit

/* eslint-disable @typescript-eslint/no-unused-vars */
/**
 * This monstrosity is here to check that there are no ASTNodeTypeString without
 * a corresponding ASTNode, no ASTNode without a corresponding ASTNodeTypeString,
 * no ASTVisitorEnter callback without a corresponding ASTNode,
 * no ASTVisitorExit callback without a corresponding ASTVisitorEnter callback,
 * and so on, and so on.
 *
 * There are probably some ways to simplify this by deriving some types
 * from others.
 */
function checkTypes() {
  const astNodeType: ASTNode['type'] = '' as any
  const astNodeTypeString: ASTNodeTypeString = '' as any
  const astVisitorEnterKey: keyof ASTVisitorEnter = '' as any

  let assignAstNodeType: ASTNode['type'] = astNodeTypeString
  assignAstNodeType = astVisitorEnterKey

  let assignAstNodeTyeString: ASTNodeTypeString = astNodeType
  assignAstNodeTyeString = astVisitorEnterKey

  let assignAstVisitorEnterKey: keyof ASTVisitorEnter = astNodeType
  assignAstVisitorEnterKey = astNodeTypeString

  const astNodeTypeExit: `${ASTNode['type']}:exit` = '' as any
  const astNodeTypeStringExit: `${ASTNodeTypeString}:exit` = '' as any
  const astVisitorEnterKeyExit: `${keyof ASTVisitorEnter}:exit` = '' as any
  const astVisitorExitKey: keyof ASTVisitorExit = '' as any

  let letAstNodeTypeExit: `${ASTNode['type']}:exit` = astNodeTypeStringExit
  letAstNodeTypeExit = astVisitorEnterKeyExit
  letAstNodeTypeExit = astVisitorExitKey

  let assignAstNodeTypeStringExit: `${ASTNodeTypeString}:exit` = astNodeTypeExit
  assignAstNodeTypeStringExit = astVisitorEnterKeyExit
  assignAstNodeTypeStringExit = astVisitorExitKey

  let assignAstVisitorEnterKeyExit: `${keyof ASTVisitorEnter}:exit` =
    astNodeTypeExit
  assignAstVisitorEnterKeyExit = astNodeTypeStringExit
  assignAstVisitorEnterKeyExit = astVisitorExitKey

  let assignAstVisitorExitKey: keyof ASTVisitorExit = astNodeTypeExit
  assignAstVisitorExitKey = astNodeTypeStringExit
  assignAstVisitorExitKey = astVisitorEnterKeyExit
}
/* eslint-enable @typescript-eslint/no-unused-vars */
