package store

import (
	"errors"

	"github.com/cloudfoundry-incubator/ducati-daemon/models"
)

//go:generate counterfeiter -o ../fakes/store.go --fake-name Store . Store
type Store interface {
	Put(container models.Container) error
	Get(id string) (models.Container, error)
	All() ([]models.Container, error)
	Delete(id string) error
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
