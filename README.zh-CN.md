# sshw

<p align="center">
  <img src="./assets/readme/hero.svg" width="100%" alt="sshw 将 YAML 或 SSH config 中的主机配置变成可搜索的交互式 SSH 启动器">
</p>

<p align="center">
  <a href="https://cooker.github.io/sshw/">使用文档</a> ·
  <a href="https://github.com/yinheli/sshw/releases"><img src="https://img.shields.io/github/v/tag/yinheli/sshw?label=release" alt="最新版本"></a>
  <a href="./LICENSE"><img src="https://img.shields.io/github/license/yinheli/sshw" alt="许可证"></a>
  <a href="https://pkg.go.dev/github.com/yinheli/sshw"><img src="https://pkg.go.dev/badge/github.com/yinheli/sshw.svg" alt="Go package"></a>
</p>

<p align="center">
  <a href="./README.md">English</a> | <a href="./README.ja.md">日本語</a> | <a href="./README.ko.md">한국어</a> | <a href="./README.vi.md">Tiếng Việt</a> | 简体中文
</p>

`sshw` 是一个轻量的 SSH 客户端包装工具，适合经常在多台服务器之间切换的用户。你只需要在 YAML 里维护主机信息，就可以通过可搜索的终端菜单选择目标，并让 `sshw` 自动使用对应的用户、端口、密钥、密码、SSH agent、跳板机和登录后回调命令打开会话。

![sshw 交互式演示](./assets/sshw-demo.gif)

## 为什么使用

- 用交互式终端选择器浏览 SSH 目标，少输入冗长的 `ssh` 命令。
- 使用 `sshw -search <keyword>` 跨嵌套分组搜索主机。
- 使用 `sshw -config` 通过本地 Web 界面配置 `~/.sshw.yaml`。
- 支持通过别名直连，例如 `sshw dev`。
- 支持按团队、环境或网段组织嵌套分组。
- 支持密码、密钥文件、带 passphrase 的密钥、SSH agent 和 keyboard-interactive 认证。
- 支持通过跳板机连接目标主机。
- 可使用 `sshw -s` 从 `~/.ssh/config` 导入条目。
- 支持远程 shell 打开后自动发送回调命令。

## 安装

使用 Go 安装最新版本：

```bash
go install github.com/yinheli/sshw/cmd/sshw@latest
```

编译当前源码：

```bash
./build.sh
```

产物输出到 `dist/<操作系统>-<架构>/sshw`。可以通过 `GOOS`、`GOARCH`、
`VERSION` 或 `OUTPUT_DIR` 指定目标平台、版本及输出目录：

```bash
GOOS=linux GOARCH=amd64 VERSION=dev ./build.sh
```

