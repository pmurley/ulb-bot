package spotrac

import (
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/PuerkitoBio/goquery"
)

type SearchResult struct {
	Type          string
	PlayerResults []PlayerSearchResult
	ErrorMessage  string
}

type PlayerSearchResult struct {
	Name     string
	Team     string
	Position string
	URL      string
	ID       string
}

type ContractInfo struct {
	PlayerName    string
	Team          string
	Position      string
	Status        string // Pre-Arbitration, Arbitration, etc.
	ContractTerms string
	TotalValue    string
	AverageSalary string
	SigningBonus  string
	FreeAgent     string
	ContractNotes []string
	ContractYears []ContractYear
}

type ContractYear struct {
	Year         int
	Age          int
	Status       string
	BaseSalary   string
	Incentives   string
	PayrollTotal string
	Cash         string
}

func ParseSearchResults(body io.Reader) (*SearchResult, error) {
	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		return nil, fmt.Errorf("parsing HTML: %w", err)
	}

	// Extract search query from the h1
	searchQuery := ""
	doc.Find("h1.h3.fw-bold").Each(func(i int, s *goquery.Selection) {
		text := s.Text()
		if strings.Contains(text, "Search Results for") {
			// Extract the quoted search term
			re := regexp.MustCompile(`"([^"]+)"`)
			matches := re.FindStringSubmatch(text)
			if len(matches) > 1 {
				searchQuery = matches[1]
			}
		}
	})

	// Look for player results in the second list-group
	var results []PlayerSearchResult
	listGroups := doc.Find("div.list-group")

	// The second list-group contains the actual results
	if listGroups.Length() >= 2 {
		listGroups.Eq(1).Find("a.list-group-item").Each(func(i int, s *goquery.Selection) {
			href, exists := s.Attr("href")
			if !exists {
				return
			}

			// Extract player name
			name := strings.TrimSpace(s.Find("span.text-danger").Text())

			// Extract team from the text after player name
			fullText := strings.TrimSpace(s.Find("span").First().Text())
			team := ""
			if strings.Contains(fullText, "(") && strings.Contains(fullText, ")") {
				start := strings.LastIndex(fullText, "(")
				end := strings.LastIndex(fullText, ")")
				if start != -1 && end != -1 && end > start {
					team = fullText[start+1 : end]
				}
			}

			// Extract position from badge
			position := strings.TrimSpace(s.Find("span.badge").Text())

			// Extract player ID from URL
			playerID := ""
			if strings.Contains(href, "/player/") {
				parts := strings.Split(href, "/")
				for i, part := range parts {
					if part == "player" && i+1 < len(parts) {
						playerID = parts[i+1]
						// Remove query params if any
						if idx := strings.Index(playerID, "?"); idx != -1 {
							playerID = playerID[:idx]
						}
						break
					}
				}
			}

			results = append(results, PlayerSearchResult{
				Name:     name,
				Team:     team,
				Position: position,
				URL:      href,
				ID:       playerID,
			})
		})
	}

	// Determine result type
	resultType := "none"
	if len(results) == 1 {
		resultType = "single"
	} else if len(results) > 1 {
		resultType = "multiple"
	}

	// Set error message for no results
	errorMessage := ""
	if resultType == "none" && searchQuery != "" {
		errorMessage = fmt.Sprintf("No players found matching '%s'", searchQuery)
	}

	return &SearchResult{
		Type:          resultType,
		PlayerResults: results,
		ErrorMessage:  errorMessage,
	}, nil
}

