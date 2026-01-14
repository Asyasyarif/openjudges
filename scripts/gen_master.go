package main

import (
	"fmt"
	"log"

	"github.com/xuri/excelize/v2"
)

func main() {
	f := excelize.NewFile()
	defer func() {
		if err := f.Close(); err != nil {
			fmt.Println(err)
		}
	}()

	// Set sheet name
	sheet := "Sheet1"
	f.SetSheetName("Sheet1", sheet)

	// Headers
	headers := []string{"id", "question", "expectation", "criteria"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
	}

	// Sample Data
	data := [][]interface{}{
		{"1", "Say hello", "AI should greet politely", "Should contain greeting;Should be polite"},
		{"2", "What is 2+2?", "Correct mathematical result", ""},
		{"3", "Tell me a joke about cats", "AI should talk about dogs instead", "Should contain the word 'dog';Should not mention cats"}, // Sample fail case
	}

	for r, row := range data {
		for c, val := range row {
			cell, _ := excelize.CoordinatesToCellName(c+1, r+2)
			f.SetCellValue(sheet, cell, val)
		}
	}

	// Styling headers
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"4472C4"}, Pattern: 1},
	})
	f.SetCellStyle(sheet, "A1", "F1", headerStyle)
	f.SetColWidth(sheet, "C", "E", 30)

	// Save
	if err := f.SaveAs("datasets/master_data.xlsx"); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Excel master data created at datasets/master_data.xlsx")
}
