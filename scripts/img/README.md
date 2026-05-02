# 图片拷贝工具

这个工具用于将 web 目录中的所有图片文件拷贝到项目根目录的 `img` 文件夹中。

## 使用方法

### 方法一：使用 PowerShell 脚本（推荐）

#### 默认拷贝（从 imgbak 到 homeocto）

```powershell
.\copy_images.ps1
```

将 `G:\code\imgbak` 中的平铺图片文件拷贝到 `G:\code\homeocto\web` 目录的对应位置，自动恢复目录结构。

#### 反向拷贝（从 homeocto 到 imgbak）

```powershell
.\copy_images.ps1 --reverse
# 或使用短参数
.\copy_images.ps1 -r
```

将 `G:\code\homeocto\web` 目录中的图片文件扁平化拷贝到 `G:\code\imgbak` 根目录。

脚本会自动：
1. 保存当前目录
2. 检查 Go 环境
3. 编译 Go 程序
4. 执行拷贝操作
5. 清理临时文件
6. 恢复原始目录

### 方法二：直接运行 Go 程序

#### 默认拷贝

```powershell
go run copy_images.go
```

#### 反向拷贝

```powershell
go run copy_images.go --reverse
```

## 功能说明

- **默认拷贝**：将 `G:\code\imgbak` 根目录中的平铺图片文件拷贝回 `G:\code\homeocto\web` 目录的原始位置，自动创建所需的目录结构
- **反向拷贝**：将 `G:\code\homeocto\web` 目录中的所有图片文件扁平化拷贝到 `G:\code\imgbak` 根目录
- 如果文件名冲突，后面的文件会覆盖前面的文件
- 显示详细的拷贝进度和结果统计
- 脚本执行前后保持当前工作目录不变

## 当前包含的图片

工具会拷贝以下路径的图片文件：

### web/backend/dist/
- apple-touch-icon.png
- favicon-96x96.png
- favicon.ico
- favicon.svg
- lark.svg
- logo_with_text.png
- web-app-manifest-192x192.png
- web-app-manifest-512x512.png

### web/backend/
- icon.ico
- icon.png

### web/frontend/public/
- apple-touch-icon.png
- favicon-96x96.png
- favicon.ico
- favicon.svg
- lark.svg
- logo_with_text.png
- web-app-manifest-192x192.png
- web-app-manifest-512x512.png

### web/
- picoclaw-launcher.png

## 注意事项

- 如果源文件不存在，程序会跳过该文件并继续处理
- 目标目录会自动创建（如果不存在）
- 程序会显示成功和失败的文件数量统计
- **默认拷贝**：从 `G:\code\imgbak` 根目录读取平铺文件，恢复到 `G:\code\homeocto\web` 目录的原始路径结构中
- **反向拷贝**：从 `G:\code\homeocto\web` 目录读取文件，扁平化拷贝到 `G:\code\imgbak` 根目录
- 脚本执行完成后，当前工作目录保持不变
