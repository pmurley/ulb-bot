package storage

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/pmurley/go-fantrax/models"
)

const transactionFileName = "transactions.csv"

// TransactionStorage handles persistent storage of transactions
type TransactionStorage struct {
	mu       sync.RWMutex
	filePath string
}

// NewTransactionStorage creates a new transaction storage instance
func NewTransactionStorage() (*TransactionStorage, error) {
	// Ensure data directory exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	filePath := filepath.Join(dataDir, transactionFileName)
	ts := &TransactionStorage{
		filePath: filePath,
	}

	// Create file if it doesn't exist
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		if err := ts.createFile(); err != nil {
			return nil, err
		}
	}

	return ts, nil
}

// createFile creates the CSV file with headers
func (ts *TransactionStorage) createFile() error {
	file, err := os.Create(ts.filePath)
	if err != nil {
		return fmt.Errorf("failed to create transaction file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	headers := []string{
		"ID", "Type", "TeamName", "TeamID", "FromTeamName", "FromTeamID",
		"ToTeamName", "ToTeamID", "PlayerName", "PlayerID", "PlayerTeam",
		"PlayerPosition", "BidAmount", "Priority", "ProcessedDate", "Period",
		"Executed", "ExecutedBy", "TradeGroupID", "TradeGroupSize",
	}
	if err := writer.Write(headers); err != nil {
		return fmt.Errorf("failed to write headers: %w", err)
	}
	writer.Flush()

	return nil
}

// AddTransactions adds new transactions to the CSV file
func (ts *TransactionStorage) AddTransactions(transactions []models.Transaction) error {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	file, err := os.OpenFile(ts.filePath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open transaction file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	for _, transaction := range transactions {
		record := []string{
			transaction.ID,
			transaction.Type,
			transaction.TeamName,
			transaction.TeamID,
			transaction.FromTeamName,
			transaction.FromTeamID,
			transaction.ToTeamName,
			transaction.ToTeamID,
			transaction.PlayerName,
			transaction.PlayerID,
			transaction.PlayerTeam,
			transaction.PlayerPosition,
			transaction.BidAmount,
			transaction.Priority,
			transaction.ProcessedDate.Format(time.RFC3339),
			strconv.Itoa(transaction.Period),
			strconv.FormatBool(transaction.Executed),
			transaction.ExecutedBy,
			transaction.TradeGroupID,
			strconv.Itoa(transaction.TradeGroupSize),
		}

		if err := writer.Write(record); err != nil {
			return fmt.Errorf("failed to write transaction record: %w", err)
		}
	}

	return nil
}

// GetAllTransactions returns all stored transactions
func (ts *TransactionStorage) GetAllTransactions() ([]models.Transaction, error) {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	file, err := os.Open(ts.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open transaction file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read transaction file: %w", err)
	}

	var transactions []models.Transaction
	// Skip header row
	for i := 1; i < len(records); i++ {
		record := records[i]
		if len(record) < 20 {
			continue
		}

		processedDate, err := time.Parse(time.RFC3339, record[14])
		if err != nil {
			continue
		}

		period, _ := strconv.Atoi(record[15])
		executed, _ := strconv.ParseBool(record[16])
		tradeGroupSize, _ := strconv.Atoi(record[19])

		transaction := models.Transaction{
			ID:             record[0],
			Type:           record[1],
			TeamName:       record[2],
			TeamID:         record[3],
			FromTeamName:   record[4],
			FromTeamID:     record[5],
			ToTeamName:     record[6],
			ToTeamID:       record[7],
			PlayerName:     record[8],
			PlayerID:       record[9],
			PlayerTeam:     record[10],
			PlayerPosition: record[11],
			BidAmount:      record[12],
			Priority:       record[13],
			ProcessedDate:  processedDate,
			Period:         period,
			Executed:       executed,
			ExecutedBy:     record[17],
			TradeGroupID:   record[18],
			TradeGroupSize: tradeGroupSize,
		}

		transactions = append(transactions, transaction)
	}

	return transactions, nil
}

// GetTransactionIDs returns a set of all stored transaction IDs for quick lookup
func (ts *TransactionStorage) GetTransactionIDs() (map[string]bool, error) {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	file, err := os.Open(ts.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open transaction file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read transaction file: %w", err)
	}

	ids := make(map[string]bool)
	// Skip header row
	for i := 1; i < len(records); i++ {
		if len(records[i]) > 0 {
			ids[records[i][0]] = true
		}
	}

	return ids, nil
}

// GetTradeGroupIDs returns a set of all stored trade group IDs for quick lookup
func (ts *TransactionStorage) GetTradeGroupIDs() (map[string]bool, error) {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	file, err := os.Open(ts.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open transaction file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read transaction file: %w", err)
	}

	groupIDs := make(map[string]bool)
	// Skip header row
	for i := 1; i < len(records); i++ {
		if len(records[i]) >= 19 && records[i][18] != "" {
			groupIDs[records[i][18]] = true
		}
	}

	return groupIDs, nil
}

// GroupTransactionsByType groups transactions by their type
func GroupTransactionsByType(transactions []models.Transaction) map[string][]models.Transaction {
	groups := make(map[string][]models.Transaction)
	for _, tx := range transactions {
		groups[tx.Type] = append(groups[tx.Type], tx)
	}
	return groups
}

// GroupTransactionsByTeam groups transactions by team (handles both regular and trade transactions)
func GroupTransactionsByTeam(transactions []models.Transaction) map[string][]models.Transaction {
	groups := make(map[string][]models.Transaction)
	for _, tx := range transactions {
		if tx.Type == "TRADE" {
			if tx.FromTeamName != "" {
				groups[tx.FromTeamName] = append(groups[tx.FromTeamName], tx)
			}
			if tx.ToTeamName != "" {
				groups[tx.ToTeamName] = append(groups[tx.ToTeamName], tx)
			}
		} else {
			if tx.TeamName != "" {
				groups[tx.TeamName] = append(groups[tx.TeamName], tx)
			}
		}
	}
	return groups
}

// GroupTransactionsByTradeGroup groups trade transactions by their TradeGroupID
func GroupTransactionsByTradeGroup(transactions []models.Transaction) map[string][]models.Transaction {
	groups := make(map[string][]models.Transaction)
	for _, tx := range transactions {
		if tx.Type == "TRADE" && tx.TradeGroupID != "" {
			groups[tx.TradeGroupID] = append(groups[tx.TradeGroupID], tx)
		}
	}
	return groups
}
