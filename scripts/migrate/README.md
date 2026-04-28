# Picoclaw 迁移脚本使用说明

## 功能
该脚本用于将 picoclaw 项目的代码迁移到 homeocto 项目，并自动替换关键词。

## 特点
- ✅ 使用 Go 语言处理文件，避免 PowerShell 替换导致的中文乱码问题
- ✅ 保持 UTF-8 编码，正确处理中文字符
- ✅ 智能跳过二进制文件和不需要处理的目录
- ✅ 按优先级替换关键词（长字符串优先）

## 替换规则
| 原字符串 | 新字符串 | 说明 |
|---------|---------|------|
| github.com/sipeed/picoclaw | github.com/home-ai-union/homeocto | 官方仓库路径 |
| github.com/picoclaw | github.com/home-ai-union/homeocto | 简写路径 |
| Picoclaw | Homeocto | 项目名称 |
| picoclaw | homeocto | 项目名称 |
| PICOCLAW | HOMEOCTO | 项目名称 |

### 不替换的内容（外部依赖包）
- `github.com/sipeed/picoclaw/pkg` - 这是外部依赖包，不会被替换

## 使用方法

### 方式一：直接运行 Go 脚本
```powershell
cd g:\code\homeocto
go run scripts/migrate/migrate-picoclaw.go G:\code\picoclaw G:\code\homeocto
```

### 方式二：使用 PowerShell 启动脚本
```powershell
cd g:\code\homeocto
.\scripts\migrate\migrate-picoclaw.ps1
```

## 迁移内容
1. `cmd/picoclaw` → `cmd/homeocto`
2. `web` → `web`

## 注意事项
- 迁移前请确保已备份目标目录的重要文件
- 脚本会覆盖目标目录中的同名文件
- 自动跳过 `node_modules`、`.git`、`vendor` 等目录
- 二进制文件（图片、字体等）不会执行替换，直接拷贝

### 暂时跳过的文件（不会覆盖）
- `src/components/app-header.tsx`
- `src/components/app-layout.tsx`
- `src/components/app-sidebar.tsx`
- `src/routeTree.gen.ts`

## 验证迁移
迁移完成后，建议执行以下检查：
1. 检查 Go 代码是否能正常编译
2. 检查前端项目是否能正常启动
3. 搜索是否还有遗漏的 picoclaw 关键词
4. 检查中文注释是否正常显示
