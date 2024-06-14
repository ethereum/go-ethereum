/**
 * This file includes Solidity tracing heuristics for solc starting with version
 * 0.6.9.
 *
 * This solc version introduced a significant change to how sourcemaps are
 * handled for inline yul/internal functions. These were mapped to the
 * unmapped/-1 file before, which lead to many unmapped reverts. Now, they are
 * mapped to the part of the Solidity source that lead to their inlining.
 *
 * This change is a very positive change, as errors would point to the correct
 * line by default. The only problem is that we used to rely very heavily on
 * unmapped reverts to decide when our error detection heuristics were to be
 * run. In fact, this heuristics were first introduced because of unmapped
 * reverts.
 *
 * Instead of synthetically completing stack traces when unmapped reverts occur,
 * we now start from complete stack traces and adjust them if we can provide
 * more meaningful errors.
 */
import { DecodedEvmMessageTrace } from "./message-trace";
import { SolidityStackTrace } from "./solidity-stack-trace";
export declare function stackTraceMayRequireAdjustments(stackTrace: SolidityStackTrace, decodedTrace: DecodedEvmMessageTrace): boolean;
export declare function adjustStackTrace(stackTrace: SolidityStackTrace, decodedTrace: DecodedEvmMessageTrace): SolidityStackTrace;
//# sourceMappingURL=mapped-inlined-internal-functions-heuristics.d.ts.map