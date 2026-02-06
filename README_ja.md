# git-wt

`git worktree` を作成する際、設定ファイル（`.env` など）を自動的にコピーして新しいディレクトリを作成する CLI ツールです。

## 概要

Git の `worktree` 機能は便利ですが、`.gitignore` で除外されている `.env` やローカルの設定ファイルなどは、新しく作成したワークツリーには含まれません。`git-wt` を使うことで、これらを自動的にコピーし、すぐに開発やテストが可能なワークツリーを作成できます。

## 特徴

- `git worktree add` のラッパーとして動作
- `.gitignore` に指定された「無視されているファイル」を自動的に特定してコピー
- ディレクトリ構造を維持したままコピー（例: `node_modules` 内の設定ファイルなど）
- Cobra フレームワークによる柔軟なフラグ指定
- Go 言語による単一バイナリでの動作
- ブランチ名に基づくパスの自動生成（`../{プロジェクト名}-{ブランチ名}`）
- 一覧表示（`list` / `ls`）および削除（`remove` / `rm`）機能のサポート
- **カスタムフック**: ワークツリーの作成（`add`）や削除（`rm`）に連動して、複数のシェルコマンドを自然に実行

## インストール

### Homebrew

macOS での最も簡単なインストール方法は Homebrew を使用することです:

```bash
brew install mocyuto/tap/git-wt
```

### ビルド

```bash
go build -o git-wt main.go
```

### パスの設定

バイナリを `PATH` の通った場所に配置して使用します。

```bash
# mac / linux の例
sudo mv git-wt /usr/local/bin/git-wt
```

> [!TIP]
> `PATH` 内に `git-wt` という名前で配置すると、`git wt ...` という形式で呼び出すことも可能です。

## 使い方

Cobra を使用しているため、フラグ（`-b`）は引数の前後のどちらにでも記述可能です。

### 1. ブランチ名のみ指定して作成（パス自動生成）

ブランチ名のみを指定すると、自動的に `../{現在のディレクトリ名}-{ブランチ名}` というパスにワークツリーを作成します。ブランチが存在しない場合は自動的に作成されます。

```bash
git-wt add <branch>
```

**例（プロジェクト名が `pj` の場合）:**
```bash
git-wt add feature-abc
# -> ../pj-feature-abc に作成されます
```

### 2. パスを明示的に指定して作成

```bash
git-wt add <path> <branch>
```

**例:**
```bash
git-wt add ../debug-fix main
```

### 3. 新規ブランチを作成してセットアップ

```bash
git-wt add -b <new-branch> <path>
# または
git-wt add <path> -b <new-branch>
```

**例:**
```bash
git-wt add -b feature/login ../feature-login
```

### 4. ワークツリーの一覧表示

すべてのワークツリーをパス、コミットハッシュ、ブランチ名とともに表示します。`gh` CLI が利用可能な場合は、GitHub PR のステータスも表示されます。

```bash
git-wt list
# または
git-wt ls
```

### 5. ワークツリーの削除

ブランチ名（またはパス）を指定してワークツリーを削除します。デフォルトでは、関連するブランチも削除されます。

```bash
git-wt remove <branch>
# または
git-wt rm <branch>
```

**例:**
```bash
git-wt rm feature/login
```

**強制削除（未コミットの変更がある場合）:**
```bash
git-wt rm -f <branch>
```

**ブランチを残す（削除しない）:**
```bash
git-wt rm -k <branch>
# または
git-wt rm --keep-branch <branch>
```

## 設定

`git-wt` は以下の優先順位で設定ファイルを読み込みます（YAML形式）。

1. プロジェクトごとの設定（プロジェクトルートの `git-wt.config.yaml` または `.yml`）
2. グローバル設定（`~/.config/git-wt/config.yaml`）
3. `--config` フラグで明示的に指定されたパス

### プロジェクトごとの設定

プロジェクトのルートディレクトリに `git-wt.config.yaml`（または `.yml`）を作成することで、そのプロジェクト固有の設定を定義できます。`hooks` や `ignore` の設定は、グローバル設定に**追加（マージ）**されます。

```yaml
# git-wt.config.yaml
ignore:
  - "*.tmp"
  - "local-debug.log"

hooks:
  add:
    - "npm install"
```

### カスタム無視パターン

コピー時に除外したいファイルパターンを `ignore` セクションで指定できます。

```yaml
ignore:
  - ".env.production"
  - "secrets/*"
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

| プレースホルダー | 説明 |
| :--- | :--- |
| `{{.Path}}` | ワークツリーディレクトリの絶対パス |
| `{{.Branch}}` | ブランチ名 |
| `{{.Repo}}` | レポジトリ名（ベースディレクトリ名） |

#### 補足

- 単一のコマンドのみ実行する場合は、リストではなく文字列で指定可能です： `add: "echo hello"`
- コマンドは `/bin/sh -c` を介して実行されるため、パイプやステータスチェックも利用可能です。

## 注意事項

- コピー対象は、実行時のワークツリールートにおいて `git ls-files --others --ignored --exclude-standard` でリストアップされるファイルです。
- 実行には `git` がインストールされている必要があります。

## 開発・テスト

```bash
# テストの実行
go test -v .
```
