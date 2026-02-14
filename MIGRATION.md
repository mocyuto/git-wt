# Migration from `git-wt` to `zgt`

The tool has been renamed from `git-wt` to `zgt`. This document guides you through the migration process.

## 1. Binary Name

The executable is now named `zgt`.

- **Old:** `git-wt`
- **New:** `zgt`

You should update any scripts or aliases that use the old name.

## 2. Configuration Files

### Global Configuration

The global configuration directory has moved.

- **Old:** `~/.config/git-wt/`
- **New:** `~/.config/zgt/`

The configuration file name remains `config.yaml` (or `config.yml`).

#### Migration Step:

```bash
mkdir -p ~/.config/zgt
cp ~/.config/git-wt/config.yaml ~/.config/zgt/config.yaml
```

### Local Configuration

The local configuration file name has changed.

- **Old:** `git-wt.config.yaml` / `git-wt.config.yml`
- **New:** `zgt.config.yaml` / `zgt.config.yml`

#### Migration Step:

In your project root, rename the file:

```bash
mv git-wt.config.yaml zgt.config.yaml
# or
mv git-wt.config.yml zgt.config.yml
```

## 3. Environment Variables

If you are using `eval $(zgt env)`, the environment variables generated are the same, but the command to generate them has changed.

- **Old:** `eval $(git-wt env)`
- **New:** `eval $(zgt env)`
