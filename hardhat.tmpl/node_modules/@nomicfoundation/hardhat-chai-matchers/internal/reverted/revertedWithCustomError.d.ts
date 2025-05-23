/// <reference types="chai" />
import { Ssfi } from "../../utils";
export declare const REVERTED_WITH_CUSTOM_ERROR_CALLED = "customErrorAssertionCalled";
export declare function supportRevertedWithCustomError(Assertion: Chai.AssertionStatic, chaiUtils: Chai.ChaiUtils): void;
export declare function revertedWithCustomErrorWithArgs(context: any, Assertion: Chai.AssertionStatic, chaiUtils: Chai.ChaiUtils, expectedArgs: any[], ssfi: Ssfi): Promise<void>;
//# sourceMappingURL=revertedWithCustomError.d.ts.map