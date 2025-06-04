import findup from "find-up";
import * as fs from "fs-extra";
import * as path from "path";

import { HardhatRuntimeEnvironment, TaskDefinition } from "../../types";
import { HARDHAT_PARAM_DEFINITIONS } from "../core/params/hardhat-params";
import { getCacheDir } from "../util/global-dir";
import { createNonCryptographicHashBasedIdentifier } from "../util/hash";
import { mapValues } from "../util/lang";

import { ArgumentsParser } from "./ArgumentsParser";

type GlobalParam = keyof typeof HARDHAT_PARAM_DEFINITIONS;

interface Suggestion {
  name: string;
  description: string;
}

interface CompletionEnv {
  line: string;
  point: number;
}

interface Task {
  name: string;
  description: string;
  isSubtask: boolean;
  paramDefinitions: {
    [paramName: string]: {
      name: string;
      description: string;
      isFlag: boolean;
    };
  };
}

interface CompletionData {
  networks: string[];
  tasks: {
    [taskName: string]: Task;
  };
  scopes: {
    [scopeName: string]: {
      name: string;
      description: string;
      tasks: {
        [taskName: string]: Task;
      };
    };
  };
}

interface Mtimes {
  [filename: string]: number;
}

interface CachedCompletionData {
  completionData: CompletionData;
  mtimes: Mtimes;
}

export const HARDHAT_COMPLETE_FILES = "__hardhat_complete_files__";

export const REQUIRED_HH_VERSION_RANGE = "^1.0.0";

