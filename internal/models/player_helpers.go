package models

import (
	"sort"
	"strings"
)

// PlayerList represents a slice of players with helper methods
type PlayerList []Player

// FilterByTeam returns players belonging to a specific ULB team
func (pl PlayerList) FilterByTeam(teamName string) PlayerList {
	var filtered PlayerList
	teamLower := strings.ToLower(strings.TrimSpace(teamName))

	for _, p := range pl {
		if strings.ToLower(strings.TrimSpace(p.ULBTeam)) == teamLower {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

// FilterByMLBTeam returns players on a specific MLB team
func (pl PlayerList) FilterByMLBTeam(mlbTeam string) PlayerList {
	var filtered PlayerList
	mlbLower := strings.ToLower(mlbTeam)

	for _, p := range pl {
		if strings.ToLower(p.MLBTeam) == mlbLower {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

// FilterByPosition returns players who can play a specific position
func (pl PlayerList) FilterByPosition(position string) PlayerList {
	var filtered PlayerList
	posLower := strings.ToLower(position)

	// Define composite positions
	compositePositions := map[string][]string{
		"mi": {"2b", "ss", "mi"},       // Middle Infield (including players listed as just MI)
		"ci": {"1b", "3b"},             // Corner Infield
		"if": {"1b", "2b", "3b", "ss"}, // All Infield
		"of": {"lf", "cf", "rf", "of"}, // All Outfield (including generic OF)
		"ut": {"ut"},                   // Utility (players listed as UT)
	}

	// Check if this is a composite position
	validPositions := []string{posLower}
	if composites, exists := compositePositions[posLower]; exists {
		validPositions = composites
	}

	for _, p := range pl {
		positions := strings.Split(strings.ToLower(p.Position), ",")
		for _, pos := range positions {
			pos = strings.TrimSpace(pos)
			// Check if player's position matches any of the valid positions
			for _, validPos := range validPositions {
				if pos == validPos {
					filtered = append(filtered, p)
					goto nextPlayer
				}
			}
		}
	nextPlayer:
	}
	return filtered
}

// FilterByStatus returns players with a specific status (e.g., "40-Man")
func (pl PlayerList) FilterByStatus(status string) PlayerList {
	var filtered PlayerList
	statusLower := strings.ToLower(status)

	for _, p := range pl {
		if strings.ToLower(p.Status) == statusLower {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

// SearchByName returns players whose names contain the search string
func (pl PlayerList) SearchByName(search string) PlayerList {
	var matches PlayerList
	searchLower := strings.ToLower(search)

	for _, p := range pl {
		if strings.Contains(strings.ToLower(p.Name), searchLower) {
			matches = append(matches, p)
		}
	}
	return matches
}

// FindByExactName returns all players with an exact name match (case-insensitive)
func (pl PlayerList) FindByExactName(name string) []Player {
	nameLower := strings.ToLower(name)
	var matches []Player

	for _, p := range pl {
		if strings.ToLower(p.Name) == nameLower {
			matches = append(matches, p)
		}
	}
	return matches
}

// GetFreeAgents returns players who are free agents in the specified year
func (pl PlayerList) GetFreeAgents(year int) PlayerList {
	var freeAgents PlayerList

	for _, p := range pl {
		if p.IsFreeAgent(year) {
			freeAgents = append(freeAgents, p)
		}
	}
	return freeAgents
}

// GetUnownedPlayers returns players not on any ULB team
func (pl PlayerList) GetUnownedPlayers() PlayerList {
	var unowned PlayerList

	for _, p := range pl {
		if p.ULBTeam == "" {
			unowned = append(unowned, p)
		}
	}
	return unowned
}

// SortByPoints sorts players by 2024 points (descending)
func (pl PlayerList) SortByPoints() {
	sort.Slice(pl, func(i, j int) bool {
		return pl[i].Points2024 > pl[j].Points2024
	})
}

// SortBySalary sorts players by salary for a given year (descending)
func (pl PlayerList) SortBySalary(year int) {
	sort.Slice(pl, func(i, j int) bool {
		salaryI, _ := pl[i].GetSalary(year)
		salaryJ, _ := pl[j].GetSalary(year)
		return salaryI > salaryJ
	})
}

// SortByName sorts players alphabetically by name
func (pl PlayerList) SortByName() {
	sort.Slice(pl, func(i, j int) bool {
		return pl[i].Name < pl[j].Name
	})
}

// GetTopPerformers returns the top N players by 2024 points
func (pl PlayerList) GetTopPerformers(n int) PlayerList {
	sorted := make(PlayerList, len(pl))
	copy(sorted, pl)
	sorted.SortByPoints()

	if n > len(sorted) {
		n = len(sorted)
	}
	return sorted[:n]
}

// GetTopSalaries returns the top N players by salary for a given year
func (pl PlayerList) GetTopSalaries(year int, n int) PlayerList {
	sorted := make(PlayerList, len(pl))
	copy(sorted, pl)
	sorted.SortBySalary(year)

	if n > len(sorted) {
		n = len(sorted)
	}
	return sorted[:n]
}

// GetTeamPayroll calculates total payroll for a team in a given year
func (pl PlayerList) GetTeamPayroll(teamName string, year int) int {
	teamPlayers := pl.FilterByTeam(teamName)
	total := 0

	for _, p := range teamPlayers {
		if salary, ok := p.GetSalary(year); ok {
			total += salary
		}
	}
	return total
}

// GetPositionEligible returns players eligible at multiple positions
func (pl PlayerList) GetPositionEligible(positions []string) PlayerList {
	var eligible PlayerList

	for _, p := range pl {
		playerPositions := strings.Split(strings.ToLower(p.Position), ",")
		for _, reqPos := range positions {
			reqPosLower := strings.ToLower(strings.TrimSpace(reqPos))
			for _, playerPos := range playerPositions {
				if strings.TrimSpace(playerPos) == reqPosLower {
					eligible = append(eligible, p)
					goto nextPlayer
				}
			}
		}
	nextPlayer:
	}
	return eligible
}

// GroupByTeam returns a map of team name to players
func (pl PlayerList) GroupByTeam() map[string]PlayerList {
	grouped := make(map[string]PlayerList)

	for _, p := range pl {
		if p.ULBTeam != "" {
			grouped[p.ULBTeam] = append(grouped[p.ULBTeam], p)
		}
	}
	return grouped
}

// GroupByPosition returns a map of position to players
func (pl PlayerList) GroupByPosition() map[string]PlayerList {
	grouped := make(map[string]PlayerList)

	for _, p := range pl {
		positions := strings.Split(p.Position, ",")
		for _, pos := range positions {
			pos = strings.TrimSpace(pos)
			if pos != "" {
				grouped[pos] = append(grouped[pos], p)
			}
		}
	}
	return grouped
}

// Stats represents aggregate statistics for a group of players
type Stats struct {
	Count          int
	TotalPoints    float64
	AveragePoints  float64
	TotalSalary    int
	AverageSalary  int
	FreeAgentCount int
}

// GetStats returns aggregate statistics for the player list
func (pl PlayerList) GetStats(year int) Stats {
	stats := Stats{Count: len(pl)}

	salaryCount := 0
	for _, p := range pl {
		stats.TotalPoints += p.Points2024

		if salary, ok := p.GetSalary(year); ok {
			stats.TotalSalary += salary
			salaryCount++
		}

		if p.IsFreeAgent(year) {
			stats.FreeAgentCount++
		}
	}

	if stats.Count > 0 {
		stats.AveragePoints = stats.TotalPoints / float64(stats.Count)
	}
	if salaryCount > 0 {
		stats.AverageSalary = stats.TotalSalary / salaryCount
	}

	return stats
}
