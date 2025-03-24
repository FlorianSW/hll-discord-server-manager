package resources

import (
	"encoding/json"
	"errors"
	"os"
	"path"
)

type fileBackedStore[T Identifiable] struct {
	directory string
}

type Identifiable interface {
	Id() string
}

func (m *fileBackedStore[T]) Find(matchId string) (res *T, err error) {
	b, err := os.ReadFile(path.Join(m.directory, matchId))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	err = json.Unmarshal(b, &res)
	return res, err
}

func (m *fileBackedStore[T]) List() (result []string, error error) {
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

func (m *fileBackedStore[T]) Save(entity T) error {
	d, err := json.Marshal(entity)
	if err != nil {
		return err
	}
	return os.WriteFile(path.Join(m.directory, entity.Id()), d, 0644)
}

func (m *fileBackedStore[T]) Delete(id string) error {
	err := os.Remove(path.Join(m.directory, id))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
