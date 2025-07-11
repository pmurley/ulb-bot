package bot

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/pmurley/ulb-bot/internal/cache"
	"github.com/pmurley/ulb-bot/internal/config"
	"github.com/pmurley/ulb-bot/internal/discord"
	"github.com/pmurley/ulb-bot/internal/sheets"
	"github.com/pmurley/ulb-bot/internal/spotrac"
	"github.com/pmurley/ulb-bot/pkg/logger"
)

type Bot struct {
	session       *discordgo.Session
	config        *config.Config
	logger        *logger.Logger
	dataCache     *cache.Cache
	sheetsClient  *sheets.Client
	spotracClient *spotrac.Client
	handlers      *discord.HandlerManager
	stopChan      chan struct{}
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

	log.Info("Creating Sheets client")
	sheetsClient, err := sheets.NewClient(cfg.GoogleSheetsID)
	if err != nil {
		return nil, fmt.Errorf("failed to create sheets client: %w", err)
	}

	log.Info("Creating Spotrac client")
	spotracClient := spotrac.NewClient()

	log.Info("Creating bot")
	b := &Bot{
		session:       session,
		config:        cfg,
		logger:        log,
		dataCache:     cache.New(cfg.CacheDuration),
		sheetsClient:  sheetsClient,
		spotracClient: spotracClient,
		stopChan:      make(chan struct{}),
	}

	b.handlers = discord.NewHandlerManager(b.session, cfg, log, b.dataCache, sheetsClient, spotracClient)

	return b, nil
}

func (b *Bot) Start() error {
	b.handlers.RegisterHandlers()

	if err := b.session.Open(); err != nil {
		return fmt.Errorf("failed to open Discord session: %w", err)
	}

	// Load initial data
	if err := b.loadData(); err != nil {
		b.logger.Error("Failed to load initial data from sheets:", err)
	}

	// Start background data loader
	go b.backgroundDataLoader()

	// Start waiver monitor
	b.startWaiverMonitor()

	// Start transaction monitor
	b.startTransactionMonitor()

	return nil
}

func (b *Bot) Stop() error {
	close(b.stopChan)
	return b.session.Close()
}

// backgroundDataLoader runs in the background and reloads data every 30 minutes
func (b *Bot) backgroundDataLoader() {
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			b.logger.Info("Starting background data reload")
			if err := b.loadData(); err != nil {
				b.logger.Error("Background data reload failed:", err)
			} else {
				b.logger.Info("Background data reload completed successfully")
			}
		case <-b.stopChan:
			b.logger.Info("Stopping background data loader")
			return
		}
	}
}

// loadData loads data from sheets, ensuring no concurrent loads
func (b *Bot) loadData() error {
	// Check if already loading
	if b.dataCache.IsLoading() {
		b.logger.Debug("Data load already in progress, skipping")
		return nil
	}

	// Mark as loading
	b.dataCache.SetLoading(true)
	defer b.dataCache.SetLoading(false)

	// Perform the actual load
	return b.sheetsClient.LoadInitialData(b.dataCache)
}
