# Ultra League Baseball Discord Bot

## Overview
This is a Discord bot for the Ultra League Baseball fantasy league that loads player and league data from Google Sheets and provides various commands for league members to query information about players, teams, and trades.

## Key Features

### Data Management
- **Google Sheets Integration**: Loads data from public Google Sheets without requiring authentication
- **Automatic Caching**: Caches data for a configurable duration (default 5 minutes)
- **Auto-reload**: Automatically reloads data when cache expires, no manual reload needed
- **Multiple Sheet Support**: Handles multiple sheets including Master Player Pool, Standings, Accounting, Salary, and Dead Money

### Discord Commands

#### Player Information
- `!player <name>` - Look up detailed player information including contract details
  - Handles players with duplicate names by showing all matches
  - Shows position, teams, age, status, and full contract information
  
- `!players <name1>, <name2>, ...` - Look up multiple players at once
  - Returns compact player cards for efficient viewing
  - Comma-separated list of player names

#### Trade Analysis
- `!trade <players> for <players>` - Analyze trades between teams
  - Basic syntax: `!trade Juan Soto for Aaron Judge`
  - With salary retention: `!trade Ohtani (retain 25%) for Judge`
  - With cash considerations: `!trade Player, cash ($5M) for Player`
  - Add `-v` or `--verbose` flag for full multi-year contract details
  
  Features:
  - Calculates payroll impact for all teams involved
  - Handles salary retention percentages
  - Supports cash considerations that affect current year payroll
  - Shows total salaries being traded
  - In verbose mode, shows year-by-year payroll impacts

#### Utility Commands
- `!help` - Show all available commands
- `!reload` - Force reload data from Google Sheets (rarely needed due to auto-reload)

## Technical Architecture

### Project Structure
```
ulb-bot/
├── cmd/ulb-bot/          # Main application entry point
├── internal/             # Private application code
│   ├── bot/             # Bot lifecycle management
│   ├── cache/           # In-memory caching with go-cache
│   ├── config/          # Configuration management
│   ├── discord/         # Discord handlers and commands
│   ├── models/          # Data models (Player, TradedPlayer)
│   └── sheets/          # Google Sheets client
└── pkg/                 # Public packages
    └── logger/          # Custom logging utility
```

### Key Design Decisions

1. **Strong Typing Over Interfaces**: Uses concrete types rather than interfaces for simplicity and clarity, as we're unlikely to swap data sources or Discord libraries.

2. **CSV Export for Public Sheets**: Instead of using Google Sheets API (which requires authentication), the bot uses CSV export URLs for public sheets. This simplifies deployment and avoids API quotas.

3. **Smart Player Matching**: 
   - Exact name matches take precedence
   - Handles duplicate player names appropriately per command
   - Partial name search as fallback

4. **Comprehensive Trade Logic**:
   - Full salary tracking with retention
   - Cash considerations affect payroll
   - Multi-team trade support
   - Accurate payroll calculations including retained salaries

### Data Model

#### Player
- Basic info: Name, ULB Team, MLB Team, Position, Age
- Contract data: Year-by-year salaries through 2038
- Status information: 40-man roster, options, etc.
- Helper methods for salary calculations and free agency checks

#### TradedPlayer
- Extends Player with retention percentage
- Calculates retained vs traded salary amounts

### Configuration
Environment variables (via `.env` file):
- `DISCORD_TOKEN` - Bot authentication token
- `GOOGLE_SHEETS_ID` - ID of the Google Sheet
- `CACHE_DURATION_MINUTES` - How long to cache data (default: 5)
- `COMMAND_PREFIX` - Bot command prefix (default: !)
- `LOG_LEVEL` - Logging verbosity (default: info)

## Notable Implementation Details

1. **Cache Expiry Handling**: Commands automatically reload data when cache expires, providing seamless user experience.

2. **Salary Retention Display**: Shows full salary with retention clearly - e.g., "2025: Total $40M (retain 50% = $20M)"

3. **Discord Embed Limits**: Handles Discord's 10-embed limit for bulk player displays.

4. **Number Formatting**: Custom number formatter adds commas to large numbers and supports K/M suffixes.

5. **Sheet Discovery**: Since we can't enumerate sheets without API access, sheet GIDs are hardcoded after manual discovery.

## Future Considerations

- Additional sheet types (team rosters) can be added as more GIDs are discovered
- Could add more sophisticated player search (by position, team, etc.)
- Potential for scheduled announcements or automatic updates
- Could expand trade analysis to include player performance metrics