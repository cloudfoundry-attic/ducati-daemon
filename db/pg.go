package db

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func GetConnectionPool(databaseConfig string) (*sqlx.DB, error) {
	dbConn, err := sqlx.Open("postgres", databaseConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to open database connection: %s", err)
	}

	if err = dbConn.Ping(); err != nil {
		return nil, fmt.Errorf("unable to ping %s", err)
	}

	return dbConn, nil
}
