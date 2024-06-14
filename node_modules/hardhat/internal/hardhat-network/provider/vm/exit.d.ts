import type { ExceptionalHalt, SuccessReason } from "@nomicfoundation/edr";
export declare enum ExitCode {
    SUCCESS = 0,
    REVERT = 1,
    OUT_OF_GAS = 2,
    INTERNAL_ERROR = 3,
    INVALID_OPCODE = 4,
    STACK_UNDERFLOW = 5,
    CODESIZE_EXCEEDS_MAXIMUM = 6,
    CREATE_COLLISION = 7,
    STATIC_STATE_CHANGE = 8
}
export declare class Exit {
    kind: ExitCode;
    static fromEdrSuccessReason(reason: SuccessReason): Exit;
    static fromEdrExceptionalHalt(halt: ExceptionalHalt): Exit;
    constructor(kind: ExitCode);
    isError(): boolean;
    getReason(): string;
    getEdrExceptionalHalt(): ExceptionalHalt;
}
//# sourceMappingURL=exit.d.ts.map