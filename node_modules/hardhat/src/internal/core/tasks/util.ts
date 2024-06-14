import { TaskIdentifier } from "../../../types";

export function parseTaskIdentifier(taskIdentifier: TaskIdentifier): {
  scope: string | undefined;
  task: string;
} {
  if (typeof taskIdentifier === "string") {
    return {
      scope: undefined,
      task: taskIdentifier,
    };
  } else {
    return {
      scope: taskIdentifier.scope,
      task: taskIdentifier.task,
    };
  }
}
