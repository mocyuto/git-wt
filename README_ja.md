# zgt （旧称 `git-wt`）

`git worktree` を作成する際、設定ファイル（`.env` など）を自動的にコピーして新しいディレクトリを作成する CLI ツールです。

## 概要

- [移行ガイド (`git-wt` からの移行)](./MIGRATION_ja.md)

Git の `worktree` 機能は便利ですが、`.gitignore` で除外されている `.env` やローカルの設定ファイルなどは、新しく作成したワークツリーには含まれません。`zgt` を使うことで、これらを自動的にコピーし、すぐに開発やテストが可能なワークツリーを作成できます。

## 特徴

- `git worktree add` のラッパーとして動作
- `.gitignore` に指定された「無視されているファイル」を自動的に特定してコピー
- ディレクトリ構造を維持したままコピー（例: `node_modules` 内の設定ファイルなど）
- Cobra フレームワークによる柔軟なフラグ指定
- Go 言語による単一バイナリでの動作
- ブランチ名に基づくパスの自動生成（リポジトリルートと同じ階層に `{プロジェクト名}-{ブランチ名}`）
- 一覧表示（`list` / `ls`）および削除（`remove` / `rm`）機能のサポート
- **ポート管理**: 各ワークツリーに一意のポートインデックスを自動割り当てし、衝突を防止
- **カスタムフック**: ワークツリーの作成（`add`）や削除（`rm`）に連動して、複数のシェルコマンドを実行
- **設定の可視化**: `config` サブコマンドで最終的なマージされた設定を確認したり、構文チェックを行ったりできます。
- **双方向の同期**: ワークツリーで編集した無視ファイルをルートプロジェクトに同期できる `sync` サブコマンド
- **Agent Skill**: AI エージェント向けの専門スキルを配布・インストール可能

## Badges