也可以从 [GitHub Releases](https://github.com/yinheli/sshw/releases) 下载预编译二进制文件。

## 第一次运行

创建配置文件：

```bash
touch ~/.sshw.yml
```

添加至少一台主机：

```yaml
- name: dev
  alias: dev
  host: 192.168.8.35
  user: appuser
  port: 22
  keypath: ~/.ssh/id_rsa
```

打开交互式选择器：

```bash
sshw
```

或通过别名直接连接：

```bash
sshw dev
```

打开全局主机交互搜索。初始关键词可以继续编辑，删除字符会恢复之前过滤掉的主机：

```bash
sshw -search
sshw -search dev
```

启动 `~/.sshw.yaml` 的本地 Web 配置界面：

```bash
sshw -config
```

## 配置

`sshw` 会按顺序读取第一个存在的配置文件：

1. `~/.sshw`
2. `~/.sshw.yml`
3. `~/.sshw.yaml`
4. `./.sshw`
5. `./.sshw.yml`
6. `./.sshw.yaml`

如需通过本地 Web 界面创建或更新 `~/.sshw.yaml`，运行：

```bash
sshw -config
```

该命令会使用默认浏览器打开本地 URL，并同时在终端中输出地址；默认只监听
`127.0.0.1`。三栏工作区支持嵌套分组管理、按 Server 或分组字段搜索，以及
Server 的新增、编辑、复制、移动和删除。它也支持常见主机字段、跳板机、回调命令、
空分组、未保存状态提示和键盘保存。界面首次打开时会跟随浏览器语言，也可以在
英语、简体中文、日语、韩语和越南语之间切换。浏览器打开失败时，服务仍会继续
运行，可以手动访问终端中输出的地址。要编辑其他文件，可以传入 `-config-file`：

```bash
sshw -config -config-file ./.sshw.yaml
```

如需使用其他本地监听地址：

```bash
sshw -config -config-addr 127.0.0.1:9000
```

每个条目可以是一台主机，也可以是一个分组。主机字段如下：

| 字段 | 说明 |
| --- | --- |
| `name` | 选择器中展示的名称。 |
| `alias` | 可选快捷名，用于 `sshw <alias>`。 |
| `host` | 主机名或 IP 地址。 |
| `user` | SSH 用户名。省略时默认使用 `root`。 |
| `port` | SSH 端口。省略时默认使用 `22`。 |
| `password` | 密码认证值。 |
| `keypath` | 私钥路径，支持展开 `~`。 |
| `passphrase` | 配置密钥的 passphrase。 |
| `agentpath` | SSH agent socket 路径。 |
| `jump` | 跳板机定义。 |
| `children` | 嵌套主机或分组。 |
| `callback-shells` | shell 启动后发送的命令。 |

## 示例

### 常见主机

```yaml
- { name: dev with password, alias: dev, user: appuser, host: 192.168.8.35, port: 22, password: 123456 }
- { name: dev with key, user: appuser, host: 192.168.8.35, keypath: ~/.ssh/id_rsa }
- { name: default root and port, host: 192.168.8.35 }
- { name: "emoji host", alias: spark, host: 192.168.8.35 }
```

### 跳板机

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

### 分组

```yaml
- name: production
  children:
    - { name: app-1, user: root, host: 192.168.1.2 }
    - { name: app-2, user: root, host: 192.168.1.3 }
    - { name: app-3, user: root, host: 192.168.1.4 }
```

### 回调命令

`callback-shells` 会在登录后向远程 shell 发送命令。`delay` 的单位是毫秒。

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

## 使用 `~/.ssh/config`

如果想从本地 SSH config 构建选择器，而不是读取 `.sshw.yml`，运行：

```bash
sshw -s
```

`sshw` 会读取 `~/.ssh/config` 中的 `Host`、`HostName`、`User`、`Port`、`IdentityFile` 和 `IdentityAgent`。

## 命令

```text
sshw             打开交互式选择器
sshw <alias>     连接到配置好的别名
sshw -s          从 ~/.ssh/config 加载条目
sshw -search [关键词] 按名称、别名、用户、主机或端口交互搜索主机
sshw -q [关键词] -search 的短命令
sshw -config     通过本地 Web 界面编辑 ~/.sshw.yaml
sshw -c          -config 的短命令
sshw -version    输出构建版本和 Go 版本
sshw -v          -version 的短命令
sshw -help       显示参数
sshw -h          -help 的短命令
```

## 注意事项

- 不要把真实密码、私钥或生产主机信息提交到公开仓库。
- 如果省略 `user`，`sshw` 使用 `root`。
- 如果省略 `port`，`sshw` 使用 `22`。
- 跳板机支持目前使用配置中的第一个 `jump` 条目。

## 支持项目

如果 `sshw` 为你节省了时间，欢迎通过小额赞赏支持项目持续维护。感谢你的认可与支持。

<p align="center">
  <img src="./docs/wx.jpg" width="190" alt="微信支付和支付宝赞赏码">
  &nbsp;&nbsp;&nbsp;&nbsp;
  <img src="./docs/usdt.jpg" width="240" alt="USDT TRON 网络收款二维码">
</p>

<p align="center">
  <sub><strong>微信支付 / 支付宝</strong> · <strong>USDT（TRON）</strong>——转账 USDT 时请仅使用 TRON 网络。</sub>
</p>

## 许可证

[MIT](./LICENSE)
