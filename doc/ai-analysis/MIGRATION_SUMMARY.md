# GitHub Actions 配置迁移总结

## ✅ 已完成的修改

### 1. Workflow 文件修改

#### `.github/workflows/pr.yml`
- ✅ 移除了 picoclaw 特定的 build tags 参数
- ✅ 保留了 lint、安全扫描和测试流程

#### `.github/workflows/release.yml`
- ✅ 移除了 Android Bundle 构建配置
- ✅ 移除了 macOS 签名和公证配置（如需要可后续添加）
- ✅ 保留了核心的 GoReleaser 发布流程

#### `.github/workflows/nightly.yml`
- ✅ 移除了 Android Bundle 配置
- ✅ 移除了 macOS 签名配置
- ✅ 移除了 `build/picoclaw-android-universal.zip` 文件引用
- ✅ 保留了夜间构建核心流程

#### `.github/workflows/docker-build.yml`
- ✅ 镜像名称从 `picoclaw` 改为 `homeocto`

#### `.github/workflows/upload-tos.yml`
- ✅ TOS bucket 从 `picoclaw-downloads` 改为 `homeocto-downloads`

### 2. GoReleaser 配置

#### `.goreleaser.yaml` (新建)
- ✅ 模块路径: `github.com/sipeed/picoclaw` → `github.com/home-ai-union/homeocto`
- ✅ 主程序入口: `./cmd/picoclaw` → `./cmd/homeocto`
- ✅ 启动器入口: `./web/backend` (保持不变)
- ✅ 移除了 `picoclaw-launcher-tui` 构建目标
- ✅ 构建 ID 重命名:
  - `picoclaw` → `homeocto`
  - `picoclaw-launcher` → `homeocto-launcher`
- ✅ Docker 镜像名称更新为 `homeocto`
- ✅ Linux 包管理器配置更新 (rpm/deb)
- ✅ 桌面文件引用更新

### 3. Docker 文件修改

#### `docker/Dockerfile.goreleaser`
- ✅ 二进制文件路径: `picoclaw` → `homeocto`

#### `docker/Dockerfile.goreleaser.launcher`
- ✅ 二进制文件路径更新:
  - `picoclaw` → `homeocto`
  - `picoclaw-launcher` → `homeocto-launcher`
- ✅ 移除了 `picoclaw-launcher-tui` 引用
- ✅ 启动命令更新为 `homeocto-launcher`

### 4. Issue 模板修改

#### `.github/ISSUE_TEMPLATE/bug_report.md`
- ✅ 版本名称: `PicoClaw Version` → `HomeOcto Version`
- ✅ Go 版本示例: `go 1.22` → `go 1.25`
- ✅ 操作系统示例: 移除 `Android Termux`，添加 `Windows 11`

### 5. 桌面文件

#### `web/homeocto-launcher.desktop` (新建)
- ✅ Exec 命令: `picoclaw-launcher` → `homeocto-launcher`
- ✅ Icon 名称: `picoclaw-launcher` → `homeocto-launcher`
- ✅ Keywords: `picoclaw` → `homeocto`

### 6. Windows 资源文件

#### `web/backend/winres/winres.json`
- ✅ 应用名称: `PicoClaw Launcher` → `HomeOcto Launcher`
- ✅ 描述: `PicoClaw Launcher` → `HomeOcto Launcher`

## 📋 配置清单

### 已配置的工作流
1. ✅ **PR 检查** (`pr.yml`) - 自动 lint、安全扫描、测试
2. ✅ **主分支构建** (`build.yml`) - main 分支自动构建
3. ✅ **版本发布** (`release.yml`) - 手动触发正式发布
4. ✅ **夜间构建** (`nightly.yml`) - 每日自动构建
5. ✅ **Docker 构建** (`docker-build.yml`) - 可复用的 Docker 镜像构建
6. ✅ **TOS 上传** (`upload-tos.yml`) - 上传到火山引擎 TOS
7. ✅ **DMG 创建** (`create_dmg.yml`) - macOS 安装包打包

### 依赖管理
- ✅ **Dependabot** (`dependabot.yml`) - 每周自动更新 Go/npm/Actions 依赖

### 模板
- ✅ **Issue 模板** (3个) - Bug 报告、功能请求、任务跟踪
- ✅ **PR 模板** - 标准化的 PR 描述模板

## 🔧 使用前准备

### 必需的 GitHub Secrets
在 GitHub 仓库设置 → Secrets and variables → Actions 中配置：

1. **DockerHub 推送** (可选)
   - `DOCKERHUB_USERNAME` - DockerHub 用户名
   - `DOCKERHUB_TOKEN` - DockerHub 访问令牌

2. **火山引擎 TOS** (可选)
   - `VOLC_TOS_ACCESS_KEY` - TOS 访问密钥
   - `VOLC_TOS_SECRET_KEY` - TOS _SECRET 密钥

3. **macOS 签名** (可选，如需 macOS 代码签名)
   - `MACOS_SIGN_P12` - P12 证书
   - `MACOS_SIGN_PASSWORD` - 证书密码
   - `MACOS_NOTARY_ISSUER_ID` - 公证 Issuer ID
   - `MACOS_NOTARY_KEY_ID` - 公证 Key ID
   - `MACOS_NOTARY_KEY` - 公证 Key

### 必需的 GitHub Variables
- `DOCKERHUB_REPOSITORY` - DockerHub 仓库名称 (如 `yourorg/homeocto`)

## 🚀 使用方法

### 触发 PR 检查
创建 Pull Request 时自动触发

### 手动触发版本发布
1. 进入 Actions → Create Tag and Release
2. 点击 "Run workflow"
3. 输入版本号 (如 `v0.1.0`)
4. 选择是否标记为 pre-release 或 draft
5. 选择是否上传到 TOS
6. 点击运行

### 手动触发夜间构建
1. 进入 Actions → Nightly Build
2. 点击 "Run workflow"

### 手动触发 macOS DMG 构建
1. 进入 Actions → Create macOS DMG
2. 点击 "Run workflow"

## 📝 注意事项

1. **构建标签**: 当前配置使用 `goolm,stdjson` 标签，如不需要可在 `.goreleaser.yaml` 和 `pr.yml` 中移除
2. **Android 构建**: 已移除 Android 相关配置，如需要可参考 picoclaw 原配置添加
3. **macOS 签名**: 已移除 macOS 代码签名配置，正式发布时建议添加
4. **TOS 上传**: 如不使用火山引擎 TOS，可在 release.yml 中设置 `upload_tos: false`
5. **Docker 镜像**: 需要先在 DockerHub 创建对应的仓库

## 🔄 后续优化建议

1. 添加代码覆盖率报告
2. 配置自动化版本语义化检查
3. 添加 release notes 自动生成
4. 配置 Slack/Discord 通知
5. 添加性能基准测试
6. 配置自动化 E2E 测试

---

迁移完成时间: 2026-05-02
源项目: picoclaw (github.com/sipeed/picoclaw)
目标项目: homeocto (github.com/home-ai-union/homeocto)