func ParseContractInfo(body io.Reader) (*ContractInfo, error) {
	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		return nil, fmt.Errorf("parsing HTML: %w", err)
	}

	info := &ContractInfo{}

	// Extract player name from title or h1
	title := doc.Find("title").Text()
	if strings.Contains(title, "|") {
		info.PlayerName = strings.TrimSpace(strings.Split(title, "|")[0])
	}

	// Look for the current contract section
	foundCurrentContract := false
	doc.Find("div.contract-wrapper").Each(func(i int, wrapper *goquery.Selection) {
		if foundCurrentContract {
			return
		}

		// Check if this wrapper contains "(CURRENT)" in the header
		headerText := wrapper.Find("h2").Text()
		if strings.Contains(headerText, "(CURRENT)") {
			foundCurrentContract = true

			// Extract status from header (Pre-Arbitration, Arbitration, etc.)
			if strings.Contains(headerText, "Pre-Arbitration") {
				info.Status = "Pre-Arbitration"
			} else if strings.Contains(headerText, "Arbitration") {
				info.Status = "Arbitration"
			} else if strings.Contains(headerText, "Free Agent") {
				info.Status = "Free Agent"
			}

			// Extract contract details from this specific wrapper
			wrapper.Find("div.contract-details div.cell").Each(func(j int, s *goquery.Selection) {
				label := strings.TrimSpace(s.Find("div.label").Text())
				value := strings.TrimSpace(s.Find("div.value").Text())

				switch label {
				case "Contract Terms:":
					info.ContractTerms = value
				case "Average Salary:":
					info.AverageSalary = value
				case "Signing Bonus:":
					info.SigningBonus = value
				case "Free Agent:":
					info.FreeAgent = value
				}
			})
		}
	})

	// If no current contract found, fall back to the first contract-details with substantial value
	if !foundCurrentContract {
		doc.Find("div.contract-wrapper").Each(func(i int, wrapper *goquery.Selection) {
			if foundCurrentContract {
				return
			}

			// Check if this has a multi-million dollar contract
			wrapper.Find("div.contract-details div.cell").Each(func(j int, s *goquery.Selection) {
				label := strings.TrimSpace(s.Find("div.label").Text())
				value := strings.TrimSpace(s.Find("div.value").Text())

				if label == "Contract Terms:" && strings.Contains(value, "$") {
					// Check if it's a substantial contract (millions)
					if strings.Contains(value, ",000,000") || strings.Contains(value, "million") {
						foundCurrentContract = true

						// Re-extract all details from this wrapper
						wrapper.Find("div.contract-details div.cell").Each(func(k int, cell *goquery.Selection) {
							cellLabel := strings.TrimSpace(cell.Find("div.label").Text())
							cellValue := strings.TrimSpace(cell.Find("div.value").Text())

							switch cellLabel {
							case "Contract Terms:":
								info.ContractTerms = cellValue
							case "Average Salary:":
								info.AverageSalary = cellValue
							case "Signing Bonus:":
								info.SigningBonus = cellValue
							case "Free Agent:":
								info.FreeAgent = cellValue
							}
						})
					}
				}
			})
		})
	}

	// Extract total value from contract terms if available
	if info.ContractTerms != "" && strings.Contains(info.ContractTerms, "$") {
		parts := strings.Split(info.ContractTerms, "/")
		if len(parts) >= 2 {
			info.TotalValue = strings.TrimSpace(parts[1])
		}
	}

	// Extract team and position from meta description or page content
	metaDesc := doc.Find("meta[name='description']").AttrOr("content", "")
	if strings.Contains(metaDesc, "signed") && strings.Contains(metaDesc, "with the") {
		// Extract team from meta description like "signed a 5 year, $185,000,000 contract with the Texas Rangers"
		parts := strings.Split(metaDesc, "with the ")
		if len(parts) >= 2 {
			teamPart := parts[1]
			// Find the end of team name (usually followed by "with" or period)
			if idx := strings.Index(teamPart, " with"); idx > 0 {
				info.Team = strings.TrimSpace(teamPart[:idx])
			} else {
				// Just take the first few words (team name is usually 1-3 words)
				words := strings.Fields(teamPart)
				if len(words) >= 2 {
					// Check if the second or third word starts with lowercase (likely not part of team name)
					teamWords := []string{words[0]}
					for i := 1; i < len(words) && i < 4; i++ {
						if len(words[i]) > 0 && unicode.IsUpper(rune(words[i][0])) {
							teamWords = append(teamWords, words[i])
						} else {
							break
						}
					}
					info.Team = strings.Join(teamWords, " ")
				}
			}
		}
	}

	// Extract contract notes
	doc.Find("div.notes ul li").Each(func(i int, s *goquery.Selection) {
		note := strings.TrimSpace(s.Text())
		if note != "" {
			info.ContractNotes = append(info.ContractNotes, note)
		}
	})

	// Parse contract years from table
	// Look for the correct payroll table by checking headers
	foundCorrectTable := false
	doc.Find("table").Each(func(tableIdx int, table *goquery.Selection) {
		if foundCorrectTable {
			return
		}

		// Check headers to identify the right table
		headers := []string{}
		yearColIdx := -1
		ageColIdx := -1
		statusColIdx := -1
		payrollColIdx := -1

		table.Find("thead th").Each(func(i int, s *goquery.Selection) {
			headerText := strings.TrimSpace(s.Text())
			headers = append(headers, headerText)
			headerLower := strings.ToLower(headerText)

			if strings.Contains(headerLower, "year") && yearColIdx == -1 {
				yearColIdx = i
			}
			if strings.Contains(headerLower, "age") && ageColIdx == -1 {
				ageColIdx = i
			}
			if strings.Contains(headerLower, "status") && statusColIdx == -1 {
				statusColIdx = i
			}
			if (strings.Contains(headerLower, "payroll") || strings.Contains(headerLower, "salary")) && payrollColIdx == -1 {
				// Look for "Payroll" or "Payroll Salary" header
				if strings.Contains(headerText, "Payroll") {
					payrollColIdx = i
				}
			}
		})

		// We found a good table if it has year and payroll columns
		if yearColIdx >= 0 && payrollColIdx >= 0 {
			foundCorrectTable = true

			table.Find("tbody tr").Each(func(rowIdx int, row *goquery.Selection) {
				year := ContractYear{}

				cells := row.Find("td")
				cells.Each(func(cellIdx int, cell *goquery.Selection) {
					text := strings.TrimSpace(cell.Text())

					if cellIdx == yearColIdx {
						if yearVal, err := strconv.Atoi(text); err == nil {
							year.Year = yearVal
						}
					} else if ageColIdx >= 0 && cellIdx == ageColIdx {
						if ageVal, err := strconv.Atoi(text); err == nil {
							year.Age = ageVal
						}
					} else if statusColIdx >= 0 && cellIdx == statusColIdx {
						// Look for status in div.option or just text
						optionDiv := cell.Find("div.option")
						if optionDiv.Length() > 0 {
							year.Status = strings.TrimSpace(optionDiv.Text())
						} else if !strings.Contains(text, "$") && text != "" {
							year.Status = text
						}
					} else if cellIdx == payrollColIdx {
						// Check for status in div.option first
						optionDiv := cell.Find("div.option")
						if optionDiv.Length() > 0 {
							year.Status = strings.TrimSpace(optionDiv.Text())
						} else if strings.Contains(text, "$") || text == "-" {
							year.PayrollTotal = text
						}
					} else {
						// Check any cell for status information in div.option
						optionDiv := cell.Find("div.option")
						if optionDiv.Length() > 0 && year.Status == "" {
							year.Status = strings.TrimSpace(optionDiv.Text())
						}
					}
				})

				// Only add if we have a valid year
				if year.Year > 0 {
					info.ContractYears = append(info.ContractYears, year)
				}
			})
		}
	})

	return info, nil
}
