# discord-runner
🌐 [English](../README.md)

Discord のスラッシュコマンドから**ローカルのコマンドを実行する**ボットです。
設定ファイルに定義したスクリプトごとにスラッシュコマンドが登録され、Discord 上から `ls` や `git status` などのコマンドを走らせ、その出力をメッセージとして受け取ることができます。

## 特徴

- スラッシュコマンドごとに実行ディレクトリ・引数・タイムアウトを個別設定
- `~` でホームディレクトリを解決
- コマンドの標準出力をコードブロックとして返す（2000 文字を超えると自動的に切り詰め）
- タイムアウト超過時は強制終了、失敗時はステータスと標準エラーを表示
- `config init` でテンプレートから設定ファイルを生成、`config flatten` で設定を `key=value` 形式で出力

## 要件

- Go 1.26 以降
- Discord のアプリケーション（ボットトークン、クライアント ID）
- Nix（開発シェルの使用を推奨。任意）

## インストール

```sh
go build -o discord-runner .
```

または Nix 開発シェルからビルドします。

```sh
nix develop
make build
```

## 使い方

```sh
# 設定ファイルを与えてボットを起動（サブコマンド省略可）
./discord-runner start config.json
./discord-runner config.json          # ↑と同じ

# 設定ファイルの生成（未生成時のみ。既存ファイルは上書きしない）
./discord-runner config init
./discord-runner config init my.json

# 設定ファイルを key=value 形式で出力
./discord-runner config flatten config.json
```

`--help` / `-h` を指定しても同じ利用方法が表示されます。

## 設定ファイル

JSON 形式です。`token` と `client_id` と少なくとも 1 つのスクリプトが必要です。
`config init` でテンプレートから生成できるほか、本レポジトリのテンプレートは埋め込まれています。

```json
{
  "token": ">>Your Token<<",
  "client_id": ">>Your Client ID<<",
  "scripts": {
    "script1": {
      "cwd": "~",
      "cmd": ["echo", "If you defined like this, you can call this command with `/script1`."],
      "timeout": 60
    }
  }
}
```

| キー | 意味 |
| --- | --- |
| `token` | ボットのトークン ([公式ドキュメントで確認](https://docs.discord.com/developers/quick-start/getting-started)) |
| `client_id` | Discord アプリケーションのクライアント ID ([確認方法](https://support-dev.discord.com/hc/en-us/articles/360028717192-Where-can-I-find-my-Application-Team-Server-ID)) |
| `scripts` | スラッシュコマンド名 → スクリプトのマップ |

各スクリプト:

| フィールド | 意味 |
| --- | --- |
| `cmd` | 実行するコマンド `[実行可能ファイル, 引数...]` |
| `cwd` | 作業ディレクトリ（`~` はホームディレクトリに展開） |
| `timeout` | 最大実行秒数（0 以下で無効化） |

両方ともDiscordアプリから取得できます。**Application ID** は*General Information*ページ、ボットの**Token** は*Bot*ページで確認できます。アプリの作成と値の取得方法は[Getting Startedガイド](https://docs.discord.com/developers/quick-start/getting-started)をご覧ください。

## 開発

Makefile を使います。

| ターゲット | 内容 |
| --- | --- |
| `make build` | バイナリをビルド |
| `make test` | テストを実行 |
| `make race` | レース検出器付きでテスト |
| `make check` | `fmtcheck` + `vet` + `test`（デフォルトの `all`） |
| `make run` | `config.json` でボットを起動 |
| `make init` | 設定ファイルを生成 |
| `make flatten` | 設定ファイルを `key=value` で出力 |
| `make fmt` | gofmt で整形 |
| `make clean` | バイナリを削除 |

Go 1.26 以降が必要です。開発シェルは Nix フレイク経由で用意でき、`gofmt`・`nixfmt`・`git-secrets` の pre-commit フックが有効になります。

```sh
nix develop   # フック付きの開発シェル
```

## プロジェクト構造

```
.
├── main.go                          # CLI エントリポイント
├── internal/
│   ├── config/                      # 設定の読み込み・生成・平坦化
│   └── bot/                         # Discord ボットの動作
└── Makefile
```
