"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.Bytecode = exports.Instruction = exports.CustomError = exports.ContractFunction = exports.Contract = exports.SourceLocation = exports.SourceFile = exports.ContractFunctionVisibility = exports.ContractFunctionType = exports.ContractType = exports.JumpType = void 0;
const ethereumjs_util_1 = require("@nomicfoundation/ethereumjs-util");
const abi_helpers_1 = require("../../util/abi-helpers");
/* eslint-disable @nomicfoundation/hardhat-internal-rules/only-hardhat-error */
var JumpType;
(function (JumpType) {
    JumpType[JumpType["NOT_JUMP"] = 0] = "NOT_JUMP";
    JumpType[JumpType["INTO_FUNCTION"] = 1] = "INTO_FUNCTION";
    JumpType[JumpType["OUTOF_FUNCTION"] = 2] = "OUTOF_FUNCTION";
    JumpType[JumpType["INTERNAL_JUMP"] = 3] = "INTERNAL_JUMP";
})(JumpType = exports.JumpType || (exports.JumpType = {}));
var ContractType;
(function (ContractType) {
    ContractType[ContractType["CONTRACT"] = 0] = "CONTRACT";
    ContractType[ContractType["LIBRARY"] = 1] = "LIBRARY";
})(ContractType = exports.ContractType || (exports.ContractType = {}));
var ContractFunctionType;
(function (ContractFunctionType) {
    ContractFunctionType[ContractFunctionType["CONSTRUCTOR"] = 0] = "CONSTRUCTOR";
    ContractFunctionType[ContractFunctionType["FUNCTION"] = 1] = "FUNCTION";
    ContractFunctionType[ContractFunctionType["FALLBACK"] = 2] = "FALLBACK";
    ContractFunctionType[ContractFunctionType["RECEIVE"] = 3] = "RECEIVE";
    ContractFunctionType[ContractFunctionType["GETTER"] = 4] = "GETTER";
    ContractFunctionType[ContractFunctionType["MODIFIER"] = 5] = "MODIFIER";
    ContractFunctionType[ContractFunctionType["FREE_FUNCTION"] = 6] = "FREE_FUNCTION";
})(ContractFunctionType = exports.ContractFunctionType || (exports.ContractFunctionType = {}));
var ContractFunctionVisibility;
(function (ContractFunctionVisibility) {
    ContractFunctionVisibility[ContractFunctionVisibility["PRIVATE"] = 0] = "PRIVATE";
    ContractFunctionVisibility[ContractFunctionVisibility["INTERNAL"] = 1] = "INTERNAL";
    ContractFunctionVisibility[ContractFunctionVisibility["PUBLIC"] = 2] = "PUBLIC";
    ContractFunctionVisibility[ContractFunctionVisibility["EXTERNAL"] = 3] = "EXTERNAL";
})(ContractFunctionVisibility = exports.ContractFunctionVisibility || (exports.ContractFunctionVisibility = {}));
class SourceFile {
    constructor(sourceName, content) {
        this.sourceName = sourceName;
        this.content = content;
        this.contracts = [];
        this.functions = [];
    }
    addContract(contract) {
        if (contract.location.file !== this) {
            throw new Error("Trying to add a contract from another file");
        }
        this.contracts.push(contract);
    }
    addFunction(func) {
        if (func.location.file !== this) {
            throw new Error("Trying to add a function from another file");
        }
        this.functions.push(func);
    }
    getContainingFunction(location) {
        // TODO: Optimize this with a binary search or an internal tree
        for (const func of this.functions) {
            if (func.location.contains(location)) {
                return func;
            }
        }
        return undefined;
    }
}
exports.SourceFile = SourceFile;
class SourceLocation {
    constructor(file, offset, length) {
        this.file = file;
        this.offset = offset;
        this.length = length;
    }
    getStartingLineNumber() {
        if (this._line === undefined) {
            this._line = 1;
            for (const c of this.file.content.slice(0, this.offset)) {
                if (c === "\n") {
                    this._line += 1;
                }
            }
        }
        return this._line;
    }
    getContainingFunction() {
        return this.file.getContainingFunction(this);
    }
    contains(other) {
        if (this.file !== other.file) {
            return false;
        }
        if (other.offset < this.offset) {
            return false;
        }
        return other.offset + other.length <= this.offset + this.length;
    }
    equals(other) {
        return (this.file === other.file &&
            this.offset === other.offset &&
            this.length === other.length);
    }
}
exports.SourceLocation = SourceLocation;
class Contract {
    constructor(name, type, location) {
        this.name = name;
        this.type = type;
        this.location = location;
        this.localFunctions = [];
        this.customErrors = [];
        this._selectorHexToFunction = new Map();
    }
    get constructorFunction() {
        return this._constructor;
    }
    get fallback() {
        return this._fallback;
    }
    get receive() {
        return this._receive;
    }
    addLocalFunction(func) {
        if (func.contract !== this) {
            throw new Error("Function isn't local");
        }
        if (func.visibility === ContractFunctionVisibility.PUBLIC ||
            func.visibility === ContractFunctionVisibility.EXTERNAL) {
            if (func.type === ContractFunctionType.FUNCTION ||
                func.type === ContractFunctionType.GETTER) {
                this._selectorHexToFunction.set((0, ethereumjs_util_1.bytesToHex)(func.selector), func);
            }
            else if (func.type === ContractFunctionType.CONSTRUCTOR) {
                this._constructor = func;
            }
            else if (func.type === ContractFunctionType.FALLBACK) {
                this._fallback = func;
            }
            else if (func.type === ContractFunctionType.RECEIVE) {
                this._receive = func;
            }
        }
        this.localFunctions.push(func);
    }
    addCustomError(customError) {
        this.customErrors.push(customError);
    }
    addNextLinearizedBaseContract(baseContract) {
        if (this._fallback === undefined && baseContract._fallback !== undefined) {
            this._fallback = baseContract._fallback;
        }
        if (this._receive === undefined && baseContract._receive !== undefined) {
            this._receive = baseContract._receive;
        }
        for (const baseContractFunction of baseContract.localFunctions) {
            if (baseContractFunction.type !== ContractFunctionType.GETTER &&
                baseContractFunction.type !== ContractFunctionType.FUNCTION) {
                continue;
            }
            if (baseContractFunction.visibility !== ContractFunctionVisibility.PUBLIC &&
                baseContractFunction.visibility !== ContractFunctionVisibility.EXTERNAL) {
                continue;
            }
            const selectorHex = (0, ethereumjs_util_1.bytesToHex)(baseContractFunction.selector);
            if (!this._selectorHexToFunction.has(selectorHex)) {
                this._selectorHexToFunction.set(selectorHex, baseContractFunction);
            }
        }
    }
    getFunctionFromSelector(selector) {
        return this._selectorHexToFunction.get((0, ethereumjs_util_1.bytesToHex)(selector));
    }
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
    correctSelector(functionName, selector) {
        const functions = Array.from(this._selectorHexToFunction.values()).filter((cf) => cf.name === functionName);
        if (functions.length !== 1) {
            return false;
        }
        const functionToCorrect = functions[0];
        if (functionToCorrect.selector !== undefined) {
            this._selectorHexToFunction.delete((0, ethereumjs_util_1.bytesToHex)(functionToCorrect.selector));
        }
        functionToCorrect.selector = selector;
        this._selectorHexToFunction.set((0, ethereumjs_util_1.bytesToHex)(selector), functionToCorrect);
        return true;
    }
}
exports.Contract = Contract;
class ContractFunction {
    constructor(name, type, location, contract, visibility, isPayable, selector, paramTypes) {
        this.name = name;
        this.type = type;
        this.location = location;
        this.contract = contract;
        this.visibility = visibility;
        this.isPayable = isPayable;
        this.selector = selector;
        this.paramTypes = paramTypes;
        if (contract !== undefined && !contract.location.contains(location)) {
            throw new Error("Incompatible contract and function location");
        }
    }
    isValidCalldata(calldata) {
        if (this.paramTypes === undefined) {
            // if we don't know the param types, we just assume that the call is valid
            return true;
        }
        return abi_helpers_1.AbiHelpers.isValidCalldata(this.paramTypes, calldata);
    }
}
exports.ContractFunction = ContractFunction;
class CustomError {
    /**
     * Return a CustomError from the given ABI information: the name
     * of the error and its inputs. Returns undefined if it can't build
     * the CustomError.
     */
    static fromABI(name, inputs) {
        const selector = abi_helpers_1.AbiHelpers.computeSelector(name, inputs);
        if (selector !== undefined) {
            return new CustomError(selector, name, inputs);
        }
    }
    constructor(selector, name, paramTypes) {
        this.selector = selector;
        this.name = name;
        this.paramTypes = paramTypes;
    }
}
exports.CustomError = CustomError;
class Instruction {
    constructor(pc, opcode, jumpType, pushData, location) {
        this.pc = pc;
        this.opcode = opcode;
        this.jumpType = jumpType;
        this.pushData = pushData;
        this.location = location;
    }
    /**
     * Checks equality with another Instruction.
     */
    equals(other) {
        if (this.pc !== other.pc) {
            return false;
        }
        if (this.opcode !== other.opcode) {
            return false;
        }
        if (this.jumpType !== other.jumpType) {
            return false;
        }
        if (this.pushData !== undefined) {
            if (other.pushData === undefined) {
                return false;
            }
            if (!this.pushData.equals(other.pushData)) {
                return false;
            }
        }
        else if (other.pushData !== undefined) {
            return false;
        }
        if (this.location !== undefined) {
            if (other.location === undefined) {
                return false;
            }
            if (!this.location.equals(other.location)) {
                return false;
            }
        }
        else if (other.location !== undefined) {
            return false;
        }
        return true;
    }
}
exports.Instruction = Instruction;
class Bytecode {
    constructor(contract, isDeployment, normalizedCode, instructions, libraryAddressPositions, immutableReferences, compilerVersion) {
        this.contract = contract;
        this.isDeployment = isDeployment;
        this.normalizedCode = normalizedCode;
        this.instructions = instructions;
        this.libraryAddressPositions = libraryAddressPositions;
        this.immutableReferences = immutableReferences;
        this.compilerVersion = compilerVersion;
        this._pcToInstruction = new Map();
        for (const inst of instructions) {
            this._pcToInstruction.set(inst.pc, inst);
        }
    }
    getInstruction(pc) {
        const inst = this._pcToInstruction.get(pc);
        if (inst === undefined) {
            throw new Error(`There's no instruction at pc ${pc}`);
        }
        return inst;
    }
    hasInstruction(pc) {
        return this._pcToInstruction.has(pc);
    }
    /**
     * Checks equality with another Bytecode.
     */
    equals(other) {
        if (this._pcToInstruction.size !== other._pcToInstruction.size) {
            return false;
        }
        for (const [key, val] of this._pcToInstruction) {
            const otherVal = other._pcToInstruction.get(key);
            if (otherVal === undefined || !val.equals(otherVal)) {
                return false;
            }
        }
        return true;
    }
}
exports.Bytecode = Bytecode;
//# sourceMappingURL=model.js.map