import {
	Patch,
	PatchListener,
	Drafted,
	Immer,
	DRAFT_STATE,
	ImmerState,
	ArchType,
	getPlugin
} from "../internal"

/** Each scope represents a `produce` call. */

export interface ImmerScope {
	patches_?: Patch[]
	inversePatches_?: Patch[]
	canAutoFreeze_: boolean
	drafts_: any[]
	parent_?: ImmerScope
	patchListener_?: PatchListener
	immer_: Immer
	unfinalizedDrafts_: number
}

let currentScope: ImmerScope | undefined

export function getCurrentScope() {
	return currentScope!
}

function createScope(
	parent_: ImmerScope | undefined,
	immer_: Immer
): ImmerScope {
	return {
		drafts_: [],
		parent_,
		immer_,
		// Whenever the modified draft contains a draft from another scope, we
		// need to prevent auto-freezing so the unowned draft can be finalized.
		canAutoFreeze_: true,
		unfinalizedDrafts_: 0
	}
}

export function usePatchesInScope(
	scope: ImmerScope,
	patchListener?: PatchListener
) {
	if (patchListener) {
		getPlugin("Patches") // assert we have the plugin
		scope.patches_ = []
		scope.inversePatches_ = []
		scope.patchListener_ = patchListener
	}
}

export function revokeScope(scope: ImmerScope) {
	leaveScope(scope)
	scope.drafts_.forEach(revokeDraft)
	// @ts-ignore
	scope.drafts_ = null
}

export function leaveScope(scope: ImmerScope) {
	if (scope === currentScope) {
		currentScope = scope.parent_
	}
}

export function enterScope(immer: Immer) {
	return (currentScope = createScope(currentScope, immer))
}

function revokeDraft(draft: Drafted) {
	const state: ImmerState = draft[DRAFT_STATE]
	if (state.type_ === ArchType.Object || state.type_ === ArchType.Array)
		state.revoke_()
	else state.revoked_ = true
}
