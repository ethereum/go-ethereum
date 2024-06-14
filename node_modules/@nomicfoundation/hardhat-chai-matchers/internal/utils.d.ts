/// <reference types="chai" />
import { AssertWithSsfi, Ssfi } from "../utils";
export declare function assertIsNotNull<T>(value: T, valueName: string): asserts value is Exclude<T, null>;
export declare function preventAsyncMatcherChaining(context: object, matcherName: string, chaiUtils: Chai.ChaiUtils, allowSelfChaining?: boolean): void;
export declare function assertArgsArraysEqual(Assertion: Chai.AssertionStatic, expectedArgs: any[], actualArgs: any[], tag: string, assertionType: "event" | "error", assert: AssertWithSsfi, ssfi: Ssfi): void;
//# sourceMappingURL=utils.d.ts.map