# ボットをサーバに追加する

discord-runner は設定したスクリプトをそれぞれスラッシュコマンドとして登録します。
Discord 上で使用するには、まず Discord アプリケーションとボットを作成し、
スラッシュコマンドの権限を有効にしてボットをサーバへ招待してください。

主な手順は Discord 公式ドキュメントで説明されています（下線はリンク）。
このページでは、このプロジェクトで必要な設定のみを示します。

## 1. Discord アプリケーションとボットの作成

アプリケーションを作成し、ボットを追加してから、**Application ID** と**ボットトークン**をコピーします。

- ガイド: [Getting Started](https://docs.discord.com/developers/quick-start/getting-started)
- アプリの新規作成: [https://discord.com/developers/applications?new_application=true](https://discord.com/developers/applications?new_application=true)

## 2. 「スラッシュコマンドの使用」権限でボットを招待

スラッシュコマンドには「**スラッシュコマンドの使用**」権限が必要です。
ボットを招待する際に有効にしてください — これがこのプロジェクトで必要な唯一の設定です。

- 招待フローと権限の仕組み: [OAuth2 and Permissions](https://docs.discord.com/developers/topics/oauth2)

この権限は `USE_APPLICATION_COMMANDS`（`1 << 31`、値は `2147483648`）です。
有効化した招待リンクは次のようになります:

```
https://discord.com/oauth2/authorize?client_id=CLIENT_ID&scope=bot%20applications.commands&permissions=2147483648
```

`CLIENT_ID` をあなたの Application ID に置き換えてください。

### より簡単: インストールページ

アプリケーションのインストールページでスコープとこの権限を設定し、
招待リンクを生成することもできます:

- [https://discord.com/developers/applications/select/<APP_ID>/installation](https://discord.com/developers/applications/select/<APP_ID>/installation)

## 値の設定場所

`client_id` とボットトークンは設定ファイルに記述します。設定ファイルの形式は[メインの README](../README.md)を確認してください。