[![Release](https://img.shields.io/github/release/mocyuto/zgt.svg?style=for-the-badge)](https://github.com/mocyuto/zgt/releases/latest)
[![Software License](https://img.shields.io/badge/license-MIT-brightgreen.svg?style=for-the-badge)](/LICENSE.md)
[![Build status](https://img.shields.io/github/actions/workflow/status/mocyuto/zgt/ci.yml?style=for-the-badge&branch=main)](https://github.com/mocyuto/zgt/actions?workflow=ci)

## インストール

### Homebrew

macOS での最も簡単なインストール方法は Homebrew を使用することです:

```bash
brew install mocyuto/tap/zgt
```

### ビルド

```bash
go build -o zgt .
```

### パスの設定

バイナリを `PATH` の通った場所に配置して使用します。

```bash
# mac / linux の例
sudo mv zgt /usr/local/bin/zgt
```

## 開発フローと挙動

`zgt` は単なるコマンドのラッパーではなく、開発者のコンテキスト切り替えをシームレスにすることを目的としています。

### 1. 新しい機能の開発を始める (`add`)

`git worktree add` を実行すると、通常は `.env` などの設定ファイルが欠落した状態で新しいディレクトリが作成されます。`zgt` は以下の手順を自動化します：

1. **ワークツリーの作成**: 指定されたパスにディレクトリを作成し、ブランチをチェックアウトします。
2. **自動パス生成**: ブランチ名だけを指定すれば、リポジトリルートと同じ階層に `{プロジェクト名}-{ブランチ名}` として自動で配置されます。
3. **設定の同期**: 元のディレクトリにある「無視されているファイル（`.env` など）」を特定し、構造を維持したままコピーします。
4. **ポートの割り当て**: そのワークツリー専用の「ポートインデックス」を予約します。
5. **自動セットアップ**: フック（`hooks.add`）に `npm install` などを設定しておけば、依存関係の解決まで自動で行われます。

```bash
# feature-login ブランチを新しいワークツリーで開始
zgt add feature-login
```

### 2. 複数プロジェクトを同時に動かす (`env` / `ports`)

複数のワークツリーでサーバーを同時に立ち上げる際、ポートの衝突が問題になります。`zgt` はこれを解決します。

- **挙動**: 各ワークツリーには `0, 1, 2...` とインデックスが振られます。
- **動的な計算**: ポート番号は、そのワークツリーのプロジェクトルートにある `zgt.config.yml` の `ports` 設定（ベースポート）にインデックスを加算して算出されます。これにより、プロジェクトごとに異なるベースポートを設定していても、正しく計算されます。
- **活用**: 設定ファイルで `api: 8080` と定義しておけば、`zgt env` はそのワークツリーに応じたポート（`8080`, `8081`...）を環境変数として出力します。

```bash
# ワークツリーに移動して環境変数を読み込む
cd ../my-project-feature-login
eval "$(zgt env)"

# これで $API_PORT が 8081 など、他のワークツリーと被らない値になります
npm start
```

### 3. 作業が終わったら片付ける (`remove`)

作業が完了したワークツリーの削除も一括で行えます。

- **挙動**: ワークツリーのディレクトリを削除すると同時に、関連するローカルブランチも削除します（オプションで変更可能）。
- **クリーンアップ**: ポートインデックスも返却され、再利用可能になります。削除後にフック（`hooks.rm`）を実行することも可能です。

```bash
# 作業完了。ディレクトリとブランチを削除
zgt rm feature-login
```

### 4. 無視ファイルの同期 (`sync`)

ワークツリー内で `.env` などの設定ファイルを編集し、その内容をメインプロジェクトのルートディレクトリに反映させたい場合：

- **対話モード**: `zgt sync` を実行すると TUI（`rivo/tview` を利用）が起動し、同期したいファイルを個別に選択できます。
- **一括同期**: `-a` / `--all`（または `--force`）フラグを使用すると、すべての無視ファイルを即座に同期します。

```bash
# 変更内容を選択してルートに同期
zgt sync
```

### 5. 割り当て状況を確認する (`list` / `ports`)

- `list`: どのワークツリーがどのブランチで動いているか、GitHub PR の状態や未コミットの変更があるか（`[DIRTY]`）を表示します。割り当てられているポート番号も一覧できます。
- `ports`: 現在どのパスにどのインデックス・ポート番号が割り当てられているかを確認できます。`-a` / `--all` フラグを使うと、現在のプロジェクトだけでなく、すべてのプロジェクトの割り当て状況を表示します。
- `ports update`: 現在のプロジェクトのポート割当を最新の設定に基づいて更新します。不足している割当の追加や、設定から消えた割当の削除を自動で行います。
- `config`: マージされた最終的な設定（グローバル + プロジェクト + フラグ）をYAML形式で表示します。`--check` フラグを使うことで設定ファイルの構文エラーをチェックでき、`--raw` フラグを使うことでプレースホルダー置換をスキップできます。
- `config edit`: システムエディタを使用して設定ファイルを編集します。デフォルトでローカルプロジェクトの設定を編集します。
- `version`: `zgt` のバージョン番号を表示します。
- `skill install`: カレントリポジトリの `skills/` ディレクトリ内のスキルを、グローバルなエージェントスキルディレクトリ (`~/.claude/skills/`) にインストールします。

## 設定

`zgt` は以下の優先順位で設定ファイルを読み込みます（YAML形式）。

1. プロジェクトごとの設定（プロジェクトルートの `zgt.config.yaml` または `.yml`）
2. グローバル設定（`~/.config/zgt/config.yaml`）
3. `--config` フラグで明示的に指定されたパス

### 設定ファイルの生成 (`init`)

プロジェクトのルートディレクトリで以下のコマンドを実行することで、デフォルトの設定ファイル（`zgt.config.yml`）を生成できます。

```bash
zgt init
```

このコマンドは以下の動作を行います：

1.  ポート管理や環境変数のテンプレート、サンプルフックを含む設定ファイル（`zgt.config.yml`）を作成します。
2.  `zgt.config.yml` および `zgt.config.yaml` を自動的に `.gitignore` ファイルに追記し、誤ってコミットされないようにします。

すでに `zgt.config.yml` または `zgt.config.yaml` が存在する場合は、上書きを防止するためにスキップされます。

### 設定の編集 (`config edit`)

CLI から直接、好みのエディタ（`$EDITOR` または `vi`）を使用して設定ファイルを編集できます。

```bash
# ローカル設定を編集
zgt config edit --local

# グローバル設定を編集
zgt config edit --global
```

保存前に YAML の構文チェックが自動的に行われます。

### 設定 (Configuration)

`zgt` はグローバル (`~/.config/zgt/config.yaml`) またはローカル (プロジェクトのルートにある `zgt.config.yml`) で設定可能です。

```yaml
# 利用可能な設定オプションの例
add:
  from_default: true # 新しい worktree を常にデフォルトブランチ (例: main) から作成する
  auto_pull: true # worktree 作成前にデフォルトブランチの更新を pull する

ignore:
  - "*.tmp"
  - "local-debug.log"
  - ".env"

hooks:
  add:
    - "npm install"

ports:
  api: 8080
  web: 3000

tmux:
  enabled: true
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

### カスタム環境変数

`env` セクションで独自の環境変数を定義できます。これらの値にはプレースホルダーを使用でき、`zgt env` を介して自動的にエクスポートされます。

```yaml
env:
  COMPOSE_PROJECT_NAME: "zgt-{{.Repo}}"
  DEBUG: "true"
```

実行例：

```bash
eval "$(zgt env)"
echo $COMPOSE_PROJECT_NAME # zgt-myrepo
```

これらの環境変数は、[カスタムフック](#カスタムフック) の実行時にも適用されます。

### カスタム無視パターン

コピー時に除外したいファイルパターンを `ignore` セクションで指定できます。

```yaml
ignore:
  - ".env.production"
  - "secrets/*"
```

### 削除後のデフォルトブランチPULL

ワークツリー削除後に、リモートからデフォルトブランチを自動的にプルするには、`rm` フックに git コマンドを追加します。

```yaml
hooks:
  rm:
    - "git pull origin main:main"
```

### カスタムフック

フックを使用すると、ワークツリーの管理時にシェルコマンドを自動実行できます。

```yaml
hooks:
  # 'add'（作成）後に実行するコマンド
  add:
    - "tmux new-window -n [{{.Repo}}]{{.Branch}} -c {{.Path}}"
    - "echo 'Welcome to {{.Repo}}'"
  # 'remove'（削除）後に実行するコマンド
  rm:
    - "echo 'Cleanup for {{.Branch}}'"
```

#### 使用可能なプレースホルダー

プレースホルダーは `hooks`、`env`、および `tmux.window_name` で利用可能です。

| プレースホルダー     | 説明                                             |
| :------------------- | :----------------------------------------------- |
| `{{.Path}}`          | ワークツリーディレクトリの絶対パス               |
| `{{.Repo}}`          | メインプロジェクトのルートディレクトリ名         |
| `{{.CurrentDir}}`    | 現在の作業ディレクトリ名                         |
| `{{.Branch}}`        | 対象ブランチ名（引数で指定されたもの）           |
| `{{.TargetBranch}}`  | `{{.Branch}}` のエイリアス                       |
| `{{.CurrentBranch}}` | `zgt` 実行時にチェックアウトされていたブランチ名 |

### Tmux 連携

`zgt` は、`add` を実行したときに自動的に tmux のウィンドウを作成し、複数のペインに分割してコマンドを実行することができます。`id` を指定することで、特定のペインをターゲットにして分割することが可能です。

```yaml
tmux:
  enabled: true
  panes:
    - id: main
      commands: ["yarn"]
    - id: side
      target: main # main を左右に分割
      split: horizontal
      size: 50%
      commands: ["yarn dev"]
    - target: side # side を上下に分割
      split: vertical
      commands: ["yarn watch"]
    - target: main # main を上下に分割
      split: vertical
      commands: ["tail -f logs/app.log"]
```

#### ペインのプロパティ

| プロパティ    | 説明                                                               |
| :------------ | :----------------------------------------------------------------- |
| `window_name` | (任意) tmux ウィンドウ名のプレースホルダー。                       |
| `id`          | (任意) `target` から参照するための、ペインの一意識別子。           |
| `target`      | (任意) 分割対象とするペインの ID。省略時は最後に作成されたペイン。 |
| `commands`    | そのペインで実行するコマンドのリスト。                             |
| `split`       | 分割方向: `horizontal` (h) または `vertical` (v)。                 |
| `size`        | ペインのサイズ (例: `20%` でパーセント指定、`20` で行/列数指定)。  |

`enabled` が `true` の場合、`zgt` は以下の動作を行います：

1. `window_name` で指定された名前（デフォルトは `[repo]branch`）の新しい tmux ウィンドウを作成します。
2. `panes` のリストに従って分割を作成します。各分割は指定された `target` または直前に作成されたペインを対象とします。
3. 各ペインで `commands` を実行し、終了後もシェルを開いたままにします。

#### 注意事項

- `tmux` がインストールされており、tmux セッションが実行中である必要があります。

#### 補足

- 単一のコマンドのみ実行する場合は、リストではなく文字列で指定可能です： `add: "echo hello"`
- コマンドは `/bin/sh -c` を介して実行されるため、パイプやステータスチェックも利用可能です。

## Agent Skill の管理

`zgt` は、AI エージェント向けの「エキスパートスキル」を管理する機能を提供します。スキルとは、エージェントの能力を拡張するための指示書やリソースのセットです。

### スキルのインストール (`skill install`)

カレントリポジトリにあるスキルをインストールして、AI コラボレーター（Claude など）が利用できるようにします。

```bash
zgt skill install
```

実行すると、以下の 4 つのインストール先からインタラクティブに（チェックボックス形式で）選択できます。

- **Local .claude**: `./.claude/skills/`
- **Local .agent**: `./.agent/skills/`
- **Global .claude**: `~/.claude/skills/`
- **Global .agent**: `~/.agent/skills/`

`-a` または `--all` フラグを使用すると、すべてのターゲットに即座にインストールされます。

## 注意事項

- コピー対象は、実行時のワークツリールートにおいて `git ls-files --others --ignored --exclude-standard` でリストアップされるファイルです。
- 実行には `git` と　`gh` がインストールされている必要があります。

## 開発・テスト

```bash
# テストの実行
go test -v ./...
```
