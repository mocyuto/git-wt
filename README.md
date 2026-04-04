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
- **Path Automation**: Automatically generates worktree paths based on branch names (`{project}-{branch}`) adjacent to the repository root. Use `--path` for custom locations, `--base` to specify a base branch, and `--from-default` to force using the default branch as base.
- **Lifecycle Management**: Support for listing (`list`/`ls`) and removing (`remove`/`rm`) worktrees.
- **Port Management**: Automatically assigns unique port indexes to each worktree to prevent port collisions.
- **Custom Hooks**: Execute multiple shell commands naturally after creating (`add`) or removing (`rm`) worktrees.
- **Configuration Visibility**: `config` subcommand to inspect the final merged configuration and validate syntax.
- **Bi-directional Sync**: `sync` subcommand to synchronize ignored files from worktree back to the project root.
- **Agent Skills**: Distribute and install expert skills for AI agents.

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
2. **Auto-Path Generation**: Provide just the branch name, and it will be placed as `{project}-{branch}` at the same level as your repository root automatically. You can specify a custom path with `--path` or `-p`.
3. **Config Synchronization**: Identifies "ignored files" (like `.env`) in your main tree and copies them over, maintaining the directory structure.
4. **Port Reservation**: Reserves a unique "Port Index" for this specific worktree.
5. **Automated Setup**: If you define hooks like `npm install` in `hooks.add`, they run immediately after creation.

```bash
# Start a new feature in a fresh worktree (automated path)
zgt add feature-login

# Start a new feature from a specific base branch
zgt add feature-login --base develop

# Start a new feature forcing the default branch as base
zgt add feature-login --from-default

# Start a new feature in a custom directory
zgt add feature-login --path ./experimental-worktree
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
- **Path Filtering**: Use `-p` / `--path` to filter the list of files by a specific path or substring.

```bash
# Selectively sync changes back to root
zgt sync

# Sync files matching a specific path
zgt sync --path internal/config
```

### 5. Monitoring Assignments (`list` / `ports`)

- `list`: Shows which worktrees are active, their branch status, GitHub PR details, assigned ports, and whether they have uncommitted changes (`[DIRTY]`).
- `ports`: Shows the mapping of worktree paths to their assigned port indexes and actual port numbers. Use `-a` / `--all` to see assignments across all projects.

### 6. Other Commands

- `tmux ls`: Shows tmux sessions, windows, and panes status (Running/Waiting) in a hierarchical tree structure.
- `tmux open`: Opens or activates the tmux window for the specified worktree. If the window exists, it switches to it; otherwise, it creates it according to the configuration.
- `tmux save [save-name]`: Saves current tmux windows to the state file under the specified name. Use `-s` or `--session` to specify a source session (defaults to current).
- `tmux restore [save-name]`: Restores tmux windows from the state using the specified save name.
- `tmux close`: Gracefully closes the tmux window for the specified worktree (sends SIGTERM and waits).
- `ports update`: Synchronizes port assignments for the current project with the latest configuration. It adds missing port assignments and removes those no longer present in the configuration.
- `config`: Displays the final merged configuration (global + project + flags) in YAML format. Use `--check` to validate configuration syntax, or `--raw` to skip placeholder replacement.
- `config edit`: Edits the configuration file using the system editor. Defaults to editing the local project configuration.
- `version`: Prints the version number of `zgt`.
- `skill install`: Installs skills from the current repository's `skills/` directory to the global agent skills directory (`~/.claude/skills/`).

## Configuration

You can configure `zgt` globally (`~/.config/zgt/config.yaml`) or locally (`zgt.config.yml` in project root).

```yaml
# Examples of available configuration options
add:
  from_default: true # Always base new worktrees on the default branch (e.g., main)
  auto_pull: true # Pull updates on the default branch before creating a new worktree (automatically stashes uncommitted changes)

ignore:
  - .env
```

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

tmux:
  enabled: true
  keep_open: true # Do not close the tmux window on 'rm' (default: false)
  panes:
    - id: main
      commands: ["yarn"]
    - id: dev
      target: main
      split: horizontal
      size: 50%
      commands: ["yarn dev"]
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

### Pull Default Branch After Removal

You can automatically pull the default branch from the remote repository after removing a worktree by adding a git command to your `rm` hooks.

```yaml
hooks:
  rm:
    - "git pull origin main:main"
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

