# Android Bundle 构建流程详细分析

## 📦 最终产物

```
build/homeocto-android-universal.zip
```

这是一个包含 Android 平台所有必需二进制文件的压缩包，供 Android 应用使用。

---

## 🔄 完整构建流程

### 步骤 1: 触发构建

在 GitHub Actions 中通过环境变量触发：
```yaml
INCLUDE_ANDROID_BUNDLE: "true"
```

GoReleaser 的 before hook 执行：
```bash
sh -c 'if [ "${INCLUDE_ANDROID_BUNDLE:-}" = "true" ]; then make build-android-bundle; fi'
```

### 步骤 2: 主 Makefile 入口

**文件**: `Makefile` (第 311-325 行)

```makefile
build-android-bundle: generate
	@echo "Building core for all Android architectures..."
	@mkdir -p $(BUILD_DIR)
	
	# 2.1 构建 homeocto 核心二进制文件 (ARM64)
	GOOS=android GOARCH=arm64 $(GO) build \
	  -tags stdjson \
	  -ldflags "$(LDFLAGS)" \
	  -o $(BUILD_DIR)/homeocto-android-arm64 \
	  ./cmd/homeocto
	
	# 2.2 构建 launcher (调用 web/Makefile)
	@echo "Building launcher for Android arm64..."
	@$(MAKE) build-launcher-android-arm64
```

**关键参数**:
- `GOOS=android`: 目标操作系统为 Android
- `GOARCH=arm64`: 目标架构为 ARM64 (64位)
- `-tags stdjson`: 使用标准 JSON 库
- `-ldflags "$(LDFLAGS)"`: 注入版本信息

### 步骤 3: 构建 Launcher

**文件**: `Makefile` (第 301-309 行)

```makefile
build-launcher-android-arm64:
	@echo "Building homeocto-launcher for android/arm64..."
	@mkdir -p $(BUILD_DIR)
	
	# 调用 web/Makefile 的 build-android-arm64 目标
	@$(MAKE) -C web build-android-arm64 \
		OUTPUT_ANDROID_ARM64="$(CURDIR)/$(BUILD_DIR)/homeocto-launcher-android-arm64" \
		GO='$(GO)' \
		LDFLAGS='$(LDFLAGS)'
```

**说明**:
- `-C web`: 切换到 `web/` 目录执行 Makefile
- `OUTPUT_ANDROID_ARM64`: 指定输出文件路径
- 传递 `GO` 和 `LDFLAGS` 变量保持一致性

### 步骤 4: Web 子模块构建

**文件**: `web/Makefile` (第 98-100 行)

```makefile
build-android-arm64: build-frontend
	@mkdir -p $(BUILD_DIR)
	GOOS=android GOARCH=arm64 $(GO) build \
	  -tags stdjson \
	  -ldflags "$(LDFLAGS)" \
	  -o "$(OUTPUT_ANDROID_ARM64)" \
	  ./backend/
```

**依赖**: `build-frontend` (必须先构建前端)

#### 步骤 4.1: 前端构建

**文件**: `web/Makefile` (第 108-119 行)

```makefile
build-frontend:
	# 检查依赖是否需要安装
	@expected_stamp="$$(cat $(FRONTEND_DIR)/package.json $(FRONTEND_DIR)/pnpm-lock.yaml | cksum | awk '{print $$1 ":" $$2}')"; \
	if [ ! -d $(FRONTEND_DIR)/node_modules ] || \
		[ ! -x $(FRONTEND_DIR)/node_modules/.bin/tsc ] || \
		[ ! -f $(FRONTEND_INSTALL_STAMP) ] || \
		[ "$$(cat $(FRONTEND_INSTALL_STAMP) 2>/dev/null)" != "$$expected_stamp" ]; then \
		echo "Installing frontend dependencies..."; \
		(cd $(FRONTEND_DIR) && CI=true pnpm install --frozen-lockfile) && \
		printf '%s\n' "$$expected_stamp" > $(FRONTEND_INSTALL_STAMP); \
	fi
	
	# 构建前端并输出到 backend/dist
	@echo "Building frontend..."
	@cd $(FRONTEND_DIR) && pnpm build:backend
```

**流程**:
1. 检查 `package.json` 和 `pnpm-lock.yaml` 是否变化
2. 如果需要，执行 `pnpm install --frozen-lockfile`
3. 执行 `pnpm build:backend` 构建前端资源
4. 前端产物输出到 `web/backend/dist/` 目录

**前端构建产物**:
```
web/backend/dist/
├── index.html
├── assets/
│   ├── index-xxx.js
│   ├── index-xxx.css
│   └── ...
└── ...
```

这些文件会被 Go 的 `embed.go` 嵌入到最终的二进制文件中。

### 步骤 5: 打包 Android Bundle

回到主 `Makefile` (第 318-325 行):

```makefile
	@echo "Staging JNI libs..."
	
	# 5.1 创建临时目录
	@rm -rf $(BUILD_DIR)/android-staging
	@mkdir -p $(BUILD_DIR)/android-staging/arm64-v8a
	
	# 5.2 复制 core 二进制文件 (重命名为 .so 库)
	@cp $(BUILD_DIR)/homeocto-android-arm64 \
	  $(BUILD_DIR)/android-staging/arm64-v8a/libpicoclaw.so
	
	# 5.3 复制 launcher 二进制文件 (重命名为 .so 库)
	@cp $(BUILD_DIR)/homeocto-launcher-android-arm64 \
	  $(BUILD_DIR)/android-staging/arm64-v8a/libpicoclaw-web.so
	
	# 5.4 打包为 zip
	@cd $(BUILD_DIR)/android-staging && \
	  zip -r ../picoclaw-android-universal.zip .
	
	# 5.5 清理临时目录
	@rm -rf $(BUILD_DIR)/android-staging
	
	@echo "All Android builds complete: $(BUILD_DIR)/homeocto-android-universal.zip"
```

