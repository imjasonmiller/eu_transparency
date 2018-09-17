package main

import (
	"os"
	"path"
	"reflect"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	t.Run("wrong path", func(t *testing.T) {
		_, err := loadConfig(path.Join("fixtures", "config", "null.env"))
		if err == nil {
			t.Error("file not found, but no error was returned")
		}
	})

	t.Run("keys and values", func(t *testing.T) {
		tests := map[string]struct {
			path     string
			expected string
		}{
			"missing user key": {path.Join("fixtures", "config", "nouserkey.env"), "environment variable DB_USER is required"},
			"missing user val": {path.Join("fixtures", "config", "nouserval.env"), "environment variable DB_USER is required"},
			"missing pass key": {path.Join("fixtures", "config", "nopasskey.env"), "environment variable DB_PASS is required"},
			"missing host key": {path.Join("fixtures", "config", "nohostkey.env"), "environment variable DB_HOST is required"},
			"missing host val": {path.Join("fixtures", "config", "nohostval.env"), "environment variable DB_HOST is required"},
			"missing port key": {path.Join("fixtures", "config", "noportkey.env"), "environment variable DB_PORT is required"},
			"missing port val": {path.Join("fixtures", "config", "noportval.env"), "environment variable DB_PORT is required"},
		}

		for name, test := range tests {
			// Unset env before test
			os.Clearenv()

			t.Run(name, func(t *testing.T) {
				_, err := loadConfig(test.path)
				if err != nil && err.Error() != test.expected {
					t.Errorf("expected %s to be %s, got %s", test.path, test.expected, err.Error())
				}
			})
		}
	})

	t.Run("pass input", func(t *testing.T) {
		tests := map[string]struct {
			path     string
			expected map[string]string
		}{
			"missing pass val": {path.Join("fixtures", "config", "nopassval.env"), map[string]string{
				"DB_USER": "postgres",
				"DB_PASS": "",
				"DB_HOST": "localhost",
				"DB_PORT": "5432",
			}},
			"found pass val": {path.Join("fixtures", "config", ".env"), map[string]string{
				"DB_USER": "postgres",
				"DB_PASS": "89asd76034Xs!3$",
				"DB_HOST": "localhost",
				"DB_PORT": "5432",
			}},
		}

		for name, test := range tests {
			// Unset env before test
			os.Clearenv()

			t.Run(name, func(t *testing.T) {
				output, _ := loadConfig(test.path)

				if !reflect.DeepEqual(output, test.expected) {
					t.Errorf("expected %s to be %+v, got %+v", test.path, test.expected, output)
				}
			})
		}
	})
}
