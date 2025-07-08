package storage

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/pmurley/ulb-bot/internal/models"
)

const (
	waiverFileName = "waivers.csv"
	dataDir        = "./data"
)

// WaiverStorage handles persistent storage of waivers
type WaiverStorage struct {
	mu       sync.RWMutex
	filePath string
}

// NewWaiverStorage creates a new waiver storage instance
func NewWaiverStorage() (*WaiverStorage, error) {
	// Ensure data directory exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	filePath := filepath.Join(dataDir, waiverFileName)
	ws := &WaiverStorage{
		filePath: filePath,
	}

	// Create file if it doesn't exist
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		if err := ws.createFile(); err != nil {
			return nil, err
		}
	}

	return ws, nil
}

// createFile creates the CSV file with headers
func (ws *WaiverStorage) createFile() error {
	file, err := os.Create(ws.filePath)
	if err != nil {
		return fmt.Errorf("failed to create waiver file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	headers := []string{"PlayerName", "TeamName", "UserID", "StartTime", "EndTime", "MessageID", "ChannelID", "Processed"}
	if err := writer.Write(headers); err != nil {
		return fmt.Errorf("failed to write headers: %w", err)
	}
	writer.Flush()

	return nil
}

// AddWaiver adds a new waiver to the CSV file
func (ws *WaiverStorage) AddWaiver(waiver *models.Waiver) error {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	file, err := os.OpenFile(ws.filePath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open waiver file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	record := []string{
		waiver.PlayerName,
		waiver.TeamName,
		waiver.UserID,
		waiver.StartTime.Format(time.RFC3339),
		waiver.EndTime.Format(time.RFC3339),
		waiver.MessageID,
		waiver.ChannelID,
		strconv.FormatBool(waiver.Processed),
	}

	if err := writer.Write(record); err != nil {
		return fmt.Errorf("failed to write waiver record: %w", err)
	}
	writer.Flush()

	return nil
}

// GetActiveWaivers returns all unprocessed waivers
func (ws *WaiverStorage) GetActiveWaivers() ([]*models.Waiver, error) {
	ws.mu.RLock()
	defer ws.mu.RUnlock()

	file, err := os.Open(ws.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open waiver file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read waiver file: %w", err)
	}

	var waivers []*models.Waiver
	// Skip header row
	for i := 1; i < len(records); i++ {
		record := records[i]
		if len(record) < 8 {
			continue
		}

		processed, _ := strconv.ParseBool(record[7])
		if processed {
			continue // Skip already processed waivers
		}

		startTime, err := time.Parse(time.RFC3339, record[3])
		if err != nil {
			continue
		}

		endTime, err := time.Parse(time.RFC3339, record[4])
		if err != nil {
			continue
		}

		waiver := &models.Waiver{
			PlayerName: record[0],
			TeamName:   record[1],
			UserID:     record[2],
			StartTime:  startTime,
			EndTime:    endTime,
			MessageID:  record[5],
			ChannelID:  record[6],
			Processed:  processed,
		}

		waivers = append(waivers, waiver)
	}

	return waivers, nil
}

// MarkWaiverProcessed updates a waiver as processed
func (ws *WaiverStorage) MarkWaiverProcessed(messageID string) error {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	// Read all records
	file, err := os.Open(ws.filePath)
	if err != nil {
		return fmt.Errorf("failed to open waiver file: %w", err)
	}

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		file.Close()
		return fmt.Errorf("failed to read waiver file: %w", err)
	}
	file.Close()

	// Update the processed flag for all matching waivers
	updated := false
	for i := 1; i < len(records); i++ {
		if len(records[i]) >= 8 && records[i][5] == messageID {
			records[i][7] = "true"
			updated = true
		}
	}

	if !updated {
		return fmt.Errorf("waiver with message ID %s not found", messageID)
	}

	// Write all records back
	file, err = os.Create(ws.filePath)
	if err != nil {
		return fmt.Errorf("failed to create waiver file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	if err := writer.WriteAll(records); err != nil {
		return fmt.Errorf("failed to write waiver records: %w", err)
	}
	writer.Flush()

	return nil
}
