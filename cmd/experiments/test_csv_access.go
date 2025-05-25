package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

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

	fmt.Printf("Testing CSV access for sheet: %s\n\n", sheetsID)

	// Use the actual GID from the sheet URL
	gid := "633435137"
	url := fmt.Sprintf("https://docs.google.com/spreadsheets/d/%s/export?format=csv&gid=%s", sheetsID, gid)
	
	fmt.Printf("Attempting to fetch: %s\n", url)
	
	resp, err := http.Get(url)
	if err != nil {
		log.Fatal("Failed to fetch:", err)
	}
	defer resp.Body.Close()

	fmt.Printf("Status: %s\n", resp.Status)
	
	if resp.StatusCode != http.StatusOK {
		log.Fatal("Failed with status:", resp.StatusCode)
	}

	// Read CSV
	reader := csv.NewReader(resp.Body)
	
	fmt.Println("\nFirst 5 rows of data:")
	fmt.Println("--------------------")
	
	for i := 0; i < 5; i++ {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal("Failed to read CSV:", err)
		}
		
		fmt.Printf("Row %d: %v\n", i+1, record)
	}
}