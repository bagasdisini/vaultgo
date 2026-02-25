package config

import (
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/joho/godotenv"
)

type Config struct {
	Port string

	MongoURI string
	DBName   string
}

var envOnce sync.Once

func Load() *Config {
	envOnce.Do(findRootEnv)

	return &Config{
		Port: getEnv("PORT", "8080"),

		MongoURI: getEnv("MONGO_URI", "mongodb://localhost:27017/?replicaSet=rs0"),
		DBName:   getEnv("DB_NAME", "vaultgo"),
	}
}

func findRootEnv() {
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal("Could not get working directory:", err)
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			_ = godotenv.Load(filepath.Join(dir, ".env"))
			return
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return
		}
		dir = parent
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
