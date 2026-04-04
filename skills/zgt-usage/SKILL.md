---
name: zgt-usage
description: Instructions for AI agents on how to use `zgt` to manage Git worktrees and development environments effectively.
---

# ZGT Usage Skill

This skill provides instructions for AI agents on how to use `zgt` to manage Git worktrees and development environments effectively.

## Overview

`zgt` is a CLI tool that extends `git worktree add` by automatically copying ignored configuration files (like `.env`) to the new directory. It also manages ports and tmux sessions.

For detailed configuration options, see the [Configuration Guide](./config.md).

## Core Commands

### 1. Adding a Worktree (`add`)

Use `zgt add <branch>` to create a new worktree.

- It automatically generates a path: `../{project}-{branch}`.
- Use `--path <dir>` to specify a custom target directory.
- Use `--base <branch>` (or `-b`) to specify a base branch to create the worktree from.
- Use `--from-default` to force using the default branch as base.
- It copies ignored files specified in `.gitignore` or `zgt.config.yml`.
- It executes `hooks.add` (e.g., `npm install`).
- **Existing Branch**: If the branch already exists, it uses it. Note that a branch cannot be checked out in multiple worktrees at once.

### 2. Removing a Worktree (`remove` / `rm`)

Use `zgt rm <branch>` to cleanup.

- It deletes the worktree directory from your filesystem.
- It deletes the associated local branch (controllable via `--keep-branch`).
- It safely closes the associated tmux window (if `tmux.keep_open` is false).
- It releases the assigned port index.

### 3. Synchronizing Ignored Files (`sync`)

Use `zgt sync` to bring changes from a worktree back to the project root.

- Use `zgt sync -a` for immediate sync of all files.
- Use `zgt sync -p <string>` / `--path <string>` to filter files by path.
- Without flags, it opens an interactive TUI.

### 4. Environment and Ports (`env` / `ports` / `tmux`)

- `zgt env`: Outputs export commands for environment variables (eval "$(zgt env)").
- `zgt ports`: Shows port assignments.
- `zgt tmux open [branch]`: Opens or activates the tmux window for a worktree. Opens an interactive TUI if no branch is specified.
- `zgt tmux close <branch>`: Gracefully closes the tmux window for a worktree.

### 5. Global Flags

- `--verbose` / `-v`: Shows detailed output and prints the current merged configuration at the start of any command execution.

## Best Practices for Agents

1. **New Feature Development**: When starting a new task, always check if a worktree is needed. Use `zgt add` to set up a clean environment.
2. **Path Context**: After `zgt add`, remember that the project root is sibling to your current repository.
3. **Configuration**: If the user needs specific files copied, suggest adding them to the `ignore` list in `zgt.config.yml`.
4. **Cleanup**: Proactively suggest `zgt rm` when a task is completed and the PR is merged.
5. **Hostname Compatibility**: If using placeholders for hostnames (e.g., in tmux window names or environment variables), use the `hostname` function to replace illegal characters: `{{.Branch | hostname}}`.
