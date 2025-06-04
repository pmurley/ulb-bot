package bot

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/pmurley/ulb-bot/internal/models"
	"github.com/pmurley/ulb-bot/internal/storage"
)

const waiverCheckInterval = 2 * time.Minute

// startWaiverMonitor starts the background waiver monitoring process
func (b *Bot) startWaiverMonitor() {
	go b.waiverMonitorLoop()
}

// waiverMonitorLoop runs in the background and checks for expired waivers
func (b *Bot) waiverMonitorLoop() {
	b.logger.Info("Starting waiver monitor")

	// Initial check on startup
	b.checkExpiredWaivers()

	ticker := time.NewTicker(waiverCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			b.checkExpiredWaivers()
		case <-b.stopChan:
			b.logger.Info("Stopping waiver monitor")
			return
		}
	}
}

// checkExpiredWaivers checks for expired waivers and sends notifications
func (b *Bot) checkExpiredWaivers() {
	b.logger.Debug("Checking for expired waivers")

	// Create waiver storage instance
	waiverStorage, err := storage.NewWaiverStorage()
	if err != nil {
		b.logger.Error("Failed to create waiver storage:", err)
		return
	}

	// Get all active waivers
	activeWaivers, err := waiverStorage.GetActiveWaivers()
	if err != nil {
		b.logger.Error("Failed to get active waivers:", err)
		return
	}

	// Check each waiver
	for _, waiver := range activeWaivers {
		if waiver.IsExpired() {
			b.processExpiredWaiver(waiver, waiverStorage)
		}
	}
}

// processExpiredWaiver handles an expired waiver
func (b *Bot) processExpiredWaiver(waiver *models.Waiver, waiverStorage *storage.WaiverStorage) {
	b.logger.Info("Processing expired waiver for player", waiver.PlayerName)

	// Create the notification message
	message := fmt.Sprintf("<@%s> The waiver period has expired for %s -- Would you like to assign them to the minors?",
		waiver.UserID, waiver.PlayerName)

	// Create a reference to the original message
	reference := &discordgo.MessageReference{
		MessageID: waiver.MessageID,
		ChannelID: waiver.ChannelID,
	}

	// Send the notification as a reply
	if _, err := b.session.ChannelMessageSendReply(waiver.ChannelID, message, reference); err != nil {
		b.logger.Error("Failed to send waiver expiration notification:", err)
		return
	}

	// Mark the waiver as processed
	if err := waiverStorage.MarkWaiverProcessed(waiver.MessageID); err != nil {
		b.logger.Error("Failed to mark waiver as processed:", err)
	}
}
