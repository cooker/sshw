# sshw

<p align="center">
  <img src="./assets/readme/hero.svg" width="100%" alt="sshw biến cấu hình máy chủ trong YAML hoặc SSH config thành trình khởi chạy SSH tương tác có thể tìm kiếm">
</p>

<p align="center">
  <a href="https://github.com/yinheli/sshw/releases"><img src="https://img.shields.io/github/v/tag/yinheli/sshw?label=release" alt="Bản phát hành mới nhất"></a>
  <a href="./LICENSE"><img src="https://img.shields.io/github/license/yinheli/sshw" alt="Giấy phép"></a>
  <a href="https://pkg.go.dev/github.com/yinheli/sshw"><img src="https://pkg.go.dev/badge/github.com/yinheli/sshw.svg" alt="Go package"></a>
</p>

<p align="center">
  <a href="./README.md">English</a> | <a href="./README.ja.md">日本語</a> | <a href="./README.ko.md">한국어</a> | Tiếng Việt | <a href="./README.zh-CN.md">简体中文</a>
</p>

`sshw` là một wrapper SSH client nhỏ dành cho người thường xuyên chuyển đổi giữa nhiều máy chủ. Bạn chỉ cần định nghĩa host một lần trong YAML, chọn mục tiêu từ menu terminal có thể tìm kiếm, rồi để `sshw` mở phiên với đúng user, port, khóa, mật khẩu, SSH agent, jump host và các lệnh khởi động tùy chọn.

![Bản demo tương tác của sshw](./assets/sshw-demo.gif)

## Vì sao nên dùng

- Duyệt các mục tiêu SSH trong bộ chọn terminal tương tác thay vì gõ các lệnh `ssh` dài.
- Tìm kiếm trên các nhóm host lồng nhau bằng `sshw -search <keyword>`.
- Cấu hình `~/.sshw.yaml` bằng web UI cục bộ với `sshw -config`.
- Kết nối trực tiếp bằng alias, ví dụ `sshw dev`.
- Sắp xếp host thành các nhóm lồng nhau theo nhóm làm việc, môi trường hoặc phân đoạn mạng.
- Hỗ trợ xác thực bằng mật khẩu, tệp khóa, khóa có passphrase, SSH agent hoặc keyboard-interactive.
- Kết nối qua jump host đã cấu hình.
- Nhập mục từ `~/.ssh/config` bằng `sshw -s`.
- Chạy các lệnh callback tùy chọn sau khi remote shell mở.

## Cài đặt

Cài đặt phiên bản mới nhất bằng Go:

```bash
go install github.com/yinheli/sshw/cmd/sshw@latest
```

Biên dịch mã nguồn hiện tại:

```bash
./build.sh
```

Binary được ghi vào `dist/<os>-<arch>/sshw`. Có thể dùng `GOOS`, `GOARCH`,
`VERSION` hoặc `OUTPUT_DIR` để chỉ định nền tảng, phiên bản và thư mục đầu ra:

```bash
GOOS=linux GOARCH=amd64 VERSION=dev ./build.sh
```

