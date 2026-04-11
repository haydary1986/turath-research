package main

import (
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func newMigrate(dbURL string) (*migrate.Migrate, error) {
	return migrate.New("file:///migrations", dbURL)
}