Placeholders can be used in `hooks`, `env`, and `tmux.window_name` values.

| Placeholder          | Description                                                   |
| :------------------- | :------------------------------------------------------------ |
| `{{.Path}}`          | Absolute path of the worktree directory.                      |
| `{{.Repo}}`          | Name of the main project root directory.                      |
| `{{.CurrentDir}}`    | Name of the current working directory.                        |
| `{{.Branch}}`        | Name of the target branch (provided as argument).             |
| `{{.TargetBranch}}`  | Alias for `{{.Branch}}`.                                      |
| `{{.CurrentBranch}}` | Name of the branch you are currently on when executing `zgt`. |

#### Template Functions

You can use functions to transform placeholder values.

| Function   | Description                                              | Example                                |
| :--------- | :------------------------------------------------------- | :------------------------------------- |
| `hostname` | Replaces hostname-unsafe characters (`_`, `/`) with `-`. | `{{.Branch \| hostname}}`->`feat-test` |

### Tmux Integration

`zgt` can automatically set up a tmux window with multiple panes and execute commands in each when you run `add`. You can explicitly target panes for splitting using IDs.

```yaml
tmux:
  enabled: true
  panes:
    - id: main
      commands: ["yarn"]
    - id: side
      target: main
      split: horizontal
      size: 50%
      commands: ["yarn dev"]
    - target: side
      split: vertical
      commands: ["yarn watch"]
    - target: main
      split: vertical
      commands: ["tail -f logs/app.log"]
```

#### Pane Properties

| Property      | Description                                                                   |
| :------------ | :---------------------------------------------------------------------------- |
| `enabled`     | Whether to enable tmux integration (`true` / `false`).                        |
| `keep_open`   | Whether to NOT close the window when running `zgt rm` (default: `false`).     |
| `window_name` | (Optional) Template for the tmux window name.                                 |
| `id`          | (Optional) Unique ID for the pane to be referenced as a `target`.             |
| `target`      | (Optional) ID of the pane to split. If omitted, splits the last created pane. |
| `commands`    | List of commands to execute in the pane.                                      |
| `split`       | Split direction: `horizontal` (h) or `vertical` (v).                          |
| `size`        | Pane size (e.g., `20%` for percentage or `20` for lines/columns).             |

If `enabled` is `true`, `zgt` will:

1. Create a new tmux window named as specified in `window_name` (defaults to `[repo]branch`).
2. Follow the `panes` list to create splits. Each split targets the specified `target` or the last created pane.
3. Execute the `commands` in each pane and keep the shell open.

#### Note

- Requires `tmux` to be installed and a tmux session to be running.

#### Note

- If you only need a single command, you can use a string instead of a list: `add: "echo hello"`.
- Commands are executed via `/bin/sh -c`, allowing for pipes and status checks.

## Agent Skill Management

`zgt` provides a way to manage "Expert Skills" for AI agents. Skills are sets of instructions and resources that extend an agent's capabilities.

### Installing Skills (`skill install`)

To install the skills from the current repository so that your AI collaborator (like Claude) can use them:

```bash
zgt skill install
```

This will open an interactive TUI to choose from 4 installation targets:

- **Local .claude**: `./.claude/skills/`
- **Local .agents**: `./.agents/skills/`
- **Global .claude**: `~/.claude/skills/`
- **Global .agents**: `~/.agents/skills/`

Use `-a` or `--all` to install to all targets immediately.

## Requirements

- `git` and `gh` must be installed and available in your environment.

## Development & Testing

```bash
# Run tests
go test -v ./...
```

## FAQ

### Q: What happens if I `add` a branch that already exists?

`zgt` will check if the branch exists locally or on the remote (`origin`).

- If it exists, `zgt` will use the existing branch for the new worktree.
- Note that Git does not allow the same branch to be checked out in multiple worktrees simultaneously. If the branch is already in use elsewhere, the command will fail with a Git error.

### Q: Does the `rm` command delete the created directory?

Yes, `zgt rm` (or `remove`) will:

1. Delete the worktree directory from your filesystem.
2. Unregister the worktree from Git.
3. Delete the associated local branch (unless `--keep-branch` is used).
   If there are uncommitted changes, Git will protect the directory from deletion unless `-f` (`--force`) is used.
