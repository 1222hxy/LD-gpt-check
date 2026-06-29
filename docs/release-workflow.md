# GitHub Release 工作流

本项目通过 GitHub Actions 自动构建 CLI 二进制并发布到 GitHub Release。工作流文件在 `.github/workflows/release.yml`。

## 触发方式

推荐使用语义化 tag 触发：

```bash
git checkout main
git pull --ff-only
git tag -a v0.2.0 -m "Release v0.2.0"
git push origin v0.2.0
```

也可以在 GitHub Actions 页面手动运行 `Release CLI` workflow。手动运行时需要填写已经存在的 tag，例如 `v0.2.0`。工作流会校验 tag 存在，不会自动为任意提交创建 tag。

## 工作流做什么

1. 运行 `go test ./...`。
2. 用 tag 注入 CLI 版本号，例如 `v0.2.0` 会构建出 `ld-gpt-check version` 首行返回 `0.2.0`，并附带构建 commit 与最近提交记录。
3. 构建以下平台的裸二进制文件：

```text
ld-gpt-check_windows_amd64.exe
ld-gpt-check_windows_arm64.exe
ld-gpt-check_darwin_amd64
ld-gpt-check_darwin_arm64
ld-gpt-check_linux_amd64
ld-gpt-check_linux_arm64
ld-gpt-check_linux_armv7
ld-gpt-check_linux_armv6
```

4. 生成 `SHA256SUMS.txt`。
5. 创建或更新对应的 GitHub Release，并上传所有二进制文件和校验和文件。
6. 如果配置了 Cloudflare R2 secrets，则把同一批文件镜像到 R2。

## Release 文件

Release 不再上传 zip 或 tar.gz，用户直接下载对应系统的二进制文件：

- Windows 下载 `.exe` 后直接运行。
- Linux/macOS 下载后先执行 `chmod +x ld-gpt-check_*`，再运行对应文件。
- `SHA256SUMS.txt` 用于校验下载文件。

## 可选：同步到 Cloudflare R2

GitHub Release 是默认下载源。R2 只是可选镜像。需要在 GitHub repository secrets 中配置：

```text
CLOUDFLARE_R2_ACCOUNT_ID
CLOUDFLARE_R2_BUCKET
CLOUDFLARE_R2_ACCESS_KEY_ID
CLOUDFLARE_R2_SECRET_ACCESS_KEY
```

配置后，工作流会同步到：

```text
ld-gpt-check/<version>/
ld-gpt-check/latest/
```

例如 tag `v0.2.0` 会同步到 `ld-gpt-check/0.2.0/` 和 `ld-gpt-check/latest/`。

如果 R2 bucket 绑定了公开下载域名，还可以配置 repository variable：

```text
CLOUDFLARE_R2_PUBLIC_BASE_URL
```

当前 R2 下载域配置为：

```text
https://download.yhklab.com
```

把 R2 域名放到 README 或首页作为主下载源前，先确认二进制文件不会触发 Cloudflare challenge：

```bash
curl -I https://download.yhklab.com/ld-gpt-check/latest/ld-gpt-check_linux_amd64
```

如果响应里有 `cf-mitigated: challenge` 或状态码是 `403`，需要在 Cloudflare 给 `download.yhklab.com/ld-gpt-check/*` 跳过 challenge/安全规则；否则普通下载工具会拿到 challenge HTML，而不是二进制文件。

## 本地发布前检查

发布 tag 前建议至少执行：

```bash
go test ./...
```

如果改过前端、Dashboard 或 Worker，也要执行对应项目的构建和测试，再发布 tag。

## 回滚

如果 release 资产有误，可以修复代码后创建新的 patch tag，例如 `v0.2.1`。不建议复用已经公开传播的 tag；如果必须重发同一个 tag，工作流会覆盖同名 release assets，但使用者本地缓存可能仍然拿到旧文件。
