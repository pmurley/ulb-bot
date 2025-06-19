package spotrac

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type Client struct {
	httpClient *http.Client
}

func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return fmt.Errorf("too many redirects")
				}
				// Allow redirects but we'll track them
				return nil
			},
		},
	}
}

func (c *Client) Search(query string) (*SearchResult, error) {
	searchURL := fmt.Sprintf("https://www.spotrac.com/search?q=%s", url.QueryEscape(query))

	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("performing request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	// Check if we were redirected to a player page
	finalURL := resp.Request.URL.String()
	if strings.Contains(finalURL, "/player/_/id/") && !strings.Contains(finalURL, "/search") {
		// We were redirected to a specific player
		// Extract player info from the URL
		parts := strings.Split(finalURL, "/")
		var playerID string
		var playerName string

		for i, part := range parts {
			if part == "id" && i+1 < len(parts) {
				playerID = parts[i+1]
			}
			if i == len(parts)-1 && part != "" {
				// Last part is usually the player name slug
				playerName = strings.ReplaceAll(part, "-", " ")
				playerName = strings.Title(playerName)
			}
		}

		return &SearchResult{
			Type: "single",
			PlayerResults: []PlayerSearchResult{
				{
					Name: playerName,
					URL:  finalURL,
					ID:   playerID,
				},
			},
		}, nil
	}

	// Parse search results page
	return ParseSearchResults(bytes.NewReader(body))
}

func (c *Client) GetPlayerContract(playerURL string) (*ContractInfo, error) {
	// Handle redirect URLs
	if strings.Contains(playerURL, "/redirect/player/") {
		// This is a redirect URL, we need to follow it
		req, err := http.NewRequest("GET", playerURL, nil)
		if err != nil {
			return nil, fmt.Errorf("creating request: %w", err)
		}

		c.setHeaders(req)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("performing request: %w", err)
		}
		defer resp.Body.Close()

		// Get the final URL after redirects
		playerURL = resp.Request.URL.String()
	}

	req, err := http.NewRequest("GET", playerURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("performing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	return ParseContractInfo(bytes.NewReader(body))
}

func (c *Client) SearchAndSaveHTML(query string, outputFile string) error {
	searchURL := fmt.Sprintf("https://www.spotrac.com/search?q=%s", url.QueryEscape(query))

	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("performing request: %w", err)
	}
	defer resp.Body.Close()

	// Save response info
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response body: %w", err)
	}

	// Write HTML to file
	err = os.WriteFile(outputFile, body, 0644)
	if err != nil {
		return fmt.Errorf("writing to file: %w", err)
	}

	// Also print some debug info
	fmt.Printf("Initial URL: %s\n", searchURL)
	fmt.Printf("Final URL: %s\n", resp.Request.URL.String())
	fmt.Printf("Status Code: %d\n", resp.StatusCode)
	fmt.Printf("Response saved to: %s\n", outputFile)

	return nil
}

func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/137.0.0.0 Safari/537.36")
	req.Header.Set("Sec-Ch-Ua", `"Google Chrome";v="137", "Chromium";v="137", "Not/A)Brand";v="24"`)
	req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	req.Header.Set("Sec-Ch-Ua-Platform", `"macOS"`)
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Sec-Fetch-User", "?1")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
}
