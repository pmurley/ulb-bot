package discord

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/pmurley/ulb-bot/internal/models"
)

// handleTrade analyzes a trade between two teams
func (hm *HandlerManager) handleTrade(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	if len(args) == 0 {
		helpMsg := "Usage: `!trade <team1 players> for <team2 players>`\n" +
			"Example: `!trade Juan Soto, Aaron Judge for Shohei Ohtani`\n" +
			"With retention: `!trade Ohtani (retain 25%) for Judge`\n" +
			"With cash: `!trade Player, cash ($5M) for Player`\n" +
			"Add `-v` or `--verbose` for full contract details"
		s.ChannelMessageSend(m.ChannelID, helpMsg)
		return
	}

	// Check for verbose flag
	verbose := false
	if args[0] == "-v" || args[0] == "--verbose" {
		verbose = true
		args = args[1:] // Remove flag from args
		if len(args) == 0 {
			s.ChannelMessageSend(m.ChannelID, "Please specify players after the verbose flag.")
			return
		}
	}

	// Join args and look for "for" separator
	tradeStr := strings.Join(args, " ")
	parts := strings.Split(strings.ToLower(tradeStr), " for ")
	if len(parts) != 2 {
		s.ChannelMessageSend(m.ChannelID, "Invalid format. Use: `!trade <players> for <players>`")
		return
	}

	// Get the original case version
	originalParts := strings.Split(tradeStr, " for ")
	if len(originalParts) != 2 {
		originalParts = strings.Split(tradeStr, " FOR ")
	}

	// Parse player lists with retention
	side1Info := parsePlayerList(originalParts[0])
	side2Info := parsePlayerList(originalParts[1])

	if len(side1Info) == 0 || len(side2Info) == 0 {
		s.ChannelMessageSend(m.ChannelID, "Please specify at least one player on each side of the trade.")
		return
	}

	// Get players from cache (auto-reload if needed)
	players, err := hm.ensurePlayersLoaded()
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "Failed to load player data: "+err.Error())
		return
	}

	// Find players for each side with retention info
	side1Players, side1Cash, side1NotFound := findPlayersWithRetention(players, side1Info)
	side2Players, side2Cash, side2NotFound := findPlayersWithRetention(players, side2Info)

	// Report not found players
	if len(side1NotFound) > 0 || len(side2NotFound) > 0 {
		msg := "**Players not found:**\n"
		if len(side1NotFound) > 0 {
			msg += fmt.Sprintf("Side 1: %s\n", strings.Join(side1NotFound, ", "))
		}
		if len(side2NotFound) > 0 {
			msg += fmt.Sprintf("Side 2: %s\n", strings.Join(side2NotFound, ", "))
		}
		s.ChannelMessageSend(m.ChannelID, msg)
		return
	}

	// Analyze the trade
	analysis := hm.analyzeTrade(side1Players, side2Players, side1Cash, side2Cash, verbose)

	// Create embed
	embed := buildTradeEmbed(analysis, verbose)
	s.ChannelMessageSendEmbed(m.ChannelID, embed)
}

// PlayerWithRetention represents a player name with optional retention percentage
type PlayerWithRetention struct {
	Name             string
	RetentionPercent float64
	IsCash           bool
	CashAmount       int
}

// parsePlayerList splits a comma-separated list of player names with optional retention
func parsePlayerList(input string) []PlayerWithRetention {
	names := strings.Split(input, ",")
	var result []PlayerWithRetention

	for _, entry := range names {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		pwr := PlayerWithRetention{}

		// Check if this is cash considerations
		lowerEntry := strings.ToLower(entry)
		if strings.HasPrefix(lowerEntry, "cash") && strings.Contains(entry, "($") {
			pwr.IsCash = true
			pwr.Name = "Cash Considerations"

			// Extract cash amount
			dollarIdx := strings.Index(entry, "($")
			if dollarIdx != -1 {
				amountStr := entry[dollarIdx+2:]
				endIdx := strings.Index(amountStr, ")")
				if endIdx != -1 {
					amountStr = amountStr[:endIdx]
					// Remove commas and parse
					amountStr = strings.ReplaceAll(amountStr, ",", "")
					// Handle M or K suffix
					if strings.HasSuffix(strings.ToUpper(amountStr), "M") {
						if val, err := strconv.ParseFloat(amountStr[:len(amountStr)-1], 64); err == nil {
							pwr.CashAmount = int(val * 1000000)
						}
					} else if strings.HasSuffix(strings.ToUpper(amountStr), "K") {
						if val, err := strconv.ParseFloat(amountStr[:len(amountStr)-1], 64); err == nil {
							pwr.CashAmount = int(val * 1000)
						}
					} else {
						if val, err := strconv.Atoi(amountStr); err == nil {
							pwr.CashAmount = val
						}
					}
				}
			}
		} else if strings.Contains(entry, "(retain") {
			// Check for retention syntax: "Player Name (retain X%)"
			retainIdx := strings.Index(entry, "(retain")
			pwr.Name = strings.TrimSpace(entry[:retainIdx])

			// Extract retention percentage
			retainPart := entry[retainIdx:]
			var percent float64
			n, _ := fmt.Sscanf(retainPart, "(retain %f%%)", &percent)
			if n == 1 && percent > 0 && percent <= 100 {
				pwr.RetentionPercent = percent
			}
		} else {
			pwr.Name = entry
			pwr.RetentionPercent = 0
		}

		result = append(result, pwr)
	}
	return result
}

