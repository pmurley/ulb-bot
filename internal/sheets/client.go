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
			Timeout: 30 * time.Second,
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

// GetSheetDataCSV fetches data from a specific sheet tab as CSV
func (c *Client) GetSheetDataCSV(gid string) ([][]string, error) {
	url := fmt.Sprintf("https://docs.google.com/spreadsheets/d/%s/export?format=csv&gid=%s", c.spreadsheetID, gid)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch sheet data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	reader := csv.NewReader(resp.Body)
	var data [][]string

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read CSV: %w", err)
		}
		data = append(data, record)
	}

	return data, nil
}
