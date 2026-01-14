package tools

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// ReadResult represents the result of a read operation
type ReadResult struct {
	Content string `json:"content"`
	Total   int    `json:"total_lines"`
}

// Read reads a file with line numbers, offset, and limit
func Read(path string, offset, limit int) (*ReadResult, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	lineNum := 1
	for scanner.Scan() {
		if lineNum > offset && (limit <= 0 || lineNum <= offset+limit) {
			lines = append(lines, fmt.Sprintf("%d: %s", lineNum, scanner.Text()))
		}
		lineNum++
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return &ReadResult{
		Content: strings.Join(lines, "\n"),
		Total:   lineNum - 1,
	}, nil
}

// Write writes content to a file
func Write(path, content string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0644)
}

// Edit replaces a unique string in a file
func Edit(path, oldText, newText string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	content := string(data)
	count := strings.Count(content, oldText)
	if count == 0 {
		return fmt.Errorf("text not found: %s", oldText)
	}
	if count > 1 {
		return fmt.Errorf("text is not unique (found %d occurrences): %s", count, oldText)
	}

	newContent := strings.Replace(content, oldText, newText, 1)
	return os.WriteFile(path, []byte(newContent), 0644)
}

// Glob finds files by pattern, sorted by mtime
func Glob(pattern string) ([]string, error) {
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}

	type fileInfo struct {
		path  string
		mtime int64
	}

	var infos []fileInfo
	for _, m := range matches {
		stat, err := os.Stat(m)
		if err != nil {
			continue
		}
		if !stat.IsDir() {
			infos = append(infos, fileInfo{path: m, mtime: stat.ModTime().Unix()})
		}
	}

	sort.Slice(infos, func(i, j int) bool {
		return infos[i].mtime > infos[j].mtime // Newest first
	})

	var result []string
	for _, info := range infos {
		result = append(result, info.path)
	}

	return result, nil
}

// Grep searches files for regex
func Grep(pattern, path string) ([]string, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}

	var matches []string
	err = filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		file, err := os.Open(p)
		if err != nil {
			return nil // Skip files we can't read
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		lineNum := 1
		for scanner.Scan() {
			line := scanner.Text()
			if re.MatchString(line) {
				matches = append(matches, fmt.Sprintf("%s:%d:%s", p, lineNum, line))
			}
			lineNum++
		}
		return nil
	})

	return matches, err
}