export async function complete({
  line,
  point,
}: CompletionEnv): Promise<Suggestion[] | typeof HARDHAT_COMPLETE_FILES> {
  const completionData = await getCompletionData();

  if (completionData === undefined) {
    return [];
  }

  const { networks, tasks, scopes } = completionData;

  const words = line.split(/\s+/).filter((x) => x.length > 0);

  const wordsBeforeCursor = line.slice(0, point).split(/\s+/);
  // 'prev' and 'last' variables examples:
  // `hh compile --network|` => prev: "compile" last: "--network"
  // `hh compile --network |` => prev: "--network" last: ""
  // `hh compile --network ha|` => prev: "--network" last: "ha"
  const [prev, last] = wordsBeforeCursor.slice(-2);

  const startsWithLast = (completion: string) => completion.startsWith(last);

  const coreParams = Object.values(HARDHAT_PARAM_DEFINITIONS)
    .map((param) => ({
      name: ArgumentsParser.paramNameToCLA(param.name),
      description: param.description ?? "",
    }))
    .filter((x) => !words.includes(x.name));

  // Get the task or scope if the user has entered one
  let taskName: string | undefined;
  let scopeName: string | undefined;

  let index = 1;
  while (index < words.length) {
    const word = words[index];

    if (isGlobalFlag(word)) {
      index += 1;
    } else if (isGlobalParam(word)) {
      index += 2;
    } else if (word.startsWith("--")) {
      index += 1;
    } else {
      // Possible scenarios:
      // - no task or scope: `hh `
      // - only a task: `hh task `
      // - only a scope: `hh scope `
      // - both a scope and a task (the task always follow the scope): `hh scope task `
      // Between a scope and a task there could be other words, e.g.: `hh scope --flag task `
      if (scopeName === undefined) {
        if (tasks[word] !== undefined) {
          taskName = word;
          break;
        } else if (scopes[word] !== undefined) {
          scopeName = word;
        }
      } else {
        taskName = word;
        break;
      }

      index += 1;
    }
  }

  // If a task or a scope is found and it is equal to the last word,
  // this indicates that the cursor is positioned after the task or scope.
  // In this case, we ignore the task or scope. For instance, if you have a task or a scope named 'foo' and 'foobar',
  // and the line is 'hh foo|', we want to suggest the value for 'foo' and 'foobar'.
  // Possible scenarios:
  // - no task or scope: `hh ` -> task and scope already undefined
  // - only a task: `hh task ` -> task set to undefined, scope already undefined
  // - only a scope: `hh scope ` -> scope set to undefined, task already undefined
  // - both a scope and a task (the task always follow the scope): `hh scope task ` -> task set to undefined, scope stays defined
  if (taskName === last || scopeName === last) {
    if (taskName !== undefined && scopeName !== undefined) {
      [taskName, scopeName] = [undefined, scopeName];
    } else {
      [taskName, scopeName] = [undefined, undefined];
    }
  }

  if (prev === "--network") {
    return networks.filter(startsWithLast).map((network) => ({
      name: network,
      description: "",
    }));
  }

  const scopeDefinition =
    scopeName === undefined ? undefined : scopes[scopeName];

  const taskDefinition =
    taskName === undefined
      ? undefined
      : scopeDefinition === undefined
      ? tasks[taskName]
      : scopeDefinition.tasks[taskName];

  // if the previous word is a param, then a value is expected
  // we don't complete anything here
  if (prev.startsWith("-")) {
    const paramName = ArgumentsParser.cLAToParamName(prev);

    const globalParam = HARDHAT_PARAM_DEFINITIONS[paramName as GlobalParam];
    if (globalParam !== undefined && !globalParam.isFlag) {
      return HARDHAT_COMPLETE_FILES;
    }

    const isTaskParam =
      taskDefinition?.paramDefinitions[paramName]?.isFlag === false;

    if (isTaskParam) {
      return HARDHAT_COMPLETE_FILES;
    }
  }

  // If there's no task or scope, we complete either tasks and scopes or params
  if (taskDefinition === undefined && scopeDefinition === undefined) {
    if (last.startsWith("-")) {
      return coreParams.filter((param) => startsWithLast(param.name));
    }

    const taskSuggestions = Object.values(tasks)
      .filter((x) => !x.isSubtask)
      .map((x) => ({
        name: x.name,
        description: x.description,
      }));

    const scopeSuggestions = Object.values(scopes).map((x) => ({
      name: x.name,
      description: x.description,
    }));

    return taskSuggestions
      .concat(scopeSuggestions)
      .filter((x) => startsWithLast(x.name));
  }

  // If there's a scope but not a task, we complete with the scopes'tasks
  if (taskDefinition === undefined && scopeDefinition !== undefined) {
    return Object.values(scopes[scopeName!].tasks)
      .filter((x) => !x.isSubtask)
      .map((x) => ({
        name: x.name,
        description: x.description,
      }))
      .filter((x) => startsWithLast(x.name));
  }

  if (!last.startsWith("-")) {
    return HARDHAT_COMPLETE_FILES;
  }

  const taskParams =
    taskDefinition === undefined
      ? []
      : Object.values(taskDefinition.paramDefinitions)
          .map((param) => ({
            name: ArgumentsParser.paramNameToCLA(param.name),
            description: param.description,
          }))
          .filter((x) => !words.includes(x.name));

  return [...taskParams, ...coreParams].filter((suggestion) =>
    startsWithLast(suggestion.name)
  );
}

