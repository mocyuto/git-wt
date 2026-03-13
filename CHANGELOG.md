# Changelog

## [v0.11.1](https://github.com/mocyuto/zgt/compare/v0.11.0...v0.11.1) - 2026-03-13
- feat: add autostash support for autopull in add command by @mocyuto in https://github.com/mocyuto/zgt/pull/61
- fix: safely update base branch and refactor AutoPull logic by @mocyuto in https://github.com/mocyuto/zgt/pull/63

## [v0.11.0](https://github.com/mocyuto/zgt/compare/v0.10.1...v0.11.0) - 2026-03-01
- feat: add FAQ and --path option to zgt add by @mocyuto in https://github.com/mocyuto/zgt/pull/57
- feat: print merged configuration when --verbose is specified by @mocyuto in https://github.com/mocyuto/zgt/pull/59
- feat: add tmux graceful shutdown (default true) and close command by @mocyuto in https://github.com/mocyuto/zgt/pull/56

## [v0.10.1](https://github.com/mocyuto/zgt/compare/v0.10.0...v0.10.1) - 2026-03-01
- fix README by @mocyuto in https://github.com/mocyuto/zgt/pull/52
- Change tmux command execution timing to post-shell startup by @mocyuto in https://github.com/mocyuto/zgt/pull/54

## [v0.10.0](https://github.com/mocyuto/zgt/compare/v0.9.2...v0.10.0) - 2026-02-27
- feat: implement hostname-safe placeholder function by @mocyuto in https://github.com/mocyuto/zgt/pull/50

## [v0.9.2](https://github.com/mocyuto/zgt/compare/v0.9.1...v0.9.2) - 2026-02-24
- Rename .agent to .agents and fix skill installation permissions by @mocyuto in https://github.com/mocyuto/zgt/pull/48

## [v0.9.1](https://github.com/mocyuto/zgt/compare/v0.9.0...v0.9.1) - 2026-02-24
- feat(skill): support embedded skill installation for cross-project use by @mocyuto in https://github.com/mocyuto/zgt/pull/46

## [v0.9.0](https://github.com/mocyuto/zgt/compare/v0.8.1...v0.9.0) - 2026-02-24
- feat: hierarchical tmux ls display and interactive open command by @mocyuto in https://github.com/mocyuto/zgt/pull/43
- Implement Agent Skill plugin system and installation command by @mocyuto in https://github.com/mocyuto/zgt/pull/44

## [v0.8.1](https://github.com/mocyuto/zgt/compare/v0.8.0...v0.8.1) - 2026-02-18
- Enhance config command: add --raw flag and change edit default to local by @mocyuto in https://github.com/mocyuto/zgt/pull/41

## [v0.8.0](https://github.com/mocyuto/zgt/compare/v0.7.0...v0.8.0) - 2026-02-17
- feat: support tmux window name configuration by @mocyuto in https://github.com/mocyuto/zgt/pull/37
- feat: Expand and clarify placeholders, centralize context creation by @mocyuto in https://github.com/mocyuto/zgt/pull/39

## [v0.7.0](https://github.com/mocyuto/zgt/compare/v0.6.0...v0.7.0) - 2026-02-16
- fix: release workflow trigger and tagpr token usage by @mocyuto in https://github.com/mocyuto/zgt/pull/33
- docs: add example for pulling default branch and CI updates by @mocyuto in https://github.com/mocyuto/zgt/pull/35
- Refactor CreateWorktree and add support for creating worktrees from default branch by @mocyuto in https://github.com/mocyuto/zgt/pull/36

## [v0.6.0](https://github.com/mocyuto/zgt/compare/v0.5.1...v0.6.0) - 2026-02-14
- feat: change default worktree path to be adjacent to repository root by @mocyuto in https://github.com/mocyuto/zgt/pull/28
- fix(ci): avoid detached HEAD in tagpr workflow by @mocyuto in https://github.com/mocyuto/zgt/pull/30
- feat: multi-pane tmux integration support by @mocyuto in https://github.com/mocyuto/zgt/pull/31
- fix: ensure tagpr runs on main branch for labeled events by @mocyuto in https://github.com/mocyuto/zgt/pull/32

