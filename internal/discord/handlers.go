package discord

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/pmurley/ulb-bot/internal/cache"
	"github.com/pmurley/ulb-bot/internal/config"
	"github.com/pmurley/ulb-bot/internal/models"
	"github.com/pmurley/ulb-bot/internal/sheets"
	"github.com/pmurley/ulb-bot/internal/spotrac"
	"github.com/pmurley/ulb-bot/pkg/logger"
)

type HandlerManager struct {
	session       *discordgo.Session
	config        *config.Config
	logger        *logger.Logger
	cache         *cache.Cache
	sheetsClient  *sheets.Client
	spotracClient *spotrac.Client
	commands      map[string]CommandHandler
}

type CommandHandler func(s *discordgo.Session, m *discordgo.MessageCreate, args []string)

func NewHandlerManager(
	session *discordgo.Session,
	config *config.Config,
	logger *logger.Logger,
	cache *cache.Cache,
	sheetsClient *sheets.Client,
	spotracClient *spotrac.Client,
) *HandlerManager {
	hm := &HandlerManager{
		session:       session,
		config:        config,
		logger:        logger,
		cache:         cache,
		sheetsClient:  sheetsClient,
		spotracClient: spotracClient,
		commands:      make(map[string]CommandHandler),
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
	hm.commands["dfa"] = hm.handleDFA
	hm.commands["spotrac"] = hm.handleSpotrac
	hm.commands["getfile"] = hm.handleGetFile
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
		hm.logger.Info("Processing command: ", command, " with args: ", args)
		handler(s, m, args)
	} else {
		hm.logger.Warn("Unknown command: ", command)
	}
}

func (hm *HandlerManager) handleHelp(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	helpMessage := `**Ultra League Baseball Bot Commands:**
` + "```" + `
!help          - Show this help message
!reload        - Force reload data from Google Sheets
!player <name> - Look up player information
!players <name1>, <name2>, ... - Look up multiple players
!spotrac <name> - Look up player contract information from Spotrac
!team <name>   - Show team roster and payroll (defaults to 40-man roster)
  Options:
    --status=<40-man|minors|all> - Filter by roster status (default: 40-man)
    --position=<pos>             - Filter by position (C, 1B, SS, OF, SP, MI, CI, IF, UT, etc)
    --age=<range>                - Filter by age (e.g., 20-25, 25+, 30, 22-)
    --contracts                  - Show contract details for each player
  Example: !team Berries --status=all --position=SP --age=25+
!trade <players> for <players> - Analyze a trade
!dfa <playerName> - Designate a player for assignment (only in #dfa-waivers channel)
  Examples:
    !trade Ohtani for Judge
    !trade Ohtani (retain 25%) for Judge
    !trade Judge, cash ($5M) for Soto
  Use -v for full contract details
` + "```"

	s.ChannelMessageSend(m.ChannelID, helpMessage)
}

func (hm *HandlerManager) handleReload(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	// Check if already loading
	if hm.cache.IsLoading() {
		s.ChannelMessageSend(m.ChannelID, "Data reload already in progress...")
		return
	}

	// Mark as loading
	hm.cache.SetLoading(true)
	defer hm.cache.SetLoading(false)

	if err := hm.sheetsClient.LoadInitialData(hm.cache); err != nil {
		s.ChannelMessageSend(m.ChannelID, "Failed to reload data: "+err.Error())
		return
	}
	s.ChannelMessageSend(m.ChannelID, "Data reloaded successfully!")
}

// ensurePlayersLoaded returns cached player data without blocking
func (hm *HandlerManager) ensurePlayersLoaded() (models.PlayerList, error) {
	players, found := hm.cache.GetPlayers()
	if !found {
		return nil, fmt.Errorf("player data not available yet, please try again in a moment")
	}
	return players, nil
}

func (hm *HandlerManager) handleGetFile(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	if len(args) < 1 {
		s.ChannelMessageSend(m.ChannelID, "Usage: !getfile <filepath>")
		return
	}

	filePath := strings.Join(args, " ")

	// Clean the path to prevent directory traversal
	cleanPath := filepath.Clean(filePath)

	// Check if file exists
	fileInfo, err := os.Stat(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			s.ChannelMessageSend(m.ChannelID, "File not found: "+cleanPath)
		} else {
			s.ChannelMessageSend(m.ChannelID, "Error accessing file: "+err.Error())
		}
		return
	}

	// Check if it's a directory
	if fileInfo.IsDir() {
		s.ChannelMessageSend(m.ChannelID, "Cannot send a directory")
		return
	}

	// Check file size (Discord has a limit of 8MB for free servers, 50MB for boosted)
	const maxFileSize = 8 * 1024 * 1024 // 8MB
	if fileInfo.Size() > maxFileSize {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("File too large (%d bytes). Maximum size is %d bytes", fileInfo.Size(), maxFileSize))
		return
	}

	// Open the file
	file, err := os.Open(cleanPath)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "Failed to open file: "+err.Error())
		return
	}
	defer file.Close()

	// Send the file
	_, err = s.ChannelFileSend(m.ChannelID, filepath.Base(cleanPath), file)
	if err != nil {
		hm.logger.Error("Failed to send file: ", err)
		s.ChannelMessageSend(m.ChannelID, "Failed to send file: "+err.Error())
		return
	}

	hm.logger.Info("File sent successfully: ", cleanPath)
}
