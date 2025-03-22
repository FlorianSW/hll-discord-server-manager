package resources

import (
	"encoding/json"
	"errors"
	"os"
	"path"
)

type servers struct {
	directory string
}

func NewServers(d string) *servers {
	return &servers{directory: d}
}

func (m *servers) Find(matchId string) (*Server, error) {
	b, err := os.ReadFile(path.Join(m.directory, matchId))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	var match Server
	err = json.Unmarshal(b, &match)
	return &match, err
}

func (m *servers) List() (result []string, error error) {
	b, err := os.ReadDir(m.directory)
	if err != nil {
		return nil, err
	}
	for _, entry := range b {
		if entry.IsDir() {
			continue
		}
		result = append(result, entry.Name())
	}
	return result, err
}

func (m *servers) Save(match Server) error {
	d, err := json.Marshal(match)
	if err != nil {
		return err
	}
	return os.WriteFile(path.Join(m.directory, match.ServerId), d, 0644)
}

func (m *servers) Delete(serverId string) error {
	err := os.Remove(path.Join(m.directory, serverId))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
