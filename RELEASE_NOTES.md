# sshw vX.Y.Z — 搜索与 Config 管理体验升级

本次版本重点升级了交互搜索和 `config` Web 管理界面，同时修复配置保存及 GitHub Actions 构建问题。

## 主要更新

### 更灵活的交互搜索

- `sshw -search` 现在可以不带关键词，直接进入全局主机搜索。
- 支持在搜索界面持续输入、删除和修改关键词。
- 初始关键词可以回退；删除字符后，之前被过滤的主机会重新出现。
- 即使暂时没有匹配结果，也可以继续修改搜索词，无需退出重试。
- 搜索范围覆盖名称、Alias、User、Host 和 Port，并支持嵌套分组中的主机。

```bash
# 打开全局交互搜索
sshw -search

# 使用初始关键词进入搜索，关键词仍可继续编辑
sshw -search production

# 短命令
sshw -q production
```

### 全新的 Config Web 管理界面

`sshw -config` 现在提供三栏式 Server 管理工作区：

- 左侧：嵌套分组树与 Server 数量。
- 中间：当前分组及子分组中的 Server 列表和即时搜索。
- 右侧：分组或 Server 的详细编辑器。

支持的管理功能：

- 创建、重命名、移动和删除嵌套分组。
- 新增、编辑、复制、移动和删除 Server。
- 按名称、Alias、Host、User、Port 或完整分组路径搜索。
- 编辑认证信息、Jump Host 和 Callback Commands。
- 显示未保存状态，并在离开页面前提醒。
- 使用 `Cmd/Ctrl+S` 保存、`Cmd/Ctrl+K` 聚焦搜索。
- 默认自动打开系统浏览器；打开失败时仍可使用终端输出的本地 URL。

```bash
# 编辑默认配置文件 ~/.sshw.yaml
sshw -config

# 短命令
sshw -c

# 编辑指定配置文件
sshw -config -config-file ./.sshw.yaml

# 指定本地监听地址
sshw -config -config-addr 127.0.0.1:9000
```

### Config 界面多语言支持

界面首次打开时会自动匹配浏览器语言，也可以随时手动切换：

- English
- 简体中文
- 日本語
- 한국어
- Tiếng Việt

静态标签、搜索结果、保存状态、校验错误和删除确认均已本地化。语言选择会在刷新后保留，切换语言不会修改配置数据或触发保存。

### 更可靠的配置保存

- 使用同目录临时文件、同步写入和原子替换，降低部分写入造成配置损坏的风险。
- 新配置文件权限设置为 `0600`。
- 自动创建缺失的配置目录。
- 支持替换只读配置文件。
- 写入失败时保留原配置并清理临时文件。
- 拒绝覆盖符号链接配置，避免意外修改链接目标。
- 增加递归数据校验，拒绝非法 Port、负数 Callback Delay 和 `null` 节点。
- 保留嵌套分组、空分组及未展示的额外 Jump Host 数据。

### 短命令

新增以下快捷参数：

| 完整命令 | 短命令 | 功能 |
| --- | --- | --- |
| `-search` | `-q` | 交互搜索主机 |
| `-config` | `-c` | 打开 Config Web 界面 |
| `-version` | `-v` | 显示版本信息 |
| `-help` | `-h` | 显示帮助 |

### 构建与 CI

- 新增项目编译脚本 `build.sh`。
- 支持通过 `GOOS`、`GOARCH`、`CGO_ENABLED`、`VERSION` 和 `OUTPUT_DIR` 自定义构建。
- 构建产物输出到 `dist/<os>-<arch>/sshw`。
- 修复 GitHub Actions 和 GoReleaser 只编译 `main.go` 导致的错误：

```text
cmd/sshw/main.go:73:13: undefined: runConfigUI
```

构建入口现已统一为完整的 `./cmd/sshw` package。

```bash
# 编译当前平台
./build.sh

# 交叉编译 Linux AMD64
GOOS=linux GOARCH=amd64 VERSION=vX.Y.Z ./build.sh
```

## 兼容性

- 现有 YAML 配置格式保持不变。
- `GET /api/config` 和 `POST /api/config` 协议保持不变。
- 原有命令和完整参数继续可用。
- `port: 0` 仍表示使用默认 SSH 端口。

## 验证

本次更新已通过：

- 全量 Go 单元测试。
- Race detector。
- `go vet`。
- Config API 保存失败及数据校验测试。
- 浏览器端分组、搜索、Server 移动、保存重载和多语言切换测试。
- Darwin ARM64 本地构建验证。

## 升级提示

替换旧版 `sshw` 二进制即可升级。首次使用新版 Config 界面时，建议先确认终端输出的配置文件路径，再进行编辑和保存。
