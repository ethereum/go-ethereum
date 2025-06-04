### 0.5.0 (Unreleased)

 * Remove `ParameterList` and `Parameter` node types. Parameters are now always
   of type `VariableDeclaration` and lists of parameters are represented as
   lists of nodes of type `VariableDeclaration`. This is a breaking change.

### 0.4.12 (Unreleased)

 * Fix type name expressions to also support user-defined type names.

### 0.4.11

 * Bugfix release

### 0.4.9

 * Fix parsing of inheritance specifier with no arguments.

### 0.4.8

 * Fix parsing of string literals with escaped characters.

### 0.4.7

 * Fix parsing of underscores in number literals.

### 0.4.6

 * Add support for the `type` keyword.
 * Add support for underscores in number literals.

### 0.4.5

 * Improve TypeScript type definitions.

### 0.4.4

 * Add missing `storageLocation` to variables in VariableDeclarationStatement.
 * Return `null` for `arguments` instead of `[]` when `ModifierInvocation`
   contains no arguments and no parentheses to distinguish the two cases.
 * Improve TypeScript type definitions.

### 0.4.3

 * Improve TypeScript type definitions, thanks @Leeleo3x and @yxliang01.

### 0.4.2

 * Fix parsing of assembly function definitions with no args or return args.

### 0.4.1

 * Fix parsing of for loops with missing initial and condition statements.

### 0.4.0

 * Correctly handle non-existent tuple components. Thanks @maxsam4
 * Accept calldata as identifier

### 0.3.3

 * Add support for `address payable` typename.

### 0.3.2

 * Fix parsing of hex numbers with uppercase X.

### 0.3.1

 * Fix parsing of zero-component tuples.

### 0.3.0

 * Use `components` for all `TupleExpression` nodes. Earlier versions
   incorrectly stored tuple components under the `elements` key.
 * Fix parsing of decimal literals without integer part.
