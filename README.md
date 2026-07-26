# sshw

<p align="center">
  <img src="./assets/readme/hero.svg" width="100%" alt="sshw turns YAML or SSH config entries into an interactive SSH launcher">
</p>

<p align="center">
  <a href="https://github.com/yinheli/sshw/releases"><img src="https://img.shields.io/github/v/tag/yinheli/sshw?label=release" alt="Latest release"></a>
  <a href="./LICENSE"><img src="https://img.shields.io/github/license/yinheli/sshw" alt="License"></a>
  <a href="https://pkg.go.dev/github.com/yinheli/sshw"><img src="https://pkg.go.dev/badge/github.com/yinheli/sshw.svg" alt="Go package"></a>
</p>

<p align="center">
  English | <a href="./README.ja.md">日本語</a> | <a href="./README.ko.md">한국어</a> | <a href="./README.vi.md">Tiếng Việt</a> | <a href="./README.zh-CN.md">简体中文</a>
</p>

`sshw` is a small SSH client wrapper for people who jump between many servers. Define hosts once in YAML, pick one from a searchable terminal menu, and let `sshw` open the session with the right user, port, key, password, agent, jump host, and optional startup commands.

![Interactive sshw demo](./assets/sshw-demo.gif)

## Why use it

- Browse SSH targets in an interactive terminal selector instead of typing long `ssh` commands.
- Search across nested host groups with `sshw -search <keyword>`.
- Configure `~/.sshw.yaml` from a local web UI with `sshw -config`.
- Connect directly by alias, for example `sshw dev`.
- Organize hosts into nested groups for teams, environments, or network segments.
- Use password, key file, passphrase key, SSH agent, or keyboard-interactive authentication.
- Route through a configured jump host.
- Import entries from `~/.ssh/config` with `sshw -s`.
- Run optional callback commands after the remote shell opens.

## Install

Install the latest version with Go:

```bash
go install github.com/yinheli/sshw/cmd/sshw@latest
```

Build the current checkout:

```bash
./build.sh
```

The binary is written to `dist/<os>-<arch>/sshw`. Set `GOOS`, `GOARCH`,
`VERSION`, or `OUTPUT_DIR` to override the target, version, or output directory:

```bash
GOOS=linux GOARCH=amd64 VERSION=dev ./build.sh
```

You can also download prebuilt binaries from [GitHub Releases](https://github.com/yinheli/sshw/releases).

## First run

Create a config file:

```bash
touch ~/.sshw.yml
```

Add at least one host:

```yaml
- name: dev
  alias: dev
  host: 192.168.8.35
  user: appuser
  port: 22
  keypath: ~/.ssh/id_rsa
```

Open the selector:

```bash
sshw
```

Or connect by alias:

```bash
sshw dev
```

Open the interactive global host search. The optional initial keyword remains
editable, and deleting characters restores previously filtered hosts:

```bash
sshw -search
sshw -search dev
```

Start the local web config editor for `~/.sshw.yaml`:

```bash
sshw -config
```

## Configuration

`sshw` loads the first config file it finds in this order:

1. `~/.sshw`
2. `~/.sshw.yml`
3. `~/.sshw.yaml`
4. `./.sshw`
5. `./.sshw.yml`
6. `./.sshw.yaml`

To create or update `~/.sshw.yaml` from a local web UI, run:

```bash
sshw -config
```

The editor opens the local URL in your default browser and also prints it in
the terminal. It serves only on `127.0.0.1` by default. Its three-pane workspace
supports nested group management, scoped search by server or group fields, and
adding, editing, duplicating, moving, and deleting servers. It also supports
common host fields, jump hosts, callback commands, empty groups, unsaved-change
tracking, and keyboard save. The interface follows the browser language on first
use and can switch between English, Simplified Chinese, Japanese, Korean, and
Vietnamese. If the browser cannot be opened, the server keeps running and you
can open the printed URL manually.
To edit another file, pass `-config-file`:

```bash
sshw -config -config-file ./.sshw.yaml
```

To use a different local bind address:

```bash
sshw -config -config-addr 127.0.0.1:9000
```

Each entry can be a host or a group. Host fields:

| Field | Purpose |
| --- | --- |
| `name` | Display name in the selector. |
| `alias` | Optional shortcut used by `sshw <alias>`. |
| `host` | Hostname or IP address. |
| `user` | SSH username. Defaults to `root` when omitted. |
| `port` | SSH port. Defaults to `22` when omitted. |
| `password` | Password authentication value. |
| `keypath` | Private key path. `~` is expanded. |
| `passphrase` | Passphrase for the configured key. |
| `agentpath` | SSH agent socket path. |
| `jump` | One or more jump host definitions. |
| `children` | Nested hosts or groups. |
| `callback-shells` | Commands sent after the shell starts. |

## Examples

### Common hosts

```yaml
- { name: dev with password, alias: dev, user: appuser, host: 192.168.8.35, port: 22, password: 123456 }
- { name: dev with key, user: appuser, host: 192.168.8.35, keypath: ~/.ssh/id_rsa }
- { name: default root and port, host: 192.168.8.35 }
- { name: "emoji host", alias: spark, host: 192.168.8.35 }
```

### Jump host

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

### Groups

```yaml
- name: production
  children:
    - { name: app-1, user: root, host: 192.168.1.2 }
    - { name: app-2, user: root, host: 192.168.1.3 }
    - { name: app-3, user: root, host: 192.168.1.4 }
```

### Callback commands

`callback-shells` sends commands to the remote shell after login. `delay` is measured in milliseconds.

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

## Use `~/.ssh/config`

To build the selector from your local SSH config instead of `.sshw.yml`, run:

```bash
sshw -s
```

`sshw` reads `Host`, `HostName`, `User`, `Port`, `IdentityFile`, and `IdentityAgent` from `~/.ssh/config`.

## Commands

```text
sshw             open the interactive selector
sshw <alias>     connect to a configured alias
sshw -s          load entries from ~/.ssh/config
sshw -search [keyword] interactively search by name, alias, user, host, or port
sshw -q [keyword] shorthand for -search
sshw -config     edit ~/.sshw.yaml from a local web UI
sshw -c          shorthand for -config
sshw -version    print build and Go version
sshw -v          shorthand for -version
sshw -help       show flags
sshw -h          shorthand for -help
```

## Notes

- Do not commit real passwords, private keys, or production host details to a public repository.
- If `user` is omitted, `sshw` uses `root`.
- If `port` is omitted, `sshw` uses `22`.
- Jump host support currently uses the first configured jump entry.

## License

[MIT](./LICENSE)
