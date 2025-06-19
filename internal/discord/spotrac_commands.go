package discord

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/pmurley/ulb-bot/internal/spotrac"
)

// handleSpotrac looks up a player's contract information from Spotrac
func (hm *HandlerManager) handleSpotrac(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	if len(args) == 0 {
		s.ChannelMessageSend(m.ChannelID, "Usage: `!spotrac <player name>`")
		return
	}

	// Join args to handle multi-word names
	playerName := strings.Join(args, " ")

	// Search for player on Spotrac
	result, err := hm.spotracClient.Search(playerName)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "Failed to search Spotrac: "+err.Error())
		return
	}

	switch result.Type {
	case "none":
		message := "No players found on Spotrac"
		if result.ErrorMessage != "" {
			message = result.ErrorMessage
		}
		s.ChannelMessageSend(m.ChannelID, message)
		return

	case "multiple":
		// Show multiple results and ask user to be more specific
		embed := buildSpotracMultipleResultsEmbed(result, playerName)
		s.ChannelMessageSendEmbed(m.ChannelID, embed)
		return

	case "single":
		// Get detailed contract information
		player := result.PlayerResults[0]
		contract, err := hm.spotracClient.GetPlayerContract(player.URL)
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "Failed to get contract information: "+err.Error())
			return
		}

		// Build and send contract embed
		embed := buildSpotracContractEmbed(contract)
		s.ChannelMessageSendEmbed(m.ChannelID, embed)
	}
}

// buildSpotracMultipleResultsEmbed creates an embed for multiple search results
func buildSpotracMultipleResultsEmbed(result *spotrac.SearchResult, query string) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{
		Title: fmt.Sprintf("Multiple players found for '%s'", query),
		Color: 0xFFA500, // Orange
	}

	var description strings.Builder
	description.WriteString("Please be more specific. Found players:\n\n")

	// Show up to 20 results to avoid hitting Discord's limits
	maxResults := len(result.PlayerResults)
	if maxResults > 20 {
		maxResults = 20
	}

	for i := 0; i < maxResults; i++ {
		player := result.PlayerResults[i]
		description.WriteString(fmt.Sprintf("**%s**", player.Name))
		if player.Team != "" {
			description.WriteString(fmt.Sprintf(" (%s)", player.Team))
		}
		if player.Position != "" {
			description.WriteString(fmt.Sprintf(" - %s", player.Position))
		}
		description.WriteString("\n")
	}

	if len(result.PlayerResults) > 20 {
		description.WriteString(fmt.Sprintf("\n*...and %d more results*", len(result.PlayerResults)-20))
	}

	embed.Description = description.String()
	return embed
}

// buildSpotracContractEmbed creates an embed for a player's contract information
func buildSpotracContractEmbed(contract *spotrac.ContractInfo) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{
		Title: fmt.Sprintf("%s - Contract Information", contract.PlayerName),
		Color: 0x00FF00, // Green
	}

	var fields []*discordgo.MessageEmbedField

	// Basic contract info
	if contract.Team != "" {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "Team",
			Value:  contract.Team,
			Inline: true,
		})
	}

	// Contract details
	if contract.ContractTerms != "" && contract.ContractTerms != "1 yr(s)" {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "Contract",
			Value:  contract.ContractTerms,
			Inline: true,
		})

		if contract.AverageSalary != "" {
			fields = append(fields, &discordgo.MessageEmbedField{
				Name:   "Average Salary",
				Value:  contract.AverageSalary,
				Inline: true,
			})
		}

		if contract.SigningBonus != "" && contract.SigningBonus != "N/A" {
			fields = append(fields, &discordgo.MessageEmbedField{
				Name:   "Signing Bonus",
				Value:  contract.SigningBonus,
				Inline: true,
			})
		}
	} else {
		// Player has no multi-year contract
		statusValue := "No Major League Contract"
		if contract.Status != "" {
			statusValue = contract.Status
		} else if len(contract.ContractYears) > 0 {
			currentYear := contract.ContractYears[0]
			if currentYear.Status != "" {
				statusValue = currentYear.Status
			} else if currentYear.PayrollTotal != "" && currentYear.PayrollTotal != "-" {
				statusValue = fmt.Sprintf("2025 Salary: %s", currentYear.PayrollTotal)
			}
		}

		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "Status",
			Value:  statusValue,
			Inline: false,
		})
	}

	if contract.FreeAgent != "" {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "Free Agent",
			Value:  contract.FreeAgent,
			Inline: true,
		})
	}

	// Contract notes
	if len(contract.ContractNotes) > 0 {
		var notesText strings.Builder
		for i, note := range contract.ContractNotes {
			if i > 0 {
				notesText.WriteString("\n")
			}
			notesText.WriteString(fmt.Sprintf("• %s", note))
		}

		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "Contract Notes",
			Value:  notesText.String(),
			Inline: false,
		})
	}

	// Contract breakdown (year-by-year) as ASCII table in code block
	if len(contract.ContractYears) > 0 {
		var breakdown strings.Builder
		breakdown.WriteString("```\n")
		breakdown.WriteString("Year  Age  Status      Salary\n")
		breakdown.WriteString("----  ---  ----------  ----------------\n")

		for _, year := range contract.ContractYears {
			ageStr := ""
			if year.Age > 0 {
				ageStr = strconv.Itoa(year.Age)
			}

			statusStr := year.Status
			if len(statusStr) > 10 {
				statusStr = statusStr[:10]
			}

			salaryStr := year.PayrollTotal
			if salaryStr == "" {
				salaryStr = "-"
			}

			breakdown.WriteString(fmt.Sprintf("%-4d  %-3s  %-10s  %s\n",
				year.Year, ageStr, statusStr, salaryStr))
		}
		breakdown.WriteString("```")

		// Discord field values have a limit of 1024 characters
		breakdownText := breakdown.String()
		if len(breakdownText) > 1024 {
			// Truncate if too long
			breakdownText = breakdownText[:1020] + "..."
		}

		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "Contract Breakdown",
			Value:  breakdownText,
			Inline: false,
		})
	}

	embed.Fields = fields
	embed.Footer = &discordgo.MessageEmbedFooter{
		Text: "Data from Spotrac.com",
	}

	return embed
}
