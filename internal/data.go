package internal

import "github.com/floriansw/hll-discord-server-watcher/resources"

type Servers interface {
	Find(serverId string) (*resources.Server, error)
	Save(server resources.Server) error
	Delete(serverId string) error
	List() ([]string, error)
}
