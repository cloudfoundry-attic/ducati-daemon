package store

import (
	"errors"
	"fmt"

	"github.com/cloudfoundry-incubator/ducati-daemon/models"
)

//go:generate counterfeiter -o ../fakes/store.go --fake-name Store . Store
type Store interface {
	Put(container models.Container) error
	Get(id string) (models.Container, error)
	All() ([]models.Container, error)
}

var NotFoundError = errors.New("record not found")

type store struct {
	containers map[string]models.Container
}

func New() Store {
	return &store{
		containers: map[string]models.Container{},
	}
}

func (s *store) Put(container models.Container) error {
	s.containers[container.ID] = container
	return nil
}

func (s *store) Get(id string) (models.Container, error) {
	container, ok := s.containers[id]
	if ok {
		return container, nil
	}

	return models.Container{}, fmt.Errorf("container not found: %s", id)
}

func (s *store) All() ([]models.Container, error) {
	containers := []models.Container{}

	for _, c := range s.containers {
		containers = append(containers, c)
	}

	return containers, nil
}
