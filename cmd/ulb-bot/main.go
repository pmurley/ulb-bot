package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/pmurley/ulb-bot/internal/bot"
	"github.com/pmurley/ulb-bot/internal/config"
	"github.com/pmurley/ulb-bot/pkg/logger"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load configuration:", err)
	}

	log := logger.New(cfg.LogLevel)

	b, err := bot.New(cfg, log)
	if err != nil {
		log.Fatal("Failed to create bot:", err)
	}

	if err := b.Start(); err != nil {
		log.Fatal("Failed to start bot:", err)
	}

	log.Info("Bot is running. Press CTRL+C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	log.Info("Shutting down...")
	if err := b.Stop(); err != nil {
		log.Error("Error during shutdown:", err)
	}
}
