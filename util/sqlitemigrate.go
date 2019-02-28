package util

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/pressly/goose"
)

func SqliteConnect(connectionString string) (*sqlx.DB, error) {
	var db *sqlx.DB
	var err error

	db, err = sqlx.Connect("sqlite3", connectionString)

	if err != nil {
		if db != nil {
			db.Close()
		}
		log.Println("Warning in sqlite: ", err)
	}

	return db, err
}

func SqliteMigrateUp(connectionString string, migrateDir string) error {
	goose.SetDialect("sqlite3")
	db, err := SqliteConnect(connectionString)
	if err != nil {
		log.Printf("Failed to connect to sqlite with connectionString: %s \n %v", connectionString, err)
		return err
	}

	return goose.Up(db.DB, migrateDir)
}
func SqliteMigrateDown(connectionString string, migrateDir string) error {
	goose.SetDialect("sqlite3")
	db, err := SqliteConnect(connectionString)
	if err != nil {
		return err
	}

	return goose.Down(db.DB, migrateDir)
}

func SqliteMigrateUpTo(connectionString string, migrateDir string, toVersion int64) error {
	goose.SetDialect("sqlite3")
	db, err := SqliteConnect(connectionString)
	if err != nil {
		return err
	}

	return goose.UpTo(db.DB, migrateDir, toVersion)
}

func SqliteMigrateDownTo(connectionString string, migrateDir string, toVersion int64) error {
	goose.SetDialect("sqlite3")
	db, err := SqliteConnect(connectionString)
	if err != nil {
		return err
	}

	return goose.DownTo(db.DB, migrateDir, toVersion)
}
