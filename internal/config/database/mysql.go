package database

import (
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var dbmysql *gorm.DB

func DatabaseMysql() *gorm.DB {
	// Try to load .env file, but don't fail if it doesn't exist (for production)
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Try Railway MySQL variables first, fallback to individual variables
	mysqlURL := os.Getenv("MYSQL_URL")
	log.Println("MYSQL_URL:", mysqlURL)

	var dsn string

	if mysqlURL != "" && !strings.Contains(mysqlURL, "switchyard.proxy.rlwy.net") {
		// Use Railway MySQL URL only if it's not the external proxy
		dsn = mysqlURL
		log.Println("Using Railway MySQL URL")
	} else {
		// Fallback to individual variables
		dbHost := os.Getenv("MYSQL_HOST")
		if dbHost == "" {
			dbHost = os.Getenv("DB_HOST")
		}
		if dbHost == "" {
			dbHost = "localhost"
		}

		dbUser := os.Getenv("MYSQL_USER")
		if dbUser == "" {
			dbUser = os.Getenv("DB_USER")
		}

		dbPassword := os.Getenv("MYSQL_PASSWORD")
		if dbPassword == "" {
			dbPassword = os.Getenv("DB_PASSWORD")
		}

		dbName := os.Getenv("MYSQL_DATABASE")
		if dbName == "" {
			dbName = os.Getenv("DB_NAME")
		}
		if dbName == "" {
			dbName = "iqgncnzy_skripsi" // Default to your imported schema
		}

		dbPort := os.Getenv("MYSQL_PORT")
		if dbPort == "" {
			dbPort = os.Getenv("DB_PORT")
		}
		if dbPort == "" {
			dbPort = "3306"
		}

		// Create DSN from individual variables
		dsn = dbUser + ":" + dbPassword + "@tcp(" + dbHost + ":" + dbPort + ")/" + dbName + "?charset=utf8mb4&parseTime=True&loc=Local"
		log.Println("Using individual variables - Host:", dbHost, "Port:", dbPort, "Database:", dbName)
		log.Println("Connecting to database:", dbHost+":"+dbPort)
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
    DisableForeignKeyConstraintWhenMigrating: true,
	})

	if err != nil {
		panic("failed to connect database")
	}

	// Migrate the schema
	// db.AutoMigrate(&User{}, &Product{})

	dbmysql = db

	return db
}

func GetDB() *gorm.DB {
	if dbmysql == nil {
		log.Println("Reload Connection", dbmysql)
		DatabaseMysql()
	}
	return dbmysql
}
