package crcon

const (
	ActionMatchEnded = action("MATCH ENDED")
	ActionMatchStart = action("MATCH START")
)

type action string

type getRecentLogsRequest struct {
	End             int      `json:"end"`
	FilterActions   []action `json:"filter_action"`
	FilterPlayer    []string `json:"filter_player"`
	InclusiveFilter bool     `json:"inclusive_filter"`
}

type setMapRequest struct {
	MapId string `json:"map_name"`
}

type setTeamSwitchCooldownRequest struct {
	Minutes int  `json:"minutes"`
	Forward bool `json:"forward"`
}

type setAutoBalanceThresholdRequest struct {
	MaxDiff int  `json:"max_diff"`
	Forward bool `json:"forward"`
}

type setWelcomeMessage struct {
	Message string `json:"message"`
	Forward bool   `json:"forward"`
}

type setProfanities struct {
	Profanities []string `json:"profanities"`
}

type setAutoBroadcastMessages struct {
	Enabled   bool               `json:"enabled"`
	Randomize bool               `json:"randomize"`
	Messages  []broadcastMessage `json:"messages"`
}

type broadcastMessage struct {
	Message string `json:"message"`
	Time    int    `json:"time_sec"`
}

type messagePlayerRequest struct {
	Message  string `json:"message"`
	PlayerId string `json:"player_id"`
}
