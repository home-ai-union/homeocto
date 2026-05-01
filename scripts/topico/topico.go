package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// 配置结构体
type SyncConfig struct {
	// 源目录（homeocto）
	SrcDir string
	// 目标目录（picoclaw）
	DstDir string
	// 需要同步的文件列表
	Files []string
	// 需要同步的目录列表
	Dirs []string
	// 路径替换规则（用于处理不同目录名的情况，如 cmd/homeocto -> cmd/picoclaw）
	PathReplacements []PathReplacement
}

// 路径替换规则
type PathReplacement struct {
	SrcPrefix string // 源路径前缀
	DstPrefix string // 目标路径前缀
}

// 默认配置 - 需要同步的文件和目录
func getDefaultConfig() SyncConfig {
	return SyncConfig{
		Files: []string{
			"web\\frontend\\src\\components\\app-header.tsx",
			"web\\frontend\\src\\components\\app-layout.tsx",
			"web\\frontend\\src\\components\\app-sidebar.tsx",
			"web\\frontend\\src\\routeTree.gen.ts",
			"web\\frontend\\src\\i18n\\index.ts",
			"web\\frontend\\src\\homeocto\\api\\device-control-websocket.ts",
			"web\\backend\\utils\\runtime.go",
			"web\\backend\\api\\homeocto_api.go",
			"web\\backend\\api\\router.go",
			"web\\picoclaw-launcher.desktop",
			"cmd\\homeocto\\internal\\gateway\\command.go",
		},
		Dirs: []string{
			"web\\frontend\\src\\homeocto",
			"web\\frontend\\src\\i18n\\locales\\homeocto",
			"web\\frontend\\src\\routes\\smart-home",
			"web\\backend\\homeocto",
			"web\\backend\\api\\homeocto",
		},
		// 路径替换规则：cmd\homeocto -> cmd\picoclaw
		PathReplacements: []PathReplacement{
			{
				SrcPrefix: "cmd\\homeocto",
				DstPrefix: "cmd\\picoclaw",
			},
		},
	}
}

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: go run scripts/topico/topico.go <homeocto-root> <picoclaw-root> [config-file]\n")
		fmt.Fprintf(os.Stderr, "Example: go run scripts/topico/topico.go G:\\code\\homeocto G:\\code\\picoclaw\n")
		os.Exit(1)
	}

	homeoctoRoot := filepath.Clean(os.Args[1])
	picoclawRoot := filepath.Clean(os.Args[2])

	// 验证源目录存在
	if _, err := os.Stat(homeoctoRoot); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: Source directory does not exist: %s\n", homeoctoRoot)
		os.Exit(1)
	}

	// 验证目标目录存在
	if _, err := os.Stat(picoclawRoot); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: Target directory does not exist: %s\n", picoclawRoot)
		os.Exit(1)
	}

	// 加载配置
	config := loadConfig(homeoctoRoot, picoclawRoot)

	fmt.Printf("Source (homeocto): %s\n", homeoctoRoot)
	fmt.Printf("Target (picoclaw): %s\n\n", picoclawRoot)

	// 同步文件
	if err := syncFiles(config); err != nil {
		fmt.Fprintf(os.Stderr, "Error syncing files: %v\n", err)
		os.Exit(1)
	}

	// 同步目录
	if err := syncDirs(config); err != nil {
		fmt.Fprintf(os.Stderr, "Error syncing directories: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("=== Sync completed successfully! ===")
}

// 加载配置（可以从配置文件或默认配置加载）
func loadConfig(homeoctoRoot, picoclawRoot string) SyncConfig {
	config := getDefaultConfig()
	config.SrcDir = homeoctoRoot
	config.DstDir = picoclawRoot
	return config
}

// 应用路径替换规则
func applyPathReplacements(path string, replacements []PathReplacement) string {
	result := path
	for _, rule := range replacements {
		// 检查路径是否以源前缀开头
		if len(result) >= len(rule.SrcPrefix) && result[:len(rule.SrcPrefix)] == rule.SrcPrefix {
			// 替换前缀
			result = rule.DstPrefix + result[len(rule.SrcPrefix):]
		}
	}
	return result
}

// 同步指定的文件
func syncFiles(config SyncConfig) error {
	fmt.Println("=== Syncing files ===")

	for _, file := range config.Files {
		srcPath := filepath.Join(config.SrcDir, file)
		// 应用路径替换规则
		dstFile := applyPathReplacements(file, config.PathReplacements)
		dstPath := filepath.Join(config.DstDir, dstFile)

		if _, err := os.Stat(srcPath); os.IsNotExist(err) {
			fmt.Printf("⚠ Warning: Source file not found: %s\n", srcPath)
			continue
		}

		// 创建目标目录
		if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
			return fmt.Errorf("create directory %s: %w", filepath.Dir(dstPath), err)
		}

		// 直接覆盖文件
		if err := copyFile(srcPath, dstPath); err != nil {
			return fmt.Errorf("copy file %s: %w", file, err)
		}
		fmt.Printf("✓ Copied: %s -> %s\n", file, dstFile)
	}

	fmt.Println()
	return nil
}

// 同步指定的目录
func syncDirs(config SyncConfig) error {
	fmt.Println("=== Syncing directories ===")

	for _, dir := range config.Dirs {
		srcDir := filepath.Join(config.SrcDir, dir)
		dstDir := filepath.Join(config.DstDir, dir)

		if _, err := os.Stat(srcDir); os.IsNotExist(err) {
			fmt.Printf("⚠ Warning: Source directory not found: %s\n", srcDir)
			continue
		}

		// 删除旧目录
		if err := os.RemoveAll(dstDir); err != nil {
			return fmt.Errorf("remove old directory %s: %w", dstDir, err)
		}

		// 拷贝新目录
		if err := copyDir(srcDir, dstDir); err != nil {
			return fmt.Errorf("copy directory %s: %w", dir, err)
		}
		fmt.Printf("✓ Copied directory: %s\n", dir)
	}

	fmt.Println()
	return nil
}

// 判断是否应该跳过某个目录
func shouldSkipDirectory(name string) bool {
	skipDirs := []string{
		"node_modules",
		".git",
		"vendor",
		"dist",
		"build",
		".cache",
	}

	for _, skip := range skipDirs {
		if name == skip {
			return true
		}
	}
	return false
}

// 拷贝整个目录
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 跳过某些目录
		if info.IsDir() && shouldSkipDirectory(info.Name()) {
			return filepath.SkipDir
		}

		// 计算相对路径
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		targetPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(targetPath, info.Mode())
		}

		return copyFile(path, targetPath)
	})
}

// 拷贝单个文件
func copyFile(src, dst string) error {
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
