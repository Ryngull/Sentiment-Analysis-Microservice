package database

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"runtime"

	"gateway/models"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// DB is the shared GORM handle initialized by InitDB.
var DB *gorm.DB

// InitDB loads database configuration, opens PostgreSQL, and migrates the schema.
func InitDB() error {
	// Resolve the optional local env file independently of the process working directory.
	_, b, _, _ := runtime.Caller(0)
	envPath := filepath.Join(filepath.Dir(b), "..", "inf.env")

	// Existing process variables take precedence over values in the local file.
	_ = godotenv.Load(envPath)

	dbUser, err := requiredEnv("DB_USER")
	if err != nil {
		return err
	}
	dbPassword, err := requiredEnv("DB_PASSWORD")
	if err != nil {
		return err
	}
	dbName, err := requiredEnv("DB_NAME")
	if err != nil {
		return err
	}
	dbHost, err := requiredEnv("DB_HOST")
	if err != nil {
		return err
	}
	dbPort, err := requiredEnv("DB_PORT")
	if err != nil {
		return err
	}

	query := url.Values{}
	query.Set("sslmode", envOrDefault("DB_SSLMODE", "disable"))
	query.Set("TimeZone", envOrDefault("DB_TIMEZONE", "Asia/Tokyo"))
	dsn := (&url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(dbUser, dbPassword),
		Host:     net.JoinHostPort(dbHost, dbPort),
		Path:     dbName,
		RawQuery: query.Encode(),
	}).String()

	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	fmt.Printf("Connected to PostgreSQL at %s:%s/%s\n", dbHost, dbPort, dbName)

	err = DB.AutoMigrate(&models.User{}, &models.AnalysisRecord{})
	if err != nil {
		return fmt.Errorf("failed to migrate database schema: %w", err)
	}

	return nil
}

func requiredEnv(key string) (string, error) {
	value := os.Getenv(key)
	if value == "" {
		return "", fmt.Errorf("required environment variable %s is not set", key)
	}
	return value, nil
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
