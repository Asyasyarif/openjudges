package testcase

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/xuri/excelize/v2"
)

// Load detects file type and loads test cases accordingly
func Load(path string) ([]TestCase, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".xlsx", ".xlsm":
		return LoadFromExcel(path)
	default:
		return nil, fmt.Errorf("unsupported file extension: %s (only .xlsx and .xlsm are supported)", ext)
	}
}

// LoadFromExcel loads test cases from an Excel file
// Expected columns in the first sheet: id, name, prompt, ai_response, expectation, criteria
func LoadFromExcel(path string) ([]TestCase, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Get all the rows in the first sheet
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("no sheets found in excel file")
	}
	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return nil, err
	}

	if len(rows) < 1 {
		return nil, fmt.Errorf("excel file is empty")
	}

	// Map column indices from header (first row)
	header := rows[0]
	colIndex := make(map[string]int)
	for i, col := range header {
		name := strings.TrimSpace(strings.ToLower(col))
		colIndex[name] = i
		// Aliases
		if name == "question" {
			colIndex["prompt"] = i
		}
	}

	var testCases []TestCase

	// Process data rows
	for i := 1; i < len(rows); i++ {
		record := rows[i]
		tc := TestCase{
			ID:          getField(record, colIndex, "id"),
			Name:        getField(record, colIndex, "name"),
			Prompt:      getField(record, colIndex, "prompt"),
			AIResponse:  getField(record, colIndex, "ai_response"),
			Expectation: getField(record, colIndex, "expectation"),
			RawData:     make(map[string]string),
		}

		// Store all columns in RawData
		for hIdx, colName := range header {
			colName = strings.TrimSpace(strings.ToLower(colName))
			if hIdx < len(record) {
				tc.RawData[colName] = strings.TrimSpace(record[hIdx])
			}
		}

		// Validation: Mandatory fields
		if tc.Prompt == "" || tc.Expectation == "" {
			continue // Skip invalid records
		}

		// Parse criteria (semicolon-separated)
		criteriaStr := getField(record, colIndex, "criteria")
		if criteriaStr != "" {
			criteria := strings.Split(criteriaStr, ";")
			for _, c := range criteria {
				c = strings.TrimSpace(c)
				if c != "" {
					tc.Criteria = append(tc.Criteria, c)
				}
			}
		}

		testCases = append(testCases, tc)
	}

	return testCases, nil
}

func getField(record []string, colIndex map[string]int, field string) string {
	if idx, ok := colIndex[field]; ok && idx < len(record) {
		return strings.TrimSpace(record[idx])
	}
	return ""
}
