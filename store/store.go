package store

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/cloudfoundry-incubator/ducati-daemon/models"
	"github.com/lib/pq"
)

const schema = `
CREATE TABLE IF NOT EXISTS container (
   id text PRIMARY KEY,
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

//go:generate counterfeiter -o ../fakes/db.go --fake-name Db . db
type db interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	NamedExec(query string, arg interface{}) (sql.Result, error)
	Get(dest interface{}, query string, args ...interface{}) error
	Select(dest interface{}, query string, args ...interface{}) error
}

var RecordNotFoundError = errors.New("record not found")
var RecordExistsError = errors.New("record already exists")

type store struct {
	conn db
}

func New(dbConnectionPool db) (Store, error) {
	err := setupTables(dbConnectionPool)
	if err != nil {
		return nil, fmt.Errorf("setting up tables: %s", err)
	}

	return &store{
		conn: dbConnectionPool,
	}, nil
}

func (s *store) Create(container models.Container) error {
	_, err := s.conn.NamedExec("INSERT INTO container (id, ip, mac, host_ip) VALUES (:id, :ip, :mac, :host_ip)", &container)
	if err != nil {
		pqErr, ok := err.(*pq.Error)
		if !ok {
			return fmt.Errorf("insert: %s", err)
		}
		if pqErr.Code.Name() == "unique_violation" {
			return RecordExistsError
		}
		return fmt.Errorf("insert: %s", pqErr.Code.Name())
	}

	return nil
}

func (s *store) Get(id string) (models.Container, error) {
	var container models.Container
	err := s.conn.Get(&container, "SELECT * FROM container WHERE id=$1", id)
	if err != nil {
		if err == sql.ErrNoRows {
			return models.Container{}, RecordNotFoundError
		}
		return container, fmt.Errorf("getting record: %s", err)
	}

	return container, nil
}

func (s *store) All() ([]models.Container, error) {
	containers := []models.Container{}
	err := s.conn.Select(&containers, "SELECT * FROM container")
	if err != nil {
		return nil, fmt.Errorf("listing all: %s", err)
	}

	return containers, nil
}

func (s *store) Delete(id string) error {
	execResult, err := s.conn.Exec("DELETE FROM container WHERE id=$1", id)
	if err != nil {
		return fmt.Errorf("deleting: %s", err)
	}
	rowsAffected, err := execResult.RowsAffected()
	if err != nil {
		return fmt.Errorf("deleting: rows affected: %s", err)
	}
	if rowsAffected == 0 {
		return RecordNotFoundError
	} else if rowsAffected != 1 {
		return fmt.Errorf("deleting: rows affected: %d", rowsAffected)
	}

	return nil
}

func setupTables(dbConnectionPool db) error {
	_, err := dbConnectionPool.Exec(schema)
	return err
}
