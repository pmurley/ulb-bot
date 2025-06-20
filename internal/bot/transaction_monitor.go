package bot

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/pmurley/go-fantrax/models"
	"github.com/pmurley/ulb-bot/internal/fantrax"
	"github.com/pmurley/ulb-bot/internal/storage"
)

const (
	transactionCheckInterval = 2 * time.Minute
	waiverChannelName        = "dfa-waivers"
	tradeChannelName         = "trades"
	promotionsChannelName    = "40-man-promotions"
	signingsChannelName      = "signings"
)

// startTransactionMonitor starts the background transaction monitoring process
func (b *Bot) startTransactionMonitor() {
	go b.transactionMonitorLoop()
}

// transactionMonitorLoop runs in the background and checks for new transactions
func (b *Bot) transactionMonitorLoop() {
	b.logger.Info("Starting transaction monitor")

	// Initial check on startup
	b.checkNewTransactions()

	ticker := time.NewTicker(transactionCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			b.checkNewTransactions()
		case <-b.stopChan:
			b.logger.Info("Stopping transaction monitor")
			return
		}
	}
}

// checkNewTransactions fetches transactions from Fantrax and posts new ones to Discord
func (b *Bot) checkNewTransactions() {
	// Create transaction storage instance
	transactionStorage, err := storage.NewTransactionStorage()
	if err != nil {
		b.logger.Error("Failed to create transaction storage:", err)
		return
	}

	// Check if this is the first run (empty CSV)
	existingTransactions, err := transactionStorage.GetAllTransactions()
	if err != nil {
		b.logger.Error("Failed to get existing transactions:", err)
		return
	}

	isFirstRun := len(existingTransactions) == 0
	if isFirstRun {
		b.logger.Info("First run detected - initializing transaction storage without Discord notifications")
		b.initializeTransactionStorage(transactionStorage)
		return
	}

	// Get existing transaction IDs and trade group IDs for quick lookup
	existingTxIDs, err := transactionStorage.GetTransactionIDs()
	if err != nil {
		b.logger.Error("Failed to get existing transaction IDs:", err)
		return
	}

	existingTradeGroupIDs, err := transactionStorage.GetTradeGroupIDs()
	if err != nil {
		b.logger.Error("Failed to get existing trade group IDs:", err)
		return
	}

	// Create Fantrax client
	fantraxClient, err := fantrax.NewFantraxClient(os.Getenv("FANTRAX_LEAGUE_ID"), false)
	if err != nil {
		b.logger.Error("Failed to create Fantrax client:", err)
		return
	}

	// Fetch all transactions from Fantrax
	allTransactions, err := fantraxClient.GetTransactionsFromFantrax()
	if err != nil {
		b.logger.Error("Failed to fetch transactions from Fantrax:", err)
		return
	}

	// Filter for new transactions
	var newTransactions []models.Transaction
	var newTradeGroups = make(map[string][]models.Transaction)

	for _, tx := range allTransactions {
		// For non-trade transactions, check if we've seen this transaction ID
		if tx.Type != "TRADE" {
			if !existingTxIDs[tx.ID] {
				newTransactions = append(newTransactions, tx)
			}
		} else {
			// For trade transactions, group by TradeGroupID and check if we've seen the group
			if tx.TradeGroupID != "" && !existingTradeGroupIDs[tx.TradeGroupID] {
				// Add to newTradeGroups - this automatically handles grouping multiple transactions
				// with the same TradeGroupID together
				newTradeGroups[tx.TradeGroupID] = append(newTradeGroups[tx.TradeGroupID], tx)
			}
		}
	}

	// Add individual new transactions (non-trades)
	if len(newTransactions) > 0 {
		if err := transactionStorage.AddTransactions(newTransactions); err != nil {
			b.logger.Error("Failed to store new transactions:", err)
			return
		}

		// Post each new transaction to Discord
		for _, tx := range newTransactions {
			b.postTransactionToDiscord(tx)
		}
	}

	// Add new trade groups
	for tradeGroupID, tradeTransactions := range newTradeGroups {
		if err := transactionStorage.AddTransactions(tradeTransactions); err != nil {
			b.logger.Error("Failed to store new trade transactions for group", tradeGroupID, ":", err)
			continue
		}

		// Post the trade group to Discord
		b.postTradeToDiscord(tradeTransactions)
	}

	if len(newTransactions) > 0 || len(newTradeGroups) > 0 {
		b.logger.Info("Processed ", len(newTransactions), " new transactions and ", len(newTradeGroups), " new trades")
	}
}

