package db

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

func GetConnectionPool(databaseConfig string) (*sql.DB, error) {
	dbConn, err := sql.Open("postgres", databaseConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to open database connection: %s", err)
	}

	if err = dbConn.Ping(); err != nil {
		return nil, fmt.Errorf("unable to ping %s", err)
	}

	return dbConn, nil
}
