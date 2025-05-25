package main

import (
	"fmt"
	"log"
	"sort"
	"strings"

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

	fmt.Printf("\nLoaded %d players\n", len(players))
	fmt.Println("===================")

	// Show some statistics
	teamCounts := make(map[string]int)
	positionCounts := make(map[string]int)
	var totalSalary2025 int
	var playersWithSalary2025 int
	var freeAgents2025 int

	for _, p := range players {
		// Count by ULB team
		if p.ULBTeam != "" {
			teamCounts[p.ULBTeam]++
		}

		// Count by position (handle multiple positions)
		positions := strings.Split(p.Position, ",")
		for _, pos := range positions {
			pos = strings.TrimSpace(pos)
			if pos != "" {
				positionCounts[pos]++
			}
		}

		// Check 2025 salary
		if salary, ok := p.GetSalary(2025); ok {
			totalSalary2025 += salary
			playersWithSalary2025++
		}

		// Check if free agent in 2025
		if p.IsFreeAgent(2025) {
			freeAgents2025++
		}
	}

	// Display team counts
	fmt.Println("\nPlayers by ULB Team:")
	fmt.Println("-------------------")
	teams := make([]string, 0, len(teamCounts))
	for team := range teamCounts {
		teams = append(teams, team)
	}
	sort.Strings(teams)
	for _, team := range teams {
		fmt.Printf("%-30s: %d players\n", team, teamCounts[team])
	}

	// Display position counts
	fmt.Println("\nPlayers by Position:")
	fmt.Println("-------------------")
	positions := make([]string, 0, len(positionCounts))
	for pos := range positionCounts {
		positions = append(positions, pos)
	}
	sort.Strings(positions)
	for _, pos := range positions {
		fmt.Printf("%-10s: %d\n", pos, positionCounts[pos])
	}

	// Display salary info
	fmt.Println("\n2025 Salary Information:")
	fmt.Println("------------------------")
	fmt.Printf("Players with 2025 salary: %d\n", playersWithSalary2025)
	fmt.Printf("Total 2025 salaries: $%s\n", formatNumber(totalSalary2025))
	if playersWithSalary2025 > 0 {
		fmt.Printf("Average 2025 salary: $%s\n", formatNumber(totalSalary2025/playersWithSalary2025))
	}
	fmt.Printf("Free agents in 2025: %d\n", freeAgents2025)

	// Show top 10 players by 2024 points
	fmt.Println("\nTop 10 Players by 2024 Points:")
	fmt.Println("-------------------------------")
	sort.Slice(players, func(i, j int) bool {
		return players[i].Points2024 > players[j].Points2024
	})
	for i := 0; i < 10 && i < len(players); i++ {
		p := players[i]
		fmt.Printf("%2d. %-25s %-25s %7.2f pts\n", 
			i+1, p.Name, p.ULBTeam, p.Points2024)
	}

	// Show top 10 salaries for 2025
	fmt.Println("\nTop 10 Salaries for 2025:")
	fmt.Println("-------------------------")
	type playerSalary struct {
		player models.Player
		salary int
	}
	var salaries []playerSalary
	for _, p := range players {
		if sal, ok := p.GetSalary(2025); ok {
			salaries = append(salaries, playerSalary{p, sal})
		}
	}
	sort.Slice(salaries, func(i, j int) bool {
		return salaries[i].salary > salaries[j].salary
	})
	for i := 0; i < 10 && i < len(salaries); i++ {
		ps := salaries[i]
		fmt.Printf("%2d. %-25s %-25s $%s\n", 
			i+1, ps.player.Name, ps.player.ULBTeam, formatNumber(ps.salary))
	}
}

// formatNumber adds commas to large numbers
func formatNumber(n int) string {
	str := fmt.Sprintf("%d", n)
	if len(str) <= 3 {
		return str
	}
	
	// Add commas from right to left
	result := ""
	for i, digit := range str {
		if i > 0 && (len(str)-i)%3 == 0 {
			result += ","
		}
		result += string(digit)
	}
	return result
}