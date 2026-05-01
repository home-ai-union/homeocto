# Homeocto to Picoclaw Sync Tool

## 功能
该工具用于将 homeocto 项目的指定文件和目录同步到 picoclaw 项目。

## 特点
- ✅ 支持配置文件列表和文件夹列表
- ✅ 文件直接覆盖，不做任何替换
- ✅ 文件夹先删除旧的，再拷贝新的
- ✅ 智能跳过不需要的目录（node_modules、.git 等）
- ✅ 简单高效，无复杂逻辑

## 使用方法

### 方式一：直接运行 Go 脚本
```powershell
cd g:\code\homeocto
go run scripts/topico/topico.go G:\code\homeocto G:\code\picoclaw
```

### 方式二：使用 PowerShell 启动脚本
```powershell
cd g:\code\homeocto
.\scripts\topico\topico.ps1
```

## 配置说明

### 默认配置
工具默认配置为空，你可以在 `topico.go` 中的 `getDefaultConfig()` 函数中自定义需要同步的文件和目录：

```go
func getDefaultConfig() SyncConfig {
    return SyncConfig{
        Files: []string{
            // 添加需要同步的文件（相对于源目录的路径）
            "pkg/third/clients.go",
            "pkg/config/config.go",
        },
        Dirs: []string{
            // 添加需要同步的目录（相对于源目录的路径）
            "cmd/homeocto/internal/agent",
            "pkg/third/miio",
        },
    }
}
```

### 配置示例
```go
func getDefaultConfig() SyncConfig {
    return SyncConfig{
        Files: []string{
            "pkg/data/types.go",
            "pkg/event/center.go",
        },
        Dirs: []string{
            "cmd/homeocto/internal/skills",
            "pkg/workflow",
        },
    }
}
```

## 注意事项
- ⚠️ 同步前请务必备份目标目录的重要文件
- ⚠️ 文件会直接覆盖，不做任何合并
- ⚠️ 文件夹会先删除旧的，再拷贝新的
- 自动跳过 `node_modules`、`.git`、`vendor`、`dist`、`build`、`.cache` 等目录

## 验证同步
同步完成后，建议执行以下检查：
1. 检查 Go 代码是否能正常编译
2. 检查前端项目是否能正常启动
3. 搜索是否还有遗漏的 homeocto 关键词
4. 检查中文注释是否正常显示