// findPlayersWithRetention looks up players and creates TradedPlayer objects with retention info
func findPlayersWithRetention(allPlayers models.PlayerList, playerInfo []PlayerWithRetention) ([]models.TradedPlayer, int, []string) {
	var found []models.TradedPlayer
	var notFound []string
	var totalCash int

	for _, info := range playerInfo {
		// Handle cash separately
		if info.IsCash {
			totalCash += info.CashAmount
			continue
		}

		// Try exact match first
		exactMatches := allPlayers.FindByExactName(info.Name)

		if len(exactMatches) > 0 {
			// For trades, if multiple exact matches, we need to disambiguate
			if len(exactMatches) > 1 {
				// Add all matches to not found with disambiguation info
				disambigInfo := fmt.Sprintf("%s (found %d players with this name - please specify team)", info.Name, len(exactMatches))
				notFound = append(notFound, disambigInfo)
				continue
			}
			// Single exact match
			tradedPlayer := models.TradedPlayer{
				Player:           exactMatches[0],
				RetentionPercent: info.RetentionPercent,
			}
			found = append(found, tradedPlayer)
		} else {
			// No exact match, try partial search
			matches := allPlayers.SearchByName(info.Name)
			if len(matches) == 0 {
				notFound = append(notFound, info.Name)
				continue
			}
			// Use first match
			tradedPlayer := models.TradedPlayer{
				Player:           matches[0],
				RetentionPercent: info.RetentionPercent,
			}
			found = append(found, tradedPlayer)
		}
	}

	return found, totalCash, notFound
}

// TradeAnalysis contains the analysis of a trade
type TradeAnalysis struct {
	Side1Players         []models.TradedPlayer
	Side2Players         []models.TradedPlayer
	Side1Teams           map[string][]models.TradedPlayer
	Side2Teams           map[string][]models.TradedPlayer
	Side1Cash            int // Cash being sent by side 1
	Side2Cash            int // Cash being sent by side 2
	PayrollChanges       map[string]PayrollChange
	YearlyPayrollChanges map[string]map[int]PayrollChange // team -> year -> change
}

// PayrollChange tracks how a team's payroll changes
type PayrollChange struct {
	TeamName      string
	PayrollBefore int // Total team payroll before trade
	PayrollAfter  int // Total team payroll after trade
	NetChange     int // PayrollAfter - PayrollBefore
}