Bạn cũng có thể tải binary dựng sẵn từ [GitHub Releases](https://github.com/yinheli/sshw/releases).

## Chạy lần đầu

Tạo tệp cấu hình:

```bash
touch ~/.sshw.yml
```

Thêm ít nhất một host:

```yaml
- name: dev
  alias: dev
  host: 192.168.8.35
  user: appuser
  port: 22
  keypath: ~/.ssh/id_rsa
```

Mở bộ chọn:

```bash
sshw
```

Hoặc kết nối bằng alias:

```bash
sshw dev
```

Tìm kiếm tương tác trên tất cả host. Từ khóa ban đầu có thể chỉnh sửa và việc
xóa ký tự sẽ khôi phục các host đã bị lọc:

```bash
sshw -search
sshw -search dev
```

Khởi động trình chỉnh sửa cấu hình web cục bộ cho `~/.sshw.yaml`:

```bash
sshw -config
```

## Cấu hình

`sshw` đọc tệp cấu hình đầu tiên tìm thấy theo thứ tự sau:

1. `~/.sshw`
2. `~/.sshw.yml`
3. `~/.sshw.yaml`
4. `./.sshw`
5. `./.sshw.yml`
6. `./.sshw.yaml`

Để tạo hoặc cập nhật `~/.sshw.yaml` từ web UI cục bộ, hãy chạy:

```bash
sshw -config
```

Trình chỉnh sửa mở URL cục bộ trong trình duyệt mặc định và cũng in địa chỉ ra terminal. Mặc định công cụ chỉ phục vụ trên `127.0.0.1`, đồng thời có thể thêm, sửa, xóa, lưu host và nhóm. Công cụ hỗ trợ các trường host phổ biến, một jump host, lệnh callback, nhóm lồng nhau và nhóm rỗng. Nếu không thể mở trình duyệt, máy chủ vẫn tiếp tục chạy và bạn có thể mở URL đã in theo cách thủ công. Để chỉnh sửa tệp khác, truyền `-config-file`:

```bash
sshw -config -config-file ./.sshw.yaml
```

Để dùng địa chỉ bind cục bộ khác:

```bash
sshw -config -config-addr 127.0.0.1:9000
```

Mỗi mục có thể là host hoặc nhóm. Các trường host:

| Trường | Mục đích |
| --- | --- |
| `name` | Tên hiển thị trong bộ chọn. |
| `alias` | Lối tắt tùy chọn dùng bởi `sshw <alias>`. |
| `host` | Tên host hoặc địa chỉ IP. |
| `user` | Tên người dùng SSH. Mặc định là `root` khi bỏ trống. |
| `port` | Cổng SSH. Mặc định là `22` khi bỏ trống. |
| `password` | Giá trị xác thực bằng mật khẩu. |
| `keypath` | Đường dẫn khóa riêng. `~` sẽ được mở rộng. |
| `passphrase` | Passphrase cho khóa đã cấu hình. |
| `agentpath` | Đường dẫn socket của SSH agent. |
| `jump` | Định nghĩa jump host. |
| `children` | Host hoặc nhóm lồng nhau. |
| `callback-shells` | Lệnh được gửi sau khi shell khởi động. |

## Ví dụ

### Host phổ biến

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

### Nhóm

```yaml
- name: production
  children:
    - { name: app-1, user: root, host: 192.168.1.2 }
    - { name: app-2, user: root, host: 192.168.1.3 }
    - { name: app-3, user: root, host: 192.168.1.4 }
```

### Lệnh callback

`callback-shells` gửi lệnh đến remote shell sau khi đăng nhập. `delay` được tính bằng mili giây.

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

## Dùng `~/.ssh/config`

Để tạo bộ chọn từ SSH config cục bộ thay vì `.sshw.yml`, hãy chạy:

```bash
sshw -s
```

`sshw` đọc `Host`, `HostName`, `User`, `Port`, `IdentityFile` và `IdentityAgent` từ `~/.ssh/config`.

## Lệnh

```text
sshw             mở bộ chọn tương tác
sshw <alias>     kết nối đến alias đã cấu hình
sshw -s          tải các mục từ ~/.ssh/config
sshw -search [keyword] tìm kiếm tương tác theo tên, alias, user, host hoặc port
sshw -q [keyword] dạng rút gọn của -search
sshw -config     chỉnh sửa ~/.sshw.yaml từ web UI cục bộ
sshw -c          dạng rút gọn của -config
sshw -version    in phiên bản build và phiên bản Go
sshw -v          dạng rút gọn của -version
sshw -help       hiển thị các flag
sshw -h          dạng rút gọn của -help
```

## Ghi chú

- Không commit mật khẩu thật, khóa riêng hoặc thông tin host production vào repository công khai.
- Nếu bỏ trống `user`, `sshw` dùng `root`.
- Nếu bỏ trống `port`, `sshw` dùng `22`.
- Hỗ trợ jump host hiện dùng mục `jump` đầu tiên trong cấu hình.

## Giấy phép

[MIT](./LICENSE)
