package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// 替换规则 - 按长度降序排列,确保长字符串优先匹配
var replacements = []struct {
	oldStr string
	newStr string
}{
	{"github.com/sipeed/picoclaw", "github.com/home-ai-union/homeocto"},
}

// cmd 目录专用的替换规则
var cmdReplacements = []struct {
	oldStr string
	newStr string
}{
	{"github.com/sipeed/picoclaw/cmd/picoclaw", "github.com/home-ai-union/homeocto/cmd/homeocto"},
	{"github.com/sipeed/picoclaw/cmd", "github.com/home-ai-union/homeocto/cmd"},
	{"github.com/sipeed/picoclaw", "github.com/home-ai-union/homeocto"},
}

// web/backend 目录专用的替换规则
var webBackendReplacements = []struct {
	oldStr string
	newStr string
}{
	{"github.com/sipeed/picoclaw/web/backend/", "github.com/home-ai-union/homeocto/web/backend/"},
	{"github.com/sipeed/picoclaw", "github.com/home-ai-union/homeocto"},
}

// 不替换的路径前缀(外部依赖包)
var skipReplacementPrefixes = []string{
	"github.com/sipeed/picoclaw/pkg", // 外部依赖包,保持原样
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

	// 检查 homeocto 是否有未提交的更改
	fmt.Println("=== Checking Git status for homeocto ===")
	if err := checkGitStatus(homeoctoRoot); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		fmt.Fprintf(os.Stderr, "Please commit or stash your changes before running migration.\n")
		os.Exit(1)
	}
	fmt.Println("✓ Git working directory is clean\n")

	// 1. 处理 cmd 目录
	if err := processCmdDir(picoclawRoot, homeoctoRoot); err != nil {
		fmt.Fprintf(os.Stderr, "Error processing cmd directory: %v\n", err)
		os.Exit(1)
	}

	// 2. 处理 web/backend 目录
	if err := processWebBackendDir(picoclawRoot, homeoctoRoot); err != nil {
		fmt.Fprintf(os.Stderr, "Error processing web/backend directory: %v\n", err)
		os.Exit(1)
	}

	// 3. 处理 web/frontend 目录
	if err := processWebFrontendDir(picoclawRoot, homeoctoRoot); err != nil {
		fmt.Fprintf(os.Stderr, "Error processing web/frontend directory: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("=== Migration completed successfully! ===")
}

// 检查 Git 工作目录是否有未提交的更改
func checkGitStatus(repoPath string) error {
	// 检查是否是 Git 仓库
	gitDir := filepath.Join(repoPath, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		fmt.Println("⚠ Warning: Not a Git repository, skipping check")
		return nil
	}

	// 执行 git status --porcelain 检查是否有未提交的更改
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to run git status: %w", err)
	}

	// 如果输出不为空，说明有未提交的更改
	if len(strings.TrimSpace(string(output))) > 0 {
		fmt.Println("⚠ Found uncommitted changes in homeocto:")
		fmt.Println(string(output))
		return fmt.Errorf("working directory has uncommitted changes")
	}

	return nil
}

// 处理 cmd 目录迁移
func processCmdDir(picoclawRoot, homeoctoRoot string) error {
	srcCmdDir := filepath.Join(picoclawRoot, "cmd", "picoclaw")
	dstCmdDir := filepath.Join(homeoctoRoot, "cmd", "homeocto")

	if _, err := os.Stat(srcCmdDir); os.IsNotExist(err) {
		fmt.Println("⚠ Warning: cmd/picoclaw not found in source\n")
		return nil
	}

	fmt.Println("=== Processing cmd/picoclaw -> cmd/homeocto ===")

	// 先删除 homeocto 的 cmd 目录下的所有内容
	fmt.Println("  🗑 Cleaning cmd directory in homeocto...")
	if err := cleanDirectory(filepath.Join(homeoctoRoot, "cmd")); err != nil {
		return fmt.Errorf("clean cmd directory: %w", err)
	}

	// 拷贝并替换
	if err := copyCmdWithReplace(srcCmdDir, dstCmdDir); err != nil {
		return fmt.Errorf("copy cmd directory: %w", err)
	}

	fmt.Println("✓ cmd directory copied and replaced successfully\n")
	return nil
}

