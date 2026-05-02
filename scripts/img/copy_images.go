package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// 定义需要拷贝的图片文件列表（相对于项目根目录的路径）
var imageFiles = []string{
	"web/backend/dist/apple-touch-icon.png",
	"web/backend/dist/favicon-96x96.png",
	"web/backend/dist/favicon.ico",
	"web/backend/dist/favicon.svg",
	"web/backend/dist/lark.svg",
	"web/backend/dist/logo_with_text.png",
	"web/backend/dist/web-app-manifest-192x192.png",
	"web/backend/dist/web-app-manifest-512x512.png",
	"web/backend/icon.ico",
	"web/backend/icon.png",
	"web/frontend/public/apple-touch-icon.png",
	"web/frontend/public/favicon-96x96.png",
	"web/frontend/public/favicon.ico",
	"web/frontend/public/favicon.svg",
	"web/frontend/public/lark.svg",
	"web/frontend/public/logo_with_text.png",
	"web/frontend/public/web-app-manifest-192x192.png",
	"web/frontend/public/web-app-manifest-512x512.png",
	"web/picoclaw-launcher.png",
}

func main() {
	// 检查命令行参数
	reverse := false
	if len(os.Args) > 1 && (os.Args[1] == "--reverse" || os.Args[1] == "-r") {
		reverse = true
	}

	// 源目录和目标目录
	var projectRoot, targetDir string
	if reverse {
		// 反向拷贝：从 homeocto 拷贝到 imgbak
		projectRoot = `G:\code\homeocto`
		targetDir = `G:\code\imgbak`
		fmt.Println("[模式] 反向拷贝：从 homeocto -> imgbak")
	} else {
		// 默认拷贝：从 imgbak 拷贝回 homeocto
		projectRoot = `G:\code\imgbak`
		targetDir = `G:\code\homeocto`
		fmt.Println("[模式] 默认拷贝：从 imgbak -> homeocto")
	}

	// 创建目标目录
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		fmt.Printf("创建目标目录失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("项目根目录: %s\n", projectRoot)
	fmt.Printf("目标目录: %s\n", targetDir)
	fmt.Println(strings.Repeat("-", 50))

	successCount := 0
	failCount := 0

	for _, relPath := range imageFiles {
		var srcPath, dstPath string
		fileName := filepath.Base(relPath)

		if reverse {
			// 反向拷贝（homeocto -> imgbak）：从 web 目录的原始路径读取，拷贝到 imgbak 根目录
			srcPath = filepath.Join(projectRoot, relPath)
			dstPath = filepath.Join(targetDir, fileName)
		} else {
			// 默认拷贝（imgbak -> homeocto）：从 imgbak 根目录读取平铺文件，恢复到 web 目录的原始路径
			srcPath = filepath.Join(projectRoot, fileName)
			dstPath = filepath.Join(targetDir, relPath)
			// 确保目标目录存在
			dstDir := filepath.Dir(dstPath)
			if err := os.MkdirAll(dstDir, 0755); err != nil {
				fmt.Printf("[失败] 创建目录失败: %s\n  错误: %v\n", dstDir, err)
				failCount++
				continue
			}
		}

		// 检查源文件是否存在
		if _, err := os.Stat(srcPath); os.IsNotExist(err) {
			fmt.Printf("[跳过] 源文件不存在: %s\n", relPath)
			continue
		}

		// 拷贝文件
		if err := copyFile(srcPath, dstPath); err != nil {
			fmt.Printf("[失败] 拷贝文件失败: %s\n  错误: %v\n", srcPath, err)
			failCount++
		} else {
			if reverse {
				fmt.Printf("[成功] %s -> %s\n", relPath, fileName)
			} else {
				fmt.Printf("[成功] %s -> %s\n", fileName, relPath)
			}
			successCount++
		}
	}

	fmt.Println(strings.Repeat("-", 50))
	fmt.Printf("拷贝完成! 成功: %d, 失败: %d\n", successCount, failCount)

	if failCount > 0 {
		os.Exit(1)
	}
}

// copyFile 拷贝单个文件
func copyFile(src, dst string) error {
	// 打开源文件
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("打开源文件失败: %w", err)
	}
	defer srcFile.Close()

	// 创建目标文件
	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("创建目标文件失败: %w", err)
	}
	defer dstFile.Close()

	// 拷贝内容
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return fmt.Errorf("拷贝内容失败: %w", err)
	}

	// 同步到磁盘
	err = dstFile.Sync()
	if err != nil {
		return fmt.Errorf("同步文件失败: %w", err)
	}

	return nil
}
