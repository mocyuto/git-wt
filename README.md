# git-wt

English | [日本語](./README_ja.md)

A CLI tool that extends `git worktree add` by automatically copying ignored configuration files (like `.env`) to the new directory.

## Overview

Git's `worktree` feature is powerful, but files ignored by `.gitignore` (such as `.env` or local configs) are not included in the newly created worktree. `git-wt` automates the process of copying these files, allowing you to start development and testing immediately.

## Features

- **Standard Wrapper**: Works as a wrapper for `git worktree add`.
- **Auto-Discovery**: Automatically identifies and copies "ignored files" specified in `.gitignore`.
- **Structural Integrity**: Maintains directory structure during copy (e.g., config files inside `node_modules`).
- **Flexible Interface**: Powered by the Cobra framework for robust flag handling.
- **Path Automation**: Automatically generates worktree paths based on branch names (`../{project}-{branch}`).
- **Lifecycle Management**: Support for listing (`list`/`ls`) and removing (`remove`/`rm`) worktrees.
- **Port Management**: Automatically assigns unique port indexes to each worktree to prevent port collisions.
- **Custom Hooks**: Execute multiple shell commands naturally after creating (`add`) or removing (`rm`) worktrees.

## Installation

### Homebrew

The easiest way to install on macOS is via Homebrew:

```bash
brew install mocyuto/tap/git-wt
```

### Build

```bash
go build -o git-wt main.go
```

### Move to PATH

Place the binary in a directory included in your `PATH`.

```bash
# Example for macOS / Linux
sudo mv git-wt /usr/local/bin/git-wt
```

> [!TIP]
> If you name the binary `git-wt` and place it in your `PATH`, you can also call it as `git wt ...`.

## Usage

Since it uses Cobra, flags (like `-b`) can be placed either before or after the positional arguments.

### 1. Create with Branch Name Only (Auto-Path)

Providing only a branch name will automatically create a worktree at `../{current_dir}-{branch}`. If the branch doesn't exist, it will be created automatically.

```bash
git-wt add <branch>
```

**Example (if project is `pj`):**

```bash
git-wt add feature-abc
# -> Created at ../pj-feature-abc
```

### 2. Create with Explicit Path

```bash
git-wt add <path> <branch>
```

**Example:**

```bash
git-wt add ../debug-fix main
```

### 3. Create with New Branch

```bash
git-wt add -b <new-branch> <path>
# or
git-wt add <path> -b <new-branch>
```

**Example:**

```bash
git-wt add -b feature/login ../feature-login
```

### 4. List Worktrees

Lists all worktrees with their paths, commit hashes, branch names, and GitHub PR status (if `gh` CLI is available).

```bash
git-wt list
# or
git-wt ls
```

### 5. Remove Worktree

Remove a worktree by providing the branch name (or path). By default, the associated branch is also deleted.

```bash
git-wt remove <branch>
# or
git-wt rm <branch>
```

**Example:**

```bash
git-wt rm feature/login
```

**Force Removal (if there are uncommitted changes):**

```bash
git-wt rm -f <branch>
```

**Keep the Branch (don't delete it):**

```bash
git-wt rm -k <branch>
# or
git-wt rm --keep-branch <branch>
```

### 6. Export Port Variables

Generates shell export commands for variables defined in your config, offset by the worktree's assigned index.

```bash
eval $(git-wt env)
```

### 7. List Port Assignments

Lists all worktree paths and their assigned port indexes.

```bash
git-wt ports
```

## Configuration

`git-wt` loads configuration from three sources in this priority:

1. Local project configuration (`git-wt.config.yaml` or `git-wt.config.yml` in project root)
2. Global configuration (`~/.config/git-wt/config.yaml`)
3. Explicit configuration path provided via `--config` flag

### Project-Specific Configuration

You can create a `git-wt.config.yaml` (or `.yml`) in your project's root directory to define settings specific to that project. Local settings for `hooks` and `ignore` will be **appended** to the global settings.

```yaml
# git-wt.config.yaml
ignore:
  - "*.tmp"
  - "local-debug.log"

hooks:
  add:
    - "npm install"

ports:
  api: 8080
  web: 3000
```

### Port Management Logic

When you add a worktree with `git-wt add`, it is assigned a unique `PortIndex` (starting from 0).
Calling `git-wt env` will export environment variables using the pattern `UPPER_NAME_PORT = BasePort + PortIndex`.

Example for `PortIndex: 1` with the config above:

- `API_PORT=8081`
- `WEB_PORT=3001`

### Custom Ignore Patterns

You can specify additional file patterns to be ignored during the copy process. These patterns follow the same format as `.gitignore` (using `filepath.Match`).

```yaml
ignore:
  - ".env.production"
  - "secrets/*"
```

### Custom Hooks

Hooks allow you to run automated shell commands when worktrees are managed.

```yaml
hooks:
  # Commands to run after 'add'
  add:
    - "tmux new-window -n [{{.Repo}}]{{.Branch}} -c {{.Path}}"
    - "echo 'Welcome to {{.Repo}}'"
  # Commands to run after 'remove'
  rm:
    - "echo 'Cleanup for {{.Branch}}'"
```

#### Available Placeholders

| Placeholder   | Description                                   |
| :------------ | :-------------------------------------------- |
| `{{.Path}}`   | Absolute path of the worktree directory.      |
| `{{.Branch}}` | Name of the branch.                           |
| `{{.Repo}}`   | Name of the repository (base directory name). |

#### Note

- If you only need a single command, you can use a string instead of a list: `add: "echo hello"`.
- Commands are executed via `/bin/sh -c`, allowing for pipes and status checks.

## Requirements

- `git` must be installed and available in your environment.

## Development & Testing

```bash
# Run tests
go test -v .
```
