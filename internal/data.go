package internal

import "github.com/floriansw/hll-discord-server-watcher/resources"

const (
	HLLGameId = "1098726659"
	HLLModId  = "0"
	HLLFileId = "1"
)

type Servers interface {
	Find(serverId string) (*resources.Server, error)
	Save(server resources.Server) error
	Delete(serverId string) error
	List() ([]string, error)
}
