import type { ContractABI, HintFn, HookFn } from './decoder.ts';

export function addHint<T extends ContractABI>(abi: ContractABI, name: string, fn: HintFn): T {
  const res = [];
  for (const elm of abi) {
    if (elm.name === name) res.push({ ...elm, hint: fn });
    else res.push(elm);
  }
  return res as unknown as T;
}

export function addHints<T extends ContractABI>(abi: T, map: Record<string, HintFn>): T {
  const res = [];
  for (const elm of abi) {
    if (['event', 'function'].includes(elm.type) && elm.name && map[elm.name]) {
      res.push({ ...elm, hint: map[elm.name!] });
    } else res.push(elm);
  }
  return res as unknown as T;
}

export function addHook<T extends ContractABI>(abi: T, name: string, fn: HookFn): T {
  const res = [];
  for (const elm of abi) {
    if (elm.type === 'function' && elm.name === name) res.push({ ...elm, hook: fn });
    else res.push(elm);
  }
  return res as unknown as T;
}
