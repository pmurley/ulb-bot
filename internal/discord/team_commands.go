package discord

import (
	"fmt"
	"sort"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/pmurley/ulb-bot/internal/models"
)

// TeamFilters represents filtering options for team roster
type TeamFilters struct {
	Status   string // "40-man" or "minors"
	Position string // Position to filter by
	MinAge   int    // Minimum age
	MaxAge   int    // Maximum age
}

// handleTeam displays the roster for a specific team with optional filters
func (hm *HandlerManager) handleTeam(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	if len(args) == 0 {
		s.ChannelMessageSend(m.ChannelID, "Usage: `!team <team name> [--status=<40-man|minors>] [--position=<pos>] [--age=<min-max>]`")
		return
	}

	// Parse args to separate team name from filters
	teamNameParts := []string{}
	filters := TeamFilters{}
	
	for _, arg := range args {
		if strings.HasPrefix(arg, "--") {
			// Parse filter
			parts := strings.SplitN(arg, "=", 2)
			if len(parts) == 2 {
				switch parts[0] {
				case "--status":
					filters.Status = strings.ToLower(parts[1])
				case "--position", "--pos":
					filters.Position = strings.ToUpper(parts[1])
				case "--age":
					// Parse age range (e.g., "20-25" or "25+")
					if strings.Contains(parts[1], "-") {
						ageParts := strings.Split(parts[1], "-")
						if len(ageParts) == 2 {
							fmt.Sscanf(ageParts[0], "%d", &filters.MinAge)
							fmt.Sscanf(ageParts[1], "%d", &filters.MaxAge)
						}
					} else if strings.HasSuffix(parts[1], "+") {
						fmt.Sscanf(parts[1], "%d+", &filters.MinAge)
						filters.MaxAge = 99
					} else {
						// Single age value
						var age int
						fmt.Sscanf(parts[1], "%d", &age)
						filters.MinAge = age
						filters.MaxAge = age
					}
				}
			}
		} else {
			teamNameParts = append(teamNameParts, arg)
		}
	}
	
	if len(teamNameParts) == 0 {
		s.ChannelMessageSend(m.ChannelID, "Please specify a team name")
		return
	}

	teamName := strings.Join(teamNameParts, " ")

	// Get players from cache (auto-reload if needed)
	players, err := hm.ensurePlayersLoaded()
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "Failed to load player data: " + err.Error())
		return
	}

	// Find team (case-insensitive)
	teamPlayers := players.FilterByTeam(teamName)
	
	if len(teamPlayers) == 0 {
		// Try to find similar team names
		allTeams := getAllTeamNames(players)
		suggestions := findSimilarTeams(teamName, allTeams)
		
		msg := fmt.Sprintf("No team found matching '%s'", teamName)
		if len(suggestions) > 0 {
			msg += "\n\nDid you mean:\n"
			for _, team := range suggestions {
				msg += fmt.Sprintf("• %s\n", team)
			}
		}
		s.ChannelMessageSend(m.ChannelID, msg)
		return
	}

	// Apply filters
	filteredPlayers := applyTeamFilters(teamPlayers, filters)
	
	if len(filteredPlayers) == 0 {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("No players found for %s with the specified filters", teamName))
		return
	}

	// Build team roster embed
	embed := buildTeamRosterEmbed(teamName, filteredPlayers, filters)
	s.ChannelMessageSendEmbed(m.ChannelID, embed)
}

// applyTeamFilters applies the specified filters to the player list
func applyTeamFilters(players models.PlayerList, filters TeamFilters) models.PlayerList {
	filtered := players
	
	// Filter by status
	if filters.Status != "" {
		var statusFiltered models.PlayerList
		for _, p := range filtered {
			statusLower := strings.ToLower(p.Status)
			if filters.Status == "40-man" && strings.Contains(statusLower, "40") {
				statusFiltered = append(statusFiltered, p)
			} else if filters.Status == "minors" && !strings.Contains(statusLower, "40") {
				statusFiltered = append(statusFiltered, p)
			}
		}
		filtered = statusFiltered
	}
	
	// Filter by position
	if filters.Position != "" {
		filtered = filtered.FilterByPosition(filters.Position)
	}
	
	// Filter by age
	if filters.MinAge > 0 || filters.MaxAge > 0 {
		var ageFiltered models.PlayerList
		for _, p := range filtered {
			if filters.MinAge > 0 && p.Age < filters.MinAge {
				continue
			}
			if filters.MaxAge > 0 && p.Age > filters.MaxAge {
				continue
			}
			ageFiltered = append(ageFiltered, p)
		}
		filtered = ageFiltered
	}
	
	return filtered
}

