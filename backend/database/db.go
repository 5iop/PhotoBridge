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
	if _, err := os.Stat(config.AppConfig.DatabasePath); os.IsNotExist(err) {
		log.Printf("%s Database file does not exist, will be created: %s", shortname, config.AppConfig.DatabasePath)
	} else if err != nil {
		log.Fatalf("%s Failed to check database file: %v", shortname, err)
	} else {
		log.Printf("%s Database file exists: %s", shortname, config.AppConfig.DatabasePath)
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
