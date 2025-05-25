package discord

import (
	"fmt"
	"sort"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/pmurley/ulb-bot/internal/models"
)

// handleTeam displays the roster for a specific team
func (hm *HandlerManager) handleTeam(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	if len(args) == 0 {
		s.ChannelMessageSend(m.ChannelID, "Usage: `!team <team name>`")
		return
	}

	// Join args to handle multi-word team names
	teamName := strings.Join(args, " ")

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

	// Build team roster embed
	embed := buildTeamRosterEmbed(teamName, teamPlayers)
	s.ChannelMessageSendEmbed(m.ChannelID, embed)
}

// buildTeamRosterEmbed creates a rich embed for team roster
func buildTeamRosterEmbed(teamName string, players models.PlayerList) *discordgo.MessageEmbed {
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

	// Build embed
	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("%s Roster", teamName),
		Color:       getTeamColor(teamName),
		Description: fmt.Sprintf("**%d Players | 2025 Payroll: $%s**", len(players), formatNumber(totalPayroll)),
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

