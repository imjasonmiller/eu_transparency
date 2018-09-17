package main

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

func loadConfig(path string) (map[string]string, error) {
	keys := []string{"DB_USER", "DB_PASS", "DB_HOST", "DB_PORT"}
	cfg := map[string]string{}

	err := godotenv.Load(path)
	if err != nil {
		return cfg, fmt.Errorf("could not load .env")
	}

	for _, key := range keys {
		val := os.Getenv(key)

		// All values, besides the password, should be present.
		if val == "" && key != "DB_PASS" {
			return cfg, fmt.Errorf("environment variable %s is required", key)
		}

		cfg[key] = val
	}

	return cfg, nil
}
