# git-wt

`git worktree` を作成する際、設定ファイル（`.env` など）をメインツリーから自動的にコピーする CLI ツールです。

## 概要

Git の `worktree` 機能は便利ですが、`.gitignore` で除外されている `.env` やローカルの設定ファイルなどは新しいワークツリーに引き継がれません。`git-wt` を使うことで、これらを自動的にコピーし、すぐに開発やテストが可能なワークツリーを作成できます。

## 特徴

- `git worktree add` のラッパーとして動作
- `.gitignore` に指定された「無視されているファイル」を自動的に特定してコピー
- ディレクトリ構造を維持したままコピー（例: `node_modules` 内の特定ファイルなど）
- Go 言語による実装のため、高速かつ単一バイナリで動作

## インストール

### リポジトリのクローンとビルド

```bash
git clone <repository-url>
cd git-wt
go build -o git-wt main.go
```

### Git サブコマンドとしての登録

バイナリを `PATH` の通った場所に `git-wt` という名前で配置すると、`git wt` として呼び出せるようになります。

```bash
mv git-wt /usr/local/bin/git-wt
```

## 使い方

### 基本的な使い方

既存のブランチを指定してワークツリーを作成します。

```bash
git wt <path> <branch>
```

例:
```bash
git wt ../debug-fix main
```

### 新規ブランチの作成

`-b` オプションを使用して、新しいブランチを作成しながらワークツリーをセットアップします。

```bash
git wt -b <new-branch> <path>
```

例:
```bash
git wt -b feature/login ../feature-login
```

## 動作の仕組み

1. `git worktree add` を実行し、指定されたパスにワークツリーを作成します。
2. `git ls-files --others --ignored --exclude-standard` を実行し、現在のプロジェクトで無視されているファイルのリストを取得します。
3. リストアップされたファイルを、新しいワークツリーの対応する場所にコピーします。
