import {
	ImmerState,
	Patch,
	Drafted,
	ImmerBaseState,
	AnyMap,
	AnySet,
	ArchType,
	die
} from "../internal"

/** Plugin utilities */
const plugins: {
	Patches?: {
		generatePatches_(
			state: ImmerState,
			basePath: PatchPath,
			patches: Patch[],
			inversePatches: Patch[]
		): void
		generateReplacementPatches_(
			base: any,
			replacement: any,
			patches: Patch[],
			inversePatches: Patch[]
		): void
		applyPatches_<T>(draft: T, patches: Patch[]): T
	}
	MapSet?: {
		proxyMap_<T extends AnyMap>(target: T, parent?: ImmerState): T
		proxySet_<T extends AnySet>(target: T, parent?: ImmerState): T
	}
} = {}

type Plugins = typeof plugins

export function getPlugin<K extends keyof Plugins>(
	pluginKey: K
): Exclude<Plugins[K], undefined> {
	const plugin = plugins[pluginKey]
	if (!plugin) {
		die(0, pluginKey)
	}
	// @ts-ignore
	return plugin
}

export function loadPlugin<K extends keyof Plugins>(
	pluginKey: K,
	implementation: Plugins[K]
): void {
	if (!plugins[pluginKey]) plugins[pluginKey] = implementation
}
/** Map / Set plugin */

export interface MapState extends ImmerBaseState {
	type_: ArchType.Map
	copy_: AnyMap | undefined
	assigned_: Map<any, boolean> | undefined
	base_: AnyMap
	revoked_: boolean
	draft_: Drafted<AnyMap, MapState>
}

export interface SetState extends ImmerBaseState {
	type_: ArchType.Set
	copy_: AnySet | undefined
	base_: AnySet
	drafts_: Map<any, Drafted> // maps the original value to the draft value in the new set
	revoked_: boolean
	draft_: Drafted<AnySet, SetState>
}

/** Patches plugin */

export type PatchPath = (string | number)[]
