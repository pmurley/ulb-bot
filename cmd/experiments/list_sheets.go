package main

import (
	"context"
	"fmt"
	"log"

	"github.com/joho/godotenv"
	"github.com/pmurley/ulb-bot/internal/config"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
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

	if cfg.GoogleAPIKey == "" {
		log.Fatal("GOOGLE_API_KEY not set in .env file")
	}

	ctx := context.Background()
	
	// Create sheets service with API key
	srv, err := sheets.NewService(ctx, option.WithAPIKey(cfg.GoogleAPIKey))
	if err != nil {
		log.Fatal("Unable to create sheets service:", err)
	}

	// Get spreadsheet metadata
	spreadsheet, err := srv.Spreadsheets.Get(cfg.GoogleSheetsID).Do()
	if err != nil {
		log.Fatal("Unable to retrieve spreadsheet metadata:", err)
	}

	fmt.Printf("Spreadsheet Title: %s\n", spreadsheet.Properties.Title)
	fmt.Printf("Spreadsheet ID: %s\n", cfg.GoogleSheetsID)
	fmt.Printf("Public URL: https://docs.google.com/spreadsheets/d/%s\n\n", cfg.GoogleSheetsID)
	
	fmt.Println("Sheets in this spreadsheet:")
	fmt.Println("----------------------------")
	
	for i, sheet := range spreadsheet.Sheets {
		props := sheet.Properties
		fmt.Printf("%d. Name: %s\n", i+1, props.Title)
		fmt.Printf("   GID: %d\n", props.SheetId)
		fmt.Printf("   Rows: %d, Columns: %d\n", props.GridProperties.RowCount, props.GridProperties.ColumnCount)
		
		if props.Hidden {
			fmt.Printf("   (Hidden)\n")
		}
		fmt.Println()
	}
}