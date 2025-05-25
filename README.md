# Ultra League Baseball Discord Bot

A Discord bot for the Ultra League Baseball fantasy league that loads data from Google Sheets and provides league information through Discord commands.

## Project Structure

```
ulb-bot/
├── cmd/ulb-bot/        # Application entry point
├── internal/           # Private application code
│   ├── bot/           # Bot initialization and lifecycle
│   ├── cache/         # Data caching layer
│   ├── config/        # Configuration management
│   ├── discord/       # Discord handlers and commands
│   ├── models/        # Data models (to be defined based on sheet data)
│   └── sheets/        # Google Sheets client
├── pkg/               # Public packages
│   └── logger/        # Logging utilities
├── configs/           # Configuration files
└── scripts/           # Utility scripts
```

## Setup

1. Clone the repository
2. Copy `.env.example` to `.env` and fill in your values:
   - `DISCORD_TOKEN`: Your Discord bot token
   - `GOOGLE_SHEETS_ID`: The ID of your public Google Sheet
3. Run `make deps` to download dependencies
4. Run `make build` to build the bot
5. Run `./ulb-bot` or `make run` to start the bot

## Commands

- `!help` - Show available commands
- `!reload` - Force reload data from Google Sheets

## Development

- `make run` - Run the bot
- `make test` - Run tests
- `make lint` - Run linter
- `make clean` - Clean build artifacts