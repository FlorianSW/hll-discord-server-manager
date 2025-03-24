package resources

type Template struct {
	TemplateId           string             `json:"id"`
	Name                 string             `json:"name"`
	TeamSwitchCooldown   int                `json:"team_switch_cooldown"`
	AutoBalanceThreshold int                `json:"auto_balance_threshold"`
	ServerNameTemplate   string             `json:"server_name_template"`
	WelcomeMessage       string             `json:"welcome_message"`
	BroadcastMessage     []BroadcastMessage `json:"broadcast_message"`
	ProfanityFilter      []string           `json:"profanity_filter"`
}

func (t Template) Id() string {
	return t.TemplateId
}

type BroadcastMessage struct {
	Time    int    `json:"time"`
	Message string `json:"message"`
}
