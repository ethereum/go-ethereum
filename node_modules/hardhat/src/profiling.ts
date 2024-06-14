import { assertHardhatInvariant } from "./internal/core/errors";

/**
 * Utility to create ad-hoc profiles when computing flamegraphs. You can think
 * of these as virtual tasks that execute the function `f`.
 *
 * This is an **unstable** feature, only meant for development. DO NOT use in
 * production code nor plugins.
 *
 * @param name The name of the profile. Think of it as a virtual task name.
 * @param f The function you want to profile.
 */
export async function adhocProfile<T>(
  name: string,
  f: () => Promise<T>
): Promise<T> {
  const globalAsAny = global as any;
  assertHardhatInvariant(
    "adhocProfile" in globalAsAny,
    "adhocProfile is missing. Are you running with --flamegraph?"
  );
  return globalAsAny.adhocProfile(name, f);
}

/**
 * Sync version of adhocProfile
 *
 * @see adhocProfile
 */
export function adhocProfileSync<T>(name: string, f: () => T): T {
  const globalAsAny = global as any;
  assertHardhatInvariant(
    "adhocProfileSync" in globalAsAny,
    "adhocProfileSync is missing. Are you running with --flamegraph?"
  );
  return globalAsAny.adhocProfileSync(name, f);
}