// postTransactionToDiscord posts a single transaction to the appropriate channel
func (b *Bot) postTransactionToDiscord(tx models.Transaction) {
	channelName := b.getChannelForTransaction(tx)
	channelID := b.findChannelByName(channelName)
	if channelID == "" {
		b.logger.Error("Could not find channel:", channelName)
		return
	}

	embed := b.createTransactionEmbed(tx)

	_, err := b.session.ChannelMessageSendEmbed(channelID, embed)
	if err != nil {
		b.logger.Error("Failed to send transaction message to Discord:", err)
	}
}

// postTradeToDiscord posts a trade group to the bot-testing channel
func (b *Bot) postTradeToDiscord(tradeTransactions []models.Transaction) {
	channelID := b.findChannelByName(tradeChannelName)
	if channelID == "" {
		b.logger.Error("Could not find bot-testing channel")
		return
	}

	embed := b.createTradeEmbed(tradeTransactions)

	_, err := b.session.ChannelMessageSendEmbed(channelID, embed)
	if err != nil {
		b.logger.Error("Failed to send trade message to Discord:", err)
	}
}

// createTransactionEmbed creates a Discord embed for a single transaction
func (b *Bot) createTransactionEmbed(tx models.Transaction) *discordgo.MessageEmbed {
	var color int
	var title string
	var description strings.Builder

	// Check if executed by commissioner
	isCommissioner := strings.ToLower(tx.ExecutedBy) == "commissioner"

	switch tx.Type {
	case "CLAIM":
		switch tx.ClaimType {
		case "FA": // Free Agent
			if isCommissioner {
				color = 0x9932cc // Purple for 40-man promotion
				title = "⬆️ 40-Man Promotion"
				description.WriteString(fmt.Sprintf("**%s** promoted **%s** (%s - %s) to 40-man roster",
					tx.TeamName, tx.PlayerName, tx.PlayerPosition, tx.PlayerTeam))
			} else {
				color = 0x00ff00 // Green for free agent signing
				title = "✍️ Free Agent Signing"
				description.WriteString(fmt.Sprintf("**%s** signed **%s** (%s - %s)",
					tx.TeamName, tx.PlayerName, tx.PlayerPosition, tx.PlayerTeam))
				if tx.BidAmount != "" {
					description.WriteString(fmt.Sprintf("\n💰 Bid Amount: $%s", tx.BidAmount))
				}
			}
		case "WW": // Waiver Wire
			if isCommissioner {
				color = 0x9932cc // Purple for 40-man promotion
				title = "⬆️ 40-Man Promotion"
				description.WriteString(fmt.Sprintf("**%s** promoted **%s** (%s - %s) to 40-man roster",
					tx.TeamName, tx.PlayerName, tx.PlayerPosition, tx.PlayerTeam))
			} else {
				color = 0x00ff00 // Green for waiver claim
				title = "🔄 Waiver Claim"
				description.WriteString(fmt.Sprintf("**%s** claimed **%s** (%s - %s)",
					tx.TeamName, tx.PlayerName, tx.PlayerPosition, tx.PlayerTeam))
			}
		default:
			// For backwards compatibility
			color = 0x00ff00 // Green
			title = "🔄 Waiver Claim (Transaction Type: " + tx.ClaimType + ")"
			description.WriteString(fmt.Sprintf("**%s** claimed **%s** (%s - %s)",
				tx.TeamName, tx.PlayerName, tx.PlayerPosition, tx.PlayerTeam))
		}

	case "DROP":
		color = 0xff0000 // Red
		title = "❌ Player Drop"
		description.WriteString(fmt.Sprintf("**%s** dropped **%s** (%s - %s)",
			tx.TeamName, tx.PlayerName, tx.PlayerPosition, tx.PlayerTeam))
	default:
		color = 0x0099ff // Blue
		title = fmt.Sprintf("📋 %s Transaction", tx.Type)
		description.WriteString(fmt.Sprintf("**%s**: %s (%s - %s)",
			tx.TeamName, tx.PlayerName, tx.PlayerPosition, tx.PlayerTeam))
	}

	embed := &discordgo.MessageEmbed{
		Title:       title,
		Description: description.String(),
		Color:       color,
		Timestamp:   tx.ProcessedDate.Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Period %d", tx.Period),
		},
	}

	if tx.ExecutedBy != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Executed By",
			Value:  tx.ExecutedBy,
			Inline: true,
		})
	}

	return embed
}

