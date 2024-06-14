import { FSWatcher } from "chokidar";
import { EIP1193Provider, ProjectPathsConfig } from "../../types";
export type Watcher = FSWatcher;
export declare function watchCompilerOutput(provider: EIP1193Provider, paths: ProjectPathsConfig): Promise<Watcher>;
//# sourceMappingURL=watch.d.ts.map