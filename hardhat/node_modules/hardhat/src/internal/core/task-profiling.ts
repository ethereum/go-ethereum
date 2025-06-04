export interface TaskProfile {
  name: string;
  start: bigint;
  end?: bigint;
  children: TaskProfile[];
  parallel?: boolean;
}

export function createTaskProfile(name: string): TaskProfile {
  return {
    name,
    start: process.hrtime.bigint(),
    children: [],
  };
}

export function completeTaskProfile(taskProfile: TaskProfile) {
  taskProfile.end = process.hrtime.bigint();
}

export function createParentTaskProfile(taskProfile: TaskProfile): TaskProfile {
  return createTaskProfile(`super::${taskProfile.name}`);
}

/**
 * Sets `parallel` to `true` to any children that was running at the same time
 * of another.
 *
 * We assume `children[]` is in chronological `start` order.
 */
export function flagParallelChildren(
  profile: TaskProfile,
  isParentParallel = false
) {
  if (isParentParallel) {
    profile.parallel = true;
    for (const child of profile.children) {
      child.parallel = true;
    }
  } else {
    for (const [i, child] of profile.children.entries()) {
      if (i === 0) {
        continue;
      }
      const prevChild = profile.children[i - 1];
      if (child.start < prevChild.end!) {
        prevChild.parallel = true;
        child.parallel = true;
      }
    }
  }

  for (const child of profile.children) {
    flagParallelChildren(child, child.parallel);
  }
}
