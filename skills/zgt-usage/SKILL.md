# ZGT Usage Skill

This skill provides instructions for AI agents on how to use `zgt` to manage Git worktrees and development environments effectively.

## Overview

`zgt` is a CLI tool that extends `git worktree add` by automatically copying ignored configuration files (like `.env`) to the new directory. It also manages ports and tmux sessions.

## Core Commands

### 1. Adding a Worktree (`add`)

Use `zgt add <branch>` to create a new worktree.

- It automatically generates a path: `../{project}-{branch}`.
- It copies ignored files specified in `.gitignore` or `zgt.config.yml`.
- It executes `hooks.add` (e.g., `npm install`).

### 2. Removing a Worktree (`remove` / `rm`)

Use `zgt rm <branch>` to cleanup.

- It deletes the worktree directory.
- It deletes the associated local branch.
- It releases the assigned port index.

### 3. Synchronizing Ignored Files (`sync`)

Use `zgt sync` to bring changes from a worktree back to the project root.

- Use `zgt sync -a` for immediate sync of all files.
- Without flags, it opens an interactive TUI.

### 4. Environment and Ports (`env` / `ports`)

- `zgt env`: Outputs export commands for environment variables (eval "$(zgt env)").
- `zgt ports`: Shows port assignments.

## Best Practices for Agents

1. **New Feature Development**: When starting a new task, always check if a worktree is needed. Use `zgt add` to set up a clean environment.
2. **Path Context**: After `zgt add`, remember that the project root is sibling to your current repository.
3. **Configuration**: If the user needs specific files copied, suggest adding them to the `ignore` list in `zgt.config.yml`.
4. **Cleanup**: Proactively suggest `zgt rm` when a task is completed and the PR is merged.
