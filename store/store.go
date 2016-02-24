package store

import (
	"errors"
	"fmt"

	"github.com/cloudfoundry-incubator/ducati-daemon/models"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

const schema = `
CREATE TABLE IF NOT EXISTS container (
   id text,
   ip text,
   mac text,
   host_ip text
);
`

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
	conn       *sqlx.DB
}

func New(dbConnectionPool *sqlx.DB) (Store, error) {
	err := setupTables(dbConnectionPool)
	if err != nil {
		return nil, fmt.Errorf("unable to setup tables: %s", err)
	}
	return &store{
		containers: map[string]models.Container{},
		conn:       dbConnectionPool,
	}, nil
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

func setupTables(dbConnectionPool *sqlx.DB) error {
	_, err := dbConnectionPool.Exec(schema)
	return err
}
