package components

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// FileInfo represents a file or directory
type FileInfo struct {
	Name    string
	Path    string
	IsDir   bool
	Size    int64
	ModTime string
}

// FileBrowser is a file browser component
type FileBrowser struct {
	Select       Select
	rootDir      string
	currentDir   string
	extension    string
	allowNew     bool
	selectedPath string
}

// NewFileBrowser creates a new file browser
func NewFileBrowser(rootDir, extension string, width, height int) FileBrowser {
	if rootDir == "" {
		rootDir, _ = os.Getwd()
	}

	fb := FileBrowser{
		rootDir:    rootDir,
		currentDir: rootDir,
		extension:  extension,
		allowNew:   false,
	}

	// Load initial files
	files := fb.loadFiles()
	fb.Select = NewSelect("Select a file", files, width, height)

	if extension != "" {
		fb.Select.list.Title = "Select a " + extension + " file"
	}

	return fb
}

// WithAllowNew allows manual path entry
func (fb FileBrowser) WithAllowNew() FileBrowser {
	fb.allowNew = true

	// Add "Enter custom path" option
	items := fb.Select.list.Items()
	customItem := NewSelectItem("✏ Enter custom path", "Type a file path manually", "__custom__")
	newItems := append([]list.Item{customItem}, items...)
	fb.Select.list.SetItems(newItems)

	return fb
}

// WithBackOption adds a back option
func (fb FileBrowser) WithBackOption() FileBrowser {
	fb.Select = fb.Select.WithBackOption()
	return fb
}

// loadFiles loads files from the current directory
func (fb *FileBrowser) loadFiles() []SelectItem {
	var items []SelectItem

	type fileWithInfo struct {
		item SelectItem
		info os.FileInfo
	}
	var fileList []fileWithInfo

	// Walk directory
	filepath.Walk(fb.currentDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || strings.HasPrefix(info.Name(), ".") {
			return nil
		}

		// Check root dir boundary - only show files within the root
		relPath, _ := filepath.Rel(fb.rootDir, path)
		if strings.HasPrefix(relPath, "..") {
			return nil
		}

		if fb.extension != "" && !strings.HasSuffix(info.Name(), fb.extension) {
			return nil
		}

		fileList = append(fileList, fileWithInfo{
			item: NewSelectItem(relPath, formatFileSize(info.Size()), path),
			info: info,
		})
		return nil
	})

	// Sort: newest first
	sort.Slice(fileList, func(i, j int) bool {
		return fileList[i].info.ModTime().After(fileList[j].info.ModTime())
	})

	for _, f := range fileList {
		items = append(items, f.item)
	}

	// If no files found, show a single disabled message instead of keeping it in the list with "Back"
	if len(fileList) == 0 {
		return []SelectItem{
			NewDisabledSelectItem("No files found", "The results directory is empty"),
		}
	}

	return items
}

// SelectedPath returns the selected file path
func (fb FileBrowser) SelectedPath() string {
	item := fb.Select.SelectedItem()
	if item == nil {
		return ""
	}

	if item.Value() == "__custom__" {
		return "__custom__"
	}

	return item.Value()
}

// Init implements tea.Model
func (fb FileBrowser) Init() tea.Cmd {
	return fb.Select.Init()
}

// Update implements tea.Model
func (fb FileBrowser) Update(msg tea.Msg) (FileBrowser, tea.Cmd) {
	var cmd tea.Cmd
	fb.Select, cmd = fb.Select.Update(msg)
	return fb, cmd
}

// View implements tea.Model
func (fb FileBrowser) View() string {
	return fb.Select.View()
}

// formatFileSize formats file size for display
func formatFileSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}