// analyzeTrade performs analysis on the trade
func (hm *HandlerManager) analyzeTrade(side1, side2 []models.TradedPlayer, side1Cash, side2Cash int, verbose bool) TradeAnalysis {
	analysis := TradeAnalysis{
		Side1Players:         side1,
		Side2Players:         side2,
		Side1Cash:            side1Cash,
		Side2Cash:            side2Cash,
		Side1Teams:           make(map[string][]models.TradedPlayer),
		Side2Teams:           make(map[string][]models.TradedPlayer),
		PayrollChanges:       make(map[string]PayrollChange),
		YearlyPayrollChanges: make(map[string]map[int]PayrollChange),
	}

	// Get all players from cache to calculate full team payrolls
	allPlayers, err := hm.ensurePlayersLoaded()
	if err != nil {
		// If we can't load players, we can't calculate payrolls accurately
		// But we can still show the trade structure
		allPlayers = models.PlayerList{}
	}
	year := 2025

	// Group players by team and track which teams are involved
	involvedTeams := make(map[string]bool)
	for _, tp := range side1 {
		if tp.Player.ULBTeam != "" {
			analysis.Side1Teams[tp.Player.ULBTeam] = append(analysis.Side1Teams[tp.Player.ULBTeam], tp)
			involvedTeams[tp.Player.ULBTeam] = true
		}
	}
	for _, tp := range side2 {
		if tp.Player.ULBTeam != "" {
			analysis.Side2Teams[tp.Player.ULBTeam] = append(analysis.Side2Teams[tp.Player.ULBTeam], tp)
			involvedTeams[tp.Player.ULBTeam] = true
		}
	}

	// Calculate current payroll for each involved team
	for team := range involvedTeams {
		change := PayrollChange{TeamName: team}

		// Calculate total payroll before trade
		teamPlayers := allPlayers.FilterByTeam(team)
		for _, p := range teamPlayers {
			if salary, ok := p.GetSalary(year); ok {
				change.PayrollBefore += salary
			}
		}

		// Start with current payroll
		change.PayrollAfter = change.PayrollBefore

		// Subtract outgoing players (full salary) and add back retained salary
		if outgoingPlayers, exists := analysis.Side1Teams[team]; exists {
			for _, tp := range outgoingPlayers {
				if salary, ok := tp.Player.GetSalary(year); ok {
					// Remove full salary
					change.PayrollAfter -= salary
					// Add back retained portion
					change.PayrollAfter += tp.GetRetainedSalary(year)
				}
			}
		}
		if outgoingPlayers, exists := analysis.Side2Teams[team]; exists {
			for _, tp := range outgoingPlayers {
				if salary, ok := tp.Player.GetSalary(year); ok {
					// Remove full salary
					change.PayrollAfter -= salary
					// Add back retained portion
					change.PayrollAfter += tp.GetRetainedSalary(year)
				}
			}
		}

		// Add incoming players (only the portion not retained)
		// If this team is trading away side1 players, they receive side2 players
		if _, isSide1Team := analysis.Side1Teams[team]; isSide1Team {
			for _, tp := range side2 {
				// Add only the traded portion of salary
				change.PayrollAfter += tp.GetTradedSalary(year)
			}
			// Also receive side2's cash
			change.PayrollAfter += analysis.Side2Cash
			// And pay side1's cash
			change.PayrollAfter -= analysis.Side1Cash
		}
		// If this team is trading away side2 players, they receive side1 players
		if _, isSide2Team := analysis.Side2Teams[team]; isSide2Team {
			for _, tp := range side1 {
				// Add only the traded portion of salary
				change.PayrollAfter += tp.GetTradedSalary(year)
			}
			// Also receive side1's cash
			change.PayrollAfter += analysis.Side1Cash
			// And pay side2's cash
			change.PayrollAfter -= analysis.Side2Cash
		}

		change.NetChange = change.PayrollAfter - change.PayrollBefore
		analysis.PayrollChanges[team] = change
	}

	// If verbose, calculate payroll changes for each year
	if verbose {
		for team := range involvedTeams {
			analysis.YearlyPayrollChanges[team] = make(map[int]PayrollChange)

			// Calculate for years 2025-2030 (or until all players are FA)
			for year := 2025; year <= 2030; year++ {
				change := PayrollChange{TeamName: team}

				// Calculate total payroll before trade for this year
				teamPlayers := allPlayers.FilterByTeam(team)
				for _, p := range teamPlayers {
					if salary, ok := p.GetSalary(year); ok {
						change.PayrollBefore += salary
					}
				}

				// Start with current payroll
				change.PayrollAfter = change.PayrollBefore

				// Subtract outgoing players (full salary) and add back retained salary
				if outgoingPlayers, exists := analysis.Side1Teams[team]; exists {
					for _, tp := range outgoingPlayers {
						if salary, ok := tp.Player.GetSalary(year); ok {
							change.PayrollAfter -= salary
							change.PayrollAfter += tp.GetRetainedSalary(year)
						}
					}
				}
				if outgoingPlayers, exists := analysis.Side2Teams[team]; exists {
					for _, tp := range outgoingPlayers {
						if salary, ok := tp.Player.GetSalary(year); ok {
							change.PayrollAfter -= salary
							change.PayrollAfter += tp.GetRetainedSalary(year)
						}
					}
				}

				// Add incoming players (only the traded portion)
				if _, isSide1Team := analysis.Side1Teams[team]; isSide1Team {
					for _, tp := range side2 {
						change.PayrollAfter += tp.GetTradedSalary(year)
					}
				}
				if _, isSide2Team := analysis.Side2Teams[team]; isSide2Team {
					for _, tp := range side1 {
						change.PayrollAfter += tp.GetTradedSalary(year)
					}
				}

				change.NetChange = change.PayrollAfter - change.PayrollBefore
				analysis.YearlyPayrollChanges[team][year] = change
			}
		}
	}

	return analysis
}

