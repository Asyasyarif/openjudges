package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// RelativePath returns a relative path from the current working directory.
// If the path cannot be made relative or would require going up many directories,
// it returns the absolute path instead.
func RelativePath(path string) string {
	// Get absolute path first
	absPath, err := filepath.Abs(path)
	if err != nil {
		return path
	}

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return absPath
	}

	// Calculate relative path
	rel, err := filepath.Rel(cwd, absPath)
	if err != nil {
		return absPath
	}

	// If relative path starts with too many "../", return absolute path
	if strings.HasPrefix(rel, ".."+string(filepath.Separator)+".."+string(filepath.Separator)) {
		return absPath
	}

	return rel
}

// DisplayPath returns a nicely formatted path for display.
// It shows relative paths when reasonable, otherwise absolute paths.
func DisplayPath(path string) string {
	return RelativePath(path)
}

// TruncatePath truncates a path to a maximum length, preserving the filename.
// For example: "/very/long/path/to/file.txt" becomes "...path/to/file.txt"
func TruncatePath(path string, maxLen int) string {
	if len(path) <= maxLen {
		return path
	}

	// Always preserve the filename
	filename := filepath.Base(path)
	if len(filename) >= maxLen-3 {
		// If filename itself is too long, truncate it
		return "..." + filename[len(filename)-(maxLen-3):]
	}

	// Calculate how much of the directory path we can show
	availableLen := maxLen - len(filename) - 4 // 4 for ".../"

	// Get the directory path
	dir := filepath.Dir(path)
	if len(dir) <= availableLen {
		return dir + string(filepath.Separator) + filename
	}

	// Truncate from the beginning of the directory path
	truncatedDir := dir[len(dir)-availableLen:]

	return "..." + truncatedDir + string(filepath.Separator) + filename
}

// FormatTokens formats a token count in a human-readable way
func FormatTokens(n int) string {
	if n >= 1000000 {
		return fmt.Sprintf("%.1fM", float64(n)/1000000)
	}
	if n >= 1000 {
		return fmt.Sprintf("%.1fK", float64(n)/1000)
	}
	return fmt.Sprintf("%d", n)
}

// Truncate truncates a string to a maximum length with ellipsis
func Truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}
