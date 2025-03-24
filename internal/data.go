package internal

import "github.com/floriansw/hll-discord-server-watcher/resources"

const (
	HLLGameId = "1098726659"
	HLLModId  = "0"
	HLLFileId = "1"
)

type Storage[T resources.Identifiable] interface {
	Find(id string) (*T, error)
	Save(entity T) error
	Delete(id string) error
	List() ([]string, error)
}
