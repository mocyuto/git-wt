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

## Installation

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

Providing only a branch name will automatically create a worktree at `../{current_dir}-{branch}`.

```bash
git-wt <branch>
```

**Example (if project is `pj`):**
```bash
git-wt feature-abc
# -> Created at ../pj-feature-abc
```

### 2. Create with Explicit Path

```bash
git-wt <path> <branch>
```

**Example:**
```bash
git-wt ../debug-fix main
```

### 3. Setup with New Branch

```bash
git-wt -b <new-branch> <path>
# or
git-wt <path> -b <new-branch>
```

### 4. List Worktrees

```bash
git-wt list
# or
git-wt ls
# or
git-wt -l
```

### 5. Remove Worktree

Remove a worktree by providing the branch name (or path).

```bash
git-wt remove <branch>
# or
git-wt rm <branch>
```

**Force Removal (if there are changes):**
```bash
git-wt rm -f <branch>
```

## Requirements

- `git` must be installed and available in your environment.

## Development & Testing

```bash
# Run tests
go test -v .
```
