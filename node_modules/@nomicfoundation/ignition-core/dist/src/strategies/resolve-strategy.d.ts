import { ExecutionStrategy } from "../internal/execution/types/execution-strategy";
import { StrategyConfig } from "../types/deploy";
export declare function resolveStrategy<StrategyT extends keyof StrategyConfig>(strategyName: StrategyT | undefined, strategyConfig: StrategyConfig[StrategyT] | undefined): ExecutionStrategy;
//# sourceMappingURL=resolve-strategy.d.ts.map