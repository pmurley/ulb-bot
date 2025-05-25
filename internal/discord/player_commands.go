package discord

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/pmurley/ulb-bot/internal/models"
)

// handlePlayer looks up a player by name and displays their info
func (hm *HandlerManager) handlePlayer(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	if len(args) == 0 {
		s.ChannelMessageSend(m.ChannelID, "Usage: `!player <player name>`")
		return
	}

	// Join args to handle multi-word names
	playerName := strings.Join(args, " ")

	// Get players from cache (auto-reload if needed)
	players, err := hm.ensurePlayersLoaded()
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "Failed to load player data: " + err.Error())
		return
	}

	// Try exact match first
	exactMatches := players.FindByExactName(playerName)
	
	// If exact matches found, display all of them
	if len(exactMatches) > 0 {
		if len(exactMatches) == 1 {
			// Single exact match
			embed := buildPlayerEmbed(&exactMatches[0])
			s.ChannelMessageSendEmbed(m.ChannelID, embed)
		} else {
			// Multiple exact matches - show all
			var embeds []*discordgo.MessageEmbed
			for _, player := range exactMatches {
				p := player // capture for pointer
				embed := buildPlayerEmbed(&p)
				embeds = append(embeds, embed)
			}
			// Discord allows up to 10 embeds per message
			for i := 0; i < len(embeds); i += 10 {
				end := i + 10
				if end > len(embeds) {
					end = len(embeds)
				}
				s.ChannelMessageSendEmbeds(m.ChannelID, embeds[i:end])
			}
		}
		return
	}
	
	// No exact match, try partial search
	matches := players.SearchByName(playerName)
	if len(matches) == 0 {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("No player found matching '%s'", playerName))
		return
	}
	
	// If multiple matches, show them
	if len(matches) > 1 {
		msg := fmt.Sprintf("Multiple players found matching '%s':\n", playerName)
		for i, p := range matches {
			if i >= 10 { // Limit to 10 results
				msg += fmt.Sprintf("... and %d more\n", len(matches)-10)
				break
			}
			msg += fmt.Sprintf("• %s (%s, %s)\n", p.Name, p.Position, p.MLBTeam)
		}
		msg += "\nPlease be more specific."
		s.ChannelMessageSend(m.ChannelID, msg)
		return
	}
	
	// Single match found
	embed := buildPlayerEmbed(&matches[0])
	s.ChannelMessageSendEmbed(m.ChannelID, embed)
}

// buildPlayerEmbed creates a rich embed for player information
func buildPlayerEmbed(p *models.Player) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{
		Title: p.Name,
		Color: getTeamColor(p.ULBTeam),
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Position",
				Value:  p.Position,
				Inline: true,
			},
			{
				Name:   "MLB Team",
				Value:  p.MLBTeam,
				Inline: true,
			},
			{
				Name:   "Age",
				Value:  fmt.Sprintf("%d", p.Age),
				Inline: true,
			},
		},
	}

	// Add ULB team if they have one
	if p.ULBTeam != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "ULB Team",
			Value:  p.ULBTeam,
			Inline: true,
		})
	} else {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "ULB Team",
			Value:  "Free Agent",
			Inline: true,
		})
	}


	// Add status
	if p.Status != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Status",
			Value:  p.Status,
			Inline: true,
		})
	}

	// Build contract info
	contractInfo := buildContractInfo(p)
	if contractInfo != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Contract",
			Value:  contractInfo,
			Inline: false,
		})
	}

	// Add contract notes if present
	if p.ContractNote != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Contract Notes",
			Value:  p.ContractNote,
			Inline: false,
		})
	}

	return embed
}

// buildContractInfo formats the player's contract information
func buildContractInfo(p *models.Player) string {
	var parts []string
	currentYear := 2025 // We should make this dynamic eventually
	
	// Look for contract info for the next few years
	for year := currentYear; year <= currentYear+5; year++ {
		if value, exists := p.Contract[year]; exists && value != "" {
			if p.IsFreeAgent(year) {
				parts = append(parts, fmt.Sprintf("%d: FREE AGENT", year))
				break // Don't show years after free agency
			} else if salary, ok := p.GetSalary(year); ok {
				parts = append(parts, fmt.Sprintf("%d: $%s", year, formatNumber(salary)))
			} else if value != "" {
				parts = append(parts, fmt.Sprintf("%d: %s", year, value))
			}
		}
	}

	if len(parts) == 0 {
		return "No contract information"
	}

	return strings.Join(parts, "\n")
}

