# zgt (formerly `git-wt`)

English | [日本語](./README_ja.md)

A CLI tool that extends `git worktree add` by automatically copying ignored configuration files (like `.env`) to the new directory.

## Overview

- [Migration Guide (from `git-wt`)](./MIGRATION.md)

Git's `worktree` feature is powerful,
but files ignored by `.gitignore` (such as `.env` or local configs) are not included in the newly created worktree. `zgt` automates the process of copying these files, allowing you to start development and testing immediately.

## Features

- **Standard Wrapper**: Works as a wrapper for `git worktree add`.
- **Auto-Discovery**: Automatically identifies and copies "ignored files" specified in `.gitignore`.
- **Structural Integrity**: Maintains directory structure during copy (e.g., config files inside `node_modules`).
- **Flexible Interface**: Powered by the Cobra framework for robust flag handling.
- **Path Automation**: Automatically generates worktree paths based on branch names (`{project}-{branch}`) adjacent to the repository root.
- **Lifecycle Management**: Support for listing (`list`/`ls`) and removing (`remove`/`rm`) worktrees.
- **Port Management**: Automatically assigns unique port indexes to each worktree to prevent port collisions.
- **Custom Hooks**: Execute multiple shell commands naturally after creating (`add`) or removing (`rm`) worktrees.
- **Configuration Visibility**: `config` subcommand to inspect the final merged configuration and validate syntax.
- **Bi-directional Sync**: `sync` subcommand to synchronize ignored files from worktree back to the project root.

## Badges

