# sshw v1.0.2：交互搜索与 Config Web 全面升级

这是一次以“更快找到服务器、更安全地管理配置”为核心的功能更新。

新版本重做了 `config` Web 界面，加入可持续编辑的全局搜索、分组与 Server 管理、多语言支持和短命令，同时修复配置保存、跨平台构建以及 GitHub Release 流程中的多个问题。

## ✨ 更新亮点

### 全局交互搜索

`-search` 现在可以不带参数直接启动，也可以携带一个可继续编辑的初始关键词：

```bash
# 搜索全部 Server
sshw -search

# 从 production 开始搜索，进入界面后仍可继续修改
sshw -search production

# 短命令
sshw -q production
```

- 支持持续输入、删除和修改关键词。
- 删除字符后会恢复之前被过滤的 Server。
- 没有匹配结果时仍可继续编辑，无需退出重试。
- 可搜索名称、Alias、User、Host 和 Port。
- 自动覆盖嵌套分组中的全部 Server。

### 全新的 Config Web

运行以下命令即可在默认浏览器中打开本地配置管理界面：

```bash
sshw -config

# 短命令
sshw -c
```

新的三栏工作区包括：

- **分组导航**：查看、新建、重命名、移动和删除嵌套分组。
- **Server 列表**：按 Server 字段或完整分组路径即时搜索。
- **详情编辑器**：新增、编辑、复制、移动和删除 Server。
- **完整连接配置**：支持认证信息、Jump Host 和 Callback Commands。
- **编辑保护**：显示未保存状态，并在离开页面前提醒。
- **键盘操作**：`Cmd/Ctrl+S` 保存，`Cmd/Ctrl+K` 聚焦搜索。

Config Web 默认只监听 `127.0.0.1`。如果浏览器无法自动打开，服务仍会继续运行，可手动访问终端中输出的 URL。

### 五种界面语言

Config Web 会在首次打开时匹配浏览器语言，也可以随时手动切换：

- English
- 简体中文
- 日本語
- 한국어
- Tiếng Việt

界面标签、状态提示、搜索结果、校验错误和删除确认均已本地化。语言偏好会保存在浏览器中，切换语言不会修改配置内容。

### 更安全的配置保存

- 在配置文件所在目录写入临时文件，再通过原子替换完成保存。
- 写入前执行同步，降低部分写入导致配置损坏的风险。
- 新配置文件权限设为 `0600`。
- 自动创建缺失的配置目录。
- 支持替换只读配置文件。
- 保存失败时保留原文件并清理临时文件。
- 拒绝覆盖符号链接配置，避免误改链接目标。
- 校验 Port、Callback Delay 和空节点等非法数据。
- 保留嵌套分组、空分组以及额外的 Jump Host 数据。

## ⌨️ 短命令速查

| 完整命令 | 短命令 | 说明 |
| --- | --- | --- |
| `sshw -search [关键词]` | `sshw -q [关键词]` | 全局交互搜索 |
| `sshw -config` | `sshw -c` | 打开 Config Web |
| `sshw -version` | `sshw -v` | 查看版本信息 |
| `sshw -help` | `sshw -h` | 查看帮助 |

原有完整命令保持兼容。

## 🔧 构建与发布

- 新增 `build.sh`，默认将产物输出到 `dist/<os>-<arch>/sshw`。
- 支持通过 `GOOS`、`GOARCH`、`CGO_ENABLED`、`VERSION` 和 `OUTPUT_DIR` 自定义构建。
- 构建入口统一为完整的 `./cmd/sshw` package，修复只编译 `main.go` 导致的 `undefined: runConfigUI`。
- GoReleaser 配置升级到 v2，并加入发布前配置检查。
- GitHub Actions 更新到 Node 24 兼容版本。
- GitHub Pages 文档站现已上线：[cooker.github.io/sshw](https://cooker.github.io/sshw/)。

```bash
# 编译当前平台
./build.sh

# 交叉编译 Linux AMD64
GOOS=linux GOARCH=amd64 VERSION=v1.0.2 ./build.sh
```

## ✅ 兼容性

- 现有 YAML 配置格式保持不变。
- 原有命令和完整参数继续可用。
- `GET /api/config` 和 `POST /api/config` 协议保持不变。
- `port: 0` 仍表示使用默认 SSH 端口。

## 📦 升级方式

从 [GitHub Releases](https://github.com/cooker/sshw/releases) 下载适合系统和 CPU 架构的压缩包，替换旧版 `sshw` 二进制即可。

已安装 Go 的用户也可以执行：

```bash
go install github.com/yinheli/sshw/cmd/sshw@latest
```

首次打开新版 Config Web 时，建议先确认终端输出的配置文件路径，再编辑并保存。

---

[查看 v1.0.1...v1.0.2 的完整变更](https://github.com/cooker/sshw/compare/v1.0.1...v1.0.2)
