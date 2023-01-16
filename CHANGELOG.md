<!--
Guiding Principles:

Changelogs are for humans, not machines.
There should be an entry for every single version.
The same types of changes should be grouped.
Versions and sections should be linkable.
The latest version comes first.
The release date of each version is displayed.
Mention whether you follow Semantic Versioning.

Usage:

Change log entries are to be added to the Unreleased section under the
appropriate stanza (see below). Each entry should ideally include a tag and
the Github issue reference in the following format:

* (<tag>) \#<issue-number> message

The issue numbers will later be link-ified during the release process so you do
not have to worry about including a link manually, but you can if you wish.

Types of changes (Stanzas):

"Features" for new features.
"Improvements" for changes in existing functionality.
"Deprecated" for soon-to-be removed features.
"Bug Fixes" for any bug fixes.
"Client Breaking" for breaking CLI commands and REST routes used by end-users.
"API Breaking" for breaking exported APIs used by developers building on SDK.
"State Machine Breaking" for any changes that result in a different AppState given same genesisState and txList.

Ref: https://keepachangelog.com/en/1.0.0/
-->

# Changelog

## Unreleased

### Improvements

* [#8](https://github.com/evmos/go-ethereum/pull/8) Add `Address` function to `PrecompiledContract` interface.
* [#7](https://github.com/evmos/go-ethereum/pull/7) Implement custom active precompiles for the EVM.
* [#6](https://github.com/evmos/go-ethereum/pull/6) Refactor `Stack` implementation
* [#3](https://github.com/evmos/go-ethereum/pull/3) Move the `JumpTable` defaults to a separate function.
* [#2](https://github.com/evmos/go-ethereum/pull/2) Define `Interpreter` interface for the EVM.