// getTeamColor returns a color for the team (you can customize these)
func getTeamColor(team string) int {
	// Default blue color
	return 0x3498db
}

// handlePlayers looks up multiple players by name and displays their info
func (hm *HandlerManager) handlePlayers(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	if len(args) == 0 {
		s.ChannelMessageSend(m.ChannelID, "Usage: `!players <player1>, <player2>, <player3>, ...`")
		return
	}

	// Join args and split by comma
	playerList := strings.Join(args, " ")
	playerNames := strings.Split(playerList, ",")

	// Get players from cache (auto-reload if needed)
	players, err := hm.ensurePlayersLoaded()
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "Failed to load player data: " + err.Error())
		return
	}

	// Build results message
	var embeds []*discordgo.MessageEmbed
	var notFound []string

	for _, name := range playerNames {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}

		// Try exact match first
		exactMatches := players.FindByExactName(name)
		
		if len(exactMatches) > 0 {
			// Add all exact matches
			for _, player := range exactMatches {
				p := player // capture for pointer
				embed := buildCompactPlayerEmbed(&p)
				embeds = append(embeds, embed)
			}
		} else {
			// No exact match, try partial search
			matches := players.SearchByName(name)
			if len(matches) == 0 {
				notFound = append(notFound, name)
				continue
			}
			// Use first match for partial searches
			embed := buildCompactPlayerEmbed(&matches[0])
			embeds = append(embeds, embed)
		}
	}

	// Send results
	if len(embeds) == 0 {
		s.ChannelMessageSend(m.ChannelID, "No players found.")
		return
	}

	// Discord has a limit of 10 embeds per message
	for i := 0; i < len(embeds); i += 10 {
		end := i + 10
		if end > len(embeds) {
			end = len(embeds)
		}
		s.ChannelMessageSendEmbeds(m.ChannelID, embeds[i:end])
	}

	// Report not found players
	if len(notFound) > 0 {
		msg := fmt.Sprintf("\n**Not found:** %s", strings.Join(notFound, ", "))
		s.ChannelMessageSend(m.ChannelID, msg)
	}
}

// buildCompactPlayerEmbed creates a more compact embed for multiple player display
func buildCompactPlayerEmbed(p *models.Player) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{
		Title: p.Name,
		Color: getTeamColor(p.ULBTeam),
	}

	// Build description with key info
	var desc []string
	
	// Basic info line
	teamInfo := p.ULBTeam
	if teamInfo == "" {
		teamInfo = "Free Agent"
	}
	desc = append(desc, fmt.Sprintf("**%s** | %s | Age %d", 
		p.Position, p.MLBTeam, p.Age))
	desc = append(desc, fmt.Sprintf("**ULB:** %s", teamInfo))

	// Contract info (simplified)
	contractParts := []string{}
	for year := 2025; year <= 2027; year++ { // Show next 3 years
		if value, exists := p.Contract[year]; exists && value != "" {
			if p.IsFreeAgent(year) {
				contractParts = append(contractParts, fmt.Sprintf("%d: FA", year))
				break
			} else if salary, ok := p.GetSalary(year); ok {
				contractParts = append(contractParts, fmt.Sprintf("%d: $%s", year, formatNumberShort(salary)))
			}
		}
	}
	
	if len(contractParts) > 0 {
		desc = append(desc, "**Contract:** "+strings.Join(contractParts, " | "))
	}

	embed.Description = strings.Join(desc, "\n")

	return embed
}

// formatNumber adds commas to large numbers
func formatNumber(n int) string {
	str := fmt.Sprintf("%d", n)
	if len(str) <= 3 {
		return str
	}
	
	result := ""
	for i, digit := range str {
		if i > 0 && (len(str)-i)%3 == 0 {
			result += ","
		}
		result += string(digit)
	}
	return result
}

// formatNumberShort formats large numbers with K/M suffix
func formatNumberShort(n int) string {
	if n >= 1000000 {
		return fmt.Sprintf("%.1fM", float64(n)/1000000)
	} else if n >= 1000 {
		return fmt.Sprintf("%.0fK", float64(n)/1000)
	}
	return fmt.Sprintf("%d", n)
}