package discord

import (
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/pmurley/ulb-bot/internal/models"
	"github.com/pmurley/ulb-bot/internal/storage"
)

const (
	dfaWaiversChannelID = "1080917219204153395"
	//dfaWaiversChannelID = "1376062221699776543"
	waiverDuration = 8 * 24 * time.Hour // 7 days
	//waiverDuration = time.Minute // 7 days
)

// handleDFA processes the !dfa command
func (hm *HandlerManager) handleDFA(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	// Check if command is in the correct channel
	if m.ChannelID != dfaWaiversChannelID {
		response := fmt.Sprintf("The !dfa command can only be used in the <#%s> channel.", dfaWaiversChannelID)
		if _, err := s.ChannelMessageSendReply(m.ChannelID, response, m.Reference()); err != nil {
			hm.logger.Error("Failed to send channel restriction message:", err)
		}
		return
	}

	// Parse the player name from the command
	if len(args) == 0 {
		if _, err := s.ChannelMessageSendReply(m.ChannelID, "Usage: !dfa <playerName>", m.Reference()); err != nil {
			hm.logger.Error("Failed to send usage message:", err)
		}
		return
	}

	playerName := strings.Join(args, " ")

	// Get players from cache (auto-reload if needed)
	players, err := hm.ensurePlayersLoaded()
	if err != nil {
		if _, err := s.ChannelMessageSendReply(m.ChannelID, "Failed to load player data: "+err.Error(), m.Reference()); err != nil {
			hm.logger.Error("Failed to send error message:", err)
		}
		return
	}

	// Search for the player using the same logic as player commands
	matches := players.SearchByName(playerName)

	// Handle no matches
	if len(matches) == 0 {
		response := fmt.Sprintf("No player found with name: %s", playerName)
		if _, err := s.ChannelMessageSendReply(m.ChannelID, response, m.Reference()); err != nil {
			hm.logger.Error("Failed to send no match message:", err)
		}
		return
	}

	// Handle multiple matches
	if len(matches) > 1 {
		var teamNames []string
		for _, p := range matches {
			teamNames = append(teamNames, p.ULBTeam)
		}
		response := fmt.Sprintf("Multiple players found with name %s. Found on teams: %s\nPlease be more specific.",
			playerName, strings.Join(teamNames, ", "))
		if _, err := s.ChannelMessageSendReply(m.ChannelID, response, m.Reference()); err != nil {
			hm.logger.Error("Failed to send multiple matches message:", err)
		}
		return
	}

	// We have exactly one match
	player := matches[0]

	// Get the user's teams
	userTeams := models.GetTeamsForOwner(m.Author.Username)

	// Check if user owns the player's team
	isOwner := false
	for _, team := range userTeams {
		if team == player.ULBTeam {
			isOwner = true
			break
		}
	}

	if !isOwner {
		response := fmt.Sprintf("Player %s does not belong to a team in the database.", player.Name)
		if player.ULBTeam != "" {
			response = fmt.Sprintf("You are not registered as an owner of the %s.", player.ULBTeam)
		}
		if _, err := s.ChannelMessageSendReply(m.ChannelID, response, m.Reference()); err != nil {
			hm.logger.Error("Failed to send ownership error message:", err)
		}
		return
	}

	// Create waiver storage instance
	waiverStorage, err := storage.NewWaiverStorage()
	if err != nil {
		hm.logger.Error("Failed to create waiver storage:", err)
		if _, err := s.ChannelMessageSendReply(m.ChannelID, "Error processing DFA. Please try again later.", m.Reference()); err != nil {
			hm.logger.Error("Failed to send storage creation error message:", err)
		}
		return
	}

	// Create waiver entry
	waiver := &models.Waiver{
		PlayerName: player.Name,
		TeamName:   player.ULBTeam,
		UserID:     m.Author.ID,
		StartTime:  time.Now(),
		EndTime:    time.Now().Add(waiverDuration),
		MessageID:  m.ID,
		ChannelID:  m.ChannelID,
		Processed:  false,
	}

	// Save to storage
	if err := waiverStorage.AddWaiver(waiver); err != nil {
		hm.logger.Error("Failed to save waiver:", err)
		if _, err := s.ChannelMessageSendReply(m.ChannelID, "Error processing DFA. Please try again later.", m.Reference()); err != nil {
			hm.logger.Error("Failed to send storage error message:", err)
		}
		return
	}

	// Send confirmation message
	response := fmt.Sprintf("%s has been designated for assignment and placed on waivers. I will notify you after 8 days when the waiver period has expired. Make sure you have dropped the player in Fantrax.", player.Name)
	if _, err := s.ChannelMessageSendReply(m.ChannelID, response, m.Reference()); err != nil {
		hm.logger.Error("Failed to send DFA confirmation:", err)
	}
}