## [v0.5.1](https://github.com/mocyuto/zgt/compare/v0.5.0...v0.5.1) - 2026-02-14
- Use GitHub App token for tagpr workflow by @mocyuto in https://github.com/mocyuto/zgt/pull/24
- docs: add migration guide and fix tagpr checkout issue by @mocyuto in https://github.com/mocyuto/zgt/pull/26
- Consolidate config paths and refactor Git root discovery by @mocyuto in https://github.com/mocyuto/zgt/pull/27

## [v0.5.0](https://github.com/mocyuto/zgt/compare/v0.4.0...v0.5.0) - 2026-02-14
- Rename CLI tool from git-wt to zgt by @mocyuto in https://github.com/mocyuto/zgt/pull/21
- feat: add config edit command and update README branding by @mocyuto in https://github.com/mocyuto/zgt/pull/22
- Rename project to zgt and update init command to handle .gitignore by @mocyuto in https://github.com/mocyuto/zgt/pull/23

## [v0.4.0](https://github.com/mocyuto/zgt/compare/v0.3.0...v0.4.0) - 2026-02-13
- add version command by @mocyuto in https://github.com/mocyuto/zgt/pull/16
- feat: add ports update command and refine port key display by @mocyuto in https://github.com/mocyuto/zgt/pull/18
- Implement YAML configuration validation and config --check option by @mocyuto in https://github.com/mocyuto/zgt/pull/19

## [v0.3.0](https://github.com/mocyuto/zgt/compare/v0.2.2...v0.3.0) - 2026-02-11
- Add init command and improve add command help by @mocyuto in https://github.com/mocyuto/zgt/pull/12
- fix(cmd): ensure list command displays the latest PR status by @mocyuto in https://github.com/mocyuto/zgt/pull/13
- feat: Add sync command and refactor git package by @mocyuto in https://github.com/mocyuto/zgt/pull/14
- docs: add init command description to README by @mocyuto in https://github.com/mocyuto/zgt/pull/15

## [v0.2.2](https://github.com/mocyuto/zgt/compare/v0.2.1...v0.2.2) - 2026-02-10
- Fix zgt env shell evaluation and improve port auto-assignment by @mocyuto in https://github.com/mocyuto/zgt/pull/11

## [v0.2.1](https://github.com/mocyuto/zgt/compare/v0.2.0...v0.2.1) - 2026-02-10
- feat: add branch name completion for remove command by @mocyuto in https://github.com/mocyuto/zgt/pull/8
- fix: handle empty global config to avoid EOF warning by @mocyuto in https://github.com/mocyuto/zgt/pull/9
- fix: normalize paths to resolve symlink mismatches by @mocyuto in https://github.com/mocyuto/zgt/pull/10

## [v0.2.0](https://github.com/mocyuto/zgt/compare/v0.1.0...v0.2.0) - 2026-02-09
- feat: add env cmd and config cmd by @mocyuto in https://github.com/mocyuto/zgt/pull/5
- feat: A new logger package by @mocyuto in https://github.com/mocyuto/zgt/pull/6
- feat: update port management by @mocyuto in https://github.com/mocyuto/zgt/pull/7

## [v0.1.0](https://github.com/mocyuto/zgt/compare/v0.0.3...v0.1.0) - 2026-02-08
- feat: add port and env command  by @mocyuto in https://github.com/mocyuto/zgt/pull/4

## [v0.0.3](https://github.com/mocyuto/zgt/compare/v0.0.2...v0.0.3) - 2026-02-06
- show PR list by @mocyuto in https://github.com/mocyuto/zgt/pull/1
- リファクタリングと機能追加 by @mocyuto in https://github.com/mocyuto/zgt/pull/2
- fix ignore pattern and show local diffs by @mocyuto in https://github.com/mocyuto/zgt/pull/3

## [v0.0.2](https://github.com/mocyuto/zgt/compare/v0.0.1...v0.0.2) - 2026-02-05

## [v0.0.1](https://github.com/mocyuto/zgt/commits/v0.0.1) - 2026-02-05
