/// <reference types="node" />
import { Opcode } from "./opcodes";
export declare enum JumpType {
    NOT_JUMP = 0,
    INTO_FUNCTION = 1,
    OUTOF_FUNCTION = 2,
    INTERNAL_JUMP = 3
}
export declare enum ContractType {
    CONTRACT = 0,
    LIBRARY = 1
}
export declare enum ContractFunctionType {
    CONSTRUCTOR = 0,
    FUNCTION = 1,
    FALLBACK = 2,
    RECEIVE = 3,
    GETTER = 4,
    MODIFIER = 5,
    FREE_FUNCTION = 6
}
export declare enum ContractFunctionVisibility {
    PRIVATE = 0,
    INTERNAL = 1,
    PUBLIC = 2,
    EXTERNAL = 3
}
export declare class SourceFile {
    readonly sourceName: string;
    readonly content: string;
    readonly contracts: Contract[];
    readonly functions: ContractFunction[];
    constructor(sourceName: string, content: string);
    addContract(contract: Contract): void;
    addFunction(func: ContractFunction): void;
    getContainingFunction(location: SourceLocation): ContractFunction | undefined;
}
export declare class SourceLocation {
    readonly file: SourceFile;
    readonly offset: number;
    readonly length: number;
    private _line;
    constructor(file: SourceFile, offset: number, length: number);
    getStartingLineNumber(): number;
    getContainingFunction(): ContractFunction | undefined;
    contains(other: SourceLocation): boolean;
    equals(other: SourceLocation): boolean;
}
export declare class Contract {
    readonly name: string;
    readonly type: ContractType;
    readonly location: SourceLocation;
    readonly localFunctions: ContractFunction[];
    readonly customErrors: CustomError[];
    private _constructor;
    private _fallback;
    private _receive;
    private readonly _selectorHexToFunction;
    constructor(name: string, type: ContractType, location: SourceLocation);
    get constructorFunction(): ContractFunction | undefined;
    get fallback(): ContractFunction | undefined;
    get receive(): ContractFunction | undefined;
    addLocalFunction(func: ContractFunction): void;
    addCustomError(customError: CustomError): void;
    addNextLinearizedBaseContract(baseContract: Contract): void;
    getFunctionFromSelector(selector: Uint8Array): ContractFunction | undefined;
    /**
     * We compute selectors manually, which is particularly hard. We do this
     * because we need to map selectors to AST nodes, and it seems easier to start
     * from the AST node. This is surprisingly super hard: things like inherited
     * enums, structs and ABIv2 complicate it.
     *
     * As we know that that can fail, we run a heuristic that tries to correct
     * incorrect selectors. What it does is checking the `evm.methodIdentifiers`
     * compiler output, and detect missing selectors. Then we take those and
     * find contract functions with the same name. If there are multiple of those
     * we can't do anything. If there is a single one, it must have an incorrect
     * selector, so we update it with the `evm.methodIdentifiers`'s value.
     */
    correctSelector(functionName: string, selector: Buffer): boolean;
}
export declare class ContractFunction {
    readonly name: string;
    readonly type: ContractFunctionType;
    readonly location: SourceLocation;
    readonly contract?: Contract | undefined;
    readonly visibility?: ContractFunctionVisibility | undefined;
    readonly isPayable?: boolean | undefined;
    selector?: Uint8Array | undefined;
    readonly paramTypes?: any[] | undefined;
    constructor(name: string, type: ContractFunctionType, location: SourceLocation, contract?: Contract | undefined, visibility?: ContractFunctionVisibility | undefined, isPayable?: boolean | undefined, selector?: Uint8Array | undefined, paramTypes?: any[] | undefined);
    isValidCalldata(calldata: Uint8Array): boolean;
}
export declare class CustomError {
    readonly selector: Uint8Array;
    readonly name: string;
    readonly paramTypes: any[];
    /**
     * Return a CustomError from the given ABI information: the name
     * of the error and its inputs. Returns undefined if it can't build
     * the CustomError.
     */
    static fromABI(name: string, inputs: any[]): CustomError | undefined;
    private constructor();
}
export declare class Instruction {
    readonly pc: number;
    readonly opcode: Opcode;
    readonly jumpType: JumpType;
    readonly pushData?: Buffer | undefined;
    readonly location?: SourceLocation | undefined;
    constructor(pc: number, opcode: Opcode, jumpType: JumpType, pushData?: Buffer | undefined, location?: SourceLocation | undefined);
    /**
     * Checks equality with another Instruction.
     */
    equals(other: Instruction): boolean;
}
interface ImmutableReference {
    start: number;
    length: number;
}
export declare class Bytecode {
    readonly contract: Contract;
    readonly isDeployment: boolean;
    readonly normalizedCode: Buffer;
    readonly instructions: Instruction[];
    readonly libraryAddressPositions: number[];
    readonly immutableReferences: ImmutableReference[];
    readonly compilerVersion: string;
    private readonly _pcToInstruction;
    constructor(contract: Contract, isDeployment: boolean, normalizedCode: Buffer, instructions: Instruction[], libraryAddressPositions: number[], immutableReferences: ImmutableReference[], compilerVersion: string);
    getInstruction(pc: number): Instruction;
    hasInstruction(pc: number): boolean;
    /**
     * Checks equality with another Bytecode.
     */
    equals(other: Bytecode): boolean;
}
export {};
//# sourceMappingURL=model.d.ts.map