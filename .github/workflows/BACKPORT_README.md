# Backport Workflow Documentation

## Overview

This repository uses an automated backport workflow to cherry-pick merged pull requests from the `main` branch to release branches. The workflow is powered by the [korthout/backport-action](https://github.com/marketplace/actions/backport-merged-pull-requests-to-selected-branches).

## How It Works

### Automatic Backporting via Labels

1. **Add a backport label** to your pull request before or after merging:
   - Label format: `backport/{version}`
   - Examples: `backport/v1.0.0`, `backport/v2.3.1`, `backport/1.0`

2. **When the PR is merged to main**, the workflow automatically:
   - Detects all `backport/*` labels
   - Transforms them to target `release/{version}` branches
   - Creates new pull requests with cherry-picked commits to each target branch

3. **Target branches** must exist:
   - For label `backport/v1.0.0` → targets branch `release/v1.0.0`
   - For label `backport/2.0` → targets branch `release/2.0`
   - The release branch must already exist in the repository

### Manual Backporting via Comments

You can also trigger backports manually by commenting on a merged PR:
- Comment `/backport` on any merged pull request
- The workflow will look for backport labels and create the appropriate PRs

## Example Workflow

1. Create a pull request to `main` with your changes
2. Add label: `backport/v1.2.3`
3. Get the PR reviewed and merged
4. The bot automatically:
   - Creates a new branch: `backport-{PR-number}-to-release-v1.2.3`
   - Cherry-picks the commits from your merged PR
   - Opens a new PR from that branch to `release/v1.2.3`
   - Adds labels: `backport`, `automated`
   - Assigns the original PR author

## Handling Multiple Backports

You can backport to multiple release branches by adding multiple labels:
- `backport/v1.0.0` - backports to `release/v1.0.0`
- `backport/v2.0.0` - backports to `release/v2.0.0`
- `backport/v3.0.0` - backports to `release/v3.0.0`

Each label will create a separate backport PR.

## Conflict Resolution

When cherry-pick conflicts occur:
1. The bot creates a **draft PR** with the conflicts committed
2. The draft PR includes instructions on how to resolve the conflicts
3. You need to:
   - Check out the backport branch locally
   - Resolve the conflicts
   - Push the resolved changes
   - Mark the PR as ready for review

## Backport PR Format

Backport PRs are created with:
- **Title**: `[Backport release/{version}] {original PR title}`
- **Description**: Includes original PR information, author, and description
- **Labels**: `backport`, `automated`
- **Assignee**: Original PR author

## Prerequisites

For the workflow to function correctly:
1. Release branches must follow the naming pattern: `release/{version}`
2. The workflow must have write permissions (already configured)
3. Release branches must exist before attempting to backport

## Troubleshooting

### Backport not triggering
- Ensure the PR is merged to `main` (not just closed)
- Verify the label format is exactly `backport/{version}`
- Check that the target `release/{version}` branch exists

### Conflicts in backport
- The bot will create a draft PR with conflicts
- Follow the instructions in the PR to resolve conflicts locally

### Multiple commits or merge commits
- The workflow is configured to skip merge commits
- Only non-merge commits will be cherry-picked
- If you need merge commits, update the `merge_commits` setting in the workflow

## Configuration

The workflow is defined in `.github/workflows/backport.yml`. Key settings:
- **Label pattern**: `backport/{version}`
- **Target branch pattern**: `release/{version}`
- **Conflict strategy**: Create draft PRs with conflicts
- **Merge commits**: Skipped (only cherry-picks non-merge commits)

## Security

- The workflow uses `GITHUB_TOKEN` for authentication
- Runs on `pull_request_target` to handle forks securely
- Only triggers on merged PRs to prevent abuse
