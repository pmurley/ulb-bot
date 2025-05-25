package discord

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/pmurley/ulb-bot/internal/cache"
	"github.com/pmurley/ulb-bot/internal/config"
	"github.com/pmurley/ulb-bot/internal/models"
	"github.com/pmurley/ulb-bot/internal/sheets"
	"github.com/pmurley/ulb-bot/pkg/logger"
)

type HandlerManager struct {
	session      *discordgo.Session
	config       *config.Config
	logger       *logger.Logger
	cache        *cache.Cache
	sheetsClient *sheets.Client
	commands     map[string]CommandHandler
}

type CommandHandler func(s *discordgo.Session, m *discordgo.MessageCreate, args []string)

func NewHandlerManager(
	session *discordgo.Session,
	config *config.Config,
	logger *logger.Logger,
	cache *cache.Cache,
	sheetsClient *sheets.Client,
) *HandlerManager {
	hm := &HandlerManager{
		session:      session,
		config:       config,
		logger:       logger,
		cache:        cache,
		sheetsClient: sheetsClient,
		commands:     make(map[string]CommandHandler),
	}

	hm.registerCommands()

	return hm
}

func (hm *HandlerManager) RegisterHandlers() {
	hm.session.AddHandler(hm.messageCreate)
}

func (hm *HandlerManager) registerCommands() {
	hm.commands["help"] = hm.handleHelp
	hm.commands["reload"] = hm.handleReload
	hm.commands["player"] = hm.handlePlayer
	hm.commands["players"] = hm.handlePlayers
	hm.commands["trade"] = hm.handleTrade
	hm.commands["team"] = hm.handleTeam
}

func (hm *HandlerManager) messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	if !strings.HasPrefix(m.Content, hm.config.CommandPrefix) {
		return
	}

	content := strings.TrimPrefix(m.Content, hm.config.CommandPrefix)
	parts := strings.Fields(content)
	if len(parts) == 0 {
		return
	}

	command := strings.ToLower(parts[0])
	args := parts[1:]

	if handler, exists := hm.commands[command]; exists {
		handler(s, m, args)
	}
}

func (hm *HandlerManager) handleHelp(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	helpMessage := `**Ultra League Baseball Bot Commands:**
` + "```" + `
!help          - Show this help message
!reload        - Force reload data from Google Sheets
!player <name> - Look up player information
!players <name1>, <name2>, ... - Look up multiple players
!team <name>   - Show team roster and payroll
!trade <players> for <players> - Analyze a trade
  Examples:
    !trade Ohtani for Judge
    !trade Ohtani (retain 25%) for Judge
    !trade Judge, cash ($5M) for Soto
  Use -v for full contract details
` + "```"

	s.ChannelMessageSend(m.ChannelID, helpMessage)
}

func (hm *HandlerManager) handleReload(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	hm.cache.Flush()
	if err := hm.sheetsClient.LoadInitialData(hm.cache); err != nil {
		s.ChannelMessageSend(m.ChannelID, "Failed to reload data: "+err.Error())
		return
	}
	s.ChannelMessageSend(m.ChannelID, "Data reloaded successfully!")
}

// ensurePlayersLoaded checks if players are in cache and auto-reloads if needed
func (hm *HandlerManager) ensurePlayersLoaded() (models.PlayerList, error) {
	players, found := hm.cache.GetPlayers()
	if !found {
		// Auto-reload if cache is empty
		hm.logger.Info("Cache expired, auto-reloading player data...")
		if err := hm.sheetsClient.LoadInitialData(hm.cache); err != nil {
			return nil, err
		}
		// Try again after reload
		players, found = hm.cache.GetPlayers()
		if !found {
			return nil, fmt.Errorf("failed to load player data after reload")
		}
	}
	return players, nil
}
