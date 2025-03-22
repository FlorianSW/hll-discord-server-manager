package crcon

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
)

var (
	ErrFailed = errors.New("request failed")
)

type response[T any] struct {
	Result T    `json:"result"`
	Failed bool `json:"failed"`
}

type getRecentLogsResponse struct {
	Logs []log `json:"logs"`
}

type log struct {
	Action     action        `json:"action"`
	EventTime  timeWithoutTZ `json:"event_time"`
	SubContent string        `json:"sub_content"`
}

type timeWithoutTZ time.Time

func (j *timeWithoutTZ) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	t, err := time.Parse("2006-01-02T15:04:05", s)
	if err != nil {
		return err
	}
	*j = timeWithoutTZ(t)
	return nil
}

type getGameStateResponse struct {
	AlliedScore   int `json:"allied_score"`
	AxisScore     int `json:"axis_score"`
	AlliedPlayers int `json:"num_allied_players"`
	AxisPlayers   int `json:"num_axis_players"`
	Map           struct {
		GameMode    string `json:"game_mode"`
		Id          string `json:"id"`
		Environment string `json:"environment"`
		Map         struct {
			PrettyName string `json:"pretty_name"`
		} `json:"map"`
	} `json:"current_map"`
}

func (g getGameStateResponse) toGameState() GameState {
	return GameState{
		Map: Map{
			Id:          g.Map.Id,
			Name:        g.Map.Map.PrettyName,
			GameMode:    g.Map.GameMode,
			Environment: g.Map.Environment,
		},
		Score: Score{
			Allied: g.AlliedScore,
			Axis:   g.AxisScore,
		},
		PlayerCount: g.AlliedPlayers + g.AxisPlayers,
	}
}

type mapRotationResponse struct {
	Id          string `json:"id"`
	GameMode    string `json:"game_mode"`
	Environment string `json:"environment"`
	Name        string `json:"pretty_name"`
}

type getServerSettings struct {
	AutoBalanceEnabled   bool `json:"autobalance_enabled"`
	AutoBalanceThreshold int  `json:"autobalance_threshold"`
	IdleAutoKickTime     int  `json:"idle_autokick_time"`
	MaxPingAutoKick      int  `json:"max_ping_autokick"`
	QueueLength          int  `json:"queue_length"`
	TeamSwitchCooldown   int  `json:"team_switch_cooldown"`
	VipSlotsNumber       int  `json:"vip_slots_num"`
	VoteKickEnabled      bool `json:"votekick_enabled"`
}

func (g getServerSettings) toServerSettings() ServerSettings {
	return ServerSettings{
		AutoBalanceEnabled:   g.AutoBalanceEnabled,
		AutoBalanceThreshold: g.AutoBalanceThreshold,
		IdleAutoKickTime:     g.IdleAutoKickTime,
		MaxPingAutoKick:      g.MaxPingAutoKick,
		QueueLength:          g.QueueLength,
		TeamSwitchCooldown:   g.TeamSwitchCooldown,
		VipSlotsNumber:       g.VipSlotsNumber,
		VoteKickEnabled:      g.VoteKickEnabled,
	}
}

type getMapRotationResponse []mapRotationResponse

func (g getMapRotationResponse) toMapRotation() (res MapRotation) {
	for _, r := range g {
		res = append(res, Map{
			Id:          r.Id,
			Name:        r.Name,
			GameMode:    r.GameMode,
			Environment: r.Environment,
		})
	}
	return
}

type getOwnPermissions struct {
	IsSupervisor bool         `json:"is_supervisor"`
	Permissions  []permission `json:"permissions"`
	Username     string       `json:"user_name"`
}

type permission struct {
	Permission string `json:"permission"`
}

func (g getOwnPermissions) toOwnPermissions() OwnPermissions {
	return OwnPermissions{
		Username:  g.Username,
		Superuser: g.IsSupervisor,
		Permissions: sliceMap(g.Permissions, func(v permission) string {
			return v.Permission
		}),
	}
}

func sliceMap[K, V any](s []K, f func(v K) V) (r []V) {
	for _, k := range s {
		r = append(r, f(k))
	}
	return
}

func asResponse[T any](res *http.Response) (T, error) {
	var result response[T]
	err := json.NewDecoder(res.Body).Decode(&result)
	if err != nil {
		return result.Result, err
	}
	if result.Failed {
		return result.Result, ErrFailed
	}
	return result.Result, nil
}
