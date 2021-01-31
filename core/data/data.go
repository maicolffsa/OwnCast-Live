// This is a centralized place to connect to the database, and hold a reference to it.
// Other packages can share this reference.  This package would also be a place to add any kind of
// persistence-related convenience methods or migrations.

package data

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/owncast/owncast/config"
	"github.com/owncast/owncast/utils"
	log "github.com/sirupsen/logrus"
)

const (
	schemaVersion = 0
)

var _db *sql.DB

func GetDatabase() *sql.DB {
	return _db
}

func SetupPersistence() error {
	file := config.Config.DatabaseFilePath

	// Create empty DB file if it doesn't exist.
	if !utils.DoesFileExists(file) {
		log.Traceln("Creating new database at", file)

		_, err := os.Create(file)
		if err != nil {
			log.Fatal(err.Error())
		}
	}

	db, err := sql.Open("sqlite3", file)
	if err != nil {
		return err
	}

	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS config (
		"key" string NOT NULL PRIMARY KEY,
		"value" TEXT
	);`); err != nil {
		return err
	}

	var version int
	err = db.QueryRow("SELECT value FROM config WHERE key='version'").
		Scan(&version)
	if err != nil {
		if err != sql.ErrNoRows {
			return err
		}

		// fresh database: initialize it with the current schema version
		_, err := db.Exec("INSERT INTO config(key, value) VALUES(?, ?)", "version", schemaVersion)
		if err != nil {
			return err
		}
		version = schemaVersion
	}

	// is database from a newer Owncast version?
	if version > schemaVersion {
		return fmt.Errorf("incompatible database version %d (versions up to %d are supported)",
			version, schemaVersion)
	}

	// is database schema outdated?
	if version < schemaVersion {
		if err := migrateDatabase(db, version, schemaVersion); err != nil {
			return err
		}
	}

	_db = db
	return nil
}

func migrateDatabase(db *sql.DB, from, to int) error {
	log.Printf("Migrating database from version %d to %d\n", from, to)
	for v := from; v < to; v++ {
		switch v {
		case 0:
			log.Printf("Migration step from %d to %d\n", v, v+1)
		default:
			panic("missing database migration step")
		}
	}

	_, err := db.Exec("UPDATE config SET value = ? WHERE key = ?", to, "version")
	if err != nil {
		return err
	}

	return nil
}
