package crcon

import (
	"errors"
	"slices"
	"time"
)

var (
	ErrForbidden = errors.New("API client is not allowed to execute the requested action")
)

const (
	GameModeSkirmish  = GameMode("skirmish")
	GameModeWarfare   = GameMode("warfare")
	GameModeOffensive = GameMode("offensive")
)

type GameMode string

type Credentials struct {
	ApiKey string
}

type Match struct {
	Start *time.Time
	End   *time.Time
	Map   string
	Score Score
}

type Score struct {
	Allied int
	Axis   int
}

type Map struct {
	Id          string
	Name        string
	GameMode    string
	Environment string
}

type GameState struct {
	Map         Map
	Score       Score
	PlayerCount int
}

type MapRotation = []Map

type ServerSettings struct {
	AutoBalanceEnabled   bool
	AutoBalanceThreshold int
	IdleAutoKickTime     int
	MaxPingAutoKick      int
	QueueLength          int
	TeamSwitchCooldown   int
	VipSlotsNumber       int
	VoteKickEnabled      bool
}

type Permissions []string

func (p Permissions) ContainsOnly(o []string) bool {
	this := slices.Clone(p)
	slices.Sort(this)
	that := slices.Clone(o)
	slices.Sort(that)
	return slices.Equal(this, that)
}

type OwnPermissions struct {
	Username    string
	Superuser   bool
	Permissions Permissions
}
