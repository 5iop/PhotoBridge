package database

import (
	"log"
	"os"
	"path/filepath"

	"photobridge/config"
	"photobridge/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

const shortname = "[Database]"

func Init() {
	var err error

	// Ensure data directory exists
	dir := filepath.Dir(config.AppConfig.DatabasePath)
	log.Printf("%s Creating database directory: %s", shortname, dir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Fatalf("%s Failed to create database directory %s: %v", shortname, dir, err)
	}
	log.Printf("%s Database directory created/verified: %s", shortname, dir)

	// Check if database file exists
	dbExists := false
	if _, err := os.Stat(config.AppConfig.DatabasePath); err == nil {
		dbExists = true
		log.Printf("%s Database file exists: %s", shortname, config.AppConfig.DatabasePath)

		// Check if database file is writable
		if f, err := os.OpenFile(config.AppConfig.DatabasePath, os.O_RDWR, 0644); err != nil {
			log.Printf("%s Database file is not writable, removing: %v", shortname, err)
			if err := os.Remove(config.AppConfig.DatabasePath); err != nil {
				log.Fatalf("%s Failed to remove readonly database file: %v", shortname, err)
			}
			dbExists = false
			log.Printf("%s Readonly database file removed", shortname)
		} else {
			f.Close()
		}
	} else if !os.IsNotExist(err) {
		log.Fatalf("%s Failed to check database file: %v", shortname, err)
	}

	if !dbExists {
		log.Printf("%s Database file does not exist, will be created: %s", shortname, config.AppConfig.DatabasePath)
	}

	log.Printf("%s Connecting to database: %s", shortname, config.AppConfig.DatabasePath)
	DB, err = gorm.Open(sqlite.Open(config.AppConfig.DatabasePath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatalf("%s Failed to connect to database %s: %v", shortname, config.AppConfig.DatabasePath, err)
	}
	log.Printf("%s Database connection established", shortname)

	// Auto migrate models
	log.Printf("%s Running database migrations", shortname)
	err = DB.AutoMigrate(
		&models.Project{},
		&models.Photo{},
		&models.ShareLink{},
		&models.PhotoExclusion{},
	)
	if err != nil {
		log.Fatalf("%s Failed to migrate database: %v", shortname, err)
	}

	log.Printf("%s Database initialized successfully", shortname)
}
