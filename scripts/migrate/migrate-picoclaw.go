package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// 替换规则 - 按长度降序排列，确保长字符串优先匹配
var replacements = []struct {
	oldStr string
	newStr string
}{
	{"github.com/sipeed/picoclaw", "github.com/home-ai-union/homeocto"},
	{"Picoclaw", "Homeocto"},
	{"picoclaw", "homeocto"},
	{"PICOCLAW", "HOMEOCTO"},
}

// 不替换的路径前缀（外部依赖包）
var skipReplacementPrefixes = []string{
	"github.com/sipeed/picoclaw/pkg", // 外部依赖包，保持原样
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "Usage: go run scripts/migrate/migrate-picoclaw.go <picoclaw-root> <homeocto-root>\n")
		fmt.Fprintf(os.Stderr, "Example: go run scripts/migrate/migrate-picoclaw.go G:\\code\\picoclaw G:\\code\\homeocto\n")
		os.Exit(1)
	}

	picoclawRoot := filepath.Clean(os.Args[1])
	homeoctoRoot := filepath.Clean(os.Args[2])

	// 验证源目录存在
	if _, err := os.Stat(picoclawRoot); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: Source directory does not exist: %s\n", picoclawRoot)
		os.Exit(1)
	}

	// 验证目标目录存在
	if _, err := os.Stat(homeoctoRoot); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: Target directory does not exist: %s\n", homeoctoRoot)
		os.Exit(1)
	}

	fmt.Printf("Source (picoclaw): %s\n", picoclawRoot)
	fmt.Printf("Target (homeocto): %s\n\n", homeoctoRoot)

	// 1. 拷贝 cmd/picoclaw -> cmd/homeocto
	srcCmdDir := filepath.Join(picoclawRoot, "cmd", "picoclaw")
	dstCmdDir := filepath.Join(homeoctoRoot, "cmd", "homeocto")

	if _, err := os.Stat(srcCmdDir); !os.IsNotExist(err) {
		fmt.Println("=== Copying cmd/picoclaw -> cmd/homeocto ===")
		if err := copyAndReplace(srcCmdDir, dstCmdDir); err != nil {
			fmt.Fprintf(os.Stderr, "Error copying cmd directory: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✓ cmd directory copied successfully\n")
	} else {
		fmt.Println("⚠ Warning: cmd/picoclaw not found in source\n")
	}

	// 2. 拷贝 web -> web
	srcWebDir := filepath.Join(picoclawRoot, "web")
	dstWebDir := filepath.Join(homeoctoRoot, "web")

	if _, err := os.Stat(srcWebDir); !os.IsNotExist(err) {
		fmt.Println("=== Copying web -> web ===")
		if err := copyAndReplace(srcWebDir, dstWebDir); err != nil {
			fmt.Fprintf(os.Stderr, "Error copying web directory: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✓ web directory copied successfully\n")
	} else {
		fmt.Println("⚠ Warning: web directory not found in source\n")
	}

	fmt.Println("=== Migration completed successfully! ===")
}

// 判断是否应该跳过某个目录
func shouldSkipDirectory(relPath string) bool {
	skipDirs := []string{
		"node_modules",
		".git",
		"vendor",
		"dist",
		"build",
		".cache",
		".next",
		".turbo",
	}

	for _, skip := range skipDirs {
		if strings.Contains(relPath, skip) {
			return true
		}
	}
	return false
}

// 判断是否应该跳过某个文件
func shouldSkipFile(relPath string) bool {
	// 跳过的文件列表（完整路径匹配）
	skipFilePaths := []string{
		"src/components/app-header.tsx",
		"src/components/app-layout.tsx",
		"src/components/app-sidebar.tsx",
		"src/routeTree.gen.ts",
	}

	// 检查完整路径匹配
	for _, skip := range skipFilePaths {
		if relPath == skip || strings.HasSuffix(relPath, "\\"+skip) || strings.HasSuffix(relPath, "/"+skip) {
			return true
		}
	}

	// 跳过的文件名（全局匹配）
	skipFiles := []string{
		".DS_Store",
		"Thumbs.db",
		".env.local",
	}

	filename := filepath.Base(relPath)
	for _, skip := range skipFiles {
		if filename == skip {
			return true
		}
	}
	return false
}

// 判断是否为文本文件
func isTextFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	textExtensions := map[string]bool{
		".go":      true,
		".mod":     true,
		".sum":     true,
		".ts":      true,
		".tsx":     true,
		".js":      true,
		".jsx":     true,
		".json":    true,
		".css":     true,
		".scss":    true,
		".html":    true,
		".md":      true,
		".yaml":    true,
		".yml":     true,
		".toml":    true,
		".sh":      true,
		".bat":     true,
		".ps1":     true,
		".psm1":    true,
		".env":     true,
		".txt":     true,
		".sql":     true,
		".xml":     true,
		".svg":     true,
		".graphql": true,
	}

	return textExtensions[ext]
}

