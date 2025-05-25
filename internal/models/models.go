package models

import (
	"strconv"
	"strings"
)

// Player represents a player in the Master Player Pool
type Player struct {
	// Basic Information
	ULBTeam      string  // Column B - The fantasy team that owns this player
	Sort         string  // Column C - Sort category
	Name         string  // Column D - Player name
	Agency       string  // Column E - Agency status
	Position     string  // Column F - Player position(s)
	MLBTeam      string  // Column G - MLB team
	Age          int     // Column H - Age
	Points2024   float64 // Column I - 2024 Points
	OptionsLeft  string  // Column J - Options remaining
	OptionUsed   string  // Column K - Option used?
	Status       string  // Column L - Player status (40-Man, etc)
	
	// Contract Information - Years 2025-2038 (Columns M-Z, AA)
	Contract     map[int]string // Year -> Salary/Status
	ContractNote string         // Column AB - Contract notes
}

// ParsePlayerRow parses a CSV row into a Player struct
func ParsePlayerRow(row []string, headerRow []string) (*Player, error) {
	if len(row) < 28 { // Minimum expected columns through AB
		return nil, nil // Skip incomplete rows
	}
	
	// Skip empty rows or rows without player names
	if strings.TrimSpace(row[3]) == "" { // Column D is player name
		return nil, nil
	}
	
	p := &Player{
		Contract: make(map[int]string),
	}
	
	// Parse basic fields
	if len(row) > 1 {
		p.ULBTeam = strings.TrimSpace(row[1]) // Column B
	}
	if len(row) > 2 {
		p.Sort = strings.TrimSpace(row[2]) // Column C
	}
	if len(row) > 3 {
		p.Name = strings.TrimSpace(row[3]) // Column D
	}
	if len(row) > 4 {
		p.Agency = strings.TrimSpace(row[4]) // Column E
	}
	if len(row) > 5 {
		p.Position = strings.TrimSpace(row[5]) // Column F
	}
	if len(row) > 6 {
		p.MLBTeam = strings.TrimSpace(row[6]) // Column G
	}
	
	// Parse age
	if len(row) > 7 {
		if age, err := strconv.Atoi(strings.TrimSpace(row[7])); err == nil {
			p.Age = age
		}
	}
	
	// Parse 2024 points
	if len(row) > 8 {
		if pts, err := strconv.ParseFloat(strings.TrimSpace(row[8]), 64); err == nil {
			p.Points2024 = pts
		}
	}
	
	// Parse remaining fields
	if len(row) > 9 {
		p.OptionsLeft = strings.TrimSpace(row[9]) // Column J
	}
	if len(row) > 10 {
		p.OptionUsed = strings.TrimSpace(row[10]) // Column K
	}
	if len(row) > 11 {
		p.Status = strings.TrimSpace(row[11]) // Column L
	}
	
	// Parse contract years (2025-2038)
	// Columns M(12) through AA(26) represent years 2025-2038
	for i := 12; i <= 26 && i < len(row); i++ {
		year := 2025 + (i - 12)
		value := strings.TrimSpace(row[i])
		if value != "" {
			p.Contract[year] = value
		}
	}
	
	// Parse contract notes
	if len(row) > 27 {
		p.ContractNote = strings.TrimSpace(row[27]) // Column AB
	}
	
	return p, nil
}

// GetSalary returns the salary for a given year, parsing the dollar amount
func (p *Player) GetSalary(year int) (int, bool) {
	contractValue, exists := p.Contract[year]
	if !exists {
		return 0, false
	}
	
	// Check if it's a salary (starts with $)
	if !strings.HasPrefix(contractValue, "$") {
		return 0, false
	}
	
	// Remove $ and commas, then parse
	salaryStr := strings.TrimPrefix(contractValue, "$")
	salaryStr = strings.ReplaceAll(salaryStr, ",", "")
	
	salary, err := strconv.Atoi(salaryStr)
	if err != nil {
		return 0, false
	}
	
	return salary, true
}

// IsFreeAgent checks if the player is a free agent in a given year
func (p *Player) IsFreeAgent(year int) bool {
	contractValue, exists := p.Contract[year]
	return exists && strings.Contains(contractValue, "FREE AGENT")
}

// HasContract checks if the player has any contract information
func (p *Player) HasContract() bool {
	for _, v := range p.Contract {
		if v != "" {
			return true
		}
	}
	return false
}