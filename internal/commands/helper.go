package commands

import (
	"github.com/floriansw/go-tcadmin"
	"github.com/floriansw/hll-discord-server-watcher/internal"
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

func tcadminClient(url, username, password string) TCAdmin {
	jar, _ := cookiejar.New(nil)
	return tcadmin.NewClient(http.Client{
		Jar: jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}, url, internal.HLLGameId, internal.HLLModId, internal.HLLFileId, tcadmin.Credentials{Username: username, Password: password})
}
