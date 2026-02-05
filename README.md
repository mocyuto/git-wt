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

## インストール

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

### 1. 既存のブランチを指定して作成

```bash
git-wt <path> <branch>
```

**例:**
```bash
git-wt ../debug-fix main
```

### 2. 新規ブランチを作成してセットアップ

```bash
git-wt -b <new-branch> <path>
# または
git-wt <path> -b <new-branch>
```

**例:**
```bash
git-wt -b feature/login ../feature-login
```

### 3. オプションなし（カレントブランチをベースに作成）

```bash
git-wt <path>
```

### 4. ワークツリーの一覧表示

```bash
git-wt list
# または
git-wt ls
# または
git-wt -l
```

## 注意事項

- コピー対象は、実行時のワークツリールートにおいて `git ls-files --others --ignored --exclude-standard` でリストアップされるファイルです。
- 実行には `git` がインストールされている必要があります。

## 開発・テスト

```bash
# テストの実行
go test -v .
```
