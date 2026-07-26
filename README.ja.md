# sshw

<p align="center">
  <img src="./assets/readme/hero.svg" width="100%" alt="sshw は YAML または SSH config のホスト設定を検索可能なインタラクティブ SSH ランチャーにします">
</p>

<p align="center">
  <a href="https://github.com/yinheli/sshw/releases"><img src="https://img.shields.io/github/v/tag/yinheli/sshw?label=release" alt="最新リリース"></a>
  <a href="./LICENSE"><img src="https://img.shields.io/github/license/yinheli/sshw" alt="ライセンス"></a>
  <a href="https://pkg.go.dev/github.com/yinheli/sshw"><img src="https://pkg.go.dev/badge/github.com/yinheli/sshw.svg" alt="Go package"></a>
</p>

<p align="center">
  <a href="./README.md">English</a> | 日本語 | <a href="./README.ko.md">한국어</a> | <a href="./README.vi.md">Tiếng Việt</a> | <a href="./README.zh-CN.md">简体中文</a>
</p>

`sshw` は、多数のサーバーを行き来するユーザー向けの小さな SSH クライアントラッパーです。ホストを YAML に一度定義すれば、検索可能なターミナルメニューから選択し、ユーザー、ポート、鍵、パスワード、SSH agent、踏み台ホスト、任意の起動後コマンドを使ってセッションを開けます。

![sshw のインタラクティブデモ](./assets/sshw-demo.gif)

## 使う理由

- 長い `ssh` コマンドを入力せず、インタラクティブなターミナル選択画面で SSH 接続先を参照できます。
- `sshw -search <keyword>` でネストしたホストグループ全体を検索できます。
- `sshw -config` で `~/.sshw.yaml` をローカル Web UI から設定できます。
- `sshw dev` のように別名で直接接続できます。
- チーム、環境、ネットワーク単位でホストをネストしたグループに整理できます。
- パスワード、鍵ファイル、パスフレーズ付き鍵、SSH agent、keyboard-interactive 認証に対応しています。
- 設定済みの踏み台ホスト経由で接続できます。
- `sshw -s` で `~/.ssh/config` から項目を取り込めます。
- リモート shell が開いた後に任意のコールバックコマンドを実行できます。

## インストール

Go で最新バージョンをインストールします。

```bash
go install github.com/yinheli/sshw/cmd/sshw@latest
```

現在のソースをビルドします。

```bash
./build.sh
```

バイナリは `dist/<os>-<arch>/sshw` に出力されます。`GOOS`、`GOARCH`、
`VERSION`、`OUTPUT_DIR` でターゲット、バージョン、出力先を指定できます。

```bash
GOOS=linux GOARCH=amd64 VERSION=dev ./build.sh
```

