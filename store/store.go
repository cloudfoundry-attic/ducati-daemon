package store

import (
	"database/sql"
	"errors"

	"github.com/cloudfoundry-incubator/ducati-daemon/models"
)

//go:generate counterfeiter -o ../fakes/store.go --fake-name Store . Store
type Store interface {
	Create(container models.Container) error
	Get(id string) (models.Container, error)
	All() ([]models.Container, error)
	Delete(id string) error
}

var NotFoundError = errors.New("record not found")
var RecordExistsError = errors.New("record already exists")

type store struct {
	containers map[string]models.Container
}

func New(dbConnectionPool *sql.DB) Store {
	return &store{
		containers: map[string]models.Container{},
	}
}

func (s *store) Create(container models.Container) error {
	if _, ok := s.containers[container.ID]; ok {
		return RecordExistsError
	}

	s.containers[container.ID] = container

	return nil
}

func (s *store) Get(id string) (models.Container, error) {
	container, ok := s.containers[id]
	if ok {
		return container, nil
	}

	return models.Container{}, NotFoundError
}

func (s *store) All() ([]models.Container, error) {
	containers := []models.Container{}

	for _, c := range s.containers {
		containers = append(containers, c)
	}

	return containers, nil
}

func (s *store) Delete(id string) error {
	if _, ok := s.containers[id]; !ok {
		return NotFoundError
	}

	delete(s.containers, id)
	return nil
}
