import { IgnitionError } from "../../errors";
import { AccountRuntimeValue, ArgumentType, RuntimeValue } from "../../types/module";
export declare function validateAccountRuntimeValue(arv: AccountRuntimeValue, accounts: string[]): IgnitionError[];
export declare function filterToAccountRuntimeValues(runtimeValues: RuntimeValue[]): AccountRuntimeValue[];
export declare function retrieveNestedRuntimeValues(args: ArgumentType[]): RuntimeValue[];
//# sourceMappingURL=utils.d.ts.map