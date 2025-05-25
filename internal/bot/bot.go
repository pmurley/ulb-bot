package bot

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/pmurley/ulb-bot/internal/cache"
	"github.com/pmurley/ulb-bot/internal/config"
	"github.com/pmurley/ulb-bot/internal/discord"
	"github.com/pmurley/ulb-bot/internal/sheets"
	"github.com/pmurley/ulb-bot/pkg/logger"
)

type Bot struct {
	session      *discordgo.Session
	config       *config.Config
	logger       *logger.Logger
	dataCache    *cache.Cache
	sheetsClient *sheets.Client
	handlers     *discord.HandlerManager
}

func New(cfg *config.Config, log *logger.Logger) (*Bot, error) {
	session, err := discordgo.New("Bot " + cfg.DiscordToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create Discord session: %w", err)
	}

	// Set intents - we need these for DMs and message content
	session.Identify.Intents = discordgo.IntentsGuildMessages | 
		discordgo.IntentsDirectMessages | 
		discordgo.IntentsDirectMessageReactions |
		discordgo.IntentsMessageContent

	sheetsClient, err := sheets.NewClient(cfg.GoogleSheetsID)
	if err != nil {
		return nil, fmt.Errorf("failed to create sheets client: %w", err)
	}

	b := &Bot{
		session:      session,
		config:       cfg,
		logger:       log,
		dataCache:    cache.New(cfg.CacheDuration),
		sheetsClient: sheetsClient,
	}

	b.handlers = discord.NewHandlerManager(b.session, cfg, log, b.dataCache, sheetsClient)

	return b, nil
}

func (b *Bot) Start() error {
	b.handlers.RegisterHandlers()

	if err := b.session.Open(); err != nil {
		return fmt.Errorf("failed to open Discord session: %w", err)
	}

	if err := b.sheetsClient.LoadInitialData(b.dataCache); err != nil {
		b.logger.Error("Failed to load initial data from sheets:", err)
	}

	return nil
}

func (b *Bot) Stop() error {
	return b.session.Close()
}