// createTradeEmbed creates a Discord embed for a trade (multiple transactions grouped)
func (b *Bot) createTradeEmbed(tradeTransactions []models.Transaction) *discordgo.MessageEmbed {
	if len(tradeTransactions) == 0 {
		return nil
	}

	// Group players by team
	teamPlayers := make(map[string][]models.Transaction)
	for _, tx := range tradeTransactions {
		teamPlayers[tx.FromTeamName] = append(teamPlayers[tx.FromTeamName], tx)
	}

	var description strings.Builder
	description.WriteString("\n")

	// Build trade description
	teams := make([]string, 0, len(teamPlayers))
	for team := range teamPlayers {
		teams = append(teams, team)
	}

	for i, team := range teams {
		if i > 0 {
			description.WriteString("\n**↓ ↑**\n\n")
		}

		description.WriteString(fmt.Sprintf("**%s** traded:\n", team))
		for _, tx := range teamPlayers[team] {
			description.WriteString(fmt.Sprintf("• %s (%s - %s)\n",
				tx.PlayerName, tx.PlayerPosition, tx.PlayerTeam))
		}
	}

	embed := &discordgo.MessageEmbed{
		Title:       "🔄 Trade Executed",
		Description: description.String(),
		Color:       0xffa500, // Orange
		Timestamp:   tradeTransactions[0].ProcessedDate.Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Period %d • %d players involved",
				tradeTransactions[0].Period, len(tradeTransactions)),
		},
	}

	if tradeTransactions[0].ExecutedBy != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Executed By",
			Value:  tradeTransactions[0].ExecutedBy,
			Inline: true,
		})
	}

	return embed
}

// findChannelByName finds a channel ID by name
func (b *Bot) findChannelByName(channelName string) string {
	for _, guild := range b.session.State.Guilds {
		channels, err := b.session.GuildChannels(guild.ID)
		if err != nil {
			continue
		}

		for _, channel := range channels {
			if channel.Name == channelName && channel.Type == discordgo.ChannelTypeGuildText {
				return channel.ID
			}
		}
	}
	return ""
}

// getChannelForTransaction determines the appropriate Discord channel for a transaction
func (b *Bot) getChannelForTransaction(tx models.Transaction) string {
	// Check if executed by commissioner (case-insensitive)
	isCommissioner := strings.ToLower(tx.ExecutedBy) == "commissioner"

	switch tx.Type {
	case "CLAIM":
		switch tx.ClaimType {
		case "FA": // Free Agent
			if isCommissioner {
				return promotionsChannelName // 40-man promotion
			}
			return signingsChannelName // Real free agent signing
		case "WW": // Waiver Wire
			if isCommissioner {
				return promotionsChannelName // 40-man promotion
			}
			return waiverChannelName // True waiver claim
		default:
			// For backwards compatibility with old records without ClaimType
			return waiverChannelName
		}
	case "DROP":
		// All drops go to the same channel regardless of claim type
		return waiverChannelName
	case "TRADE":
		return tradeChannelName
	default:
		// Default to waivers for unknown transaction types
		return waiverChannelName
	}
}

// initializeTransactionStorage populates the CSV with all historical transactions without posting to Discord
func (b *Bot) initializeTransactionStorage(transactionStorage *storage.TransactionStorage) {
	b.logger.Info("Initializing transaction storage with historical data...")

	// Create Fantrax client
	fantraxClient, err := fantrax.NewFantraxClient(os.Getenv("FANTRAX_LEAGUE_ID"), false)
	if err != nil {
		b.logger.Error("Failed to create Fantrax client during initialization:", err)
		return
	}

	// Fetch all historical transactions
	allTransactions, err := fantraxClient.GetTransactionsFromFantrax()
	if err != nil {
		b.logger.Error("Failed to fetch transactions during initialization:", err)
		return
	}

	if len(allTransactions) == 0 {
		b.logger.Info("No transactions found during initialization")
		return
	}

	// Store all transactions without posting to Discord
	if err := transactionStorage.AddTransactions(allTransactions); err != nil {
		b.logger.Error("Failed to store transactions during initialization:", err)
		return
	}

	// Log summary of what was initialized
	typeCount := make(map[string]int)
	tradeGroups := make(map[string]bool)
	for _, tx := range allTransactions {
		typeCount[tx.Type]++
		if tx.Type == "TRADE" && tx.TradeGroupID != "" {
			tradeGroups[tx.TradeGroupID] = true
		}
	}

	b.logger.Info("Transaction storage initialized successfully:")
	b.logger.Info("  Total transactions:", len(allTransactions))
	for txType, count := range typeCount {
		b.logger.Info(fmt.Sprintf("  %s: %d", txType, count))
	}
	b.logger.Info("  Trade groups:", len(tradeGroups))
	b.logger.Info("Future transaction monitoring will only post new transactions to Discord")
}