// buildTradeEmbed creates an embed for the trade analysis
func buildTradeEmbed(analysis TradeAnalysis, verbose bool) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{
		Title: "Trade Analysis",
		Color: 0x3498db,
	}

	// Build side 1 description
	side1Team := ""
	var side1Desc []string
	for _, tp := range analysis.Side1Players {
		p := tp.Player
		side1Team = p.ULBTeam
		if side1Team == "" {
			side1Team = "Unowned"
		}

		if verbose {
			// Show full contract details
			contractYears := []string{}
			for year := 2025; year <= 2038; year++ {
				if val, exists := p.Contract[year]; exists && val != "" {
					if p.IsFreeAgent(year) {
						contractYears = append(contractYears, fmt.Sprintf("%d: FA", year))
						break
					} else if sal, ok := p.GetSalary(year); ok {
						if tp.RetentionPercent > 0 {
							retained := tp.GetRetainedSalary(year)
							contractYears = append(contractYears, fmt.Sprintf("%d: Total $%s (retain %.0f%% = $%s)",
								year, formatNumberShort(sal), tp.RetentionPercent, formatNumberShort(retained)))
						} else {
							contractYears = append(contractYears, fmt.Sprintf("%d: $%s", year, formatNumberShort(sal)))
						}
					}
				}
			}
			contractInfo := strings.Join(contractYears, ", ")
			if contractInfo == "" {
				contractInfo = "No contract info"
			}

			desc := fmt.Sprintf("• **%s** (%s)\n  %s | %s\n  %s",
				p.Name, side1Team, p.Position, p.MLBTeam, contractInfo)
			if tp.RetentionPercent > 0 {
				desc += fmt.Sprintf("\n  **Retention: %.0f%%**", tp.RetentionPercent)
			}
			side1Desc = append(side1Desc, desc)
		} else {
			// Simple view - just 2025
			salary2025 := "N/A"
			if sal, ok := p.GetSalary(2025); ok {
				if tp.RetentionPercent > 0 {
					retained := tp.GetRetainedSalary(2025)
					salary2025 = fmt.Sprintf("Total $%s (retain %.0f%% = $%s)",
						formatNumberShort(sal), tp.RetentionPercent, formatNumberShort(retained))
				} else {
					salary2025 = "$" + formatNumberShort(sal)
				}
			} else if p.IsFreeAgent(2025) {
				salary2025 = "FA"
			}
			side1Desc = append(side1Desc, fmt.Sprintf("• **%s** (%s)\n  %s | %s | 2025: %s",
				p.Name, side1Team, p.Position, p.MLBTeam, salary2025))
		}
	}

	// Add cash if present
	if analysis.Side1Cash > 0 {
		side1Desc = append(side1Desc, fmt.Sprintf("• **Cash Considerations**\n  $%s", formatNumber(analysis.Side1Cash)))
	}

	// Build side 2 description
	side2Team := ""
	var side2Desc []string
	for _, tp := range analysis.Side2Players {
		p := tp.Player
		side2Team = p.ULBTeam
		if side2Team == "" {
			side2Team = "Unowned"
		}

		if verbose {
			// Show full contract details
			contractYears := []string{}
			for year := 2025; year <= 2038; year++ {
				if val, exists := p.Contract[year]; exists && val != "" {
					if p.IsFreeAgent(year) {
						contractYears = append(contractYears, fmt.Sprintf("%d: FA", year))
						break
					} else if sal, ok := p.GetSalary(year); ok {
						if tp.RetentionPercent > 0 {
							retained := tp.GetRetainedSalary(year)
							contractYears = append(contractYears, fmt.Sprintf("%d: Total $%s (retain %.0f%% = $%s)",
								year, formatNumberShort(sal), tp.RetentionPercent, formatNumberShort(retained)))
						} else {
							contractYears = append(contractYears, fmt.Sprintf("%d: $%s", year, formatNumberShort(sal)))
						}
					}
				}
			}
			contractInfo := strings.Join(contractYears, ", ")
			if contractInfo == "" {
				contractInfo = "No contract info"
			}

			desc := fmt.Sprintf("• **%s** (%s)\n  %s | %s\n  %s",
				p.Name, side2Team, p.Position, p.MLBTeam, contractInfo)
			if tp.RetentionPercent > 0 {
				desc += fmt.Sprintf("\n  **Retention: %.0f%%**", tp.RetentionPercent)
			}
			side2Desc = append(side2Desc, desc)
		} else {
			// Simple view - just 2025
			salary2025 := "N/A"
			if sal, ok := p.GetSalary(2025); ok {
				if tp.RetentionPercent > 0 {
					retained := tp.GetRetainedSalary(2025)
					salary2025 = fmt.Sprintf("Total $%s (retain %.0f%% = $%s)",
						formatNumberShort(sal), tp.RetentionPercent, formatNumberShort(retained))
				} else {
					salary2025 = "$" + formatNumberShort(sal)
				}
			} else if p.IsFreeAgent(2025) {
				salary2025 = "FA"
			}
			side2Desc = append(side2Desc, fmt.Sprintf("• **%s** (%s)\n  %s | %s | 2025: %s",
				p.Name, side2Team, p.Position, p.MLBTeam, salary2025))
		}
	}

	// Add cash if present
	if analysis.Side2Cash > 0 {
		side2Desc = append(side2Desc, fmt.Sprintf("• **Cash Considerations**\n  $%s", formatNumber(analysis.Side2Cash)))
	}

	// Add fields
	embed.Fields = []*discordgo.MessageEmbedField{
		{
			Name:   fmt.Sprintf("%s (%d player%s)", side1Team, len(analysis.Side1Players), pluralize(len(analysis.Side1Players))),
			Value:  strings.Join(side1Desc, "\n"),
			Inline: true,
		},
		{
			Name:   "⇄",
			Value:  "for",
			Inline: true,
		},
		{
			Name:   fmt.Sprintf("%s (%d player%s)", side2Team, len(analysis.Side2Players), pluralize(len(analysis.Side2Players))),
			Value:  strings.Join(side2Desc, "\n"),
			Inline: true,
		},
	}

	// Add payroll impact
	if verbose && len(analysis.YearlyPayrollChanges) > 0 {
		// Show yearly breakdown for each team
		for team, yearlyChanges := range analysis.YearlyPayrollChanges {
			var yearlyDesc []string

			for year := 2025; year <= 2030; year++ {
				if change, exists := yearlyChanges[year]; exists {
					var changeStr string
					if change.NetChange > 0 {
						changeStr = "+$" + formatNumberShort(change.NetChange)
					} else if change.NetChange < 0 {
						changeStr = "-$" + formatNumberShort(abs(change.NetChange))
					} else {
						changeStr = "$0"
					}
					yearlyDesc = append(yearlyDesc, fmt.Sprintf("%d: %s", year, changeStr))
				}
			}

			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:   fmt.Sprintf("%s Payroll Impact by Year", team),
				Value:  strings.Join(yearlyDesc, " | "),
				Inline: false,
			})
		}
	} else if len(analysis.PayrollChanges) > 0 {
		// Simple view - just 2025
		var payrollDesc []string
		for _, change := range analysis.PayrollChanges {
			var changeStr string
			if change.NetChange > 0 {
				changeStr = "+$" + formatNumber(change.NetChange)
			} else if change.NetChange < 0 {
				changeStr = "-$" + formatNumber(abs(change.NetChange))
			} else {
				changeStr = "$0"
			}

			desc := fmt.Sprintf("**%s**\n", change.TeamName)
			desc += fmt.Sprintf("Before: $%s\n", formatNumber(change.PayrollBefore))
			desc += fmt.Sprintf("After: $%s\n", formatNumber(change.PayrollAfter))
			desc += fmt.Sprintf("Change: %s", changeStr)
			payrollDesc = append(payrollDesc, desc)
		}

		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Team Payroll Impact (2025)",
			Value:  strings.Join(payrollDesc, "\n\n"),
			Inline: false,
		})
	}

	// Add total salaries (showing traded amounts)
	total1 := 0
	total2 := 0
	for _, tp := range analysis.Side1Players {
		// For side 1, show what's being traded away (full salary minus retention)
		total1 += tp.GetTradedSalary(2025)
	}
	for _, tp := range analysis.Side2Players {
		// For side 2, show what's being traded away (full salary minus retention)
		total2 += tp.GetTradedSalary(2025)
	}

	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
		Name: "2025 Salary Totals",
		Value: fmt.Sprintf("%s: $%s\n%s: $%s\nDifference: $%s",
			side1Team,
			formatNumber(total1),
			side2Team,
			formatNumber(total2),
			formatNumber(abs(total1-total2))),
		Inline: false,
	})

	return embed
}

func pluralize(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
