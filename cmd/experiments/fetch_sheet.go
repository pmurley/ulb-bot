package main

import (
	"fmt"
	"log"

	"github.com/joho/godotenv"
	"github.com/pmurley/ulb-bot/internal/config"
	"github.com/pmurley/ulb-bot/internal/sheets"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	client, err := sheets.NewClient(cfg.GoogleSheetsID)
	if err != nil {
		log.Fatal("Failed to create sheets client:", err)
	}

	// Example: Fetch data from the first sheet (gid=0)
	// You'll need to replace this with the actual GID of your sheet tab
	gid := "0"
	
	fmt.Printf("Fetching data from sheet %s, tab gid=%s...\n", cfg.GoogleSheetsID, gid)
	
	data, err := client.GetSheetDataCSV(gid)
	if err != nil {
		log.Fatal("Failed to fetch data:", err)
	}

	fmt.Printf("Fetched %d rows\n", len(data))
	
	// Print first 5 rows to see the structure
	for i, row := range data {
		if i >= 5 {
			break
		}
		fmt.Printf("Row %d: %v\n", i, row)
	}
}