# `git-wt` から `zgt` への移行ガイド

ツール名が `git-wt` から `zgt` に変更されました。このドキュメントでは移行手順を説明します。

## 1. バイナリ名

実行ファイル名が `zgt` に変更されました。

- **旧:** `git-wt`
- **新:** `zgt`

古い名前を使用しているスクリプトやエイリアスを更新してください。

## 2. 設定ファイル

### グローバル設定

グローバル設定のディレクトリが変更されました。

- **旧:** `~/.config/git-wt/`
- **新:** `~/.config/zgt/`

ファイル名は引き続き `config.yaml` (または `config.yml`) です。

#### 移行手順:

```bash
mkdir -p ~/.config/zgt
cp ~/.config/git-wt/config.yaml ~/.config/zgt/config.yaml
```

### ローカル設定

ローカル設定のファイル名が変更されました。

- **旧:** `git-wt.config.yaml` / `git-wt.config.yml`
- **新:** `zgt.config.yaml` / `zgt.config.yml`

#### 移行手順:

プロジェクトのルートディレクトリで、ファイルをリネームしてください。

```bash
mv git-wt.config.yaml zgt.config.yaml
# または
mv git-wt.config.yml zgt.config.yml
```

## 3. 環境変数

`eval $(zgt env)` を使用している場合、生成される環境変数は以前と同じですが、コマンド名が変更されました。

- **旧:** `eval $(git-wt env)`
- **新:** `eval $(zgt env)`
