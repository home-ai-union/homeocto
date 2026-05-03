# Android 构建问题修复说明

## 问题描述

在使用 Go 1.25+ 进行 Android 交叉编译时,出现以下链接错误:

```
link: github.com/wlynxg/anet: invalid reference to net.zoneCache
```

## 问题原因

### 依赖引入路径

`github.com/wlynxg/anet` 库通过以下依赖链引入:

- `github.com/AlexxIT/go2rtc` → `github.com/wlynxg/anet@v0.0.5`
- `github.com/pion/ice/v4` → `github.com/wlynxg/anet@v0.0.5`
- `github.com/pion/stun/v3` → `github.com/wlynxg/anet@v0.0.5`
- `github.com/pion/transport/v4` → `github.com/wlynxg/anet@v0.0.5`
- `github.com/pion/webrtc/v4` → `github.com/wlynxg/anet@v0.0.5`

### 根本原因

`anet` 库使用了 `//go:linkname` 机制来访问 Go 标准库的内部符号 `net.zoneCache`,以解决 Android 环境下的网络接口访问权限问题(Go issues #68082, #40569)。

从 **Go 1.23.0** 开始,Go 团队限制了 `//go:linkname` 的使用。如果不显式允许,链接器会拒绝访问内部符号,导致编译失败。

## 解决方案

### 方法:添加链接器标志

在编译 Android 版本时,添加 `-checklinkname=0` 标志到 ldflags:

```bash
go build -ldflags "-checklinkname=0 ..." ...
```

这个标志告诉链接器允许 `//go:linkname` 引用内部符号,从而解决兼容性问题。

## 已修改的文件

### 1. Makefile (根目录)

- `build-android-arm64`: 添加 `-checklinkname=0` 标志
- `build-android-bundle`: 添加 `-checklinkname=0` 标志

### 2. web/Makefile

- `build-android-arm64`: 添加 `-checklinkname=0` 标志
- `build-android-bundle`: 添加 `-checklinkname=0` 标志

### 3. .github/workflows/nightly.yml

- 将 `INCLUDE_ANDROID_BUNDLE` 从 `"false"` 改为 `"true"`
- 更新注释说明已修复兼容性问题

### 4. .github/workflows/release.yml

- 将 `INCLUDE_ANDROID_BUNDLE` 从 `"false"` 改为 `"true"`
- 更新注释说明已修复兼容性问题

## 验证结果

本地编译测试成功:

- ✅ Core binary (homeocto-android-arm64): 62.75 MB
- ✅ Launcher binary (homeocto-launcher-android-arm64): 22.31 MB

## 参考链接

- anet 项目: https://github.com/wlynxg/anet
- Go Issue #68082: https://github.com/golang/go/issues/68082
- Go Issue #40569: https://github.com/golang/go/issues/40569
- Go 1.23 Release Notes: https://go.dev/doc/go1.23

## 注意事项

1. `-checklinkname=0` 标志仅用于 Android 构建,其他平台不需要
2. 这个标志允许访问 Go 内部符号,理论上在 Go 未来版本中可能会有变化
3. 如果 anet 库更新以兼容新的 Go 版本,可能需要重新评估此解决方案