[![Release](https://img.shields.io/github/release/mocyuto/zgt.svg?style=for-the-badge)](https://github.com/mocyuto/zgt/releases/latest)
[![Software License](https://img.shields.io/badge/license-MIT-brightgreen.svg?style=for-the-badge)](/LICENSE.md)
[![Build status](https://img.shields.io/github/actions/workflow/status/mocyuto/zgt/ci.yml?style=for-the-badge&branch=main)](https://github.com/mocyuto/zgt/actions?workflow=ci)

## Installation

### Homebrew

The easiest way to install on macOS is via Homebrew:

```bash
brew install mocyuto/tap/zgt
```

### Build

```bash
go build -o zgt .
```

### Move to PATH

Place the binary in a directory included in your `PATH`.

```bash
# Example for macOS / Linux
sudo mv zgt /usr/local/bin/zgt
```

## Development Workflow & Behavior

`zgt` is designed to make context switching seamless for developers by automating the repetitive parts of managing worktrees.

### 1. Starting a New Feature (`add`)

Running `git worktree add` usually leaves you with a fresh directory missing essential local files like `.env`. `zgt` automates the entire setup:

1. **Worktree Creation**: Creates the directory and checks out the branch.
2. **Auto-Path Generation**: Provide just the branch name, and it will be placed as `{project}-{branch}` at the same level as your repository root automatically.
3. **Config Synchronization**: Identifies "ignored files" (like `.env`) in your main tree and copies them over, maintaining the directory structure.
4. **Port Reservation**: Reserves a unique "Port Index" for this specific worktree.
5. **Automated Setup**: If you define hooks like `npm install` in `hooks.add`, they run immediately after creation.

```bash
# Start a new feature in a fresh worktree
zgt add feature-login
```

### 2. Running Multiple Projects Simultaneously (`env` / `ports`)

When running multiple servers across different worktrees, port collisions are a common pain point. `zgt` solves this:

- **Behavior**: Every worktree is assigned a stable index (`0, 1, 2...`).
- **Dynamic Calculation**: Port numbers are calculated by adding the index to the base port defined in the `zgt.config.yml` of the worktree's project root. This ensures correct calculation even with different base ports across projects.
- **Usage**: Define base ports (e.g., `api: 8080`) in your config. `zgt env` generates environment variables (e.g., `8080`, `8081`...) specific to that worktree.

```bash
# Move to a worktree and load its specific environment
cd ../my-project-feature-login
eval "$(zgt env)"

# $API_PORT is now 8081, preventing collision with other worktrees
npm start
```

### 3. Cleaning Up (`remove`)

When a feature is finished, `zgt` handles the teardown in one go.

- **Behavior**: Deletes the worktree directory and its associated local branch simultaneously (configurable).
- **Cleanup**: The Port Index is released and becomes available for future worktrees. You can also trigger cleanup scripts via `hooks.rm`.

```bash
# Done with the feature. Delete both directory and branch.
zgt rm feature-login
```

### 4. Synchronizing Ignored Files (`sync`)

If you've made changes to ignored configuration files (like `.env`) within your worktree and want to reflect those changes back to the main project root:

- **Interactive Mode**: Run `zgt sync` to open a TUI (powered by `rivo/tview`) where you can selectively choose which files to sync.
- **Bulk Sync**: Use `-a` / `--all` (or `--force`) to sync all ignored files immediately.

```bash
# Selectively sync changes back to root
zgt sync
```

### 5. Monitoring Assignments (`list` / `ports`)

- `list`: Shows which worktrees are active, their branch status, GitHub PR details, assigned ports, and whether they have uncommitted changes (`[DIRTY]`).
- `ports`: Shows the mapping of worktree paths to their assigned port indexes and actual port numbers. Use `-a` / `--all` to see assignments across all projects.
- `ports update`: Synchronizes port assignments for the current project with the latest configuration. It adds missing port assignments and removes those no longer present in the configuration.
- `config`: Displays the final merged configuration (global + project + flags) in YAML format. Use `--check` to validate configuration syntax.
- `config edit`: Edits the configuration file using the system editor.
- `version`: Prints the version number of `zgt`.

## Configuration

`zgt` loads configuration from three sources in this priority:

1. Local project configuration (`zgt.config.yaml` or `zgt.config.yml` in project root)
2. Global configuration (`~/.config/zgt/config.yaml`)
3. Explicit configuration path provided via `--config` flag

### Initializing Configuration (`init`)

You can generate a default configuration file (`zgt.config.yml`) in your project's root directory:

```bash
zgt init
```

This command performs the following:

1.  Creates a `zgt.config.yml` with sensible defaults for port management, environment templates, and sample hooks.
2.  Automatically appends `zgt.config.yml` and `zgt.config.yaml` to your `.gitignore` file to ensure they are not accidentally committed.

If `zgt.config.yml` or `zgt.config.yaml` already exists, the command will skip creation to prevent overwriting your existing settings.

### Customizing Configuration (`config edit`)

You can edit your configuration files directly from the CLI using your preferred editor (defined by `$EDITOR` or `vi`):

```bash
# Edit local configuration
zgt config edit --local

# Edit global configuration
zgt config edit --global
```

`zgt` will validate the YAML syntax before saving your changes.

### Project-Specific Configuration

You can create a `zgt.config.yaml` (or `.yml`) in your project's root directory to define settings specific to that project. Local settings for `hooks` and `ignore` will be **appended** to the global settings.

```yaml
# zgt.config.yaml
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

- `WEB_PORT=3001`

### Custom Environment Variables

You can define custom environment variables in the `env` section. These variables support placeholders and automatically exported via `zgt env`.

```yaml
env:
  COMPOSE_PROJECT_NAME: "zgt-{{.Repo}}"
  DEBUG: "true"
```

In your terminal:

```bash
eval "$(zgt env)"
echo $COMPOSE_PROJECT_NAME # zgt-myrepo
```

These variables are also available during [Custom Hooks](#custom-hooks) execution.

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

Placeholders can be used in `hooks` and `env` values.

| Placeholder   | Description                                   |
| :------------ | :-------------------------------------------- |
| `{{.Path}}`   | Absolute path of the worktree directory.      |
| `{{.Branch}}` | Name of the branch.                           |
| `{{.Repo}}`   | Name of the repository (base directory name). |

#### Note

- If you only need a single command, you can use a string instead of a list: `add: "echo hello"`.
- Commands are executed via `/bin/sh -c`, allowing for pipes and status checks.

## Requirements

- `git` and `gh` must be installed and available in your environment.

## Development & Testing

```bash
# Run tests
go test -v ./...
```
