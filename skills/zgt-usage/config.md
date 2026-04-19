# ZGT Configuration Guide

`zgt` can be configured globally or per-project using YAML files.

## Configuration Files

`zgt` loads configuration files in the following order of priority:

1. **Local Project Config**: `zgt.config.yml` or `zgt.config.yaml` in the project root.
2. **Global Config**: `~/.config/zgt/config.yaml`.
3. **Explicit Config**: Path specified via the `--config` flag.

### Generating Configuration (`init`)

Run `zgt init` in your project root to generate a default `zgt.config.yml` with port management, environment templates, and sample hooks. It also adds the config files to `.gitignore`.

### Editing Configuration (`config edit`)

You can edit configuration files directly using your system editor:

- `zgt config edit --local`: Edit the local project configuration.
- `zgt config edit --global`: Edit the global configuration.

The CLI automatically performs a YAML syntax check before saving.

## Configuration Options

### `add`

Options for the `add` command.

- `from_default` (bool): If true, always create new worktrees from the default branch (e.g., `main`).
- `auto_pull` (bool): If true, pull the latest changes from the remote default branch before creating a worktree.

### `ignore`

A list of file patterns to be copied to the new worktree even if they are ignored by Git (e.g., `.env`).

```yaml
ignore:
  - ".env"
  - "config/*.local.json"
```

### `hooks`

Custom shell commands to execute during worktree lifecycle.

- `add`: Commands run after `zgt add`.
- `rm`: Commands run after `zgt rm`.

### `git_hooks`

Automatically configure Git hooks for newly created worktrees.

- `enabled` (bool): If true, `zgt add` sets `core.hooksPath` in the new worktree.
- `path` (string): Path to the hooks directory. Default is `.githooks`.
- `shared` (bool): If true (default), relative paths are resolved from the main project root and shared. If false, the hooks directory is copied to the new worktree.

### `ports`

Base port assignments for the project. `zgt` uses these to calculate unique ports for each worktree based on its index.

```yaml
ports:
  api: 8080
  web: 3000
```

### `env`

Custom environment variables that can include placeholders. These are exported via `zgt env`.

```yaml
env:
  COMPOSE_PROJECT_NAME: "zgt-{{.Repo}}"
```

### `tmux`

Tmux integration settings.

- `enabled` (bool): Enable automatic tmux window/pane creation.
- `keep_open` (bool): If true, do not close the tmux window on `zgt rm`. Default is `false` (closes by default).
- `window_name` (string): Template for the tmux window name.
- `panes` (list): List of pane configurations.
  - `id`: Unique identifier for the pane.
  - `target`: ID of the pane to split.
  - `split`: `horizontal` or `vertical`.
  - `size`: Percentage or number of lines/columns.
  - `commands`: List of commands to run in the pane.

## Placeholders

Placeholders can be used in `hooks`, `env`, and `tmux.window_name`.

| Placeholder          | Description                              |
| :------------------- | :--------------------------------------- |
| `{{.Path}}`          | Absolute path of the worktree directory. |
| `{{.Repo}}`          | Name of the main project root directory. |
| `{{.CurrentDir}}`    | Current working directory name.          |
| `{{.Branch}}`        | Target branch name.                      |
| `{{.TargetBranch}}`  | Alias for `{{.Branch}}`.                 |
| `{{.CurrentBranch}}` | Branch name where `zgt` was executed.    |

## Inspecting Configuration (`config`)

Use `zgt config` to view the final merged configuration.

- `--check`: Validate configuration for errors.
- `--raw`: Display configuration without placeholder replacement.