async function getCompletionData(): Promise<CompletionData | undefined> {
  const projectId = getProjectId();

  if (projectId === undefined) {
    return undefined;
  }

  const cachedCompletionData = await getCachedCompletionData(projectId);

  if (cachedCompletionData !== undefined) {
    if (arePreviousMtimesCorrect(cachedCompletionData.mtimes)) {
      return cachedCompletionData.completionData;
    }
  }

  const filesBeforeRequire = Object.keys(require.cache);
  let hre: HardhatRuntimeEnvironment;
  try {
    process.env.TS_NODE_TRANSPILE_ONLY = "1";
    require("../../register");
    hre = (global as any).hre;
  } catch {
    return undefined;
  }
  const filesAfterRequire = Object.keys(require.cache);
  const mtimes = getMtimes(filesBeforeRequire, filesAfterRequire);

  const networks = Object.keys(hre.config.networks);

  // we extract the tasks data explicitly to make sure everything
  // is serializable and to avoid saving unnecessary things from the HRE
  const tasks: CompletionData["tasks"] = mapValues(hre.tasks, (task) =>
    getTaskFromTaskDefinition(task)
  );

  const scopes: CompletionData["scopes"] = mapValues(hre.scopes, (scope) => ({
    name: scope.name,
    description: scope.description ?? "",
    tasks: mapValues(scope.tasks, (task) => getTaskFromTaskDefinition(task)),
  }));

  const completionData: CompletionData = {
    networks,
    tasks,
    scopes,
  };

  await saveCachedCompletionData(projectId, completionData, mtimes);

  return completionData;
}

function getTaskFromTaskDefinition(taskDef: TaskDefinition): Task {
  return {
    name: taskDef.name,
    description: taskDef.description ?? "",
    isSubtask: taskDef.isSubtask,
    paramDefinitions: mapValues(
      taskDef.paramDefinitions,
      (paramDefinition) => ({
        name: paramDefinition.name,
        description: paramDefinition.description ?? "",
        isFlag: paramDefinition.isFlag,
      })
    ),
  };
}

function getProjectId(): string | undefined {
  const packageJsonPath = findup.sync("package.json");

  if (packageJsonPath === undefined) {
    return undefined;
  }

  return createNonCryptographicHashBasedIdentifier(
    Buffer.from(packageJsonPath)
  ).toString("hex");
}

function arePreviousMtimesCorrect(mtimes: Mtimes): boolean {
  try {
    return Object.entries(mtimes).every(
      ([file, mtime]) => fs.statSync(file).mtime.valueOf() === mtime
    );
  } catch {
    return false;
  }
}

function getMtimes(filesLoadedBefore: string[], filesLoadedAfter: string[]) {
  const loadedByHardhat = filesLoadedAfter.filter(
    (f) => !filesLoadedBefore.includes(f)
  );
  const stats = loadedByHardhat.map((f) => fs.statSync(f));

  const mtimes = loadedByHardhat.map((f, i) => ({
    [f]: stats[i].mtime.valueOf(),
  }));

  if (mtimes.length === 0) {
    return {};
  }

  return Object.assign(mtimes[0], ...mtimes.slice(1));
}

async function getCachedCompletionData(
  projectId: string
): Promise<CachedCompletionData | undefined> {
  const cachedCompletionDataPath = await getCachedCompletionDataPath(projectId);

  if (fs.existsSync(cachedCompletionDataPath)) {
    try {
      const cachedCompletionData = fs.readJsonSync(cachedCompletionDataPath);
      return cachedCompletionData;
    } catch {
      // remove the file if it seems invalid
      fs.unlinkSync(cachedCompletionDataPath);
      return undefined;
    }
  }
}

async function saveCachedCompletionData(
  projectId: string,
  completionData: CompletionData,
  mtimes: Mtimes
): Promise<void> {
  const cachedCompletionDataPath = await getCachedCompletionDataPath(projectId);

  await fs.outputJson(cachedCompletionDataPath, { completionData, mtimes });
}

async function getCachedCompletionDataPath(projectId: string): Promise<string> {
  const cacheDir = await getCacheDir();

  return path.join(cacheDir, "autocomplete", `${projectId}.json`);
}

function isGlobalFlag(param: string): boolean {
  const paramName = ArgumentsParser.cLAToParamName(param);
  return HARDHAT_PARAM_DEFINITIONS[paramName as GlobalParam]?.isFlag === true;
}

function isGlobalParam(param: string): boolean {
  const paramName = ArgumentsParser.cLAToParamName(param);
  return HARDHAT_PARAM_DEFINITIONS[paramName as GlobalParam]?.isFlag === false;
}
