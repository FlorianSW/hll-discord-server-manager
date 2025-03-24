package commands

import (
	"context"
	"github.com/floriansw/go-crcon"
	"github.com/floriansw/go-tcadmin"
	"github.com/floriansw/hll-discord-server-watcher/internal"
	"github.com/floriansw/hll-discord-server-watcher/resources"
	"net/http"
	"net/http/cookiejar"
	"strings"
)

func Int(i int) *int {
	return &i
}

func String(s string) *string {
	return &s
}

func customId(components ...string) string {
	return strings.Join(components, "#")
}

func matchesId(cid string, expected string) bool {
	if cid == expected {
		return true
	}
	return strings.HasPrefix(cid, expected+"#")
}

func peekId(cid string) (peek string, rest string) {
	s := strings.Split(cid, "#")
	if len(s) == 1 {
		return s[0], ""
	} else if len(s) < 2 {
		return "", ""
	} else {
		peek = s[len(s)-1]
		s = s[:len(s)-1]
		return peek, customId(s...)
	}
}

type TCAdmin interface {
	ServerInfo(serviceId string) (*tcadmin.ServerInfo, error)
	SetServerInfo(serviceId string, name, pw string) error
	Restart(serviceId string) (string, error)
}

func tcadminClient(creds resources.TCAdminCredentials) TCAdmin {
	jar, _ := cookiejar.New(nil)
	return tcadmin.NewClient(http.Client{
		Jar: jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}, creds.BaseUrl, internal.HLLGameId, internal.HLLModId, internal.HLLFileId, tcadmin.Credentials{Username: creds.Username, Password: creds.Password})
}

type CRCon interface {
	SetTeamSwitchCooldown(ctx context.Context, minutes int) error
	SetAutoBalanceThreshold(ctx context.Context, maxDiff int) error
	SetWelcomeMessage(ctx context.Context, message string) error
	WelcomeMessage(ctx context.Context) (string, error)
	ServerSettings(ctx context.Context) (crcon.ServerSettings, error)
	PlayerIds(ctx context.Context) ([]string, error)
	OwnPermissions(ctx context.Context) (crcon.OwnPermissions, error)
}

func crconClient(creds resources.CRConCredentials) CRCon {
	return crcon.NewClient(http.Client{}, creds.BaseUrl, crcon.Credentials{ApiKey: creds.ApiKey})
}
