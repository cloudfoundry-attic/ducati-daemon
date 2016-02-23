package db

import (
	"database/sql"

	_ "github.com/lib/pq"
)

func OpenPool(databaseConfig string) (*sql.DB, error) {
	dbConn, err := sql.Open("postgres", databaseConfig)
	if err != nil {
		return nil, err
	}

	return dbConn, nil
}
