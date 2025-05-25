package main

import (
	"fmt"
	"log"

	"github.com/joho/godotenv"
	"github.com/pmurley/ulb-bot/internal/config"
	"github.com/pmurley/ulb-bot/internal/models"
	"github.com/pmurley/ulb-bot/internal/sheets"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	client, err := sheets.NewClient(cfg.GoogleSheetsID)
	if err != nil {
		log.Fatal("Failed to create sheets client:", err)
	}

	fmt.Println("Loading Master Player Pool...")
	players, err := client.LoadMasterPlayerPool()
	if err != nil {
		log.Fatal("Failed to load players:", err)
	}

	// Convert to PlayerList for helper methods
	playerList := models.PlayerList(players)
	fmt.Printf("Loaded %d players\n\n", len(playerList))

	// Test 1: Search by name
	fmt.Println("=== Search Tests ===")
	fmt.Println("Searching for 'Ohtani'...")
	matches := playerList.SearchByName("Ohtani")
	for _, p := range matches {
		fmt.Printf("  Found: %s (%s) - %s\n", p.Name, p.Position, p.ULBTeam)
	}

	// Test 2: Find exact player
	fmt.Println("\nFinding exact match for 'Juan Soto'...")
	exactMatches := playerList.FindByExactName("Juan Soto")
	if len(exactMatches) > 0 {
		for _, soto := range exactMatches {
			fmt.Printf("  Found: %s, Age %d, Team: %s, 2024 Points: %.2f\n", 
				soto.Name, soto.Age, soto.ULBTeam, soto.Points2024)
			if salary, ok := soto.GetSalary(2025); ok {
				fmt.Printf("  2025 Salary: $%s\n", formatNumber(salary))
			}
		}
	} else {
		fmt.Println("  No exact match found")
	}

	// Test 3: Filter by team
	fmt.Println("\n=== Team Filter Test ===")
	fmt.Println("Bay Area Wildcats roster:")
	wildcats := playerList.FilterByTeam("Bay Area Wildcats")
	fmt.Printf("Total players: %d\n", len(wildcats))
	
	// Show top 5 by points
	wildcats.SortByPoints()
	fmt.Println("Top 5 performers:")
	for i := 0; i < 5 && i < len(wildcats); i++ {
		p := wildcats[i]
		fmt.Printf("  %d. %s (%s) - %.2f pts\n", i+1, p.Name, p.Position, p.Points2024)
	}

	// Test 4: Position filtering
	fmt.Println("\n=== Position Filter Test ===")
	fmt.Println("Top 10 Shortstops by 2024 points:")
	shortstops := playerList.FilterByPosition("SS")
	topSS := shortstops.GetTopPerformers(10)
	for i, p := range topSS {
		fmt.Printf("  %d. %s (%s) - %.2f pts\n", i+1, p.Name, p.ULBTeam, p.Points2024)
	}

	// Test 5: Free agents
	fmt.Println("\n=== Free Agents Test ===")
	freeAgents2025 := playerList.GetFreeAgents(2025)
	fmt.Printf("Total free agents in 2025: %d\n", len(freeAgents2025))
	fmt.Println("Top 5 by 2024 performance:")
	freeAgents2025.SortByPoints()
	for i := 0; i < 5 && i < len(freeAgents2025); i++ {
		p := freeAgents2025[i]
		fmt.Printf("  %d. %s (%s) - %.2f pts\n", i+1, p.Name, p.Position, p.Points2024)
	}

	// Test 6: Unowned players
	fmt.Println("\n=== Unowned Players Test ===")
	unowned := playerList.GetUnownedPlayers()
	fmt.Printf("Total unowned players: %d\n", len(unowned))
	topUnowned := unowned.GetTopPerformers(5)
	fmt.Println("Top 5 unowned by 2024 points:")
	for i, p := range topUnowned {
		fmt.Printf("  %d. %s (%s, %s) - %.2f pts\n", i+1, p.Name, p.Position, p.MLBTeam, p.Points2024)
	}

	// Test 7: Team payroll
	fmt.Println("\n=== Team Payroll Test ===")
	teams := []string{
		"Bay Area Wildcats",
		"Havana Bananas",
		"51st State Freedom Flotilla",
		"Chicago Grand Slammers",
		"New York Roid Rage",
	}
	for _, team := range teams {
		payroll := playerList.GetTeamPayroll(team, 2025)
		fmt.Printf("%-30s: $%s\n", team, formatNumber(payroll))
	}

	// Test 8: Multi-position eligible
	fmt.Println("\n=== Multi-Position Eligible Test ===")
	fmt.Println("Players eligible at both 2B and SS:")
	eligible := playerList.GetPositionEligible([]string{"2B", "SS"})
	eligible.SortByPoints()
	for i := 0; i < 10 && i < len(eligible); i++ {
		p := eligible[i]
		fmt.Printf("  %d. %s (%s) - %s - %.2f pts\n", 
			i+1, p.Name, p.ULBTeam, p.Position, p.Points2024)
	}

	// Test 9: MLB team filter
	fmt.Println("\n=== MLB Team Filter Test ===")
	fmt.Println("Yankees players in ULB:")
	yankees := playerList.FilterByMLBTeam("NYY")
	yankees.SortByPoints()
	for i := 0; i < 5 && i < len(yankees); i++ {
		p := yankees[i]
		fmt.Printf("  %d. %s (%s) - %.2f pts\n", i+1, p.Name, p.ULBTeam, p.Points2024)
	}

	// Test 10: League-wide stats
	fmt.Println("\n=== League Stats ===")
	stats := playerList.GetStats(2025)
	fmt.Printf("Total players: %d\n", stats.Count)
	fmt.Printf("Average 2024 points: %.2f\n", stats.AveragePoints)
	fmt.Printf("Total 2025 payroll: $%s\n", formatNumber(stats.TotalSalary))
	fmt.Printf("Average 2025 salary: $%s\n", formatNumber(stats.AverageSalary))
	fmt.Printf("Free agents in 2025: %d\n", stats.FreeAgentCount)
}

// formatNumber adds commas to large numbers
func formatNumber(n int) string {
	str := fmt.Sprintf("%d", n)
	if len(str) <= 3 {
		return str
	}
	
	result := ""
	for i, digit := range str {
		if i > 0 && (len(str)-i)%3 == 0 {
			result += ","
		}
		result += string(digit)
	}
	return result
}