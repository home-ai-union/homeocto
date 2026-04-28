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
	{"github.com/sipeed/picoclaw/cmd/picoclaw", "github.com/home-ai-union/homeocto/cmd/homeocto"},
	{"github.com/sipeed/picoclaw", "github.com/home-ai-union/homeocto"},
}

// 不替换的路径前缀（外部依赖包）
var skipReplacementPrefixes = []string{
	"github.com/sipeed/picoclaw/pkg", // 外部依赖包，保持原样
}
var gitMergeFile = []string{
	"frontend/src/routeTree.gen.ts", // 前端文件使用 Git 合并
	"frontend/src/components/app-header.tsx",
	"frontend/src/components/app-layout.tsx",
	"frontend/src/components/page-header.tsx",
	"backend/utils/runtime.go", // 后端运行时文件
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
		".tanstack",
	}

	for _, skip := range skipDirs {
		if strings.Contains(relPath, skip) {
			return true
		}
	}
	return false
}

// 判断是否应该使用 Git 合并
func shouldUseGitMerge(relPath string) bool {
	// 统一转换为正斜杠，支持跨平台
	normalizedPath := filepath.ToSlash(relPath)

	for _, prefix := range gitMergeFile {
		// 也将配置中的路径统一为正斜杠
		normalizedPrefix := filepath.ToSlash(prefix)

		// 精确匹配或前缀匹配
		if normalizedPath == normalizedPrefix ||
			strings.HasPrefix(normalizedPath, normalizedPrefix) ||
			strings.Contains(normalizedPath, normalizedPrefix) {
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

// 拷贝目录并执行替换（支持 Git 合并）
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

		// 跳过某些文件（不处理）
		if shouldSkipFile(relPath) {
			return nil
		}

		// 检查目标文件是否已存在
		if _, err := os.Stat(targetPath); err == nil {
			// 目标文件已存在，检查是否需要 Git 合并
			if shouldUseGitMerge(relPath) && isTextFile(relPath) {
				// 使用 Git 合并
				return mergeTextFiles(path, targetPath)
			} else if isTextFile(relPath) {
				// 普通文本文件，直接覆盖（应用替换）
				fmt.Printf("  📝 Overwriting: %s\n", relPath)
				return copyTextFileWithReplace(path, targetPath)
			} else {
				// 二进制文件直接覆盖
				fmt.Printf("  ⚠ Overwriting binary file: %s\n", relPath)
				return copyBinaryFile(path, targetPath)
			}
		}

		// 目标文件不存在，直接复制并替换
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

// 使用简单的行级别合并
func mergeTextFiles(src, dst string) error {
	fmt.Printf("  🔄 Merging: %s\n", filepath.Base(dst))

	// 读取目标文件（当前版本）
	dstLines, err := readLines(dst)
	if err != nil {
		return fmt.Errorf("read destination file: %w", err)
	}

	// 读取源文件并应用替换
	srcContent, err := readFileWithReplace(src)
	if err != nil {
		return fmt.Errorf("read source file with replace: %w", err)
	}
	srcLines := strings.Split(srcContent, "\n")

	// 简单的合并策略：以源文件为基础，保留目标文件的独特修改
	merged := smartMerge(dstLines, srcLines)

	// 写入合并后的内容
	if err := os.WriteFile(dst, []byte(strings.Join(merged, "\n")), 0644); err != nil {
		return fmt.Errorf("write merged file: %w", err)
	}

	fmt.Printf("  ✓ Merged successfully: %s\n", filepath.Base(dst))
	return nil
}

// 读取文件行为数组
func readLines(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

// 智能合并：基于行的简单三路合并
// 策略：以 homeocto (dst) 为基础，保留其独特修改，同时应用 picoclaw (src) 的新行和替换
func smartMerge(dstLines, srcLines []string) []string {
	// 如果旧文件为空，直接返回新文件
	if len(dstLines) == 0 {
		return srcLines
	}

	// 如果新文件为空，保留旧文件
	if len(srcLines) == 0 {
		return dstLines
	}

	// 构建 dst 行的映射，用于快速查找
	dstLineSet := make(map[string]int)
	for i, line := range dstLines {
		dstLineSet[line] = i
	}

	// 构建 src 行的映射
	srcLineSet := make(map[string]int)
	for i, line := range srcLines {
		srcLineSet[line] = i
	}

	// 结果：以 dst 为基础
	result := make([]string, 0, len(dstLines))
	addedLines := make(map[int]bool) // 记录已添加的 src 行索引

	// 1. 遍历 dst 的所有行（保留 homeocto 的修改）
	for _, dstLine := range dstLines {
		result = append(result, dstLine)
	}

	// 2. 添加 src 中独有的行（picoclaw 的新内容）
	for i, srcLine := range srcLines {
		if _, exists := dstLineSet[srcLine]; !exists {
			// 这行在 dst 中不存在，是 src 的新内容
			if !addedLines[i] {
				result = append(result, srcLine)
				addedLines[i] = true
			}
		}
	}

	return result
}

// 读取文件并应用替换规则
func readFileWithReplace(src string) (string, error) {
	content, err := os.ReadFile(src)
	if err != nil {
		return "", err
	}

	result := string(content)
	for _, rule := range replacements {
		// 检查是否需要跳过替换
		shouldSkip := false
		for _, prefix := range skipReplacementPrefixes {
			if strings.Contains(result, prefix) {
				shouldSkip = true
				break
			}
		}
		if !shouldSkip {
			result = strings.ReplaceAll(result, rule.oldStr, rule.newStr)
		}
	}
	return result, nil
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
