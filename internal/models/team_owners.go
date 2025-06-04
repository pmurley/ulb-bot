package models

// TeamOwners maps team names to their Discord user IDs
// Each team can have multiple owners who are considered equal
var TeamOwners = map[string][]string{
	"San Diego Tiger Sharks":      {"jawwright88tigersharks"},
	"51st State Freedom Flotilla": {"paul3025"},
	"Oakland Expos":               {"spicycilantr0", "uebe_5"},
	"Rocky Mountain Outlaws":      {"yosepie", "dereksaich"},
	"Hoodsport Hairy Woodpeckers": {"cjacksoncowart", "emmajohnsoncowart"},
	"Havana Bananas":              {"notthe1.eth", "bmoney831"},
	"Hoboken Rat Pack":            {"rmiktus"},
	"New York Roid Rage":          {"nicebeardbro"},
	"Jacksonville Jackrabbits":    {"jowens0548"},
	"Buffalo Blue Jays":           {"stw126"},
	"Kansas City Monarchs":        {"slightlyjason"},
	"Chicago Grand Slammers":      {"skipmart7464"},
	"Tennessee Terps":             {"capcarp"},
	"Houston Colt 45s":            {"_gravez"},
	"Wichita Wranglers":           {"cooplion", "king_con67"},
	"Saskatoon Berries":           {"elloyd16"},
	"Cascadia Seduction Zone":     {"fed_00"},
	"Bay Area Wildcats":           {"uazwildcats"},
	"Seattle Weiners":             {"_jstout", "nlawson40", "4est5957"},
	"Los Angeles Deferrals":       {"dodgerdave2025"},
	"Varysburg Vandals":           {"staticjeff"},
	"Florida Marlins":             {"nftsamo"},
	"Capital City Bombers":        {"tjguy3409"},
	"Shaolin Generals":            {"melo0o"},
	"Daytona Suns":                {"strohsograc_48845"},
	"Austin Bytes":                {"cyclone852_19274"},
	"Angel Fire Wrath":            {"eephus2288"},
	"Sithcinnati Red Blades":      {"xspittoon"},
	"Bluegrass Bourbons":          {"strat0sfere_84444", "mike12_17740"},
	"Chicago Ultra Athletes":      {"tasm616"},
}

// IsTeamOwner checks if a Discord user ID is an owner of the specified team
func IsTeamOwner(teamName string, discordUserID string) bool {
	owners, exists := TeamOwners[teamName]
	if !exists {
		return false
	}

	for _, ownerID := range owners {
		if ownerID == discordUserID {
			return true
		}
	}

	return false
}

// GetTeamOwners returns the list of Discord user IDs for a team
func GetTeamOwners(teamName string) []string {
	if owners, exists := TeamOwners[teamName]; exists {
		return owners
	}
	return []string{}
}

// GetTeamsForOwner returns all teams owned by a Discord user
func GetTeamsForOwner(discordUserID string) []string {
	var teams []string

	for teamName, owners := range TeamOwners {
		for _, ownerID := range owners {
			if ownerID == discordUserID {
				teams = append(teams, teamName)
				break
			}
		}
	}

	return teams
}
