package sheets

import (
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/pmurley/ulb-bot/internal/cache"
	"github.com/pmurley/ulb-bot/internal/models"
)

// Client fetches data from public Google Sheets using CSV export
type Client struct {
	spreadsheetID string
	httpClient    *http.Client
}

func NewClient(spreadsheetID string) (*Client, error) {
	return &Client{
		spreadsheetID: spreadsheetID,
		httpClient: &http.Client{
			Timeout: 2 * time.Minute, // Increased from 30s to 2 minutes
		},
	}, nil
}

// Sheet GIDs
const (
	StandingsGID    = "633435137"
	MasterPlayerGID = "286507798"
	AccountingGID   = "396888711"
	SalaryGID       = "1669990835"
	DeadMoneyGID    = "36423663"
)

func (c *Client) LoadInitialData(cache *cache.Cache) error {
	// Load Master Player Pool
	players, err := c.LoadMasterPlayerPool()
	if err != nil {
		return fmt.Errorf("failed to load player pool: %w", err)
	}

	cache.SetPlayers(players)
	return nil
}

// LoadMasterPlayerPool loads all players from the Master Player Pool sheet
func (c *Client) LoadMasterPlayerPool() ([]models.Player, error) {
	data, err := c.GetSheetDataCSV(MasterPlayerGID)
	if err != nil {
		return nil, err
	}

	if len(data) < 3 { // Need at least header rows and one data row
		return nil, fmt.Errorf("insufficient data in player pool sheet")
	}

	// The second row contains headers
	headerRow := data[1]

	var players []models.Player

	// Start from row 3 (index 2) for actual player data
	for i := 2; i < len(data); i++ {
		player, err := models.ParsePlayerRow(data[i], headerRow)
		if err != nil {
			// Log error but continue processing other players
			continue
		}
		if player != nil {
			players = append(players, *player)
		}
	}

	return players, nil
}

// GetSheetDataCSV fetches data from a specific sheet tab as CSV with retry logic
func (c *Client) GetSheetDataCSV(gid string) ([][]string, error) {
	url := fmt.Sprintf("https://docs.google.com/spreadsheets/d/%s/export?format=csv&gid=%s", c.spreadsheetID, gid)

	return c.fetchWithRetry(url)
}

// fetchWithRetry attempts to fetch data with exponential backoff retry
func (c *Client) fetchWithRetry(url string) ([][]string, error) {
	maxRetries := 3
	baseDelay := 5 * time.Second

	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			delay := baseDelay * time.Duration(1<<(attempt-1)) // Exponential backoff: 5s, 10s, 20s
			fmt.Printf("Retrying sheet fetch in %v (attempt %d of %d)\n", delay, attempt+1, maxRetries+1)
			time.Sleep(delay)
		}

		resp, err := c.httpClient.Get(url)
		if err != nil {
			lastErr = fmt.Errorf("failed to fetch sheet data: %w", err)
			fmt.Printf("HTTP request failed: %v\n", err)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			lastErr = fmt.Errorf("unexpected status code: %d", resp.StatusCode)
			fmt.Printf("Unexpected status code: %d\n", resp.StatusCode)
			continue
		}

		reader := csv.NewReader(resp.Body)
		var data [][]string
		csvErr := false

		for {
			record, err := reader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				lastErr = fmt.Errorf("failed to read CSV: %w", err)
				fmt.Printf("Failed to read CSV: %v\n", err)
				csvErr = true
				break
			}
			data = append(data, record)
		}

		resp.Body.Close()

		// If we got here and data was read successfully, return it
		if !csvErr && len(data) > 0 {
			if attempt > 0 {
				fmt.Printf("Sheet fetch succeeded on retry attempt %d\n", attempt+1)
			}
			return data, nil
		}
	}

	return nil, fmt.Errorf("failed to fetch sheet data after %d retries: %w", maxRetries+1, lastErr)
}