---

## 📁 最终 Zip 文件结构

```
homeocto-android-universal.zip
└── arm64-v8a/
    ├── libhomeocto.so          # homeocto 核心二进制 (约 30-50MB)
    └── libhomeocto-web.so      # launcher web 后端 (约 30-50MB)
```

### 为什么命名为 `.so` 文件？

在 Android 中：
- `.so` 是 **Shared Object** (共享库) 文件
- Android 应用通过 **JNI (Java Native Interface)** 调用这些原生库
- Go 编译的 Android 二进制文件可以直接作为 `.so` 库使用
- `arm64-v8a` 是 Android 的 ARM64 架构目录名（标准命名）

---

## 🔗 完整调用链

```
GitHub Actions (release.yml / nightly.yml)
  ↓
GoReleaser before hook
  ↓
make build-android-bundle
  ↓
  ├─→ make generate (代码生成)
  ├─→ GOOS=android GOARCH=arm64 go build ./cmd/homeocto
  │     输出: build/homeocto-android-arm64
  │
  └─→ make build-launcher-android-arm64
        ↓
        make -C web build-android-arm64
          ↓
          ├─→ build-frontend
          │     ├─→ pnpm install (如果需要)
          │     └─→ pnpm build:backend
          │
          └─→ GOOS=android GOARCH=arm64 go build ./web/backend
                输出: build/homeocto-launcher-android-arm64
  
  ↓
打包为 zip:
  build/android-staging/arm64-v8a/libhomeocto.so
  build/android-staging/arm64-v8a/libhomeocto-web.so
  ↓
build/homeocto-android-universal.zip
```

---

## 🛠️ 构建参数详解

### LDFLAGS (链接标志)

```makefile
CONFIG_PKG=github.com/sipeed/picoclaw/pkg/config
LDFLAGS=-X $(CONFIG_PKG).Version=$(VERSION) \
        -X $(CONFIG_PKG).GitCommit=$(GIT_COMMIT) \
        -X $(CONFIG_PKG).BuildTime=$(BUILD_TIME) \
        -X $(CONFIG_PKG).GoVersion=$(GO_VERSION) \
        -s -w
```

**作用**:
- `-X`: 在编译时注入字符串变量值
- `-s`: 剥离符号表 (减小文件大小)
- `-w`: 剥离 DWARF 调试信息 (减小文件大小)

**注入的信息**:
| 变量 | 来源 | 示例值 |
|------|------|--------|
| Version | `git describe --tags` | `v0.1.0` 或 `v0.1.0-3-gabc1234` |
| GitCommit | `git rev-parse --short=8` | `abc12345` |
| BuildTime | 当前时间 | `2026-05-02T10:30:00+0800` |
| GoVersion | `go env GOVERSION` | `go1.25.9` |

### Build Tags

```makefile
-tags stdjson
```

**说明**:
- `stdjson`: 使用 Go 标准库 `encoding/json`
- 不使用 `goolm` (因为 Android 可能不需要本地 LLM 支持)

---

## ⚠️ 注意事项

### 1. 文件命名不一致问题

**当前问题**:
```makefile
# 在 Makefile 第 323 行
@cd $(BUILD_DIR)/android-staging && zip -r ../picoclaw-android-universal.zip .
```

**输出文件名**: `build/picoclaw-android-universal.zip` ❌

**应该是**: `build/homeocto-android-universal.zip` ✅

**需要修复的位置**:
- `Makefile` 第 323 行
- `.goreleaser.yaml` 第 207 行
- `.github/workflows/nightly.yml` 第 128 行

### 2. 架构支持

**当前仅支持**: `arm64-v8a` (64位 ARM)

**可能需要的其他架构**:
- `armeabi-v7a` (32位 ARM) - 旧设备
- `x86_64` (64位 x86) - 模拟器/平板
- `x86` (32位 x86) - 旧模拟器

### 3. 依赖要求

- ✅ Go 1.25.9+
- ✅ Node.js 22+
- ✅ pnpm 10.33.0+
- ✅ `zip` 命令 (Linux/macOS) 或 `7z` (Windows)
- ✅ Android NDK (不需要，因为 CGO_ENABLED=0)

---

## 🚀 本地测试构建

```bash
# 方法 1: 通过 Makefile
make build-android-bundle

# 方法 2: 直接调用
INCLUDE_ANDROID_BUNDLE=true goreleaser release --snapshot --clean

# 检查输出
ls -lh build/homeocto-android-universal.zip
unzip -l build/homeocto-android-universal.zip
```

---

## 📊 构建时间估算

| 步骤 | 预计时间 | 说明 |
|------|---------|------|
| 前端依赖安装 | 30-60s | 首次或依赖变化时 |
| 前端构建 | 20-40s | Vite/Rollup 构建 |
| Core 二进制编译 | 30-60s | 交叉编译到 Android |
| Launcher 编译 | 30-60s | 包含嵌入的前端资源 |
| 打包 zip | <5s | 压缩两个 .so 文件 |
| **总计** | **~2-4 分钟** | 缓存命中时更快 |

---

## 🔧 优化建议

1. **修复文件命名**: 将 `picoclaw-android-universal.zip` 改为 `homeocto-android-universal.zip`
2. **多架构支持**: 添加 `armeabi-v7a` 等架构
3. **并行构建**: 多个架构可以同时编译
4. **缓存优化**: 前端依赖可以使用 GitHub Actions 缓存

---

更新时间: 2026-05-02
