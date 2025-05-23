/// <reference types="chai" />
import type { Ssfi } from "../utils";
export declare const EMIT_CALLED = "emitAssertionCalled";
export declare function supportEmit(Assertion: Chai.AssertionStatic, chaiUtils: Chai.ChaiUtils): void;
export declare function emitWithArgs(context: any, Assertion: Chai.AssertionStatic, chaiUtils: Chai.ChaiUtils, expectedArgs: any[], ssfi: Ssfi): Promise<void>;
//# sourceMappingURL=emit.d.ts.map