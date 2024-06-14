/**
 * Creates an identifier prefixing reserved words with `_`.
 * We can only use this for function parameters and tuple element names.
 * Using it for method names would clas with runtime codegen.
 *
 * @internal
 */
export declare function createPositionalIdentifier(identifierName: string): string;
/**
 * @internal
 */
export declare function getUsedIdentifiers(identifiers: string[], sourceFile: string): string[];
/**
 * @internal
 */
export declare function createImportTypeDeclaration(identifiers: string[], moduleSpecifier: string): string;
type ModuleSpecifier = string;
type Identifier = string;
/**
 * @internal
 */
export declare function createImportsForUsedIdentifiers(possibleImports: Record<ModuleSpecifier, Identifier[]>, sourceFile: string): string;
export {};
