package models

// TradedPlayer represents a player in a trade with potential salary retention
type TradedPlayer struct {
	Player           Player
	RetentionPercent float64 // 0-100, where 50 = 50% retention
}

// GetRetainedSalary calculates how much salary is retained for a given year
func (tp *TradedPlayer) GetRetainedSalary(year int) int {
	if tp.RetentionPercent == 0 {
		return 0
	}
	
	salary, ok := tp.Player.GetSalary(year)
	if !ok {
		return 0
	}
	
	return int(float64(salary) * tp.RetentionPercent / 100.0)
}

// GetTradedSalary calculates how much salary goes to the receiving team
func (tp *TradedPlayer) GetTradedSalary(year int) int {
	salary, ok := tp.Player.GetSalary(year)
	if !ok {
		return 0
	}
	
	retained := tp.GetRetainedSalary(year)
	return salary - retained
}