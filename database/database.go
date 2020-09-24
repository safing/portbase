package database

import (
	"errors"
	"time"
)

// Database holds information about registered databases
type Database struct {
	Name         string
	Description  string
	StorageType  string
	ShadowDelete bool // Whether deleted records should be kept until purged.
	Registered   time.Time
	LastUpdated  time.Time
	LastLoaded   time.Time
}

// MigrateTo migrates the database to another storage type.
func (db *Database) MigrateTo(newStorageType string) error {
	return errors.New("not implemented yet") // TODO
}

// Loaded updates the LastLoaded timestamp.
func (db *Database) Loaded() {
	db.LastLoaded = time.Now().Round(time.Second)
}

// Updated updates the LastUpdated timestamp.
func (db *Database) Updated() {
	db.LastUpdated = time.Now().Round(time.Second)
}
