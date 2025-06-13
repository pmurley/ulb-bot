package fantrax

import (
	"fmt"
	"github.com/pmurley/go-fantrax/auth_client"
	"github.com/pmurley/go-fantrax/models"
)

type Client struct {
	Client   *auth_client.Client
	LeagueId string
}

func NewFantraxClient(leagueId string, useCache bool) (*Client, error) {
	client, err := auth_client.NewClient(leagueId, useCache)
	if err != nil {
		return nil, err
	}
	return &Client{
		Client:   client,
		LeagueId: leagueId,
	}, nil
}

func (c *Client) GetTransactionsFromFantrax() ([]models.Transaction, error) {
	transactions, err := c.Client.GetAllTransactionsIncludingTrades()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch transactions: %w", err)
	}

	return transactions, nil
}