事前ビルド済みバイナリは [GitHub Releases](https://github.com/yinheli/sshw/releases) からもダウンロードできます。

## 最初の実行

設定ファイルを作成します。

```bash
touch ~/.sshw.yml
```

少なくとも 1 台のホストを追加します。

```yaml
- name: dev
  alias: dev
  host: 192.168.8.35
  user: appuser
  port: 22
  keypath: ~/.ssh/id_rsa
```

選択画面を開きます。

```bash
sshw
```

別名で接続することもできます。

```bash
sshw dev
```

すべてのホストを対話的に検索します。初期キーワードは編集でき、文字を削除すると絞り込まれていたホストが再表示されます。

```bash
sshw -search
sshw -search dev
```

`~/.sshw.yaml` 用のローカル Web 設定画面を起動します。

```bash
sshw -config
```

## 設定

`sshw` は、次の順序で最初に見つかった設定ファイルを読み込みます。

1. `~/.sshw`
2. `~/.sshw.yml`
3. `~/.sshw.yaml`
4. `./.sshw`
5. `./.sshw.yml`
6. `./.sshw.yaml`

ローカル Web UI で `~/.sshw.yaml` を作成または更新するには、次を実行します。

```bash
sshw -config
```

エディターはデフォルトブラウザーでローカル URL を開き、ターミナルにも URL を表示します。デフォルトでは `127.0.0.1` のみで待ち受けます。ホストとグループの追加、編集、削除、保存ができ、一般的なホスト項目、1 つの踏み台ホスト、コールバックコマンド、ネストしたグループ、空のグループに対応しています。ブラウザーを開けない場合もサーバーは動作を続けるため、表示された URL を手動で開けます。別のファイルを編集するには `-config-file` を指定します。

```bash
sshw -config -config-file ./.sshw.yaml
```

別のローカル待ち受けアドレスを使う場合は、次を指定します。

```bash
sshw -config -config-addr 127.0.0.1:9000
```

各項目はホストまたはグループとして定義できます。ホスト項目は次のとおりです。

| 項目 | 説明 |
| --- | --- |
| `name` | 選択画面に表示される名前。 |
| `alias` | `sshw <alias>` で使う任意のショートカット。 |
| `host` | ホスト名または IP アドレス。 |
| `user` | SSH ユーザー名。省略時は `root` です。 |
| `port` | SSH ポート。省略時は `22` です。 |
| `password` | パスワード認証の値。 |
| `keypath` | 秘密鍵のパス。`~` は展開されます。 |
| `passphrase` | 設定した鍵のパスフレーズ。 |
| `agentpath` | SSH agent のソケットパス。 |
| `jump` | 踏み台ホスト定義。 |
| `children` | ネストしたホストまたはグループ。 |
| `callback-shells` | shell 起動後に送信するコマンド。 |

## 例

### 一般的なホスト

```yaml
- { name: dev with password, alias: dev, user: appuser, host: 192.168.8.35, port: 22, password: 123456 }
- { name: dev with key, user: appuser, host: 192.168.8.35, keypath: ~/.ssh/id_rsa }
- { name: default root and port, host: 192.168.8.35 }
- { name: "emoji host", alias: spark, host: 192.168.8.35 }
```

### 踏み台ホスト

```yaml
- name: app via bastion
  alias: app
  user: appuser
  host: 192.168.8.35
  port: 22
  keypath: ~/.ssh/id_rsa
  jump:
    - user: appuser
      host: 192.168.8.36
      port: 2222
      keypath: ~/.ssh/bastion_rsa
```

### グループ

```yaml
- name: production
  children:
    - { name: app-1, user: root, host: 192.168.1.2 }
    - { name: app-2, user: root, host: 192.168.1.3 }
    - { name: app-3, user: root, host: 192.168.1.4 }
```

### コールバックコマンド

`callback-shells` は、ログイン後にリモート shell へコマンドを送信します。`delay` の単位はミリ秒です。

```yaml
- name: dev with startup commands
  alias: dev
  user: appuser
  host: 192.168.8.35
  keypath: ~/.ssh/id_rsa
  callback-shells:
    - { cmd: "cd /srv/app" }
    - { delay: 1500, cmd: "docker ps" }
```

## `~/.ssh/config` を使う

`.sshw.yml` ではなくローカル SSH config から選択画面を作るには、次を実行します。

```bash
sshw -s
```

`sshw` は `~/.ssh/config` から `Host`、`HostName`、`User`、`Port`、`IdentityFile`、`IdentityAgent` を読み取ります。

## コマンド

```text
sshw             インタラクティブ選択画面を開く
sshw <alias>     設定済みの別名へ接続
sshw -s          ~/.ssh/config から項目を読み込む
sshw -search [keyword] 名前、別名、ユーザー、ホスト、ポートで対話検索
sshw -q [keyword] -search の短縮形
sshw -config     ローカル Web UI で ~/.sshw.yaml を編集
sshw -c          -config の短縮形
sshw -version    ビルドバージョンと Go バージョンを表示
sshw -v          -version の短縮形
sshw -help       フラグを表示
sshw -h          -help の短縮形
```

## 注意事項

- 実際のパスワード、秘密鍵、本番ホスト情報を公開リポジトリにコミットしないでください。
- `user` を省略すると、`sshw` は `root` を使用します。
- `port` を省略すると、`sshw` は `22` を使用します。
- 踏み台ホスト対応では、現在は設定内の最初の `jump` 項目を使用します。

## ライセンス

[MIT](./LICENSE)
