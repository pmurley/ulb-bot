package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	sheetsID := os.Getenv("GOOGLE_SHEETS_ID")
	if sheetsID == "" {
		log.Fatal("GOOGLE_SHEETS_ID not set")
	}

	// Master Player Pool sheet
	gid := "286507798"
	url := fmt.Sprintf("https://docs.google.com/spreadsheets/d/%s/export?format=csv&gid=%s", sheetsID, gid)
	
	fmt.Println("Analyzing Master Player Pool Sheet")
	fmt.Println("==================================")
	fmt.Printf("URL: %s\n\n", url)
	
	resp, err := http.Get(url)
	if err != nil {
		log.Fatal("Failed to fetch:", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatal("Failed with status:", resp.StatusCode)
	}

	reader := csv.NewReader(resp.Body)
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true
	
	// Read first 10 rows
	fmt.Println("First 10 rows of Master Player Pool:")
	fmt.Println("-------------------------------------")
	
	for i := 0; i < 10; i++ {
		record, err := reader.Read()
		if err == io.EOF {
			fmt.Printf("Reached end of file at row %d\n", i)
			break
		}
		if err != nil {
			fmt.Printf("Error reading row %d: %v\n", i+1, err)
			continue
		}
		
		fmt.Printf("\nRow %d (columns A-AB, total %d columns):\n", i+1, len(record))
		
		// Show each column with its letter designation
		for j, cell := range record {
			colLetter := getColumnLetter(j)
			cell = strings.TrimSpace(cell)
			
			// Only show non-empty cells, truncate long values
			if cell != "" {
				if len(cell) > 40 {
					cell = cell[:37] + "..."
				}
				fmt.Printf("  %s: %s\n", colLetter, cell)
			}
		}
		
		// For the first row (likely headers), also show a summary
		if i == 0 {
			fmt.Printf("\nColumn count: %d (A through %s)\n", len(record), getColumnLetter(len(record)-1))
		}
	}
	
	// Count total rows
	fmt.Println("\nCounting total rows...")
	rowCount := 10 // We already read 10
	for {
		_, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}
		rowCount++
	}
	
	fmt.Printf("\nTotal rows in sheet: %d\n", rowCount)
}

// Convert column index to Excel-style letter (0=A, 1=B, ..., 26=AA, 27=AB, etc.)
func getColumnLetter(index int) string {
	result := ""
	for index >= 0 {
		result = string(rune('A'+index%26)) + result
		index = index/26 - 1
	}
	return result
}