// 拷贝目录并执行替换
func copyAndReplace(srcDir, dstDir string) error {
	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 计算相对路径
		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		// 目标路径
		targetPath := filepath.Join(dstDir, relPath)

		if info.IsDir() {
			// 跳过某些目录
			if shouldSkipDirectory(relPath) {
				return filepath.SkipDir
			}
			// 创建目录
			return os.MkdirAll(targetPath, info.Mode())
		}

		// 跳过某些文件
		if shouldSkipFile(relPath) {
			return nil
		}

		// 处理文件
		if isTextFile(relPath) {
			return copyTextFileWithReplace(path, targetPath)
		} else {
			return copyBinaryFile(path, targetPath)
		}
	})
}

// 拷贝文本文件并执行替换（保持 UTF-8 编码，避免中文乱码）
func copyTextFileWithReplace(src, dst string) error {
	// 打开源文件
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open source file %s: %w", src, err)
	}
	defer srcFile.Close()

	// 创建目标目录
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("create directory %s: %w", filepath.Dir(dst), err)
	}

	// 创建目标文件
	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create destination file %s: %w", dst, err)
	}
	defer dstFile.Close()

	// 使用带缓冲的读写器
	reader := bufio.NewReader(srcFile)
	writer := bufio.NewWriter(dstFile)
	defer writer.Flush()

	// 逐行读取并替换
	for {
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return fmt.Errorf("read line from %s: %w", src, err)
		}

		// 检查是否为有效的UTF-8
		if !utf8.ValidString(line) {
			// 如果不是UTF-8，直接复制原始内容（不替换）
			if _, err := writer.WriteString(line); err != nil {
				return fmt.Errorf("write line to %s: %w", dst, err)
			}
		} else {
			// 检查是否包含需要跳过的路径前缀（如外部依赖包）
			shouldSkip := false
			for _, prefix := range skipReplacementPrefixes {
				if strings.Contains(line, prefix) {
					shouldSkip = true
					break
				}
			}

			var replacedLine string
			if shouldSkip {
				// 跳过替换，保持原样
				replacedLine = line
			} else {
				// 执行替换（按顺序执行，确保长字符串优先）
				replacedLine = line
				for _, rule := range replacements {
					replacedLine = strings.ReplaceAll(replacedLine, rule.oldStr, rule.newStr)
				}
			}

			if _, err := writer.WriteString(replacedLine); err != nil {
				return fmt.Errorf("write line to %s: %w", dst, err)
			}
		}

		if err == io.EOF {
			break
		}
	}

	return nil
}

// 拷贝二进制文件（不执行替换）
func copyBinaryFile(src, dst string) error {
	// 创建目标目录
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("create directory %s: %w", filepath.Dir(dst), err)
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open source file %s: %w", src, err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create destination file %s: %w", dst, err)
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return fmt.Errorf("copy file %s -> %s: %w", src, dst, err)
	}

	return nil
}