// buildTeamRosterEmbed creates a rich embed for team roster
func buildTeamRosterEmbed(teamName string, players models.PlayerList, filters TeamFilters) *discordgo.MessageEmbed {
	// Group players by position
	positionGroups := make(map[string][]models.Player)
	positionOrder := []string{"C", "1B", "2B", "3B", "SS", "OF", "DH", "SP", "RP"}
	
	for _, player := range players {
		// Parse positions (could be comma-separated)
		positions := strings.Split(player.Position, ",")
		primaryPos := strings.TrimSpace(positions[0])
		
		// Normalize position grouping
		if primaryPos == "LF" || primaryPos == "CF" || primaryPos == "RF" {
			primaryPos = "OF"
		}
		
		positionGroups[primaryPos] = append(positionGroups[primaryPos], player)
	}

	// Sort players within each position by salary (descending)
	year := 2025
	for pos := range positionGroups {
		sort.Slice(positionGroups[pos], func(i, j int) bool {
			salaryI, _ := positionGroups[pos][i].GetSalary(year)
			salaryJ, _ := positionGroups[pos][j].GetSalary(year)
			return salaryI > salaryJ
		})
	}

	// Calculate team payroll
	totalPayroll := 0
	playerCount := 0
	for _, player := range players {
		if salary, ok := player.GetSalary(year); ok {
			totalPayroll += salary
			playerCount++
		}
	}

	// Build filter description
	filterDesc := ""
	if filters.Status != "" || filters.Position != "" || filters.MinAge > 0 || filters.MaxAge > 0 {
		filterParts := []string{}
		if filters.Status != "" {
			filterParts = append(filterParts, fmt.Sprintf("Status: %s", filters.Status))
		}
		if filters.Position != "" {
			filterParts = append(filterParts, fmt.Sprintf("Position: %s", filters.Position))
		}
		if filters.MinAge > 0 || filters.MaxAge > 0 {
			if filters.MinAge == filters.MaxAge {
				filterParts = append(filterParts, fmt.Sprintf("Age: %d", filters.MinAge))
			} else if filters.MaxAge == 99 {
				filterParts = append(filterParts, fmt.Sprintf("Age: %d+", filters.MinAge))
			} else {
				filterParts = append(filterParts, fmt.Sprintf("Age: %d-%d", filters.MinAge, filters.MaxAge))
			}
		}
		filterDesc = "\n*Filters: " + strings.Join(filterParts, ", ") + "*"
	}

	// Build embed
	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("%s Roster", teamName),
		Color:       getTeamColor(teamName),
		Description: fmt.Sprintf("**%d Players | 2025 Payroll: $%s**%s", len(players), formatNumber(totalPayroll), filterDesc),
		Fields:      []*discordgo.MessageEmbedField{},
	}

	// Add fields for each position group
	for _, pos := range positionOrder {
		if players, exists := positionGroups[pos]; exists && len(players) > 0 {
			fieldValue := ""
			for _, player := range players {
				salary2025 := "N/A"
				if sal, ok := player.GetSalary(year); ok {
					salary2025 = "$" + formatNumberShort(sal)
				} else if player.IsFreeAgent(year) {
					salary2025 = "FA"
				}
				
				// Include age and MLB team
				fieldValue += fmt.Sprintf("**%s** (%d, %s) - %s\n", 
					player.Name, player.Age, player.MLBTeam, salary2025)
			}
			
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:   fmt.Sprintf("%s (%d)", pos, len(players)),
				Value:  fieldValue,
				Inline: true,
			})
		}
	}

	// Add any players with unknown positions
	var unknownPos []models.Player
	for _, player := range players {
		if player.Position == "" || player.Position == "N/A" {
			unknownPos = append(unknownPos, player)
		}
	}
	
	if len(unknownPos) > 0 {
		fieldValue := ""
		for _, player := range unknownPos {
			salary2025 := "N/A"
			if sal, ok := player.GetSalary(year); ok {
				salary2025 = "$" + formatNumberShort(sal)
			}
			fieldValue += fmt.Sprintf("**%s** - %s\n", player.Name, salary2025)
		}
		
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   fmt.Sprintf("Unknown Position (%d)", len(unknownPos)),
			Value:  fieldValue,
			Inline: true,
		})
	}

	// Add footer with salary summary
	embed.Footer = &discordgo.MessageEmbedFooter{
		Text: fmt.Sprintf("Average Salary: $%s | Roster Size: %d", 
			formatNumber(totalPayroll/len(players)), len(players)),
	}

	return embed
}

// getAllTeamNames returns a list of all unique team names
func getAllTeamNames(players models.PlayerList) []string {
	teamMap := make(map[string]bool)
	for _, player := range players {
		if player.ULBTeam != "" {
			teamMap[player.ULBTeam] = true
		}
	}
	
	teams := make([]string, 0, len(teamMap))
	for team := range teamMap {
		teams = append(teams, team)
	}
	sort.Strings(teams)
	return teams
}

// findSimilarTeams finds teams with similar names
func findSimilarTeams(search string, allTeams []string) []string {
	searchLower := strings.ToLower(search)
	var matches []string
	
	for _, team := range allTeams {
		teamLower := strings.ToLower(team)
		// Check if team contains search string
		if strings.Contains(teamLower, searchLower) {
			matches = append(matches, team)
		} else if strings.Contains(searchLower, teamLower) {
			matches = append(matches, team)
		}
	}
	
	// Limit to 5 suggestions
	if len(matches) > 5 {
		matches = matches[:5]
	}
	
	return matches
}

