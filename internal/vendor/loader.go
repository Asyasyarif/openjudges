package vendor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// dirExists checks if a directory exists
func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// GetVendorsDir returns the absolute path to the vendors directory
// It tries multiple locations to find the vendors folder
func GetVendorsDir() string {
	// Get executable directory first
	execPath, err := os.Executable()
	execDir := ""
	if err == nil {
		execDir = filepath.Dir(execPath)
	}

	// Get current working directory
	cwd, _ := os.Getwd()

	// Priority order: executable dir > cwd > relative paths > home dir
	paths := []string{}

	// 1. Check executable directory
	if execDir != "" {
		paths = append(paths, filepath.Join(execDir, "vendors"))
	}

	// 2. Check current working directory
	paths = append(paths, filepath.Join(cwd, "vendors"))

	// 3. Check relative paths (only if cwd is project root)
	if execDir != "" && cwd != execDir {
		paths = append(paths, filepath.Join(execDir, "..", "vendors"))
		paths = append(paths, filepath.Join(cwd, "..", "vendors"))
	}

	// 4. Legacy relative paths (for go run)
	paths = append(paths, "./vendors", "vendors", "../vendors")

	// 5. Check home directory for installed binaries
	homeDir, _ := os.UserHomeDir()
	if homeDir != "" {
		paths = append(paths, filepath.Join(homeDir, ".openjudges", "vendors"))
	}

	for _, path := range paths {
		if dirExists(path) {
			absPath, _ := filepath.Abs(path)
			return absPath
		}
	}

	return ""
}

// LoadVendors loads all vendor configurations from a directory
// Each JSON file should contain a single VendorConfig object
func LoadVendors(vendorsDir string) ([]VendorConfig, error) {
	var vendors []VendorConfig

	// Check if directory exists
	if _, err := os.Stat(vendorsDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("vendors directory does not exist: %s", vendorsDir)
	}

	files, err := filepath.Glob(filepath.Join(vendorsDir, "*.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to read vendors directory: %w", err)
	}

	for _, file := range files {
		// Skip example files
		baseName := filepath.Base(file)
		if strings.HasPrefix(baseName, "example") {
			continue
		}

		fileVendors, err := loadVendorFile(file)
		if err != nil {
			// Log warning but continue with other files
			continue
		}
		vendors = append(vendors, fileVendors...)
	}

	return vendors, nil
}

// LoadAllVendors loads all vendor configurations including example files
func LoadAllVendors(vendorsDir string) ([]VendorConfig, error) {
	var vendors []VendorConfig

	// Try to find vendors directory if path is not provided or doesn't exist
	searchPath := vendorsDir
	if searchPath == "" || !dirExists(searchPath) {
		searchPath = GetVendorsDir()
	}

	if searchPath == "" || !dirExists(searchPath) {
		return nil, fmt.Errorf("vendors directory not found. Create vendors/ folder with JSON files")
	}

	files, err := filepath.Glob(filepath.Join(searchPath, "*.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to read vendors directory: %w", err)
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no JSON files found in vendors directory: %s", searchPath)
	}

	for _, file := range files {
		fileVendors, err := loadVendorFile(file)
		if err != nil {
			continue
		}
		vendors = append(vendors, fileVendors...)
	}

	return vendors, nil
}

// LoadAllVendorsAuto automatically finds and loads vendors from any valid location
func LoadAllVendorsAuto() ([]VendorConfig, error) {
	return LoadAllVendors("")
}

// loadVendorFile loads vendor(s) from a single JSON file
// Supports both single object and array format
func loadVendorFile(path string) ([]VendorConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}

	filename := filepath.Base(path)

	// Try single object format first (new format: 1 file per vendor)
	var vendor VendorConfig
	if err := json.Unmarshal(data, &vendor); err == nil && vendor.Name != "" {
		vendor.Filename = filename
		return []VendorConfig{vendor}, nil
	}

	// Try array format (legacy format)
	var vendors []VendorConfig
	if err := json.Unmarshal(data, &vendors); err == nil {
		// Set filename for each vendor in array
		for i := range vendors {
			vendors[i].Filename = filename
		}
		return vendors, nil
	}

	return nil, fmt.Errorf("invalid vendor JSON format in file: %s", path)
}

// GetVendorByName finds a vendor by name from a list
func GetVendorByName(vendors []VendorConfig, name string) *VendorConfig {
	for i := range vendors {
		if vendors[i].Name == name {
			return &vendors[i]
		}
	}
	return nil
}
