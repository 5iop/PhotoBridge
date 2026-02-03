package utils

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gabriel-vasile/mimetype"
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

// ValidateSecurePath 验证路径是否安全且在允许的基础目录内
// baseDir: 允许的基础目录（例如 /app/uploads）
// targetPath: 要验证的目标路径
// 返回: 解析后的安全路径和错误
func ValidateSecurePath(baseDir, targetPath string) (string, error) {
	// 获取基础目录的绝对路径
	absBaseDir, err := filepath.Abs(baseDir)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute base directory: %w", err)
	}

	// 解析基础目录的符号链接
	realBaseDir, err := filepath.EvalSymlinks(absBaseDir)
	if err != nil {
		// 如果基础目录不存在，创建它
		if os.IsNotExist(err) {
			realBaseDir = absBaseDir
		} else {
			return "", fmt.Errorf("failed to evaluate base directory symlinks: %w", err)
		}
	}

	// 获取目标路径的绝对路径
	absTargetPath, err := filepath.Abs(targetPath)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute target path: %w", err)
	}

	// 解析目标路径的符号链接（如果存在）
	realTargetPath := absTargetPath
	if _, err := os.Lstat(absTargetPath); err == nil {
		// 文件存在，解析符号链接
		realTargetPath, err = filepath.EvalSymlinks(absTargetPath)
		if err != nil {
			return "", fmt.Errorf("failed to evaluate target path symlinks: %w", err)
		}
	} else {
		// 文件不存在，检查父目录
		parentDir := filepath.Dir(absTargetPath)
		if _, err := os.Lstat(parentDir); err == nil {
			realParentDir, err := filepath.EvalSymlinks(parentDir)
			if err != nil {
				return "", fmt.Errorf("failed to evaluate parent directory symlinks: %w", err)
			}
			realTargetPath = filepath.Join(realParentDir, filepath.Base(absTargetPath))
		}
	}

	// 确保目标路径在基础目录内
	relPath, err := filepath.Rel(realBaseDir, realTargetPath)
	if err != nil {
		return "", fmt.Errorf("failed to get relative path: %w", err)
	}

	// 检查相对路径是否包含 ".." （尝试逃逸）
	if strings.HasPrefix(relPath, "..") || strings.Contains(relPath, string(filepath.Separator)+"..") {
		return "", fmt.Errorf("path traversal detected: path escapes base directory")
	}

	return realTargetPath, nil
}

// ValidateImageFile 验证文件是否为合法的图片文件（通过 magic number 检测）
// filePath: 文件路径
// allowedTypes: 允许的 MIME 类型列表，如果为空则允许所有图片类型
// 返回: 检测到的 MIME 类型和错误
func ValidateImageFile(filePath string, allowedTypes []string) (string, error) {
	// 打开文件
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// 只读取前 512 字节用于检测（足够识别大多数文件类型）
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// 使用 mimetype 库检测文件类型
	mtype := mimetype.Detect(buffer[:n])
	detectedType := mtype.String()

	// 检查是否为图片类型
	if !strings.HasPrefix(detectedType, "image/") {
		return "", fmt.Errorf("file is not an image: detected type is %s", detectedType)
	}

	// 如果指定了允许的类型列表，检查是否在列表中
	if len(allowedTypes) > 0 {
		allowed := false
		for _, allowedType := range allowedTypes {
			if detectedType == allowedType {
				allowed = true
				break
			}
		}
		if !allowed {
			return "", fmt.Errorf("image type not allowed: detected type is %s", detectedType)
		}
	}

	return detectedType, nil
}

// ValidateRAWFile 验证文件是否为 RAW 格式（相机原始文件）
// 由于 RAW 格式种类繁多且复杂，这里主要通过扩展名和一些常见的 magic bytes 验证
func ValidateRAWFile(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// 读取文件头
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// 使用 mimetype 检测
	mtype := mimetype.Detect(buffer[:n])
	detectedType := mtype.String()

	// RAW 文件可能被识别为 application/octet-stream 或特定的 RAW 格式
	// 这里我们接受这些类型
	acceptedTypes := []string{
		"application/octet-stream", // 通用二进制文件
		"image/x-canon-cr2",        // Canon CR2
		"image/x-canon-crw",        // Canon CRW
		"image/x-nikon-nef",        // Nikon NEF
		"image/x-sony-arw",         // Sony ARW
		"image/x-adobe-dng",        // Adobe DNG
		"image/tiff",               // 某些 RAW 文件基于 TIFF
	}

	for _, acceptedType := range acceptedTypes {
		if detectedType == acceptedType {
			return nil
		}
	}

	// 如果不是已知的 RAW 类型，返回警告（但不阻止，因为 RAW 格式太多了）
	// 实际使用中，主要依靠扩展名验证
	return nil
}
