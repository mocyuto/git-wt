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
- **ポート管理**: 各ワークツリーに一意のポートインデックスを自動割り当てし、衝突を防止
- **カスタムフック**: ワークツリーの作成（`add`）や削除（`rm`）に連動して、複数のシェルコマンドを自然に実行

## Badges

[![Release](https://img.shields.io/github/release/mocyuto/git-wt.svg?style=for-the-badge)](https://github.com/mocyuto/git-wt/releases/latest)
[![Software License](https://img.shields.io/badge/license-MIT-brightgreen.svg?style=for-the-badge)](/LICENSE.md)
[![Build status](https://img.shields.io/github/actions/workflow/status/mocyuto/git-wt/ci.yml?style=for-the-badge&branch=main)](https://github.com/mocyuto/git-wt/actions?workflow=ci)

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

## 開発フローと挙動

`git-wt` は単なるコマンドのラッパーではなく、開発者のコンテキスト切り替えをシームレスにすることを目的としています。

### 1. 新しい機能の開発を始める (`add`)

`git worktree add` を実行すると、通常は `.env` などの設定ファイルが欠落した状態で新しいディレクトリが作成されます。`git-wt` は以下の手順を自動化します：

1. **ワークツリーの作成**: 指定されたパスにディレクトリを作成し、ブランチをチェックアウトします。
2. **自動パス生成**: ブランチ名だけを指定すれば、`../{プロジェクト名}-{ブランチ名}` に自動で配置されます。
3. **設定の同期**: 元のディレクトリにある「無視されているファイル（`.env` など）」を特定し、構造を維持したままコピーします。
4. **ポートの割り当て**: そのワークツリー専用の「ポートインデックス」を予約します。
5. **自動セットアップ**: フック（`hooks.add`）に `npm install` などを設定しておけば、依存関係の解決まで自動で行われます。

```bash
# feature-login ブランチを新しいワークツリーで開始
git-wt add feature-login
```

### 2. 複数プロジェクトを同時に動かす (`env` / `ports`)

複数のワークツリーでサーバーを同時に立ち上げる際、ポートの衝突が問題になります。`git-wt` はこれを解決します。

- **挙動**: 各ワークツリーには `0, 1, 2...` とインデックスが振られます。
- **活用**: 設定ファイルで `api: 8080` と定義しておけば、`git-wt env` はそのワークツリーに応じたポート（`8080`, `8081`...）を環境変数として出力します。

```bash
# ワークツリーに移動して環境変数を読み込む
cd ../my-project-feature-login
eval $(git-wt env)

# これで $API_PORT が 8081 など、他のワークツリーと被らない値になります
npm start
```

### 3. 作業が終わったら片付ける (`remove`)

作業が完了したワークツリーの削除も一括で行えます。

- **挙動**: ワークツリーのディレクトリを削除すると同時に、関連するローカルブランチも削除します（オプションで変更可能）。
- **クリーンアップ**: ポートインデックスも返却され、再利用可能になります。削除後にフック（`hooks.rm`）を実行することも可能です。

```bash
# 作業完了。ディレクトリとブランチを削除
git-wt rm feature-login
```

### 4. 割り当て状況を確認する (`list` / `ports`)

- `list`: どのワークツリーがどのブランチで動いているか、GitHub PR の状態や未コミットの変更があるかを表示します。
- `ports`: 現在どのパスにどのインデックスが割り当てられているかを確認できます。

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

ports:
  api: 8080
  web: 3000
```

### ポート管理の仕組み

`git-wt add` でワークツリーを追加すると、一意の `PortIndex`（0から始まる整数）が自動的に割り当てられます。
`git-wt env` を呼び出すと、設定ファイルの `ports` セクションに基づき、`UPPER_NAME_PORT = ベースポート + PortIndex` という形式で環境変数を生成します。

例（上記設定で `PortIndex: 1` の場合）:

- `API_PORT=8081`
- `WEB_PORT=3001`

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

| プレースホルダー | 説明                                 |
| :--------------- | :----------------------------------- |
| `{{.Path}}`      | ワークツリーディレクトリの絶対パス   |
| `{{.Branch}}`    | ブランチ名                           |
| `{{.Repo}}`      | レポジトリ名（ベースディレクトリ名） |

#### 補足

- 単一のコマンドのみ実行する場合は、リストではなく文字列で指定可能です： `add: "echo hello"`
- コマンドは `/bin/sh -c` を介して実行されるため、パイプやステータスチェックも利用可能です。

## 注意事項

- コピー対象は、実行時のワークツリールートにおいて `git ls-files --others --ignored --exclude-standard` でリストアップされるファイルです。
- 実行には `git` と　`gh` がインストールされている必要があります。

## 開発・テスト

```bash
# テストの実行
go test -v .
```
