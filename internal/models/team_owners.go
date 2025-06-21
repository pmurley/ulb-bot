package models

// UsernameToUserID maps Discord usernames to their user IDs
var UsernameToUserID = map[string]string{
	"cyclone852_19274":       "1289404238228623421", // Austin Bytes
	"jawwright88tigersharks": "738581488714514522",  // San Diego Tiger Sharks
	"paul3025":               "713810246287753316",  // 51st State Freedom Flotilla
	"spicycilantr0":          "356597495989796876",  // Oakland Expos
	"uebe_5":                 "979916362354941962",  // Oakland Expos
	"yosepie":                "312382142736760833",  // Rocky Mountain Outlaws
	"dereksaich":             "812427207309000704",  // Rocky Mountain Outlaws
	"cjacksoncowart":         "403339350722740244",  // Hoodsport Hairy Woodpeckers
	"emmajohnsoncowart":      "645784540920545315",  // Hoodsport Hairy Woodpeckers
	"notthe1.eth":            "485851104857554965",  // Havana Bananas
	"bmoney831":              "259812397051543562",  // Havana Bananas
	"rmiktus":                "705965732273455104",  // Hoboken Rat Pack
	"nicebeardbro":           "233039963929706496",  // New York Roid Rage
	"jowens0548":             "464992444660973568",  // Jacksonville Jackrabbits
	"stw126":                 "830968193328873503",  // Buffalo Blue Jays
	"slightlyjason":          "235279075118284802",  // Kansas City Monarchs
	"skipmart7464":           "812829437745561650",  // Chicago Grand Slammers
	"capcarp":                "692417403421851759",  // Tennessee Terps
	"_gravez":                "177597893253791745",  // Houston Colt 45s
	"cooplion":               "247135469442301952",  // Wichita Wranglers
	"king_con67":             "1139702636594090032", // Wichita Wranglers
	"elloyd16":               "1116159788590563491", // Saskatoon Berries
	"fed_00":                 "497183764590362636",  // Cascadia Seduction Zone
	"uazwildcats":            "420083030141698051",  // Bay Area Wildcats
	"_jstout":                "380962383444574219",  // Seattle Weiners
	"nlawson40":              "426537435162345472",  // Seattle Weiners
	"4est5957":               "761087733413576744",  // Seattle Weiners
	"dodgerdave2025":         "690383495041646612",  // Los Angeles Deferrals
	"staticjeff":             "1025517794265137202", // Varysburg Vandals
	"nftsamo":                "883706505759711304",  // Florida Marlins
	"tjguy3409":              "419356094008131585",  // Capital City Bombers
	"melo0o":                 "431513303228088333",  // Shaolin Generals
	"strohsograc_48845":      "1295555375436796034", // Daytona Suns
	"eephus2288":             "532790244400168970",  // Angel Fire Wrath
	"xspittoon":              "1020526684476285021", // Sithcinnati Red Blades
	"strat0sfere_84444":      "1349172365292212305", // Bluegrass Bourbons
	"mike12_17740":           "1295440247798366209", // Bluegrass Bourbons
	"tasm616":                "283415040411959296",  // Chicago Ultra Athletes
	"bestkoreaslowmoebius":   "214135068870836225",
}

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
	"Cascadia Seduction Zone":     {"fed_00", "bestkoreaslowmoebius"},
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

// GetUserIDFromUsername converts a Discord username to user ID using the mapping
func GetUserIDFromUsername(username string) string {
	if userID, exists := UsernameToUserID[username]; exists {
		return userID
	}
	// Return the username as fallback if no mapping exists
	return username
}
