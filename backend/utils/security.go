package utils

import (
	"path/filepath"
	"regexp"
	"strings"
)

// 用于验证安全路径组件的正则表达式
var (
	// 允许字母、数字、中文、下划线、短横线、点（不在开头）、空格
	safePathPattern = regexp.MustCompile(`^[a-zA-Z0-9\p{Han}_\- ]+(\.[a-zA-Z0-9]+)?$`)
	// 危险模式
	dangerousPatterns = []string{"..", "/", "\\", "\x00"}
)

// ValidatePathComponent 验证路径组件是否安全（用于项目名、文件名等）
// 返回true表示安全，false表示不安全
func ValidatePathComponent(name string) bool {
	if name == "" {
		return false
	}

	// 检查危险字符
	for _, pattern := range dangerousPatterns {
		if strings.Contains(name, pattern) {
			return false
		}
	}

	// 检查是否以点开头（隐藏文件）
	if strings.HasPrefix(name, ".") {
		return false
	}

	// 使用正则验证允许的字符
	if !safePathPattern.MatchString(name) {
		return false
	}

	// 清理后的路径应该与原始路径相同（防止路径遍历）
	cleaned := filepath.Clean(name)
	if cleaned != name || cleaned == "." || cleaned == ".." {
		return false
	}

	return true
}

// SanitizeProjectName 清理项目名称，使其可安全用于文件路径
// 如果名称包含危险字符，返回错误
func SanitizeProjectName(name string) (string, bool) {
	name = strings.TrimSpace(name)

	if !ValidatePathComponent(name) {
		return "", false
	}

	return name, true
}

// ValidateFileName 验证文件名是否安全
func ValidateFileName(filename string) bool {
	if filename == "" {
		return false
	}

	// 提取基础文件名（去掉路径）
	base := filepath.Base(filename)
	if base != filename {
		// 如果包含路径分隔符，不安全
		return false
	}

	return ValidatePathComponent(filename)
}