// 处理 web/backend 目录迁移
func processWebBackendDir(picoclawRoot, homeoctoRoot string) error {
	srcBackendDir := filepath.Join(picoclawRoot, "web", "backend")
	dstBackendDir := filepath.Join(homeoctoRoot, "web", "backend")

	if _, err := os.Stat(srcBackendDir); os.IsNotExist(err) {
		fmt.Println("⚠ Warning: web/backend directory not found in source\n")
		return nil
	}

	fmt.Println("=== Processing web/backend -> web/backend ===")

	// 先删除 homeocto 的 web/backend 目录
	fmt.Println("  🗑 Cleaning web/backend directory in homeocto...")
	if err := os.RemoveAll(dstBackendDir); err != nil {
		return fmt.Errorf("remove web/backend directory: %w", err)
	}

	// 拷贝并替换
	if err := copyWebBackendWithReplace(srcBackendDir, dstBackendDir); err != nil {
		return fmt.Errorf("copy web/backend directory: %w", err)
	}

	fmt.Println("✓ web/backend directory copied and replaced successfully\n")
	return nil
}

// 处理 web/frontend 目录迁移
func processWebFrontendDir(picoclawRoot, homeoctoRoot string) error {
	srcFrontendDir := filepath.Join(picoclawRoot, "web", "frontend")
	dstFrontendDir := filepath.Join(homeoctoRoot, "web", "frontend")

	if _, err := os.Stat(srcFrontendDir); os.IsNotExist(err) {
		fmt.Println("⚠ Warning: web/frontend directory not found in source\n")
		return nil
	}

	fmt.Println("=== Processing web/frontend -> web/frontend ===")

	// 先删除 homeocto 的 web/frontend 目录
	fmt.Println("  🗑 Cleaning web/frontend directory in homeocto...")
	if err := os.RemoveAll(dstFrontendDir); err != nil {
		return fmt.Errorf("remove web/frontend directory: %w", err)
	}

	// 拷贝（frontend 不需要替换，直接拷贝）
	if err := copyDir(srcFrontendDir, dstFrontendDir); err != nil {
		return fmt.Errorf("copy web/frontend directory: %w", err)
	}

	fmt.Println("✓ web/frontend directory copied successfully\n")
	return nil
}

// 拷贝 web/backend 目录并执行专用替换
func copyWebBackendWithReplace(srcDir, dstDir string) error {
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
			return copyWebBackendTextFileWithReplace(path, targetPath)
		} else {
			return copyBinaryFile(path, targetPath)
		}
	})
}

// 拷贝 web/backend 文本文件并执行专用替换
func copyWebBackendTextFileWithReplace(src, dst string) error {
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
			// 如果不是UTF-8，直接复制原始内容
			if _, err := writer.WriteString(line); err != nil {
				return fmt.Errorf("write line to %s: %w", dst, err)
			}
		} else {
			// 检查是否包含需要跳过的路径前缀
			shouldSkip := false
			for _, prefix := range skipReplacementPrefixes {
				if strings.Contains(line, prefix) {
					shouldSkip = true
					break
				}
			}

			var replacedLine string
			if shouldSkip {
				replacedLine = line
			} else {
				// 使用 web/backend 专用替换规则
				replacedLine = line
				for _, rule := range webBackendReplacements {
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

// 通用目录拷贝（不执行替换）
func copyDir(srcDir, dstDir string) error {
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

		// 直接拷贝文件（不替换）
		return copyBinaryFile(path, targetPath)
	})
}

// 清理目录下的所有文件和子目录
func cleanDirectory(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			// 目录不存在，无需清理
			return nil
		}
		return fmt.Errorf("read directory %s: %w", dir, err)
	}

	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())
		if err := os.RemoveAll(path); err != nil {
			return fmt.Errorf("remove %s: %w", path, err)
		}
	}
	return nil
}

// 拷贝 cmd 目录并执行专用替换
func copyCmdWithReplace(srcDir, dstDir string) error {
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
			return copyCmdTextFileWithReplace(path, targetPath)
		} else {
			return copyBinaryFile(path, targetPath)
		}
	})
}

// 拷贝 cmd 文本文件并执行专用替换
func copyCmdTextFileWithReplace(src, dst string) error {
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
			// 如果不是UTF-8，直接复制原始内容
			if _, err := writer.WriteString(line); err != nil {
				return fmt.Errorf("write line to %s: %w", dst, err)
			}
		} else {
			// 检查是否包含需要跳过的路径前缀
			shouldSkip := false
			for _, prefix := range skipReplacementPrefixes {
				if strings.Contains(line, prefix) {
					shouldSkip = true
					break
				}
			}

			var replacedLine string
			if shouldSkip {
				replacedLine = line
			} else {
				// 使用 cmd 专用替换规则
				replacedLine = line
				for _, rule := range cmdReplacements {
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
		".tanstack",
		"onboard",
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
		".desktop": true,
	}

	return textExtensions[ext]
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
